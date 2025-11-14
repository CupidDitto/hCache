package services

import (
	"container/heap"
	"context"
	"errors"
	"main/common"
	"main/distkv"
	"main/io"
	"main/rpc/cache"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// the real cache in user space
// key: imgidx; value: pointer to Node
type importance_cache struct {
	sync.RWMutex

	Cache_size   int64 `json:"cache_size"`   // cache size
	Full         bool  `json:"full"`         // cache is full
	Cache_nitems int64 `json:"cache_nitems"` // cache n items

	cache map[int64]cacheEntry // hash map to find an element quickly

	Imgidx2mainimportance   *map[int64]float64 `json:"-"` // cur using old iv
	Imgidx2shadowimportance *map[int64]float64 `json:"-"` // new iv for next epoch

	Imgidx2mainfrequency   *map[int64]int64 `json:"-"` // cur using old frequency
	Imgidx2shadowfrequency *map[int64]int64 `json:"-"` // new frequency for next epoch

	mainheap   *indexheap // main index heap
	shadowheap *indexheap // shadow index heap

	building_chanel          chan task
	Switching_shadow_channel chan int `json:"-"`
	// building_shadow_mutex sync.Mutex
}

func init_importance_cache(semantic *Semantic_cache_mng) *importance_cache {
	var icache importance_cache
	icache.Cache_size = semantic.Importance_cache_size
	icache.Full = false
	icache.Cache_nitems = 0

	icache.cache = make(map[int64]cacheEntry)

	// initialization
	tm := make(map[int64]float64)
	tm2 := make(map[int64]float64)
	icache.Imgidx2mainimportance = &tm
	icache.Imgidx2shadowimportance = &tm2

	// initialization for frequency maps
	tf := make(map[int64]int64)
	tf2 := make(map[int64]int64)
	icache.Imgidx2mainfrequency = &tf
	icache.Imgidx2shadowfrequency = &tf2

	// init main heaps
	icache.mainheap = &indexheap{Imgidx2importance: icache.Imgidx2mainimportance, Imgidx2frequency: icache.Imgidx2mainfrequency}
	heap.Init(icache.mainheap)

	icache.shadowheap = &indexheap{Imgidx2importance: icache.Imgidx2shadowimportance, Imgidx2frequency: icache.Imgidx2shadowfrequency}
	heap.Init(icache.shadowheap)

	// icache.building_shadow = false
	icache.Switching_shadow_channel = make(chan int)
	icache.building_chanel = make(chan task, common.Config.Async_building_task) // DCRuntime.Resample_interval)
	go icache.async_build_shadow_heap_worker()

	log.Debugf("[CACHE IMPORTANCE] init_importance_cache has complete")
	return &icache
}

func (icache *importance_cache) Exists(imgidx int64) bool {
	icache.RLock()
	defer icache.RUnlock()

	if len(icache.cache) != int(icache.Cache_size) {
		log.Errorf("icache.cache len if not[%v], but [%v]", int(icache.Cache_size), len(icache.cache))
	}

	_, ok := icache.cache[imgidx]
	if ok {
		return true
	} else {
		return false
	}
}

func (icache *importance_cache) Access(imgidx int64) ([]byte, error) {

	icache.RLock()
	defer icache.RUnlock()

	cache_entry, exist := icache.cache[imgidx]
	if exist && (cache_entry.imgcontent != nil) {
		return cache_entry.imgcontent, nil
	} else {
		return nil, errors.New("not exist")
	}
}

func (icache *importance_cache) Insert(imgidx int64, imgcontent []byte) error {
	DCRuntime.Semantic_related_metadata.RLock()
	importance := DCRuntime.Semantic_related_metadata.Ivpersample[imgidx]
	frequency := DCRuntime.Semantic_related_metadata.Freqpersample[imgidx]
	DCRuntime.Semantic_related_metadata.RUnlock()
	// log.Info("release importance_related_metadata rlock")
	icache.Lock()
	// log.Info("got lock")
	defer icache.Unlock()

	if !icache.Full {
		log.Debugf("[IMPORTANCE_CACHE_MNG] cache is NOT full")
		// push data to main_help
		(*icache.mainheap.Imgidx2importance)[imgidx] = importance
		(*icache.mainheap.Imgidx2frequency)[imgidx] = frequency
		icache.cache[imgidx] = cacheEntry{imgcontent: imgcontent}
		heap.Push(icache.mainheap, heapNode{Imgidx: imgidx})

		if err := distkv.GlobalKv.Put(imgidx, distkv.Curaddr); err != nil {
			log.Fatal("[importance-design1-cachemng.go]", err.Error())
		} else {
			log.Infof("[importance-design1-cachemng.go] inserted imgidx %v to server node %v", imgidx, distkv.Curaddr)
		}

		icache.Sent_async_build_shadow_heap_task(By_Keyvalue_update, nil, 0, imgidx, importance, frequency)

		log.Debugf("[IMPORTANCE_CACHE_MNG] push new heapNodes into heaps")

		// update meta-data
		icache.Cache_nitems += 1
		icache.Full = (icache.Cache_nitems >= icache.Cache_size)
	} else {
		log.Debugf("[IMPORTANCE_CACHE_MNG] cache is full")
		// compare with first element (minimum in heap) using dual sorting (importance, frequency)
		// to decide whether to cache into mainHeap or not
		minImgidx := (*icache.mainheap).nodes[0].Imgidx
		minImportance := (*(*icache.mainheap).Imgidx2importance)[minImgidx]
		minFrequency := (*(*icache.mainheap).Imgidx2frequency)[minImgidx]

		// Dual sorting comparison: first by importance, then by frequency
		// New sample should replace if: importance > minImportance, or
		// (importance == minImportance && frequency > minFrequency)
		shouldReplace := false
		if importance > minImportance {
			shouldReplace = true
		} else if importance == minImportance && frequency > minFrequency {
			shouldReplace = true
		}

		if shouldReplace {
			log.Debug("[IMPORTANCE_CACHE_MNG] insert into heap")
			// pop min node from mainHeap and insert new node to mainHeap
			tmpheapNode := heap.Pop(icache.mainheap).(heapNode)
			delete(icache.cache, tmpheapNode.Imgidx) // GC take care of heapNode in mainHeap
			delete(*icache.Imgidx2mainimportance, tmpheapNode.Imgidx)
			delete(*icache.Imgidx2mainfrequency, tmpheapNode.Imgidx)

			if err := distkv.GlobalKv.Del(tmpheapNode.Imgidx); err != nil {
				log.Fatal("[importance-design1-cachemng.go]", err.Error())
			} else {
				log.Infof("['importance-design1-cachemng.go] deleted %v from server node %v", tmpheapNode.Imgidx, distkv.Curaddr)
			}

			(*icache.mainheap.Imgidx2importance)[imgidx] = importance
			(*icache.mainheap.Imgidx2frequency)[imgidx] = frequency
			icache.cache[imgidx] = cacheEntry{imgcontent: imgcontent}
			heap.Push(icache.mainheap, heapNode{Imgidx: imgidx})

			if err := distkv.GlobalKv.Put(imgidx, distkv.Curaddr); err != nil {
				log.Fatal("[importance-design1-cachemng.go]", err.Error())
			} else {
				log.Infof("[importance-design1-cachemng.go] inserted imgidx %v to server node %v", imgidx, distkv.Curaddr)
			}

			icache.Sent_async_build_shadow_heap_task(By_keyvalue_delete_and_update, nil, tmpheapNode.Imgidx, imgidx, importance, frequency)
		} else {
			log.Debug("[IMPORTANCE_CACHE_MNG] DO NOT insert into heap")
		}
	}
	return nil
}

func (icache *importance_cache) async_build_shadow_heap_worker() {
	for build_task := range icache.building_chanel {
		log.Debug("[IMPORTANCE_CACHE_MNG] async rebuilding shadow heap: ", build_task)

		switch build_task.op_type {
		case By_Map:
		case By_Keyvalue_update:
			// add new value to Imgidx2shadowimportance and Imgidx2shadowfrequency
			(*icache.Imgidx2shadowimportance)[build_task.imgidx] = build_task.impvalue
			(*icache.Imgidx2shadowfrequency)[build_task.imgidx] = build_task.freqvalue
		case By_keyvalue_delete_and_update:
			// delete old value and add new value to Imgidx2shadowimportance and Imgidx2shadowfrequency
			delete(*icache.Imgidx2shadowimportance, build_task.oldimgidx)
			delete(*icache.Imgidx2shadowfrequency, build_task.oldimgidx)
			(*icache.Imgidx2shadowimportance)[build_task.imgidx] = build_task.impvalue
			(*icache.Imgidx2shadowfrequency)[build_task.imgidx] = build_task.freqvalue
		case SWITCH_HEAP:
			// rebuild
			log.Debug("[IMPORTANCE_CACHE_MNG] rebuild")
			for k := range *icache.Imgidx2shadowimportance {
				(*icache.Imgidx2shadowimportance)[k] = DCRuntime.Semantic_related_metadata.New_Ivpersample[k]
				(*icache.Imgidx2shadowfrequency)[k] = DCRuntime.Semantic_related_metadata.New_Freqpersample[k]
			}
			shadowheap := &indexheap{
				Imgidx2importance: icache.Imgidx2shadowimportance,
				Imgidx2frequency:  icache.Imgidx2shadowfrequency,
			}
			heap.Init(shadowheap)

			for imgidx := range *icache.Imgidx2shadowimportance {
				heap.Push(shadowheap, heapNode{Imgidx: imgidx})
			}
			icache.shadowheap = shadowheap
			log.Warn("[IMPORTANCE_CACHE_MNG-ASYNC] switch heap!")
			icache.Switch_to_ShadowHeap()
			icache.Switching_shadow_channel <- 0
			continue
		default:
			log.Error("[IMPORTANCE_CACHE_MNG] rebuilding shadow heap - Operation does not exist", build_task)
		}
	}
}

func (icache *importance_cache) Sent_async_build_shadow_heap_task(op_type task_type, updatemap map[int64]float64, oldimgidx int64, imgidx int64, impvalue float64, freqvalue int64) {
	build_task := task{op_type: op_type, updatemap: updatemap, oldimgidx: oldimgidx, imgidx: imgidx, impvalue: impvalue, freqvalue: freqvalue}
	icache.building_chanel <- build_task
}

func (icache *importance_cache) Switch_to_ShadowHeap() error {
	for k := range *icache.Imgidx2shadowimportance {
		_, ok := icache.cache[k]
		if !ok {
			log.Infof("[IMPORTANCE_CACHE_MNG] debug%v %v", k, ok)
		}
	}
	tidx2importance := make(map[int64]float64)
	for idx, impval := range *icache.Imgidx2shadowimportance {
		tidx2importance[idx] = impval
	}
	tidx2frequency := make(map[int64]int64)
	for idx, freqval := range *icache.Imgidx2shadowfrequency {
		tidx2frequency[idx] = freqval
	}
	icache.Imgidx2mainimportance = &tidx2importance
	icache.Imgidx2mainfrequency = &tidx2frequency
	icache.mainheap.nodes = icache.shadowheap.nodes
	icache.mainheap.Imgidx2importance = icache.Imgidx2mainimportance
	icache.mainheap.Imgidx2frequency = icache.Imgidx2mainfrequency
	return nil
}

func (icache *importance_cache) Is_Full() bool {

	icache.RLock()
	defer icache.RUnlock()
	return icache.Full
}

func (icache *importance_cache) AccessAtOnce(imgidx int64) ([]byte, int64, bool) {

	icache.RLock()

	cache_entry, ok := icache.cache[imgidx]
	icache.RUnlock()
	if ok {
		// start := time.Now()
		imgcontent := cache_entry.imgcontent
		// elapsed := time.Since(start)
		// log.Printf("time read a file from local-cache [%v]:", elapsed)
		return imgcontent, DCRuntime.Imgidx2clsidx[imgidx], true
	} else {
		var imgcontent []byte
		if ip, _ := distkv.GlobalKv.Get(imgidx); ip != "" {
			log.Info("[importance-design1-cachemng.go] ip:", ip)
			// start := time.Now()
			ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
			defer cancel()
			ret, err := distkv.GrpcClients[ip].DCSubmit(ctx, &cache.DCRequest{Type: cache.OpType_readimg_byidx_t, Imgidx: imgidx, FullAccess: false})
			if err != nil {
				log.Info("[importance-design1-cachemng.go] Failed to get data from remote cache server (because recently deleted): ", ip, ". ", err.Error())
				imgpath := DCRuntime.Imgidx2imgpath[imgidx]
				imgcontent = io.Cache_from_filesystem(imgpath)
			} else {
				imgcontent = ret.GetData()
				// elapsed := time.Since(start)
				// log.Printf("time read a file from remote peer [%v]:", elapsed)
				clsidx := DCRuntime.Imgidx2clsidx[imgidx]
				return imgcontent, clsidx, false
			}
		} else {
			imgpath := DCRuntime.Imgidx2imgpath[imgidx]
			imgcontent = io.Cache_from_filesystem(imgpath)
		}
		// imgpath := DCRuntime.Imgidx2imgpath[imgidx]
		// imgcontent := io.Cache_from_filesystem(imgpath)
		// insert into cache
		icache.Insert(imgidx, imgcontent)
		clsidx := DCRuntime.Imgidx2clsidx[imgidx]
		return imgcontent, clsidx, false
	}
}

func (icache *importance_cache) AccessAtOnceIcache(imgidx int64) ([]byte, int64, bool) {
	//
	icache.RLock()
	cache_entry, ok := icache.cache[imgidx]
	icache.RUnlock()
	if ok {
		return cache_entry.imgcontent, DCRuntime.Imgidx2clsidx[imgidx], true
	} else {
		var imgcontent []byte
		if ip, _ := distkv.GlobalKv.Get(imgidx); ip != "" {
			log.Info("[importance-design1-cachemng.go] ip:", ip)
			// start := time.Now()
			ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
			defer cancel()
			ret, err := distkv.GrpcClients[ip].DCSubmit(ctx, &cache.DCRequest{Type: cache.OpType_readimg_byidx_t, Imgidx: imgidx, FullAccess: false})
			if err != nil {
				log.Info("[importance-design1-cachemng.go] Failed to get data from remote cache server (because recently deleted): ", ip, ". ", err.Error())
				imgpath := DCRuntime.Imgidx2imgpath[imgidx]
				imgcontent = io.Cache_from_filesystem(imgpath)
			} else {
				imgcontent = ret.GetData()
				// elapsed := time.Since(start)
				// log.Printf("time read a file from remote peer: [%v]", elapsed)
				clsidx := DCRuntime.Imgidx2clsidx[imgidx]
				return imgcontent, clsidx, false
			}
		} else {
			imgpath := DCRuntime.Imgidx2imgpath[imgidx]
			imgcontent = io.Cache_from_filesystem(imgpath)
		}
		// imgpath := DCRuntime.Imgidx2imgpath[imgidx]
		// imgcontent := io.Cache_from_filesystem(imgpath)
		if len(icache.cache) < int(icache.Cache_size) {
			// log.Info("start insert")
			icache.Insert(imgidx, imgcontent)
		}
		clsidx := DCRuntime.Imgidx2clsidx[imgidx]
		return imgcontent, clsidx, false
	}
}

// GetMinMetadata returns the minimum importance and frequency in the main heap
// Used by semantic-cachemng for admission control decisions
func (icache *importance_cache) GetMinMetadata() (float64, int64) {
	icache.RLock()
	defer icache.RUnlock()

	if len(icache.mainheap.nodes) == 0 {
		// Empty heap: return max values (easy to beat for admission)
		return 0.0, 0
	}

	// Min element is always at index 0 in min-heap
	minNode := icache.mainheap.nodes[0]
	minImgidx := minNode.Imgidx

	minImportance := (*icache.mainheap.Imgidx2importance)[minImgidx]
	minFrequency := (*icache.mainheap.Imgidx2frequency)[minImgidx]

	return minImportance, minFrequency
}
