# mini-datakit

一个用 Go 从零写的迷你可观测性采集端。目标是搞清楚观测云 DataKit 那类 Agent
**内部到底怎么设计的**：插件怎么注册、数据模型长什么样、counter 怎么变成使用率、
数据怎么攒批上传、存储怎么按时间切分。

不是玩具 demo：每个包都有单测，改动前先让测试变绿。

## 目录结构

```
mini-datakit/
├── cmd/agent/            采集端主程序：读配置、起插件、按周期采集
├── internal/
│   ├── point/            统一数据模型 Point + line protocol 序列化
│   └── inputs/           采集插件注册表
│       └── cpu/          CPU 采集插件（读 /proc/stat，算增量）
├── configs/agent.json    采集配置
├── scripts/verify.sh     提交前一键检查
└── Makefile
```

## 快速开始

### 1. 装 Go

到 <https://go.dev/dl/> 下载 Windows 的 `.msi`，一路下一步。装完**重开一个终端**验证：

```powershell
go version      # 期望看到 go1.22 或更高
```

### 2. 跑测试

```powershell
cd mini-datakit
go test ./...
```

期望输出：

```
ok      mini-datakit/internal/inputs
ok      mini-datakit/internal/inputs/cpu
ok      mini-datakit/internal/point
```

### 3. 跑采集端

**Windows 上跑不了**，因为 `/proc/stat` 不存在（这不是 bug，见下面的"跨平台说明"）。
在 WSL / Linux 里：

```bash
go run ./cmd/agent -config configs/agent.json
```

输出长这样（每 2 秒一行，每条是 InfluxDB line protocol 格式）：

```
cpu,core=cpu,host=myhost usage=3.5,user=2.1,system=1.2,iowait=0.1 1730000000000000000
cpu,core=cpu0,host=myhost usage=4.0,user=2.4,system=1.5,iowait=0 1730000000000000000
```

> 第一轮不会有任何输出，因为还没有上一次的采样值可以相减。这是 counter 型指标的
> 正常行为，不是坏了。

## 跨平台说明

`cpu` 插件读的是 Linux 的 `/proc/stat`。Windows 上没有这个文件，所以：

- **单测在 Windows 上完全能跑**（测试用伪造的 /proc/stat 文本，不依赖真机）。
- **要在 Windows 上拿到真实 CPU 数据**，装 WSL2 + Ubuntu，在里面跑。

## 进度

| 阶段 | 内容 | 状态 |
|---|---|---|
| W1 | 数据模型 + 插件注册 + cpu 采集 + 单测 | ✅ |
| W2 | 并发采集池 + 背压 + `-race` 全绿 | ⬜ |
| W3 | server 端接收 + HTTP 中间件 | ⬜ |
| W4 | 时间分片存储 + 二分索引 | ⬜ |
| W5 | 类 DQL 查询子集 + 聚合 | ⬜ |
| W6 | pprof 火焰图找热点并优化 | ⬜ |
| W7 | 数据打进观测云控制台 | ⬜ |
| W8 | AGENTS.md + Skills + 一键验证体系 | 🚧 |

## 为什么这么设计

- **Point 是唯一契约**：采集器只产出 Point，下游只消费 Point。加一个新采集器不用改任何已有代码。
- **插件靠 `init()` 注册**：主程序 `import _ ".../cpu"` 一行就能挂上一个插件。这是观测云 DataKit 的做法。
- **解析与 IO 分离**：`parseStat(io.Reader)` 能拿字符串直接测，`Collect` 只负责开文件。这样测试不需要真的 Linux 机器。
- **输出排序后再拼接**：Go 的 map 遍历顺序随机，不排序就会写出"每次结果都不一样"的测试。
