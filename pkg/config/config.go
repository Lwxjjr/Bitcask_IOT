package config

// Config 表示应用配置
// TODO: 从底层 Bitcask 实现配置加载
type Config struct {
	// 配置字段将在需要时添加
}

// Load 从指定路径加载配置
// TODO: 实现配置加载
func Load(configPath string) (*Config, error) {
	return &Config{}, nil
}