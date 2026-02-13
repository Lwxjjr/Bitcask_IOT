package storage

import (
	"bytes"
	"encoding/binary"
	"io"
)

// Point 是最基础的时序数据单元 (16 Bytes)
type Point struct {
	Time  int64   // Unix 毫秒/秒级时间戳
	Value float64 // 实际采集值
}

// Block 是磁盘存储的最小物理单元 (The Chunk)
// 在写入文件时，会被序列化为二进制流
type Block struct {
	SensorID uint32  // 属于哪个设备 (用于启动时从文件恢复索引)
	Points   []Point // 实际的数据列表
}

// NewBlock 创建一个新的 Block
func NewBlock(sensorID uint32, points []Point) *Block {
	return &Block{
		SensorID: sensorID,
		Points:   points,
	}
}

// BlockMeta 是内存和磁盘的纽带，存内存中
// 它是 Storage 层告诉 Index 层：“刚才那个块我写好了，位置在这里”
type BlockMeta struct {
	FileID  uint32 // 属于哪个文件
	MinTime int64
	MaxTime int64
	Offset  int64
	Size    uint32
	Count   uint16 // 数据点数：用于 Count/Downsample 预估
}

// toMeta 根据 Block 生成对应的元数据
func (b *Block) toMeta(fileID uint32, offset int64, size uint32) *BlockMeta {
	if len(b.Points) == 0 {
		return nil
	}

	return &BlockMeta{
		FileID:  fileID,
		MinTime: b.Points[0].Time,
		MaxTime: b.Points[len(b.Points)-1].Time,
		Offset:  offset,
		Size:    size,
		Count:   uint16(len(b.Points)),
	}
}

// Encode 将 Block 序列化为字节数组
// Format: [SensorID: 4][Count: 4][Point1: 16]...[PointN: 16]
func (b *Block) encode() ([]byte, error) {
	buf := new(bytes.Buffer)

	// 1. 写入 SensorID
	if err := binary.Write(buf, binary.LittleEndian, b.SensorID); err != nil {
		return nil, err
	}

	// 2. 写入数据点数量 (Count)
	count := uint32(len(b.Points))
	if err := binary.Write(buf, binary.LittleEndian, count); err != nil {
		return nil, err
	}

	// 3. 写入所有数据点
	if err := binary.Write(buf, binary.LittleEndian, b.Points); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// DecodeBlock 将字节数组反序列化为 Block
func decodeBlock(data []byte) (*Block, error) {
	buf := bytes.NewReader(data)
	block := &Block{}

	// 1. 读取 SensorID
	if err := binary.Read(buf, binary.LittleEndian, &block.SensorID); err != nil {
		return nil, err
	}

	// 2. 读取数据点数量
	var count uint32
	if err := binary.Read(buf, binary.LittleEndian, &count); err != nil {
		return nil, err
	}

	// 3. 读取所有数据点
	block.Points = make([]Point, count)
	if count > 0 {
		if err := binary.Read(buf, binary.LittleEndian, block.Points); err != nil {
			if err == io.EOF {
				return block, nil
			}
			return nil, err
		}
	}

	return block, nil
}
