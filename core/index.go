package core

import (
	"encoding/binary"
	"os"
	"sync"
)

type Index struct {
	mu sync.RWMutex
	// 核心映射表：SensorName (string) -> Series对象 (指针)
	seriesMap map[string]*Series
	idToName  map[uint32]string // 反向映射，开机的时候有用
	nextID    uint32
	// ➕ 新增：字典日志文件句柄
	catalogFd *os.File
}

func NewIndex() *Index {
	return &Index{
		seriesMap: make(map[string]*Series),
		idToName:  make(map[uint32]string),
		nextID:    1,
	}
}

// GetOrCreateSeries 是对外暴露的核心方法
// 逻辑：有就直接返回，没有就创建新的
func (idx *Index) getOrCreateSeries(name string) *Series {
	// 1. 【快速路径】：先用读锁查一下有没有
	// 99.9% 的请求都会走这里，性能极高
	idx.mu.RLock()
	s, ok := idx.seriesMap[name]
	idx.mu.RUnlock()
	if ok {
		return s
	}

	// 2. 【慢速路径】：没找到，准备注册
	idx.mu.Lock()
	defer idx.mu.Unlock()

	if s, ok = idx.seriesMap[name]; ok {
		return s
	}

	// 3. 分配 ID
	id := idx.nextID
	idx.nextID++

	// 🌟 4. 【核心新增】：立刻把 "ID -> Name" 追加到字典文件中！
	// 格式极其简单：[ID: 4字节] + [Name长度: 2字节] + [Name内容]
	if err := idx.appendCatalog(id, name); err != nil {
		// 记录严重错误，注册失败！
	}

	// 5. 创建新 Series 并存入 Map
	newSeries := newSeries(id)
	idx.seriesMap[name] = newSeries
	idx.idToName[id] = name // 顺手记下反向映射

	return newSeries
}

// 追加写字典文件
func (idx *Index) appendCatalog(id uint32, name string) error {
	if idx.catalogFd == nil {
		return nil // 防御性逻辑
	}

	nameLen := len(name)
	buf := make([]byte, 4+2+nameLen)
	binary.BigEndian.PutUint32(buf[0:4], id)
	binary.BigEndian.PutUint16(buf[4:6], uint16(nameLen))
	copy(buf[6:], name)

	_, err := idx.catalogFd.Write(buf)
	return err
}

// GetAllSeries 获取所有 Series 的快照列表
// 场景：供 Engine 的后台 Ticker 巡检使用
func (idx *Index) getAllSeries() []*Series {
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

// GetAllKeys 获取所有 SensorID (Key) 的列表
func (idx *Index) getAllKeys() []string {
	idx.mu.RLock()
	defer idx.mu.RUnlock()

	keys := make([]string, 0, len(idx.seriesMap))
	for k := range idx.seriesMap {
		keys = append(keys, k)
	}
	return keys
}
