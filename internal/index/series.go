package index

import (
	"sync"

	"github.com/bitcask-iot/engine/internal/storage"
)

// Series 代表一个传感器的时间线
// 它负责管理热数据（Buffer）和冷数据索引（Blocks）
type Series struct {
	mu           sync.RWMutex
	ID           uint32
	ActiveBuffer []storage.Point
	Blocks       []*storage.BlockMeta
}

func NewSeries(ID uint32) *Series {
	return &Series{
		ID:           ID,
		ActiveBuffer: make([]storage.Point, 0, 128),
		Blocks:       make([]*storage.BlockMeta, 0),
	}
}

func (s *Series) Append(point storage.Point) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ActiveBuffer = append(s.ActiveBuffer, point)
}

// ShouldFlush 判断是否需要将内存数据刷入磁盘
// 这里的阈值可以根据配置文件动态调整，目前硬编码为 1000
func (s *Series) ShouldFlush() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.ActiveBuffer) >= 1000
}

// Flush 将内存中的 ActiveBuffer 写入指定的 Segment，并清空缓冲区
func (s *Series) Flush(seg *storage.Segment) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if len(s.ActiveBuffer) == 0 {
		return nil
	}

	// 1. 打包成 Block
	block := &storage.Block{
		SensorID: s.ID,
		Points:   s.ActiveBuffer,
	}

	// 2. 写入磁盘
	meta, err := seg.WriteBlock(block)
	if err != nil {
		return err
	}

	// 3. 更新索引：将新的 BlockMeta 加入列表
	s.Blocks = append(s.Blocks, meta)

	// 4. 重置缓冲区（复用内存空间）
	s.ActiveBuffer = s.ActiveBuffer[:0]

	return nil
}
