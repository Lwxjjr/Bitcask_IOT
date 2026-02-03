# 项目上下文文档 (AGENTS.md)

## 项目概述

这是一个基于 **Bitcask 模型** 的嵌入式时序存储引擎，专为工业物联网 (IIoT) 边缘计算场景设计。项目采用 Go 语言开发，旨在实现高性能的时序数据采集、存储和查询，支持 OPC UA 协议对接、时间窗口管理、数据降采样等核心功能。

**项目状态**: 设计规划阶段（尚未开始代码实现）

**核心目标**:
- 在边缘设备上实现高吞吐量的时序数据存储（目标 10w+ TPS）
- 支持 OPC UA 工业协议的数据订阅与采集
- 提供灵活的时间窗口管理和数据归档能力
- 实现数据降采样算法，降低边缘端带宽压力
- 提供命令行工具和 HTTP API 进行数据查询和运维管理

## 项目架构

### 系统分层架构

```
外部配置 (Config)
    ↓
pkg/config (Viper Loader)
    ↓
+---------------------+       +---------------------------+       +---------------------+
|  工业现场 / 模拟源  |       |  Edge Gateway (Go App)    |       |  运维 / 调试 CLI    |
| (Prosys Simulation) |       |                           |       | (kv-cli)            |
+----------+----------+       |  [Logger] (pkg/logger)    |       +----------+----------+
           |                  |  记录系统运行状态/错误    |                  ^
           | [1] Subscribe    +---------------------------+                  | [5] HTTP Query
           v                               |                                 |
+---------------------+       +------------+--------------+       +----------+----------+
|  [Ingestion Layer]  |       |  [Service Layer]          |       |  [Test Layer]       |
|  internal/collector |-----> |  internal/service         | <---- |  test/benchmark     |
|  (OPC UA Client)    | [2]   |  (HTTP API & Algo)        |       |  (压测脚本)         |
+---------------------+ Put   +------------+--------------+       +---------------------+
                                           | [3] Range Scan
                                           v
                      +------------------------------------------+
                      |  [Storage Engine Layer] (internal/engine)|
                      |  - Index: BTree (Key=ID+Time)            |
                      |  - Storage: Append-Only Files            |
                      +------------------------------------------+
```

### 目录结构规划

```
bitcask-iot/
├── cmd/
│   ├── server/               # 边缘网关主程序入口
│   └── cli/                  # 命令行运维工具
│
├── configs/                  # 配置文件目录
│   └── config.yaml           # OPC UA 地址、端口、存储路径等配置
│
├── internal/                 # 核心私有代码
│   ├── collector/            # 采集层：OPC UA 订阅与数据转换
│   ├── data/                 # 协议层：LogRecord 定义与编解码
│   ├── engine/               # 引擎层：DB 核心、Merge 逻辑
│   ├── index/                # 索引层：BTree 实现与 Iterator 接口
│   ├── storage/              # IO层：文件读写管理
│   └── service/              # 业务层：降采样算法与 HTTP Handler
│
├── pkg/                      # 公共库代码
│   ├── config/               # 配置管理：Viper 加载配置
│   ├── logger/               # 日志工具：Zap + 日志轮转
│   └── utils/                # 通用工具：时间转换、文件操作辅助
│
├── test/                     # 测试相关
│   ├── benchmark/            # 基准测试：压测脚本
│   └── mock/                 # Mock 数据生成
│
├── go.mod
└── README.md
```

## 核心功能模块

### 1. 时序 Key 设计
- **格式**: `SensorID (String) + Timestamp (BigEndian Uint64)`
- **目的**: 确保相同 SensorID 的记录在 BTree 索引中按时间紧凑排列
- **编码**: 使用 `binary.BigEndian` 保证字节序比较正确

### 2. 时间窗口切分
- **策略**: 按时间段切分数据文件（非传统按文件大小切分）
- **配置项**: `DataFileRotationInterval` (如 1 小时)
- **优势**: 便于按时间归档和清理旧数据，无需遍历记录
- **文件命名**: `data-{Timestamp}.vlog`

### 3. 数据降采样
- **接口**: `QueryRange(start, end, maxPoints)`
- **算法流程**:
  1. Seek 定位起始点
  2. Scan 扫描时间范围
  3. Buffer 缓存原始数据点
  4. Aggregate 聚合（LTTB 或 Simple Average）
  5. Return 返回特征点
- **应用**: 解决边缘端带宽瓶颈，"写入全量，读取特征"

### 4. 索引层适配
- **实现**: 使用 `google/btree` 替代原生 Map
- **接口方法**:
  - `Put(key, pos)`
  - `Get(key)`
  - `Iterator(reverse bool) IndexIterator`
- **能力**: 支持 `Ascend`/`Descend` 遍历，是降采样的基础

## 技术栈

### 核心语言
- **Go**: 1.19+ (利用泛型优化部分逻辑)

### 第三方依赖

| 库 | 用途 | 核心功能 |
|---|---|---|
| `github.com/gopcua/opcua` | OPC UA 协议对接 | 连接 Prosys Server、订阅模式、解析 Variant 数据 |
| `github.com/google/btree` | 内存索引 | 有序索引、支持范围查询、降采样基础 |
| `github.com/spf13/cobra` | 命令行工具 | CLI 参数解析、子命令管理 |
| `github.com/gin-gonic/gin` | HTTP 服务 | 轻量级 API 框架，适合嵌入式环境 |

### 标准库使用
- `encoding/binary`: Key 和 LogRecord 的 BigEndian/Varint 序列化
- `hash/crc32`: LogRecord 数据完整性校验 (Crash Safety)
- `sync`: 使用 `sync.RWMutex` 保证并发安全
- `golang.org/x/exp/mmap` (可选): 内存映射读取，提升读性能

### 配置与日志
- **配置管理**: `spf13/viper`
- **日志**: `uber-go/zap` (高性能) + `lumberjack` (日志切割)
  - 区分 INFO (正常采集) 和 ERROR (OPC 断连、CRC 校验失败)
  - 日志必须包含时间戳和文件定位

## 构建与运行

### 初始化命令
```bash
# 初始化模块
go mod init github.com/yourname/bitcask-iot

# 下载核心依赖
go get github.com/gopcua/opcua
go get github.com/google/btree
go get github.com/spf13/cobra
go get github.com/gin-gonic/gin
go get github.com/spf13/viper
go get go.uber.org/zap
go get gopkg.in/natefinch/lumberjack.v2
```

### 构建命令（待实现）
```bash
# 构建边缘网关服务
go build -o bin/server cmd/server/main.go

# 构建命令行工具
go build -o bin/cli cmd/cli/main.go
```

### 运行命令（待实现）
```bash
# 启动边缘网关
./bin/server -config configs/config.yaml

# 使用 CLI 查询数据
./bin/cli query --sensor-id "sensor1" --start 1640995200 --end 1641081600 --downsample 100
```

### 测试命令（待实现）
```bash
# 运行单元测试
go test ./...

# 运行基准测试
go test -bench=. -benchmem ./test/benchmark/
```

## 配置文件

### config.yaml 示例
```yaml
opc:
  endpoint: "opc.tcp://localhost:53530"
  node_ids: ["ns=3;i=1001", "ns=3;i=1002"]

storage:
  dir_path: "/tmp/bitcask-iot"
  data_file_size: 512MB
  sync_write: false
  rotation_interval: "1h"  # 时间窗口切分间隔

logger:
  level: "info"
  output: "./logs/bitcask-iot.log"
  max_size: 100
  max_backups: 3
  max_age: 7
```

## 开发约定

### 编码规范
- 严格遵循 Go 标准工程结构
- 通用工具放入 `pkg`，核心业务逻辑放入 `internal`
- 使用 `sync.RWMutex` 保证 Bitcask 引擎的并发安全
- 所有 Key 编码使用 BigEndian 字节序

### 测试实践
- 在 `test/benchmark` 目录编写性能测试脚本
- 测试场景包括：
  - Write Benchmark: 持续写入 100 万条 1KB 数据，计算 IOPS
  - Range Benchmark: 读取过去 1 小时数据，对比开启/关闭降采样的耗时差异

### 配置管理
- 不要在代码中硬编码路径
- 使用 `configs/config.yaml` 统一管理配置
- 支持运行时配置热加载（可选）

### 日志规范
- INFO 级别：正常采集流程
- ERROR 级别：OPC 断连、CRC 校验失败等异常
- 日志必须包含时间戳和文件定位信息

## 当前项目状态

**状态**: 规划设计阶段

**已完成**:
- ✅ 系统架构设计文档
- ✅ 核心功能设计文档
- ✅ 技术栈选型文档

**待实现**:
- ⏳ 项目初始化（go mod init）
- ⏳ 目录结构创建
- ⏳ 核心模块代码实现
- ⏳ 配置文件创建
- ⏳ 测试脚本编写
- ⏳ README 文档编写

## 文档说明

本目录包含以下设计文档：

1. **01_architecture_structure.md**: 详细的系统架构设计和目录结构规范
2. **02_core_features.md**: 核心功能实现指南（时间窗口、降采样、索引适配）
3. **03_tech_stack.md**: 技术栈选型和第三方依赖说明

这些文档为项目的实现提供了完整的技术指导和规范。