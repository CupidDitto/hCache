package services

// design 2

import (
	"context"
	"errors"
	"main/distkv"
	"main/io"
	"main/rpc/cache"
	"sync"
	"time"

	log "github.com/sirupsen/logrus"
)

// define similar_cache
type similar_cache struct {
	sync.RWMutex
	cache map[int64]*Node       // cache map
	freq  map[int64]*DoubleList // freq map

	Cache_size   int64 `json:"cache_size"`   // capacity
	Full         bool  `json:"full"`         // cache is full
	Cache_nitems int64 `json:"cache_nitems"` // cache n items
	minFreq      int64 `json:"min Freq of all the items"`
}

func init_similar_cache(semantic *Semantic_cache_mng) *similar_cache {
	var similar similar_cache
	similar.Cache_size = semantic.Similar_cache_size
	similar.cache = make(map[int64]*Node)
	similar.Full = false
	similar.Cache_nitems = 0

	similar.freq = make(map[int64]*DoubleList)
	return &similar
}

func (similar *similar_cache) Exists(imgidx int64) bool {

	similar.Lock()
	defer similar.Unlock()

	// if exists, return True (NOT change meta-data, because only access/insert will change), otherwise return False.
	_, ok := similar.cache[imgidx]
	if ok {
		return true
	} else {
		return false
	}
}

func (similar *similar_cache) Access(imgidx int64) ([]byte, error) {
	// lock for cur write
	similar.Lock()
	defer similar.Unlock()

	// if in cache, return imgcontent and change metadata; if not in cache, raise keyerror exception;
	node, exist := similar.cache[imgidx]
	if exist {
		imgcontent := node.val
		similar.IncFreq(node)
		return imgcontent, nil
	} else {
		return nil, errors.New("not exist")
	}
}

func (similar *similar_cache) Insert(imgidx int64, imgcontent []byte) error {
	// lock for cur write
	similar.Lock()
	defer similar.Unlock()

	// insert an element into self.cache
	// if not full, insert it into the end
	// if full, insert it into root link node, and make the first link be new root()
	if similar.Full {
		del_node := similar.freq[similar.minFreq].RemoveLast()
		oldkey := del_node.key
		delete(similar.cache, oldkey)
		similar.Cache_nitems--

		if err := distkv.GlobalKv.Del(oldkey); err != nil {
			log.Fatal("[similar-cachemng.go]", err.Error())
		} else {
			log.Infof("[similar-cachmng.go] deleted %v from server node %v", oldkey, distkv.Curaddr)
		}

	}

	new_node := &Node{key: imgidx, val: imgcontent, freq: 1}
	similar.cache[imgidx] = new_node
	if similar.freq[1] == nil {
		similar.freq[1] = CreateDL()
	}
	similar.freq[1].AddFirst(new_node)
	similar.minFreq = 1
	similar.Cache_nitems++

	similar.Full = (similar.Cache_nitems >= similar.Cache_size)

	if err := distkv.GlobalKv.Put(imgidx, distkv.Curaddr); err != nil {
		log.Fatal("[similar-cachemng.go]", err.Error())
	} else {
		log.Infof("[similar-cachmng.go] inserted imgidx %v to server node %v", imgidx, distkv.Curaddr)
	}

	return nil
}

func (similar *similar_cache) Get_type() string {
	return "lfu"
}

func (similar *similar_cache) AccessAtOnce(imgidx int64) ([]byte, int64, bool) {
	// lock for cur write
	// start := time.Now()
	similar.RLock()
	node, exists := similar.cache[imgidx]
	similar.RUnlock()
	if exists {
		similar.Lock()
		imgcontent := node.val
		similar.IncFreq(node)
		// log.Infof("[%v] in cache, spent time [%v]", imgidx, time.Since(start))
		similar.Unlock()
		return imgcontent, DCRuntime.Imgidx2clsidx[imgidx], true
	} else {
		var imgcontent []byte
		if ip, _ := distkv.GlobalKv.Get(imgidx); ip != "" {
			log.Info("[similar-cachemng.go] ip:", ip)
			start := time.Now()
			ctx, cancel := context.WithTimeout(context.Background(), time.Hour)
			defer cancel()
			ret, err := distkv.GrpcClients[ip].DCSubmit(ctx, &cache.DCRequest{Type: cache.OpType_readimg_byidx, Imgidx: imgidx, FullAccess: false, Id: 1})
			if err != nil {
				log.Info("[similar-cachemng.go] Failed to get data from remote cache server (because recently deleted): ", ip, ". ", err.Error())
				start := time.Now()
				imgpath := DCRuntime.Imgidx2imgpath[imgidx]
				imgcontent = io.Cache_from_filesystem(imgpath)
				elapsed := time.Since(start)
				log.Infof("time read a file from file system: [%v]", elapsed)
				if len(imgcontent) == 0 {
					log.Error("0 read a NULL file from filesystem for imgidx [%v]", imgidx)
				}
			} else {
				imgcontent = ret.GetData()
				if len(imgcontent) == 0 {
					log.Info("read a null from remote peer cache")
					start := time.Now()
					imgpath := DCRuntime.Imgidx2imgpath[imgidx]
					imgcontent = io.Cache_from_filesystem(imgpath)
					elapsed := time.Since(start)
					log.Infof("time read a file from file system: [%v]", elapsed)
					if len(imgcontent) == 0 {
						log.Error("1 read a NULL file from filesystem for imgidx [%v]", imgidx)
					}
					clsidx := DCRuntime.Imgidx2clsidx[imgidx]
					return imgcontent, clsidx, false
				} else {
					elapsed := time.Since(start)
					log.Infof("time read a file from remote peer: [%v]", elapsed)
					clsidx := DCRuntime.Imgidx2clsidx[imgidx]
					return imgcontent, clsidx, false
				}
			}
		} else {
			start := time.Now()
			imgpath := DCRuntime.Imgidx2imgpath[imgidx]
			imgcontent = io.Cache_from_filesystem(imgpath)
			elapsed := time.Since(start)
			log.Infof("time read a file from file system: [%v]", elapsed)
			if len(imgcontent) == 0 {
				log.Error("2 read a NULL file from filesystem for imgidx [%v]", imgidx)
			}
			// log.Infof("[%v] not in cache, IO file spent time [%v]", imgidx, time.Since(start))
		}
		// imgpath := DCRuntime.Imgidx2imgpath[imgidx]
		// imgcontent = io.Cache_from_filesystem(imgpath)

		// insert into cache
		// start = time.Now()
		similar.Insert(imgidx, imgcontent)

		// log.Infof("[%v] not in cache, INSERT spent time [%v]", imgidx, time.Since(start))
		clsidx := DCRuntime.Imgidx2clsidx[imgidx]
		return imgcontent, clsidx, false
	}
}

func (similar *similar_cache) IncFreq(node *Node) {
	_freq := node.freq
	similar.freq[_freq].Remove(node)
	if similar.minFreq == _freq && similar.freq[_freq].IsEmpty() {
		similar.minFreq++
		delete(similar.freq, _freq)
	}
	node.freq++
	if similar.freq[node.freq] == nil {
		similar.freq[node.freq] = CreateDL()
	}
	similar.freq[node.freq].AddFirst(node)
}
