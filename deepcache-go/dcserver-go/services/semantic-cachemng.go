package services

import "sync"

type Semantic_cache_mng struct {
	sync.RWMutex

	Importance_cache_size int64 `json:"cache_size"` // cache size
	Similar_cache_size    int64 `json:"cache_size"` // cache size

	Importance_cache *importance_cache
	Similar_cache    *similar_cache

	Remote_hit int64 `json:"remote_hit_times"`
}

func init_semantic_cache_mng(dc *deepcachemodel) *Semantic_cache_mng {
	var semantic Semantic_cache_mng
	semantic.Importance_cache_size = dc.Cache_size
	semantic.Similar_cache_size = int64(float64(dc.Noimg) * dc.Us_ratio)

	semantic.Importance_cache = init_importance_cache(&semantic)
	semantic.Similar_cache = init_similar_cache(&semantic)

	semantic.Remote_hit = 0

	return &semantic
}

func (semantic *Semantic_cache_mng) Get_type() string {
	return "semantic"
}

func (semantic *Semantic_cache_mng) Exists(imgidx int64) bool {
	semantic.RLock()
	defer semantic.RUnlock()

	return semantic.Importance_cache.Exists(imgidx) || semantic.Similar_cache.Exists(imgidx)
}

func (semantic *Semantic_cache_mng) Access(imgidx int64) ([]byte, error) {
	semantic.RLock()
	defer semantic.RUnlock()

}

func (semantic *Semantic_cache_mng) Insert(imgidx int64, imgcontent []byte) error {
	semantic.Lock()
	defer semantic.Unlock()

}
