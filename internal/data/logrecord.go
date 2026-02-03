package data

import (
	"encoding/binary"
	"hash/crc32"
	"time"
)

// LogRecord 表示 Bitcask 格式中的单条日志记录
type LogRecord struct {
	Key       []byte
	Value     []byte
	Timestamp int64
	ExpiresAt int64
}

// Encode 将 LogRecord 编码为字节
// 格式: [CRC32][Timestamp][KeySize][ValueSize][Key][Value]
func (r *LogRecord) Encode() []byte {
	keySize := uint32(len(r.Key))
	valueSize := uint32(len(r.Value))

	// 计算总大小
	totalSize := 4 + // CRC32
		8 + // Timestamp
		4 + // KeySize
		4 + // ValueSize
		int(keySize) +
		int(valueSize)

	buf := make([]byte, totalSize)

	offset := 0

	// 为 CRC32 预留空间（稍后计算）
	offset += 4

	// 写入时间戳
	binary.BigEndian.PutUint64(buf[offset:], uint64(r.Timestamp))
	offset += 8

	// 写入 key 大小
	binary.BigEndian.PutUint32(buf[offset:], keySize)
	offset += 4

	// 写入 value 大小
	binary.BigEndian.PutUint32(buf[offset:], valueSize)
	offset += 4

	// 写入 key
	copy(buf[offset:], r.Key)
	offset += int(keySize)

	// 写入 value
	copy(buf[offset:], r.Value)

	// 计算 CRC32（不包括 CRC32 字段本身）
	crc := crc32.ChecksumIEEE(buf[4:])
	binary.BigEndian.PutUint32(buf[0:], crc)

	return buf
}

// Decode 将字节解码为 LogRecord
func Decode(data []byte) (*LogRecord, error) {
	if len(data) < 20 {
		return nil, ErrInvalidRecord
	}

	offset := 0

	// 读取并验证 CRC32
	crc := binary.BigEndian.Uint32(data[offset:])
	offset += 4
	actualCRC := crc32.ChecksumIEEE(data[4:])
	if crc != actualCRC {
		return nil, ErrCRCMismatch
	}

	// 读取时间戳
	timestamp := int64(binary.BigEndian.Uint64(data[offset:]))
	offset += 8

	// 读取 key 大小
	keySize := int(binary.BigEndian.Uint32(data[offset:]))
	offset += 4

	// 读取 value 大小
	valueSize := int(binary.BigEndian.Uint32(data[offset:]))
	offset += 4

	// 检查数据是否足够
	if len(data) < offset+keySize+valueSize {
		return nil, ErrInvalidRecord
	}

	// 读取 key
	key := make([]byte, keySize)
	copy(key, data[offset:])
	offset += keySize

	// 读取 value
	value := make([]byte, valueSize)
	copy(value, data[offset:])

	return &LogRecord{
		Key:       key,
		Value:     value,
		Timestamp: timestamp,
	}, nil
}

// EncodeKey 将传感器 ID 和时间戳编码为字节切片
// 格式: [SensorID (string)][Timestamp (BigEndian Uint64)]
func EncodeKey(sensorID string, timestamp int64) []byte {
	buf := make([]byte, len(sensorID)+8)
	copy(buf, sensorID)
	binary.BigEndian.PutUint64(buf[len(sensorID):], uint64(timestamp))
	return buf
}

// DecodeKey 将字节切片解码为传感器 ID 和时间戳
func DecodeKey(key []byte) (string, int64) {
	if len(key) < 8 {
		return "", 0
	}
	sensorID := string(key[:len(key)-8])
	timestamp := int64(binary.BigEndian.Uint64(key[len(key)-8:]))
	return sensorID, timestamp
}

// NewLogRecord 创建一个带有当前时间戳的新 LogRecord
func NewLogRecord(key, value []byte) *LogRecord {
	return &LogRecord{
		Key:       key,
		Value:     value,
		Timestamp: time.Now().UnixNano(),
	}
}

// 错误定义
var (
	ErrInvalidRecord = &RecordError{Msg: "invalid record"}
	ErrCRCMismatch   = &RecordError{Msg: "CRC checksum mismatch"}
)

// RecordError 表示记录错误
type RecordError struct {
	Msg string
}

func (e *RecordError) Error() string {
	return e.Msg
}