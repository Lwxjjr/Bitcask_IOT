package storage

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

// BlockMeta 是内存和磁盘的纽带，存内存中
// 它是 Storage 层告诉 Index 层：“刚才那个块我写好了，位置在这里”
type BlockMeta struct {
	MinTime int64  // 用于二分查找：范围开始
	MaxTime int64  // 用于二分查找：范围结束
	Offset  int64  // 文件偏移量：Seek(Offset)
	Size    uint32 // 数据长度：Read(Size)
	Count   uint16 // 数据点数：用于 Count/Downsample 预估
}
