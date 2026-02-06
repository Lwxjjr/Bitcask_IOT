package storage

import (
	"bytes"
	"encoding/gob"
)

// Point 是最基础的时序数据单元 (16 Bytes)
type Point struct {
	Time  int64   // Unix 毫秒/秒级时间戳
	Value float64 // 实际采集值
}

// Block 是磁盘存储的最小物理单元 (The Chunk)
// 在写入文件时，会被 gob.Encode 序列化为二进制流
type Block struct {
	SensorID uint32  // 属于哪个设备 (用于启动时从文件恢复索引)
	Points   []Point // 实际的数据列表 (Go Slice 自带长度，不用额外存 Count)
}

// Encode 将 Block 序列化为字节数组
func (b *Block) Encode() ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	if err := enc.Encode(b); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// DecodeBlock 将字节数组反序列化为 Block
func DecodeBlock(data []byte) (*Block, error) {
	var block Block
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	if err := dec.Decode(&block); err != nil {
		return nil, err
	}
	return &block, nil
}
