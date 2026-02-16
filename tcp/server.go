package tcp

import (
	"fmt"
	"net"
	"sync"

	"github.com/bitcask-iot/engine/core"
	"github.com/bitcask-iot/engine/pkg/logger"
)

// Server TCP 服务端
type Server struct {
	address string
	db      *core.DB
	logger  *logger.Logger

	listener net.Listener
	wg       sync.WaitGroup
	stopCh   chan struct{}
}

// NewServer 创建 TCP 服务端
func NewServer(address string, db *core.DB, log *logger.Logger) *Server {
	return &Server{
		address: address,
		db:      db,
		logger:  log,
		stopCh:  make(chan struct{}),
	}
}

// Start 启动 TCP 服务端
func (s *Server) Start() error {
	listener, err := net.Listen("tcp", s.address)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.address, err)
	}

	s.listener = listener
	s.logger.Info("TCP server started", "address", s.address)

	// 启动接受连接的 goroutine
	s.wg.Add(1)
	go s.acceptConnections()

	return nil
}

// Stop 停止 TCP 服务端
func (s *Server) Stop() error {
	// 关闭监听器
	if s.listener != nil {
		if err := s.listener.Close(); err != nil {
			return fmt.Errorf("failed to close listener: %w", err)
		}
	}

	// 关闭 stopCh，通知所有连接停止
	close(s.stopCh)

	// 等待所有连接处理完成
	s.wg.Wait()

	s.logger.Info("TCP server stopped")
	return nil
}

// acceptConnections 接受客户端连接
func (s *Server) acceptConnections() {
	defer s.wg.Done()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.stopCh:
				// 服务端正在关闭，正常退出
				return
			default:
				s.logger.Error("failed to accept connection", "error", err)
				continue
			}
		}

		// 为每个连接启动一个处理 goroutine
		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

// handleConnection 处理客户端连接
func (s *Server) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	clientAddr := conn.RemoteAddr().String()
	s.logger.Info("client connected", "address", clientAddr)

	// 创建 handler 处理该连接
	handler := NewHandler(conn, s.db, s.logger)

	// 处理消息循环
	err := handler.HandleLoop()

	if err != nil {
		s.logger.Error("connection error", "address", clientAddr, "error", err)
	} else {
		s.logger.Info("client disconnected", "address", clientAddr)
	}
}