package core

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

	newSeries := newSeries(id)
	idx.seriesMap[name] = newSeries

	return newSeries
}

// GetAllSeries 获取所有 Series 的快照列表
// 场景：供 Engine 的后台 Ticker 巡检使用
func (idx *Index) GetAllSeries() []*Series {
	// 1. 加读锁 (RLock)
	// 我们只读 map，不修改 map 结构，所以用 RLock，允许其他协程并发读取
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	// 2. 预分配切片容量 (Performance Tip)
	// 我们已知 map 的长度，直接申请好内存，避免 append 时的多次扩容
	list := make([]*Series, 0, len(idx.seriesMap))

	// 3. 快速拷贝指针 (Snapshot)
	// 注意：这里只拷贝 Series 的指针，速度极快（纳秒级）
	// 我们不在这里做任何耗时的逻辑，以免阻塞写锁（影响 CreateSeries）
	for _, s := range idx.seriesMap {
		list = append(list, s)
	}

	// 4. 返回快照
	// 锁在这里释放。Engine 拿到 list 后，可以在锁外慢慢遍历，
	// 此时如果有新设备注册（idx.series 写入），完全不受影响。
	return list
}
