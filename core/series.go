package core

import (
	"sync"
	"time"
)

// 阈值配置
const (
	BlockMaxPoints     = 1000             // 触发刷盘的数量阈值
	ForceFlushInterval = 60 * time.Second // 触发强制刷盘的时间阈值
)

// Series 代表一个传感器的专属时间线
type Series struct {
	ID            uint32
	mu            sync.RWMutex // 读写锁：保护下方所有字段
	ActiveBuffer  []Point      // 热数据：待落盘的点
	Blocks        []*BlockMeta // 冷索引：已落盘的数据块目录
	LastFlushTime time.Time    // 计时器：上次成功刷盘的时间
}

func NewSeries(ID uint32) *Series {
	return &Series{
		ID:            ID,
		ActiveBuffer:  make([]Point, 0, BlockMaxPoints), // 预分配容量，避免扩容开销
		Blocks:        make([]*BlockMeta, 0),
		LastFlushTime: time.Now(),
	}
}

// ==========================================
// ✍️ 写入路径 (Write Path)
// ==========================================

// Append 追加数据。如果达到阈值，会"窃取"并返回数据供调用方落盘。
func (s *Series) Append(point Point) []Point {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.ActiveBuffer = append(s.ActiveBuffer, point)

	// ⚡️ 触发条件 A：数量满了
	if len(s.ActiveBuffer) >= BlockMaxPoints {
		return s.stealLocked()
	}
	return nil // 没满，返回 nil，外部无需执行写盘
}

// CheckForTicker 供后台 Ticker 调用，检查是否因为超时需要强制刷盘
func (s *Series) CheckForTicker() []Point {
	s.mu.Lock()
	defer s.mu.Unlock()

	// ⏰ 触发条件 B：有数据，且距离上次刷盘超过了设定的最大间隔
	if len(s.ActiveBuffer) > 0 && time.Since(s.LastFlushTime) >= ForceFlushInterval {
		return s.stealLocked()
	}
	return nil
}

// stealLocked 是核心的“偷梁换柱”魔法（调用方必须持有写锁）
// 它将底层数组彻底剥离，换上新的，保证写磁盘时不会阻塞新的 Append
func (s *Series) stealLocked() []Point {
	dataToSteal := s.ActiveBuffer

	// 分配全新的底层数组
	s.ActiveBuffer = make([]Point, 0, BlockMaxPoints)
	s.LastFlushTime = time.Now() // 重置计时器

	return dataToSteal
}

// AddBlockMeta 数据成功落盘后，由外部调用此方法将元数据登记造册
func (s *Series) AddBlockMeta(meta *BlockMeta) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Blocks = append(s.Blocks, meta)
}

// ==========================================
// 🔍 查询路径 (Query Path)
// ==========================================

// GetHotData 获取尚未落盘的热数据（安全拷贝）
func (s *Series) GetHotData() []Point {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 必须做深度拷贝，防止外部读取时切片被 stealLocked 替换或修改
	result := make([]Point, len(s.ActiveBuffer))
	copy(result, s.ActiveBuffer)
	return result
}

// FindBlocks 查询冷数据索引：找出在指定时间范围内的所有 Block
func (s *Series) FindBlocks(start, end int64) []*BlockMeta {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*BlockMeta
	for _, meta := range s.Blocks {
		// 时间范围过滤
		if meta.MaxTime < start || meta.MinTime > end {
			continue
		}
		// 这里存储的是指针，外部拿到指针后去调用 Manager 读盘
		result = append(result, meta)
	}
	return result
}
