package logger

// Logger 封装日志功能
// TODO: 从底层 Bitcask 实现 logger
type Logger struct{}

// New 创建新的 logger 实例
// TODO: 实现 logger 初始化
func New() (*Logger, error) {
	return &Logger{}, nil
}

func (l *Logger) Sync() error {
	return nil
}

func (l *Logger) Info(args ...interface{}) {}
func (l *Logger) Infof(format string, args ...interface{}) {}