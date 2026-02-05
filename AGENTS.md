# Bitcask IoT 项目指南

## 📋 项目概述

**Bitcask IoT Storage Engine** 是一个基于 TSM (Time-Structured Merge Tree) 变体的嵌入式时序存储引擎，专为 IoT 场景的高频写入和范围查询优化。项目采用内存缓冲（Write Buffer）+ 数据分块（Chunking）+ 稀疏索引（Sparse Index）的架构设计。

### 核心特性
- 🚀 高吞吐写入：通过 WAL + MemBuffer 实现
- 💾 高效压缩：Delta-of-Delta 和 XOR 编码
- 🔍 快速查询：稀疏索引 + Block 级别检索
- 🔄 崩溃恢复：WAL 重放机制
- 📊 时序优化：专为 IoT 数据设计

### 技术架构
```
数据源 → 采集层 → 引擎协调 → 内存缓冲 → 压缩分块 → 磁盘存储
                 ↑                    ↑
              元数据管理           预写日志 (WAL)
```

---

## 🛠 技术栈

### 核心环境
- **语言**: Go 1.23.0+ (toolchain go1.24.12)
- **模块名**: github.com/bitcask-iot/engine

### 主要依赖库

#### 压缩与编码
- `github.com/golang/snappy` - Data Block 压缩

#### 工业协议
- `github.com/gopcua/opcua` - OPC UA 客户端

#### 命令行与配置
- `github.com/spf13/cobra` - CLI 框架
- `github.com/spf13/viper` - 配置管理

#### Web 服务
- `github.com/gin-gonic/gin` - HTTP API 服务

#### 日志
- `github.com/uber-go/zap` - 高性能日志

### 标准库深度使用
- `encoding/binary` - Block Header BigEndian 编码
- `os` & `io` - 文件 Seek, ReadAt, Append 操作
- `sync/atomic` - 无锁指标统计
- `sort` - 索引二分查找

---

## 📁 项目结构

```
bitcask-iot/
├── cmd/
│   ├── server/               # 主程序：启动采集、引擎、API
│   │   └── main.go
│   └── cli/                  # 命令行工具：查询、调试
│       └── main.go
│
├── configs/                  # 配置文件
│   └── config.yaml
│
├── internal/                 # 核心私有代码
│   ├── collector/            # [采集层] OPC UA 客户端
│   ├── compaction/           # [压缩层] 数据压缩逻辑
│   ├── engine/               # [控制层] Put/Get 协调器
│   ├── index/                # [索引层] 稀疏索引管理
│   ├── query/                # [查询层] 迭代器与降采样
│   ├── service/              # [业务层] HTTP API Handler
│   └── storage/              # [存储层] Segment 文件管理
│
├── pkg/                      # 公共库
│   ├── config/               # Viper 配置加载
│   │   └── config.go
│   ├── logger/               # Zap 日志封装
│   │   └── logger.go
│   └── utils/                # 通用工具
│
├── test/                     # 测试
│   ├── benchmark/            # 性能压测
│   └── mock/                 # 模拟数据生成
│
├── bin/                      # 编译输出
├── go.mod
├── go.sum
└── AGENTS.md                 # 本文件
```

### 模块职责

#### cmd/
- **server**: 服务入口，初始化所有组件
- **cli**: 调试工具，支持查询、统计等操作

#### internal/
- **collector**: 从 OPC UA 服务器采集数据
- **compaction**: 实现 Delta-of-Delta 和 XOR 编码
- **engine**: 协调 MemTable 和 Storage，处理 Put/Get
- **index**: 管理内存中的 Block 索引
- **query**: 实现查询迭代器和降采样算法（LTTB）
- **service**: HTTP API 端点实现
- **storage**: Segment 文件读写，Block 管理

#### pkg/
- **config**: 配置文件加载和验证
- **logger**: 统一日志接口
- **utils**: 时间对齐、ID 生成等工具函数

---

## 🎯 核心功能

### 1. ID 映射机制
- **写入**: `GetOrRegister(name string) -> uint32`
- **读取**: `GetID(name string) -> uint32`
- **优势**: 磁盘只存储 uint32，节省空间

### 2. 写入路径
```
数据点 → WAL (Crash Safe) → MemBuffer → Flush → 压缩 → Block → Segment
```
- WAL 保证数据安全
- Buffer 积攒数据（>1KB 或 >60s）
- Delta-of-Delta 压缩时间戳
- XOR 压缩数值

### 3. 物理存储格式
```
Segment 文件:
[Header: MagicNumber]
[Block 1: ID, Time, Size, Data]
[Block 2: ID, Time, Size, Data]
...
[Footer: Index Offset]
```
- 按 2 小时或 512MB 轮转文件
- 过期删除旧文件（无需 Compaction）

### 4. 查询机制
```
Query(sensorID, start, end, maxPoints)
  → 索引二分查找
  → 加载相关 Blocks
  → 解压解码
  → 过滤时间范围
  → 降采样（LTTB）
  → 返回结果
```

### 5. 崩溃恢复
1. 加载元数据（ID 映射）
2. 扫描 Segment 重建 Block 索引
3. 重放 WAL 到 MemBuffer

---

## 🔧 开发指南

### 安装依赖
```bash
# 基础框架
go get github.com/spf13/cobra
go get github.com/spf13/viper
go get github.com/gin-gonic/gin
go get github.com/uber-go/zap

# 业务依赖
go get github.com/gopcua/opcua
go get github.com/golang/snappy
```

### 编译项目
```bash
# 编译服务端
go build -o bin/server ./cmd/server

# 编译 CLI
go build -o bin/cli ./cmd/cli

# 编译所有
go build ./...
```

### 运行服务
```bash
# 使用默认配置
./bin/server

# 指定配置文件
./bin/server -c configs/config.yaml
```

### 测试
```bash
# 运行所有测试
go test ./...

# 运行特定包测试
go test ./internal/engine

# 运行基准测试
go test -bench=. ./test/benchmark

# 查看覆盖率
go test -cover ./...
```

### 代码规范
- 遵循 Go 标准工程布局
- 使用 `gofmt` 格式化代码
- 通用工具放 `pkg/`，核心业务放 `internal/`
- 所有公开 API 需要注释
- 错误处理要明确，避免 `panic`

---

## ⚙️ 配置说明

### 配置文件位置
`configs/config.yaml`

### 关键配置项

#### OPC UA 配置
```yaml
opc:
  endpoint: "opc.tcp://localhost:53530"
  node_ids:
    - "ns=3;i=1001"
  subscription_interval: "1s"
```

#### 存储配置
```yaml
storage:
  dir_path: "/tmp/bitcask-iot"
  data_file_size: 512MB
  sync_write: false
  rotation_interval: "1h"
```

#### 日志配置
```yaml
logger:
  level: "info"
  output: "./logs/bitcask-iot.log"
  max_size: 100
  max_backups: 3
  max_age: 7
```

#### HTTP 服务配置
```yaml
server:
  host: "0.0.0.0"
  port: 8080
```

---

## 📊 当前状态

### ✅ 已完成
- 项目目录结构搭建
- 基础配置文件定义
- 架构设计文档（01_architecture_structure.md）
- 核心功能设计文档（02_core_features.md）
- 技术栈选型文档（03_tech_stack.md）

### 🚧 待实现
- [ ] internal/collector - OPC UA 客户端实现
- [ ] internal/engine - 核心引擎协调器
- [ ] internal/index - 稀疏索引实现
- [ ] internal/storage - Segment 文件管理
- [ ] internal/query - 查询迭代器和降采样
- [ ] internal/service - HTTP API 实现
- [ ] pkg/config - 配置加载逻辑
- [ ] pkg/logger - 日志封装

### 🎯 优先级
1. **高优先级**: storage 和 engine（核心存储引擎）
2. **中优先级**: collector 和 query（数据采集和查询）
3. **低优先级**: service 和工具（API 和 CLI）

---

## 📝 开发注意事项

### 性能优化
- 使用 `sync.Pool` 减少内存分配
- 批量写入减少磁盘 I/O
- 索引在内存中保持紧凑
- 压缩算法选择在 CPU 和磁盘 I/O 间平衡

### 安全考虑
- WAL 必须在数据写入 Buffer 前落盘
- Segment 文件采用 Append-Only 模式
- 崩溃恢复时验证数据完整性
- 配置文件权限控制

### 测试策略
- 单元测试覆盖核心逻辑
- 基准测试验证性能指标
- 集成测试验证完整流程
- 混沌测试验证崩溃恢复

---

## 🤝 贡献指南

1. Fork 本项目
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

### 提交规范
- feat: 新功能
- fix: 修复 bug
- docs: 文档更新
- style: 代码格式调整
- refactor: 重构
- test: 测试相关
- chore: 构建/工具链相关

---

## 📞 联系方式

- **项目地址**: https://github.com/Lwxjjr/Bitcask_IOT
- **问题反馈**: 请提交 GitHub Issue

---

## 📄 许可证

本项目采用 MIT 许可证 - 详见 LICENSE 文件