下面是整篇 CLAUDE.md 的中文翻译（保留原有结构和代码块，方便你直接替换或并排使用）：

---

# CLAUDE.md - hCache/iCache 项目文档

## 项目概览

**iCache**（Importance-Sampling-Informed Cache）是一个面向 I/O 受限的深度神经网络（DNN）训练场景的分布式缓存系统。它通过重要性采样、语义相似度以及访问模式来智能缓存训练数据，从而减少训练过程中的存储 I/O 瓶颈。

**已发表研究：** IEEE HPCA 2023 - [论文链接](https://ieeexplore.ieee.org/abstract/document/10070964/)

### 这个系统解决了什么问题？

DNN 训练在从存储系统（例如并行文件系统）加载训练数据时，往往受到 I/O 操作瓶颈的限制。iCache 通过以下方式缓解该问题：

* 智能缓存最「重要」的训练样本
* 支持多种缓存替换策略（LRU、LFU、ISA、Quiver、Semantic 等）
* 支持多计算节点的分布式训练
* 根据样本重要性动态调整缓存内容

## 高层架构

系统采用 **客户端-服务器架构**，包含三个主要组件：

```text
┌─────────────────────────────────────────────────────────────┐
│                    Training Nodes (Clients)                  │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  Modified PyTorch Framework (dcclient-python)          │ │
│  │  - Custom DataLoaders with cache-aware sampling        │ │
│  │  - gRPC client for cache server communication          │ │
│  │  - Importance sampling (ISA/SBP) algorithms            │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                           │ gRPC + REST
                           ▼
┌─────────────────────────────────────────────────────────────┐
│              Cache Server Cluster (dcserver-go)              │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  Cache Management Layer                                │ │
│  │  - LRU/LFU/ISA/Quiver/Semantic cache managers         │ │
│  │  - Importance-based eviction policies                 │ │
│  │  - Dynamic cache updates                              │ │
│  └────────────────────────────────────────────────────────┘ │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  Distributed Coordination (etcd)                       │ │
│  │  - Cluster membership management                       │ │
│  │  - Distributed state synchronization                  │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                           │
                           ▼
┌─────────────────────────────────────────────────────────────┐
│          Storage Layer (Parallel File System)                │
│              OrangeFS / NFS / Other PFS                      │
└─────────────────────────────────────────────────────────────┘
```

## 编程语言与技术栈

### 主要语言

* **Go (Golang)**：缓存服务器实现（dcserver-go）
* **Python**：客户端侧 PyTorch 集成（dcclient-python）

### 关键技术

* **gRPC**：客户端-服务器 RPC 通信（Protocol Buffers）
* **etcd**：分布式键值存储，用于集群协调
* **PyTorch**：深度学习框架（针对缓存感知训练做了修改）
* **Redis**：语义缓存实现（semanticCache/）
* **Beego**：Go Web 框架，用于 REST API
* **Memberlist**：基于 gossip 的集群成员管理

## 目录结构

```text
hCache/
├── deepcache-go/                 # 主代码目录
│   ├── dcclient-python/          # Python 客户端（修改后的 PyTorch）
│   │   ├── cached_image_classification.py  # 主训练脚本
│   │   ├── dcrpc/                # gRPC 协议定义
│   │   │   └── deepcache.proto   # Protocol Buffers 定义
│   │   ├── torch_cli/            # 自定义 PyTorch 修改
│   │   │   └── utils_data/       # 自定义 DataLoader 实现
│   │   ├── torchvision_datasets/ # 修改后的 torchvision 数据集
│   │   ├── models_cifar10/       # CIFAR-10 模型结构
│   │   ├── prep-1.8.py          # PyTorch 1.8 安装/补丁脚本
│   │   └── sbp.py               # Selective BackPropagation 实现
│   │
│   └── dcserver-go/              # 基于 Go 的缓存服务器
│       ├── dcserver.go           # 服务器入口
│       ├── go.mod                # Go 模块依赖
│       ├── conf/                 # 配置文件
│       │   └── dcserver.yaml     # 服务器主配置
│       ├── services/             # 核心缓存管理逻辑
│       │   ├── cachemodel.go     # 缓存数据结构
│       │   ├── data-structure.go # ISA/语义相关数据结构
│       │   ├── lru-cachemng.go   # LRU 缓存实现
│       │   ├── lfu-cachemng.go   # LFU 缓存实现
│       │   ├── isa-design1-cachemng.go      # ISA 缓存（重要性采样）
│       │   ├── quiver-cachemng.go           # Quiver 缓存实现
│       │   ├── semantic-cachemng.go         # 语义缓存基础实现
│       │   ├── semantic-design1-cachemng.go # 语义缓存实现 1
│       │   └── semantic-design2-cachemng.go # 语义缓存实现 2
│       ├── rpc/                  # gRPC 服务器实现
│       │   ├── deepcache.go      # RPC 服务注册
│       │   ├── deepcache_rpc_imple.go  # RPC 操作处理
│       │   └── cache/            # 生成的 protobuf 代码
│       ├── restful/              # REST API 端点
│       │   ├── rest.go           # REST 服务器初始化
│       │   ├── cache.go          # 缓存信息相关接口
│       │   ├── node.go           # 节点信息接口
│       │   └── cluster.go        # 集群信息接口
│       ├── distkv/               # 分布式 KV（etcd）集成
│       │   └── distkv.go         # etcd 客户端封装
│       ├── cluster/              # 集群成员管理
│       │   └── cluster.go        # Gossip 协议实现
│       ├── common/               # 通用工具
│       │   ├── configloader.go   # YAML 配置加载
│       │   ├── configmodel.go    # 配置数据模型
│       │   └── logloader.go      # 日志初始化
│       └── io/                   # 文件 I/O 操作
│
├── semanticCache/                # 语义缓存 Python 实现
│   ├── main.py                   # 主训练入口
│   ├── config.py                 # 配置参数
│   ├── requirements.txt          # Python 依赖
│   ├── data_loader/              # 自定义数据加载
│   │   ├── dataloader.py         # 缓存感知的 DataLoader
│   │   └── datasampler.py        # 基于重要性的采样器
│   ├── models/                   # DNN 模型结构
│   │   ├── resnet.py
│   │   ├── vgg.py
│   │   └── alexnet.py
│   ├── utils/                    # 工具函数
│   │   └── redis_wrapper.py      # Redis 缓存接口
│   ├── examples/                 # 训练示例脚本
│   │   └── train_cifar.py
│   └── tests/                    # 单元测试
│
├── logs/                         # 运行时日志目录
├── etcd_name0.etcd/             # etcd 数据目录
├── README.md                     # 原始项目 README
└── .gitignore                    # Git 忽略配置
```

## 核心架构组件

### 1. 缓存服务器（dcserver-go）

**入口文件：** `deepcache-go/dcserver-go/dcserver.go`

服务器初始化顺序：

1. **Common** – 解析配置并初始化日志
2. **Services** – 初始化各缓存管理器
3. **RESTful** – 启动 REST API 服务（端口 18283）
4. **RPC** – 启动 gRPC 服务（端口 18284）
5. **DistKV** – 连接 etcd 集群（端口 2379）

**支持的缓存策略：**

* **LRU**（Least Recently Used）– 经典最近最少使用策略
* **LFU**（Least Frequently Used）– 基于访问频率的淘汰
* **ISA**（Importance Sampling Aware）– 基于样本重要性维持缓存内容
* **Quiver** – 基于数据包（package）的缓存 + 预取
* **Semantic** – 使用 HNSW 的语义相似度缓存
* **CoordL** – 协同学习（Coordinated learning）缓存策略

### 2. 缓存管理接口

所有缓存管理器都实现 `cache_mng_interface` 接口：

```go
type cache_mng_interface interface {
    Exists(int64) bool                      // 检查样本是否在缓存中
    Access(int64) ([]byte, error)           // 获取缓存中的样本
    Insert(int64, []byte) error             // 插入样本到缓存
    Get_type() string                       // 返回缓存类型标识
    AccessAtOnce(int64) ([]byte, int64, bool) // 原子访问操作
    GetRmoteHit() int64                     // 获取远程命中统计
}
```

### 3. ISA 缓存设计

**关键创新：** 双堆（dual-heap）结构，用于动态重要性更新。

* **主堆（Main Heap）**：当前 epoch 使用的活动堆，用于缓存决策
* **影子堆（Shadow Heap）**：后台构建的堆（下一个 epoch 使用）
* 在 epoch 之间切换主堆和影子堆，实现在线重要性更新
* 使用带有可配置 worker 数量的构建通道（`async_building_task`）
* 可选的动态打包机制，用于处理低重要性样本

### 4. 语义缓存设计

**特性：**

* 多级缓存（重要性缓存 + 同质性缓存 Homophily Cache）
* 使用 HNSW 算法进行相似样本选择
* 基于 Redis 的分布式缓存（端口 6379、6380、6381）
* 在淘汰策略中同时考虑重要性与访问频率两个维度

### 5. Python 客户端集成

**修改后的 PyTorch 组件：**

* 自定义 `DataLoader`，支持缓存感知数据获取
* `ISASampler` 和 `SBPSampler`（Selective BackPropagation）用于基于重要性的采样
* gRPC 客户端与缓存服务器通信
* 集成脚本 `prep-1.8.py`：将修改后的文件软链接到 PyTorch 安装目录中

### 6. 分布式协调

**etcd** 提供：

* 集群节点发现
* 分布式缓存元数据同步
* 节点间共享配置

**Gossip 协议**（memberlist）：

* 节点健康检查
* 集群成员更新
* 配置广播

## 配置

### 服务器配置（`deepcache-go/dcserver-go/conf/dcserver.yaml`）

```yaml
# 节点配置
node: "10.0.16.20"              # 服务器 IP 地址
uuid: ""                         # 可选 UUID

# 端口配置
restport: 18283                  # REST API 端口
restadminport: 18285             # 管理 REST 端口
rpcport: 18284                   # gRPC 端口
gossipport: 18282                # 集群 gossip 端口

# 缓存配置
cache_ratio: 0.2                 # 缓存的数据集比例（0.0-1.0）
cache_type: "lru"                # 缓存策略: lru|lfu|isa|quiver|semantic|coordl

# 数据配置
dcserver_data_path: "/path/to/training/data"  # 训练数据集路径

# ISA 专用参数
async_building_task: 300         # 并发堆构建任务数量
package_design: true             # 是否启用低重要性样本的动态打包
loading_cache_size: 450          # 预取缓存大小

# 分布式设置
etcd_nodes: "10.0.16.20"        # 逗号分隔的 etcd 节点列表

# 开发相关
debug: false
enableadmin: false
```

### 语义缓存配置（`semanticCache/config.py`）

关键参数示例：

* `working_set_size`：缓存容量占数据集比例（如 0.1 = 10% 数据集）
* `replication_factor`：分布式缓存复制因子
* `hnsw_ef_construction`：HNSW 索引构建参数
* `hnsw_M`：HNSW 图的连接度参数
* `similarity_threshold`：语义相似度匹配阈值

## 构建、测试与开发命令

### 先决条件

**Go 环境：**

```bash
# 安装 Go 1.18+
wget https://go.dev/dl/go1.18.2.linux-amd64.tar.gz
rm -rf /usr/local/go && tar -C /usr/local -xzf go1.18.2.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
go env -w GO111MODULE=on
```

**Python 环境：**

```bash
# 创建 conda 环境
conda create -n icache python==3.9
conda activate icache

# 安装 PyTorch
pip3 install torch==1.8.0+cu111 torchvision==0.9.0+cu111 -f https://download.pytorch.org/whl/torch_stable.html

# 安装依赖
pip3 install six grpcio==1.46.1 grpcio-tools==1.46.1 requests==2.27.1
```

**etcd 安装：**

```bash
# 下载并安装 etcd 3.4.4
wget https://github.com/etcd-io/etcd/releases/download/v3.4.4/etcd-v3.4.4-linux-amd64.tar.gz
tar -xzf etcd-v3.4.4-linux-amd64.tar.gz
export PATH=$PATH:/path/to/etcd-v3.4.4-linux-amd64
```

### 初始化命令

**1. 修改 PyTorch 框架：**

```bash
cd deepcache-go/dcclient-python/
python3 prep-1.8.py
```

**2. 生成 gRPC 代码：**

```bash
cd deepcache-go/dcclient-python/dcrpc

# 生成 Go 代码
protoc \
  --go_out=Mgrpc/service_config/service_config.proto=/internal/proto/grpc_service_config:. \
  --go-grpc_out=Mgrpc/service_config/service_config.proto=/internal/proto/grpc_service_config:. \
  --go_opt=paths=source_relative \
  --go-grpc_opt=paths=source_relative \
  deepcache.proto

# 生成 Python 代码
python -m grpc_tools.protoc --python_out=. --grpc_python_out=. -I. deepcache.proto

# 服务器端同样生成
cd deepcache-go/dcserver-go/rpc/cache/
# 运行同样的 protoc 命令
```

**3. 配置服务器：**

```bash
cd deepcache-go/dcserver-go/conf
vim dcserver.yaml  # 编辑配置
```

### 运行系统

**单节点部署：**

```bash
# 终端 1：启动 etcd
rm -rf etcd_name0.etcd && \
etcd --name etcd_name0 \
  --listen-peer-urls http://{node_ip}:2380 \
  --initial-advertise-peer-urls http://{node_ip}:2380 \
  --listen-client-urls http://{node_ip}:2379 \
  --advertise-client-urls http://{node_ip}:2379 \
  --initial-cluster etcd_name0=http://{node_ip}:2380,

# 终端 2：启动缓存服务器
cd deepcache-go/dcserver-go
go run dcserver.go
# 或使用自定义配置: go run dcserver.go -config /path/to/config.yaml

# 终端 3：启动训练
cd deepcache-go/dcclient-python
CUDA_VISIBLE_DEVICES=0 python3 -u cached_image_classification.py \
  --arch resnet18 \
  --epochs 90 \
  -b 256 \
  --worker 2 \
  -p 20 \
  --num-classes 10 \
  --req-addr http://127.0.0.1:18283/ \
  --grpc-port {node_ip}:18284

# 使用 ISA 缓存时可加：
# --sbp --reuse-factor 3 --warm-up 5
```

**双节点分布式部署：**

```bash
# 节点 0：启动 etcd
rm -rf etcd_name0.etcd && \
etcd --name etcd_name0 \
  --listen-peer-urls http://{ip0}:2380 \
  --initial-advertise-peer-urls http://{ip0}:2380 \
  --listen-client-urls http://{ip0}:2379 \
  --advertise-client-urls http://{ip0}:2379 \
  --initial-cluster etcd_name0=http://{ip0}:2380,etcd_name1=http://{ip1}:2380,

# 节点 1：启动 etcd
rm -rf etcd_name1.etcd && \
etcd --name etcd_name1 \
  --listen-peer-urls http://{ip1}:2380 \
  --initial-advertise-peer-urls http://{ip1}:2380 \
  --listen-client-urls http://{ip1}:2379 \
  --advertise-client-urls http://{ip1}:2379 \
  --initial-cluster etcd_name0=http://{ip0}:2380,etcd_name1=http://{ip1}:2380,

# 每个节点分别启动缓存服务器
# 确保 yaml 配置一致

# 在每个节点启动训练客户端并设置不同 rank
# 节点 0：
CUDA_VISIBLE_DEVICES=0 python3 -u cached_image_classification.py \
  --multiprocessing-distributed \
  --world-size 2 --rank 0 --ngpus 1 \
  --dist-url 'tcp://{ip0}:1234' \
  --req-addr http://127.0.0.1:18283/ \
  --grpc-port {ip0}:18284 \
  # ... 其他参数

# 节点 1：
CUDA_VISIBLE_DEVICES=1 python3 -u cached_image_classification.py \
  --multiprocessing-distributed \
  --world-size 2 --rank 1 --ngpus 1 \
  --dist-url 'tcp://{ip0}:1234' \
  --req-addr http://127.0.0.1:18283/ \
  --grpc-port {ip1}:18284 \
  # ... 其他参数
```

**语义缓存部署：**

```bash
# 启动 Redis 服务
redis-server --port 6379  # 重要样本缓存
redis-server --port 6380  # 非重要样本缓存
redis-server --port 6381  # 比例信息共享

# 运行训练
cd semanticCache
python main.py [train_paths] \
  --network resnet18 \
  --batch_size 256 \
  --epochs 100 \
  --cache_training_data \
  --working_set_size 0.1 \
  --cache_nodes 2 \
  --host_ip <redis_host> \
  --port_num <redis_port> \
  --port_num2 <redis_port2>
```

### 开发相关命令

**构建服务器：**

```bash
cd deepcache-go/dcserver-go
go build -o dcserver dcserver.go
```

**运行测试：**

```bash
# Go 测试（若有）
cd deepcache-go/dcserver-go
go test ./...

# Python 测试
cd semanticCache
pytest tests/
```

**查看服务器状态：**

```bash
# REST API 端点
curl http://localhost:18283/cache      # 缓存统计
curl http://localhost:18283/node       # 节点信息
curl http://localhost:18283/cluster    # 集群信息
curl http://localhost:18283/statistic  # 性能统计
curl http://localhost:18283/mjob       # 多任务信息
```

## 重要的架构模式与设计决策

### 1. 客户端-服务器分离

* **动机**：将训练逻辑与缓存管理解耦
* **好处**：多个训练节点可以共享一组缓存服务器
* **权衡**：缓存未命中时会引入网络延迟

### 2. 修改版 PyTorch 集成

* **模式**：通过软链接方式对 PyTorch 内部进行 monkey-patching
* **位置**：`prep-1.8.py` 修改 `torch.utils.data` 与 `torchvision.datasets`
* **影响**：与 PyTorch 1.8.0 强绑定
* **风险**：版本升级容易导致兼容性问题

### 3. 双堆架构（ISA 缓存）

* **目的**：在不中断训练的前提下实现在线重要性更新
* **模式**：影子堆（shadow heap）/双缓冲模式
* **切换机制**：通过通道（`Switching_shadow_channel`）控制主堆与影子堆切换
* **并发**：使用 goroutine 池异步构建堆

### 4. 接口化的缓存管理器

* **模式**：策略模式（Strategy Pattern），用于封装不同缓存替换算法
* **好处**：方便增加新的缓存策略
* **位置**：`data-structure.go` 中的 `cache_mng_interface`
* **实现**：`cachemodel.go` 中使用类似工厂模式注册各策略

### 5. 使用 gRPC 进行数据传输

* **协议**：Protocol Buffers（`deepcache.proto`）
* **操作包括**：

  * `get_cache_info`：获取缓存元数据
  * `readimg_byidx`：按索引读取图像
  * `update_ivpersample`：更新样本重要性值
  * `refresh_server_ivpsample`：刷新所有样本的重要性
* **限制**：单条消息大小限制为 4GB

### 6. etcd 做分布式协调

* **用途**：集群状态的分布式 KV 存储
* **模式**：服务发现与配置共享
* **替代方案**：使用 gossip 协议（memberlist）管理节点成员
* **权衡**：etcd 提供强一致性，gossip 提供最终一致性

### 7. 重要性采样集成

* **Selective BackPropagation (SBP)**：只对重要样本进行反向传播
* **ISA**：使用重要性值引导缓存替换
* **更新流程**：客户端 → gRPC → 服务器 → 更新堆
* **更新频率**：重要性更新周期可配置

### 8. 多维度淘汰（语义缓存）

* **维度**：重要性值 + 访问频率
* **数据结构**：带自定义比较器的最小堆（min-heap）
* **模式**：多目标决策（multi-criteria decision making）
* **实现**：`data-structure.go` 中的 `indexheap`

### 9. 冷样本打包

* **功能**：`package_design` 标志控制低重要性样本的打包
* **目的**：降低低重要性样本对内存的占用
* **模式**：分层缓存（重要 vs 非重要）
* **比例**：通过 `us_ratio` 配置非重要样本比例

### 10. 日志与监控

* **日志库**：logrus（Go）、标准 logging（Python）
* **位置**：`logs/` 目录
* **REST 接口**：通过 HTTP 暴露实时统计信息
* **管理界面**：可选的 Beego 管理 dashboard

## 关键路径与数据流

### 训练数据访问流程

```text
1. PyTorch DataLoader 请求某个样本
   ↓
2. 自定义 Dataset 通过 gRPC 查询缓存
   ↓
3. 缓存服务器（dcserver-go）
   ├─ 缓存命中 → 直接返回缓存数据
   └─ 缓存未命中 → 从存储读取，写入缓存后返回
   ↓
4. 数据返回给 DataLoader
   ↓
5. 训练继续进行
   ↓
6. 每个 epoch 后：通过 gRPC 更新样本重要性值
```

### ISA 重要性更新流程

```text
1. 训练客户端计算样本重要性值
   ↓
2. 通过 update_ivpersample RPC 发送 JSON 形式的映射
   ↓
3. 服务器接收并解析更新数据
   ↓
4. 后台异步构建影子堆
   ↓
5. 在 epoch 边界：主堆与影子堆互换
   ↓
6. 缓存淘汰开始使用新的重要性值
```

## Git 状态与近期改动

**当前分支：** main

**已修改文件：**

* `.gitignore` - 更新忽略规则
* `deepcache-go/dcserver-go/rpc/deepcache_rpc_imple.go` - RPC 处理逻辑更新
* `deepcache-go/dcserver-go/services/cachemodel.go` - 缓存模型更新
* `deepcache-go/dcserver-go/services/data-structure.go` - 数据结构更新
* `deepcache-go/dcserver-go/services/isa-design1-cachemng.go` - ISA 缓存更新

**新增文件：**

* `deepcache-go/dcserver-go/services/semantic-cachemng.go` - 新的语义缓存基础实现
* `deepcache-go/dcserver-go/services/semantic-design2-cachemng.go` - 语义缓存变体实现
* `semanticCache/` - 完整的 Python 语义缓存实现

**未跟踪文件：**

* `deepcache-go/dcserver-go/services/semantic-design1-cachemng.go` - 另一种语义缓存变体

**近期提交：**

* e0b312a - 修改 .gitignore
* 67f7cb5 - 延迟监控功能（latency monitor）
* 20920d2 - 正常运行修复
* c3414d6 - 上传代码

## 如何在本代码库上工作

### 常见开发任务

**新增缓存策略：**

1. 在新文件（如 `newcache-cachemng.go`）中实现 `cache_mng_interface`
2. 添加对应的初始化函数（如 `init_newcache_cache_mng`）
3. 在 `cachemodel.go` 的 `all_cache_mng_init_funcs` 映射表中注册
4. 在 `dcserver.yaml` 中将 `cache_type` 设置为新的策略名

**修改缓存行为：**

* LRU/LFU：编辑 `lru-cachemng.go` 或 `lfu-cachemng.go`
* ISA：编辑 `isa-design1-cachemng.go`
* 语义缓存：编辑 `semantic-design1-cachemng.go` 或 `semantic-design2-cachemng.go`

**调试：**

* 在 `dcserver.yaml` 中将 `debug: true`
* 检查 `logs/` 目录中的日志
* 使用 REST 接口进行运行时状态查看
* 监控 etcd：`etcdctl get "" --prefix`（前提是安装了 etcdctl）

**性能调优：**

* `cache_ratio`：增大可提高缓存命中率（但需要更多内存）
* `async_building_task`：提高 worker 数量加快堆构建（消耗更多 CPU）
* `loading_cache_size`：预取缓存大小
* gRPC 消息大小：对于大图像需注意 protobuf 的消息大小限制

### 测试建议

**单元测试：**

* Go：在源文件旁编写 `*_test.go`
* Python：在 `semanticCache/tests/` 使用 pytest

**集成测试：**

* 使用测试配置启动服务器
* 在小数据集上运行训练
* 通过 REST API 验证缓存命中率

**性能测试：**

* 监控缓存命中/未命中比率
* 跟踪 I/O 延迟的下降情况
* 对比有缓存 / 无缓存下的训练时间

## 依赖

### Go 依赖（go.mod）

* `github.com/beego/beego/v2` – Web 框架
* `github.com/sirupsen/logrus` – 日志库
* `go.etcd.io/etcd/client/v3` – etcd 客户端
* `google.golang.org/grpc` – gRPC
* `google.golang.org/protobuf` – Protocol Buffers
* `github.com/hashicorp/memberlist` – Gossip 协议实现
* `github.com/jinzhu/configor` – YAML 配置解析

### Python 依赖（semanticCache/requirements.txt）

* `torch>=1.7.0` – PyTorch 框架
* `redis>=5.0.0` – Redis 客户端
* `hnswlib>=0.6.0` – HNSW 相似检索库
* `wandb>=0.12.0` – 实验追踪（Weights & Biases）
* `grpcio==1.46.1` – gRPC 框架
* `pytest>=6.0.0` – 测试框架

## 已知问题与注意事项

1. **PyTorch 版本锁定**：系统与 PyTorch 1.8.0 紧耦合
2. **存储系统要求**：需要并行文件系统（推荐 OrangeFS）
3. **网络要求**：分布式部署需要低延迟网络
4. **内存使用**：缓存大小必须在内存可承受范围内
5. **冷启动问题**：首个 epoch 缓存性能较差（需要预热）
6. **etcd 依赖**：即使是单节点部署，也必须运行 etcd

## 额外资源

* **原始论文**：IEEE HPCA 2023
* **OrangeFS 搭建文档**：[https://docs.orangefs.com/quickstart/quickstart-build/](https://docs.orangefs.com/quickstart/quickstart-build/)
* **etcd 文档**：[https://etcd.io/docs/](https://etcd.io/docs/)
* **gRPC Go 教程**：[https://grpc.io/docs/languages/go/](https://grpc.io/docs/languages/go/)
* **PyTorch 自定义数据集教程**：[https://pytorch.org/tutorials/beginner/data_loading_tutorial.html](https://pytorch.org/tutorials/beginner/data_loading_tutorial.html)

---

**最近更新日期**：2025-11-13
**项目仓库**：[https://github.com/ISCS-ZJU/iCache](https://github.com/ISCS-ZJU/iCache)

如果你之后想把这份翻译再精简成「给新人看的快速上手版（只保留架构 + 启动命令 + 常用配置）」我也可以帮你再压一版。
