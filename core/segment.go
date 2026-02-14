package core

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const (
	SegmentFileNamePrefix = "seg-"
	SegmentFileNameSuffix = ".vlog"
)

// GetSegmentPath 根据目录和 ID 生成完整的段文件路径
func GetSegmentPath(dir string, id uint32) string {
	return filepath.Join(dir, fmt.Sprintf("%s%06d%s", SegmentFileNamePrefix, id, SegmentFileNameSuffix))
}

// Segment 代表一个纯粹的物理数据文件
type Segment struct {
	mu          sync.RWMutex
	ID          uint32
	Path        string
	File        *os.File
	WriteOffset int64
}

// NewSegment 打开或创建一个 Segment 文件
func NewSegment(path string, id uint32) (*Segment, error) {
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}

	return &Segment{
		ID:          id,
		Path:        path,
		File:        f,
		WriteOffset: stat.Size(),
	}, nil
}

// Write 极其纯粹的物理写入！只认字节流，不管业务逻辑
// 返回写入的起始 Offset
func (s *Segment) Write(data []byte) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	offset := s.WriteOffset

	if _, err := s.File.Write(data); err != nil {
		return 0, err
	}

	s.WriteOffset += int64(len(data))
	return offset, nil
}

// ReadAt 提供极其纯粹的物理读取
func (s *Segment) ReadAt(size uint32, offset int64) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data := make([]byte, size)
	if _, err := s.File.ReadAt(data, offset); err != nil {
		return nil, err
	}

	return data, nil
}

// Size 获取当前文件的大小（线程安全），用于 Manager 判断轮转
func (s *Segment) Size() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.WriteOffset
}

// Sync 强制将 Page Cache 刷入磁盘
func (s *Segment) Sync() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.File.Sync()
}

// Close 关闭文件
func (s *Segment) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.File.Close()
}
