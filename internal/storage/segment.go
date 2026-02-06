package storage

import (
	"os"
	"sync"
)

// BlockMeta 是内存和磁盘的纽带，存内存中
// 它是 Storage 层告诉 Index 层：“刚才那个块我写好了，位置在这里”
type BlockMeta struct {
	MinTime int64
	MaxTime int64
	Offset  int64
	Size    uint32
	Count   uint16 // 数据点数：用于 Count/Downsample 预估
}

// ToMeta 根据 Block 生成对应的元数据
func (b *Block) ToMeta(offset int64, size uint32) *BlockMeta {
	if len(b.Points) == 0 {
		return nil
	}

	return &BlockMeta{
		MinTime: b.Points[0].Time,
		MaxTime: b.Points[len(b.Points)-1].Time,
		Offset:  offset,
		Size:    size,
		Count:   uint16(len(b.Points)),
	}
}

// Segment 代表一个物理数据文件
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

// WriteBlock 将一个 Block 写入 Segment 并返回其元数据
func (s *Segment) WriteBlock(block *Block) (*BlockMeta, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	data, err := block.Encode()
	if err != nil {
		return nil, err
	}

	size := uint32(len(data))
	offset := s.WriteOffset

	if _, err := s.File.Write(data); err != nil {
		return nil, err
	}

	s.WriteOffset += int64(size)

	return block.ToMeta(offset, size), nil
}

// Close 关闭文件
func (s *Segment) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.File.Close()
}
