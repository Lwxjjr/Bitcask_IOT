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
	HintFileNameSuffix    = ".hint" // 🌟 新增：伴生索引文件的后缀
)

// 🛠️ 修改：让工具函数支持指定后缀，方便复用
func getFilePath(dir string, id uint32, suffix string) string {
	return filepath.Join(dir, fmt.Sprintf("%s%06d%s", SegmentFileNamePrefix, id, suffix))
}

// Segment 代表一个纯粹的物理数据分片 (包含数据文件和索引文件)
type Segment struct {
	mu          sync.RWMutex
	ID          uint32
	File        *os.File // 真实数据文件 (.vlog)
	HintFile    *os.File // 🌟 新增：伴生索引文件 (.hint)
	WriteOffset int64
}

// NewSegment 打开或创建一个 Segment 文件组合
// 🛠️ 修改：参数从 path 改为 dir，因为要同时创建两个文件
func newSegment(dir string, id uint32) (*Segment, error) {
	vlogPath := getFilePath(dir, id, SegmentFileNameSuffix)
	hintPath := getFilePath(dir, id, HintFileNameSuffix)

	// 1. 打开数据文件
	f, err := os.OpenFile(vlogPath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		return nil, err
	}

	// 🌟 2. 打开伴生索引文件
	hf, err := os.OpenFile(hintPath, os.O_CREATE|os.O_RDWR|os.O_APPEND, 0644)
	if err != nil {
		f.Close() // 容错：防止文件句柄泄露
		return nil, err
	}

	stat, err := f.Stat()
	if err != nil {
		f.Close()
		hf.Close()
		return nil, err
	}

	return &Segment{
		ID:          id,
		File:        f,
		HintFile:    hf, // 🌟 塞入句柄
		WriteOffset: stat.Size(),
	}, nil
}

// Write 极其纯粹的物理写入！只认字节流，不管业务逻辑
func (s *Segment) write(data []byte) (int64, error) {
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
func (s *Segment) readAt(size uint32, offset int64) ([]byte, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data := make([]byte, size)
	if _, err := s.File.ReadAt(data, offset); err != nil {
		return nil, err
	}

	return data, nil
}

// Size 获取当前文件的大小（线程安全），用于 Manager 判断轮转
func (s *Segment) size() int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.WriteOffset
}

// Sync 强制将 Page Cache 刷入磁盘
func (s *Segment) Sync() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if err := s.File.Sync(); err != nil {
		return err
	}
	// 🌟 新增：顺手把 hint 也刷入磁盘
	if s.HintFile != nil {
		return s.HintFile.Sync()
	}
	return nil
}

// Close 关闭文件
func (s *Segment) close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	var err error
	if e := s.File.Close(); e != nil {
		err = e
	}
	// 🌟 新增：关闭 hint 文件句柄
	if s.HintFile != nil {
		if e := s.HintFile.Close(); e != nil {
			err = e
		}
	}
	return err
}
