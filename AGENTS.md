# Bitcask IoT 存储引擎 - 项目上下文文档

## 项目概述

Bitcask IoT 是一个基于 LSM-Tree 变体 (TSM) 的嵌入式时序存储引擎，专为 IoT 场景的高频写入和范围查询优化。该项目采用内存缓冲 (Write Buffer) + 数据分块 (Chunking) + 稀疏索引 (Sparse Index) 的架构设计。

### 核心特性

- **ID 映射系统**: 将字符串形式的 SensorID 映射为 `uint32`，以优化存储空间和内存使用
- **写路径优化**: 通过 WAL + Buffer + Compression 的三级缓冲机制实现高效写入
- **压缩存储**: 使用 Delta-of-Delta (时间戳) 和 XOR/Snappy (数值) 编码进行数据压缩
- **稀疏索引查询**: 通过内存中的 BlockMeta 索引快速定位数据块，支持高效的范围查询
- **自动文件轮转**: 按时间周期或大小切分 Segment 文件，支持 TTL 过期清理
- **崩溃恢复**: 通过 WAL 重放机制保证数据持久性

### 架构组件

1. **Ingestion Layer** (`internal/collector`): OPC UA 数据采集客户端
2. **Engine Layer** (`internal/engine`): 数据库统一入口，协调各层操作
3. **Index Layer** (`internal/index`): 内存索引管理，维护 SensorID -> Series 映射
4. **Storage Layer** (`internal/storage`): 磁盘文件操作，管理 Segment 文件读写
5. **Query Layer** (`internal/query`): 查询执行器，支持迭代器和查询规划
6. **Service Layer** (`internal/service`): HTTP/RPC 接入层

## 技术栈

### 核心语言
- **Go 1.23+**: 利用泛型处理不同类型的数值点

### 关键依赖库 (待集成)

- **压缩**: `github.com/golang/snappy` 或 `klauspost/compress/zstd` - Data Block 压缩
- **工业协议**: `github.com/gopcua/opcua` - OPC UA 数据采集
- **CLI 框架**: `github.com/spf13/cobra` - 命令行工具
- **配置管理**: `github.com/spf13/viper` - YAML 配置文件管理
- **Web 服务**: `github.com/gin-gonic/gin` - HTTP API 服务
- **日志**: `github.com/uber-go/zap` - 结构化日志

### 标准库深度使用
- `encoding/binary`: Block Header 的 BigEndian 编码
- `os` & `io`: Segment 文件的 Seek, ReadAt, Append 操作
- `sync/atomic`: 无锁指标统计
- `sort`: 内存索引二分查找

## 项目结构

```
bitcask_iot/
├── cmd/
│   ├── server/main.go       # 服务入口，组装 Engine、Service、HTTP
│   └── cli/main.go          # 运维与调试 CLI 工具
│
├── configs/
│   └── config.yaml          # 配置文件 (OPC UA、存储、日志、服务器)
│
├── internal/                # 核心业务逻辑 (不可被外部导入)
│   ├── collector/           # OPC UA 数据采集客户端
│   ├── engine/              # 数据库协调层 (Put/Get/Close 接口、WAL、Options)
│   ├── index/               # 内存索引层 (Series、ID 映射、BlockMeta)
│   ├── storage/             # 物理存储层 (Block 压缩、文件管理、二进制协议)
│   └── service/             # HTTP/RPC Handler
│
├── pkg/                     # 公共库 (可被外部导入)
│   ├── config/              # Viper 配置加载
│   ├── logger/              # Zap 日志封装
│   └── utils/               # 通用工具函数
│
├── test/
│   ├── benchmark/           # 性能基准测试
│   └── mock/                # 测试模拟数据
│
├── go.mod                   # Go 模块定义
└── go.sum                   # 依赖版本锁定
```

## 构建与运行

### 前置依赖

```bash
# 安装 Go 1.23+
# 项目使用 Go Modules，无需额外设置 GOPATH
```

### 安装依赖

```bash
# 初始化依赖 (首次运行)
go mod download

# 安装关键依赖库
go get github.com/spf13/cobra
go get github.com/spf13/viper
go get github.com/gin-gonic/gin
go get github.com/uber-go/zap
go get github.com/gopcua/opcua
go get github.com/golang/snappy
```

### 构建项目

```bash
# 构建 server
go build -o bin/server ./cmd/server

# 构建 cli
go build -o bin/cli ./cmd/cli

# 或使用 make (如果存在 Makefile)
make build
```

### 运行服务

```bash
# 使用默认配置运行
./bin/server

# 指定配置文件
./bin/server -c configs/config.yaml
```

### 运行测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/engine

# 运行基准测试
go test -bench=. ./test/benchmark

# 带覆盖率测试
go test -cover ./...
```

### 运行 CLI 工具

```bash
# 查看帮助
./bin/cli --help

# 查询数据
./bin/cli query --sensor "Temperature" --start "2024-01-01" --end "2024-01-02"
```

## 开发规范

### 代码风格

- 遵循 [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- 使用 `gofmt` 格式化代码
- 函数命名：公共函数使用 PascalCase，私有函数使用 camelCase
- 接口命名：通常以 "er" 结尾 (如 Reader, Writer)

### 错误处理

- 始终处理返回的 error
- 使用 `errors.Is()` 和 `errors.As()` 进行错误检查
- 对于可恢复的错误，记录日志并继续
- 对于严重错误，返回给调用者处理

### 并发安全

- 存储引擎内部操作需保证并发安全
- 使用 `sync.RWMutex` 保护共享数据结构
- 使用 `sync/atomic` 进行无锁指标统计
- 避免在持有锁时进行 I/O 操作

### 测试要求

- 每个核心包应有单元测试
- 测试文件命名为 `*_test.go`
- 使用 `table-driven tests` 模式
- 关键路径必须有测试覆盖
- 集成测试放在 `test/` 目录

### 配置管理

- 所有配置项定义在 `configs/config.yaml`
- 使用 Viper 加载配置
- 支持环境变量覆盖配置项
- 敏感信息使用环境变量，不写入配置文件

### 日志规范

- 使用 Zap 结构化日志
- 日志级别：Debug (开发)、Info (生产)、Warn、Error、Fatal
- 关键操作必须记录日志 (如写入、查询、错误)
- 使用结构化字段记录上下文信息 (如 `sensorID`, `timestamp`, `count`)

## 核心文件说明

### 文档文件

- `01_architecture_structure.md`: 架构设计与目录结构指南
- `02_core_features.md`: 核心功能实现指南 (ID 映射、压缩存储、查询)
- `03_tech_stack.md`: 技术栈与依赖管理指南

### 核心模块

- `internal/engine/engine.go`: 实现数据库核心接口 (Put, Get, Close)
- `internal/index/series.go`: Series 数据结构，包含 ActiveBuffer 和 BlockMeta
- `internal/storage/log_record.go`: 磁盘二进制协议定义
- `internal/storage/file_manager.go`: Segment 文件管理

## 开发路线图

当前项目处于初始化阶段，需要从底层实现：

1. **存储层实现**
   - 实现 `log_record.go` 二进制协议
   - 实现 `block.go` 压缩/解压逻辑
   - 实现 `file_manager.go` 文件操作

2. **索引层实现**
   - 实现 `series.go` 内存缓冲和索引
   - 实现 `id_map.go` ID 映射系统

3. **引擎层实现**
   - 实现 `engine.go` 核心接口
   - 实现 `wal.go` 预写日志
   - 实现崩溃恢复逻辑

4. **数据采集层实现**
   - 实现 OPC UA 客户端连接
   - 实现数据订阅和采集

5. **服务层实现**
   - 实现 HTTP API 接口
   - 实现查询和写入端点

## 配置说明

### OPC UA 配置
- `endpoint`: OPC UA 服务器地址
- `node_ids`: 订阅的节点 ID 列表
- `subscription_interval`: 订阅刷新间隔

### 存储配置
- `dir_path`: 数据存储目录
- `data_file_size`: Segment 文件大小限制
- `sync_write`: 是否同步写入 (安全 vs 性能)
- `rotation_interval`: 文件轮转间隔

### 日志配置
- `level`: 日志级别
- `output`: 日志输出路径
- `max_size`: 单个日志文件最大大小 (MB)
- `max_backups`: 保留的日志备份数量
- `max_age`: 日志保留天数

### HTTP 服务器配置
- `host`: 监听地址
- `port`: 监听端口

## 重要提示

1. **当前状态**: 项目刚完成目录结构初始化，核心功能待实现
2. **优先级**: 从存储层开始，自底向上实现
3. **性能目标**: 支持每秒 10,000+ 点的写入吞吐量
4. **兼容性**: 支持 Linux 环境 (开发/测试)
5. **文档**: 架构、特性、技术栈文档已完备，可作为开发参考