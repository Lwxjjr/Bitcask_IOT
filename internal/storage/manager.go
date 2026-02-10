package storage

import (
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
)

// Manager 负责管理多个数据段文件
type Manager struct {
	mu            sync.RWMutex
	dirPath       string
	activeSegment *Segment
	olderSegments map[uint32]*Segment
	maxSize       int64 // 单个 Segment 的最大大小，超过则轮转
}

// NewManager 初始化并加载现有的段文件
func NewManager(dirPath string, maxSize int64) (*Manager, error) {
	mgr := &Manager{
		dirPath:       dirPath,
		olderSegments: make(map[uint32]*Segment),
		maxSize:       maxSize,
	}

	if err := mgr.loadSegments(); err != nil {
		return nil, err
	}

	return mgr, nil
}

// loadSegments 扫描目录并加载已有的文件
func (m *Manager) loadSegments() error {
	files, err := os.ReadDir(m.dirPath)
	if err != nil {
		return err
	}

	var ids []uint32
	for _, f := range files {
		if strings.HasPrefix(f.Name(), SegmentFileNamePrefix) && strings.HasSuffix(f.Name(), SegmentFileNameSuffix) {
			idStr := strings.TrimPrefix(strings.TrimSuffix(f.Name(), SegmentFileNameSuffix), SegmentFileNamePrefix)
			id, err := strconv.ParseUint(idStr, 10, 32)
			if err == nil {
				ids = append(ids, uint32(id))
			}
		}
	}

	if len(ids) == 0 {
		// 如果没有文件，创建一个新的 Active Segment
		return m.rotate(0)
	}

	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	// 加载所有旧文件为只读模式（逻辑上）
	for i := 0; i < len(ids)-1; i++ {
		path := GetSegmentPath(m.dirPath, ids[i])
		seg, err := NewSegment(path, ids[i])
		if err != nil {
			return err
		}
		m.olderSegments[ids[i]] = seg
	}

	// 最后一个作为 Active Segment 加载
	lastID := ids[len(ids)-1]
	path := GetSegmentPath(m.dirPath, lastID)
	seg, err := NewSegment(path, lastID)
	if err != nil {
		return err
	}
	m.activeSegment = seg

	return nil
}

// WriteBlock 自动选择 Active Segment 写入，并处理轮转
func (m *Manager) WriteBlock(block *Block) (*BlockMeta, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 检查是否需要轮转
	if m.activeSegment.WriteOffset >= m.maxSize {
		if err := m.rotate(m.activeSegment.ID + 1); err != nil {
			return nil, err
		}
	}

	return m.activeSegment.WriteBlock(block)
}

// ReadBlock 根据 FileID 找到对应的 Segment 并读取
func (m *Manager) ReadBlock(meta *BlockMeta) (*Block, error) {
	m.mu.RLock()
	var seg *Segment
	if m.activeSegment != nil && m.activeSegment.ID == meta.FileID {
		seg = m.activeSegment
	} else {
		seg = m.olderSegments[meta.FileID]
	}
	m.mu.RUnlock()

	if seg == nil {
		return nil, fmt.Errorf("segment %d not found", meta.FileID)
	}

	return seg.ReadBlock(meta)
}

// rotate 关闭当前活跃段，开启一个新段
func (m *Manager) rotate(nextID uint32) error {
	if m.activeSegment != nil {
		m.olderSegments[m.activeSegment.ID] = m.activeSegment
	}

	path := GetSegmentPath(m.dirPath, nextID)
	seg, err := NewSegment(path, nextID)
	if err != nil {
		return err
	}

	m.activeSegment = seg
	return nil
}

// Close 关闭所有段文件
func (m *Manager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.activeSegment != nil {
		if err := m.activeSegment.Close(); err != nil {
			return err
		}
	}

	for _, seg := range m.olderSegments {
		if err := seg.Close(); err != nil {
			return err
		}
	}
	return nil
}
