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
