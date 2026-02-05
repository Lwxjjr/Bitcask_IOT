package index

import "sync"

type Index struct {
	mu sync.RWMutex
	// 核心映射表：SensorName (string) -> Series对象 (指针)
	seriesMap map[string]*Series
	nextID    uint32
}

func NewIndex() *Index {
	return &Index{
		seriesMap: make(map[string]*Series),
		nextID:    1,
	}
}

// GetOrCreateSeries 是对外暴露的核心方法
// 逻辑：有就直接返回，没有就创建新的
func (idx *Index) GetOrCreateSeries(name string) *Series {
	// 1. 【快速路径】：先用读锁查一下有没有
	// 99.9% 的请求都会走这里，性能极高
	idx.mu.RLock()
	s, ok := idx.seriesMap[name]
	idx.mu.RUnlock()
	if ok {
		return s
	}

	// 2. 【慢速路径】：没找到，说明是新设备，准备注册
	// 加写锁，互斥
	idx.mu.Lock()
	defer idx.mu.Unlock()

	// Double Check: 防止刚才那一瞬间别的线程已经创建了
	if s, ok = idx.seriesMap[name]; ok {
		return s
	}

	// 3. 创建新 Series
	// 分配 ID -> 创建对象 -> 存入 Map
	id := idx.nextID
	idx.nextID++ // 计数器自增

	newSeries := NewSeries(id)
	idx.seriesMap[name] = newSeries

	return newSeries
}
