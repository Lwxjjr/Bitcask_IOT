package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
)

var (
	ErrInvalidMessage  = errors.New("invalid message")
	ErrBufferTooSmall  = errors.New("buffer too small")
	ErrInvalidMagic    = errors.New("invalid magic number")
)

const (
	MagicNumber uint32 = 0x42495443 // "BITC" in hex
	Version     uint8  = 0x01

	MaxMessageSize = 16 * 1024 * 1024 // 16MB
)

// MessageType 定义消息类型
type MessageType uint8

const (
	MsgTypeWrite MessageType = 0x01
	MsgTypeQuery MessageType = 0x02
	MsgTypeAck   MessageType = 0x03
	MsgTypeError MessageType = 0x04
)

// MessageHeader 定义消息头部 (LTV 格式)
type MessageHeader struct {
	Magic   uint32
	Version uint8
	Type    MessageType
	Length  uint32
}

// WriteRequest 写入请求消息
type WriteRequest struct {
	SensorID  string
	Timestamp int64
	Value     float64
}

// QueryRequest 查询请求消息
type QueryRequest struct {
	SensorID string
	Start    int64
	End      int64
}

// QueryResponse 查询响应消息
type QueryResponse struct {
	Points []Point
}

// Point 数据点
type Point struct {
	Timestamp int64
	Value     float64
}

// WriteResponse 写入响应消息
type WriteResponse struct {
	Success bool
	Message string
}

// ErrorResponse 错误响应消息
type ErrorResponse struct {
	Code    uint32
	Message string
}

// EncodeWriteRequest 编码写入请求
func EncodeWriteRequest(req *WriteRequest) ([]byte, error) {
	buf := new(bytes.Buffer)

	// 编码 SensorID (length prefixed string)
	sensorIDBytes := []byte(req.SensorID)
	if err := binary.Write(buf, binary.BigEndian, uint16(len(sensorIDBytes))); err != nil {
		return nil, err
	}
	if _, err := buf.Write(sensorIDBytes); err != nil {
		return nil, err
	}

	// 编码 Timestamp
	if err := binary.Write(buf, binary.BigEndian, req.Timestamp); err != nil {
		return nil, err
	}

	// 编码 Value
	if err := binary.Write(buf, binary.BigEndian, req.Value); err != nil {
		return nil, err
	}

	return buildMessage(MsgTypeWrite, buf.Bytes())
}

// DecodeWriteRequest 解码写入请求
func DecodeWriteRequest(data []byte) (*WriteRequest, error) {
	buf := bytes.NewReader(data)

	// 解码 SensorID
	var sensorIDLen uint16
	if err := binary.Read(buf, binary.BigEndian, &sensorIDLen); err != nil {
		return nil, err
	}

	sensorIDBytes := make([]byte, sensorIDLen)
	if _, err := buf.Read(sensorIDBytes); err != nil {
		return nil, err
	}

	// 解码 Timestamp
	var timestamp int64
	if err := binary.Read(buf, binary.BigEndian, &timestamp); err != nil {
		return nil, err
	}

	// 解码 Value
	var value float64
	if err := binary.Read(buf, binary.BigEndian, &value); err != nil {
		return nil, err
	}

	return &WriteRequest{
		SensorID:  string(sensorIDBytes),
		Timestamp: timestamp,
		Value:     value,
	}, nil
}

// EncodeQueryRequest 编码查询请求
func EncodeQueryRequest(req *QueryRequest) ([]byte, error) {
	buf := new(bytes.Buffer)

	// 编码 SensorID (length prefixed string)
	sensorIDBytes := []byte(req.SensorID)
	if err := binary.Write(buf, binary.BigEndian, uint16(len(sensorIDBytes))); err != nil {
		return nil, err
	}
	if _, err := buf.Write(sensorIDBytes); err != nil {
		return nil, err
	}

	// 编码 Start
	if err := binary.Write(buf, binary.BigEndian, req.Start); err != nil {
		return nil, err
	}

	// 编码 End
	if err := binary.Write(buf, binary.BigEndian, req.End); err != nil {
		return nil, err
	}

	return buildMessage(MsgTypeQuery, buf.Bytes())
}

// DecodeQueryRequest 解码查询请求
func DecodeQueryRequest(data []byte) (*QueryRequest, error) {
	buf := bytes.NewReader(data)

	// 解码 SensorID
	var sensorIDLen uint16
	if err := binary.Read(buf, binary.BigEndian, &sensorIDLen); err != nil {
		return nil, err
	}

	sensorIDBytes := make([]byte, sensorIDLen)
	if _, err := buf.Read(sensorIDBytes); err != nil {
		return nil, err
	}

	// 解码 Start
	var start int64
	if err := binary.Read(buf, binary.BigEndian, &start); err != nil {
		return nil, err
	}

	// 解码 End
	var end int64
	if err := binary.Read(buf, binary.BigEndian, &end); err != nil {
		return nil, err
	}

	return &QueryRequest{
		SensorID: string(sensorIDBytes),
		Start:    start,
		End:      end,
	}, nil
}

// EncodeQueryResponse 编码查询响应
func EncodeQueryResponse(resp *QueryResponse) ([]byte, error) {
	buf := new(bytes.Buffer)

	// 编码 Point 数量
	if err := binary.Write(buf, binary.BigEndian, uint32(len(resp.Points))); err != nil {
		return nil, err
	}

	// 编码每个 Point
	for _, p := range resp.Points {
		if err := binary.Write(buf, binary.BigEndian, p.Timestamp); err != nil {
			return nil, err
		}
		if err := binary.Write(buf, binary.BigEndian, p.Value); err != nil {
			return nil, err
		}
	}

	return buildMessage(MsgTypeAck, buf.Bytes())
}

// DecodeQueryResponse 解码查询响应
func DecodeQueryResponse(data []byte) (*QueryResponse, error) {
	buf := bytes.NewReader(data)

	// 解码 Point 数量
	var pointCount uint32
	if err := binary.Read(buf, binary.BigEndian, &pointCount); err != nil {
		return nil, err
	}

	points := make([]Point, pointCount)
	for i := uint32(0); i < pointCount; i++ {
		var timestamp int64
		if err := binary.Read(buf, binary.BigEndian, &timestamp); err != nil {
			return nil, err
		}

		var value float64
		if err := binary.Read(buf, binary.BigEndian, &value); err != nil {
			return nil, err
		}

		points[i] = Point{Timestamp: timestamp, Value: value}
	}

	return &QueryResponse{Points: points}, nil
}

// EncodeWriteResponse 编码写入响应
func EncodeWriteResponse(resp *WriteResponse) ([]byte, error) {
	buf := new(bytes.Buffer)

	// 编码 Success
	var success uint8
	if resp.Success {
		success = 1
	}
	if err := binary.Write(buf, binary.BigEndian, success); err != nil {
		return nil, err
	}

	// 编码 Message (length prefixed string)
	msgBytes := []byte(resp.Message)
	if err := binary.Write(buf, binary.BigEndian, uint16(len(msgBytes))); err != nil {
		return nil, err
	}
	if _, err := buf.Write(msgBytes); err != nil {
		return nil, err
	}

	return buildMessage(MsgTypeAck, buf.Bytes())
}

// DecodeWriteResponse 解码写入响应
func DecodeWriteResponse(data []byte) (*WriteResponse, error) {
	buf := bytes.NewReader(data)

	// 解码 Success
	var success uint8
	if err := binary.Read(buf, binary.BigEndian, &success); err != nil {
		return nil, err
	}

	// 解码 Message
	var msgLen uint16
	if err := binary.Read(buf, binary.BigEndian, &msgLen); err != nil {
		return nil, err
	}

	msgBytes := make([]byte, msgLen)
	if _, err := buf.Read(msgBytes); err != nil {
		return nil, err
	}

	return &WriteResponse{
		Success: success == 1,
		Message: string(msgBytes),
	}, nil
}

// EncodeErrorResponse 编码错误响应
func EncodeErrorResponse(resp *ErrorResponse) ([]byte, error) {
	buf := new(bytes.Buffer)

	// 编码 Code
	if err := binary.Write(buf, binary.BigEndian, resp.Code); err != nil {
		return nil, err
	}

	// 编码 Message (length prefixed string)
	msgBytes := []byte(resp.Message)
	if err := binary.Write(buf, binary.BigEndian, uint16(len(msgBytes))); err != nil {
		return nil, err
	}
	if _, err := buf.Write(msgBytes); err != nil {
		return nil, err
	}

	return buildMessage(MsgTypeError, buf.Bytes())
}

// DecodeErrorResponse 解码错误响应
func DecodeErrorResponse(data []byte) (*ErrorResponse, error) {
	buf := bytes.NewReader(data)

	// 解码 Code
	var code uint32
	if err := binary.Read(buf, binary.BigEndian, &code); err != nil {
		return nil, err
	}

	// 解码 Message
	var msgLen uint16
	if err := binary.Read(buf, binary.BigEndian, &msgLen); err != nil {
		return nil, err
	}

	msgBytes := make([]byte, msgLen)
	if _, err := buf.Read(msgBytes); err != nil {
		return nil, err
	}

	return &ErrorResponse{
		Code:    code,
		Message: string(msgBytes),
	}, nil
}

// buildMessage 构建完整消息（LTV 格式）
func buildMessage(msgType MessageType, payload []byte) ([]byte, error) {
	if len(payload) > MaxMessageSize {
		return nil, fmt.Errorf("message too large: %d > %d", len(payload), MaxMessageSize)
	}

	buf := new(bytes.Buffer)

	// 写入 Magic Number
	if err := binary.Write(buf, binary.BigEndian, MagicNumber); err != nil {
		return nil, err
	}

	// 写入 Version
	if err := binary.Write(buf, binary.BigEndian, Version); err != nil {
		return nil, err
	}

	// 写入 Message Type
	if err := binary.Write(buf, binary.BigEndian, uint8(msgType)); err != nil {
		return nil, err
	}

	// 写入 Payload Length
	if err := binary.Write(buf, binary.BigEndian, uint32(len(payload))); err != nil {
		return nil, err
	}

	// 写入 Payload
	if _, err := buf.Write(payload); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

// DecodeMessageHeader 解码消息头部
func DecodeMessageHeader(data []byte) (*MessageHeader, error) {
	if len(data) < 13 { // 4(Magic) + 1(Version) + 1(Type) + 4(Length) = 10 bytes
		return nil, ErrBufferTooSmall
	}

	buf := bytes.NewReader(data)

	header := &MessageHeader{}

	// 解码 Magic Number
	if err := binary.Read(buf, binary.BigEndian, &header.Magic); err != nil {
		return nil, err
	}

	// 验证 Magic Number
	if header.Magic != MagicNumber {
		return nil, ErrInvalidMagic
	}

	// 解码 Version
	if err := binary.Read(buf, binary.BigEndian, &header.Version); err != nil {
		return nil, err
	}

	// 解码 Message Type
	var msgType uint8
	if err := binary.Read(buf, binary.BigEndian, &msgType); err != nil {
		return nil, err
	}
	header.Type = MessageType(msgType)

	// 解码 Payload Length
	if err := binary.Read(buf, binary.BigEndian, &header.Length); err != nil {
		return nil, err
	}

	return header, nil
}

// GetPayloadLength 从消息头部获取 payload 长度
func GetPayloadLength(data []byte) (uint32, error) {
	header, err := DecodeMessageHeader(data)
	if err != nil {
		return 0, err
	}
	return header.Length, nil
}

// GetMessageSize 获取完整消息大小（头部 + payload）
func GetMessageSize(data []byte) (int, error) {
	payloadLen, err := GetPayloadLength(data)
	if err != nil {
		return 0, err
	}
	return 10 + int(payloadLen), nil // 10 bytes header + payload
}