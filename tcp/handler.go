package tcp

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"net"
	"time"

	"github.com/bitcask-iot/engine/core"
	"github.com/bitcask-iot/engine/pkg/logger"
	"github.com/bitcask-iot/engine/protocol"
)

var (
	ErrConnectionClosed = errors.New("connection closed by client")
	ErrReadTimeout      = errors.New("read timeout")
)

// Handler 处理客户端连接和消息
type Handler struct {
	conn   net.Conn
	db     *core.DB
	logger *logger.Logger
	reader *bufio.Reader
}

// NewHandler 创建消息处理器
func NewHandler(conn net.Conn, db *core.DB, log *logger.Logger) *Handler {
	return &Handler{
		conn:   conn,
		db:     db,
		logger: log,
		reader: bufio.NewReader(conn),
	}
}

// HandleLoop 处理消息循环
func (h *Handler) HandleLoop() error {
	for {
		// 读取消息
		msg, err := h.readMessage()
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, ErrConnectionClosed) {
				return nil
			}
			return fmt.Errorf("read message failed: %w", err)
		}

		// 处理消息
		response, err := h.handleMessage(msg)
		if err != nil {
			// 构造错误响应
			errResp := &protocol.ErrorResponse{
				Code:    500,
				Message: err.Error(),
			}
			response, err = protocol.EncodeErrorResponse(errResp)
			if err != nil {
				return fmt.Errorf("encode error response failed: %w", err)
			}
		}

		// 发送响应
		if err := h.writeMessage(response); err != nil {
			return fmt.Errorf("write response failed: %w", err)
		}
	}
}

// readMessage 读取完整消息（处理粘包）
func (h *Handler) readMessage() ([]byte, error) {
	// 读取消息头部（10 字节）
	headerBuf := make([]byte, 10)
	_, err := io.ReadFull(h.reader, headerBuf)
	if err != nil {
		return nil, err
	}

	// 解析头部，获取 payload 长度
	payloadLen, err := protocol.GetPayloadLength(headerBuf)
	if err != nil {
		return nil, fmt.Errorf("parse header failed: %w", err)
	}

	// 读取 payload
	payloadBuf := make([]byte, payloadLen)
	_, err = io.ReadFull(h.reader, payloadBuf)
	if err != nil {
		return nil, err
	}

	// 组合完整消息
	msg := append(headerBuf, payloadBuf...)
	return msg, nil
}

// writeMessage 写入消息
func (h *Handler) writeMessage(msg []byte) error {
	// 设置写超时
	h.conn.SetWriteDeadline(time.Now().Add(30 * time.Second))

	_, err := h.conn.Write(msg)
	if err != nil {
		return err
	}

	return nil
}

// handleMessage 处理单个消息
func (h *Handler) handleMessage(msg []byte) ([]byte, error) {
	// 解析消息头部
	header, err := protocol.DecodeMessageHeader(msg)
	if err != nil {
		return nil, fmt.Errorf("decode header failed: %w", err)
	}

	// 提取 payload
	payloadLen := int(header.Length)
	if len(msg) < 10+payloadLen {
		return nil, fmt.Errorf("incomplete message: expected %d, got %d", 10+payloadLen, len(msg))
	}
	payload := msg[10 : 10+payloadLen]

	// 根据消息类型分发处理
	switch header.Type {
	case protocol.MsgTypeWrite:
		return h.handleWrite(payload)
	case protocol.MsgTypeQuery:
		return h.handleQuery(payload)
	default:
		return nil, fmt.Errorf("unknown message type: %d", header.Type)
	}
}

// handleWrite 处理写入请求
func (h *Handler) handleWrite(payload []byte) ([]byte, error) {
	// 解码写入请求
	req, err := protocol.DecodeWriteRequest(payload)
	if err != nil {
		return nil, fmt.Errorf("decode write request failed: %w", err)
	}

	// 执行写入
	err = h.db.Write(req.SensorID, req.Timestamp, req.Value)
	if err != nil {
		// 构造失败响应
		resp := &protocol.WriteResponse{
			Success: false,
			Message: err.Error(),
		}
		return protocol.EncodeWriteResponse(resp)
	}

	// 构造成功响应
	resp := &protocol.WriteResponse{
		Success: true,
		Message: "ok",
	}
	return protocol.EncodeWriteResponse(resp)
}

// handleQuery 处理查询请求
func (h *Handler) handleQuery(payload []byte) ([]byte, error) {
	// 解码查询请求
	req, err := protocol.DecodeQueryRequest(payload)
	if err != nil {
		return nil, fmt.Errorf("decode query request failed: %w", err)
	}

	// 执行查询
	points, err := h.db.Query(req.SensorID, req.Start, req.End)
	if err != nil {
		return nil, fmt.Errorf("query failed: %w", err)
	}

	// 转换为协议格式
	protocolPoints := make([]protocol.Point, len(points))
	for i, p := range points {
		protocolPoints[i] = protocol.Point{
			Timestamp: p.Time,
			Value:     p.Value,
		}
	}

	// 构造响应
	resp := &protocol.QueryResponse{
		Points: protocolPoints,
	}
	return protocol.EncodeQueryResponse(resp)
}