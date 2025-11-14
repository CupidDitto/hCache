# CLAUDE.md - hCache/iCache Project Documentation

在我需要的时候，我会主动说和codex协作，你可以考虑调用codexmcp工具，否则禁止调用。

## Project Overview

**iCache** (Importance-Sampling-Informed Cache) is a distributed caching system designed to accelerate I/O-bound Deep Neural Network (DNN) model training. The system intelligently caches training data based on importance sampling, semantic similarity, and access patterns to reduce storage I/O bottlenecks during training.

**Published Research:** IEEE HPCA 2023 - [Paper Link](https://ieeexplore.ieee.org/abstract/document/10070964/)

### What Problem Does This Solve?

DNN training is often bottlenecked by I/O operations when loading training data from storage systems (e.g., parallel file systems). iCache addresses this by:
- Intelligently caching the most important training samples
- Supporting multiple cache replacement strategies (LRU, LFU, ISA, Quiver, Semantic)
- Enabling distributed training across multiple compute nodes
- Dynamically adjusting cache contents based on sample importance

## High-Level Architecture

The system uses a **client-server architecture** with three main components:

```
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

## Programming Languages & Technologies

### Primary Languages
- **Go (Golang)**: Cache server implementation (dcserver-go)
- **Python**: Client-side PyTorch integration (dcclient-python)

### Key Technologies
- **gRPC**: Client-server RPC communication (Protocol Buffers)
- **etcd**: Distributed key-value store for cluster coordination
- **PyTorch**: Deep learning framework (modified for cache-aware training)
- **Redis**: Semantic cache implementation (semanticCache/)
- **Beego**: Go web framework for REST API
- **Memberlist**: Gossip-based cluster membership

## Directory Structure

```
hCache/
├── deepcache-go/                 # Main codebase directory
│   ├── dcclient-python/          # Python client (modified PyTorch)
│   │   ├── cached_image_classification.py  # Main training script
│   │   ├── dcrpc/                # gRPC protocol definitions
│   │   │   └── deepcache.proto   # Protocol buffer definitions
│   │   ├── torch_cli/            # Custom PyTorch modifications
│   │   │   └── utils_data/       # Custom DataLoader implementations
│   │   ├── torchvision_datasets/ # Modified torchvision datasets
│   │   ├── models_cifar10/       # CIFAR-10 model architectures
│   │   ├── prep-1.8.py          # PyTorch 1.8 installation script
│   │   └── sbp.py               # Selective BackPropagation implementation
│   │
│   └── dcserver-go/              # Go-based cache server
│       ├── dcserver.go           # Main server entry point
│       ├── go.mod                # Go module dependencies
│       ├── conf/                 # Configuration files
│       │   └── dcserver.yaml     # Main server configuration
│       ├── services/             # Core cache management logic
│       │   ├── cachemodel.go     # Cache data structures
│       │   ├── data-structure.go # ISA/Semantic data structures
│       │   ├── lru-cachemng.go   # LRU cache implementation
│       │   ├── lfu-cachemng.go   # LFU cache implementation
│       │   ├── isa-design1-cachemng.go      # ISA cache (importance sampling)
│       │   ├── quiver-cachemng.go           # Quiver cache implementation
│       │   ├── semantic-cachemng.go         # Semantic cache base
│       │   ├── semantic-design1-cachemng.go # Semantic cache impl 1
│       │   └── semantic-design2-cachemng.go # Semantic cache impl 2
│       ├── rpc/                  # gRPC server implementation
│       │   ├── deepcache.go      # RPC service registration
│       │   ├── deepcache_rpc_imple.go  # RPC operation handlers
│       │   └── cache/            # Generated protobuf code
│       ├── restful/              # REST API endpoints
│       │   ├── rest.go           # REST server setup
│       │   ├── cache.go          # Cache info endpoints
│       │   ├── node.go           # Node info endpoints
│       │   └── cluster.go        # Cluster info endpoints
│       ├── distkv/               # Distributed KV (etcd) integration
│       │   └── distkv.go         # etcd client wrapper
│       ├── cluster/              # Cluster membership management
│       │   └── cluster.go        # Gossip protocol implementation
│       ├── common/               # Common utilities
│       │   ├── configloader.go   # YAML config loader
│       │   ├── configmodel.go    # Configuration data model
│       │   └── logloader.go      # Logging setup
│       └── io/                   # File I/O operations
│
├── semanticCache/                # Semantic cache Python implementation
│   ├── main.py                   # Main training entry point
│   ├── config.py                 # Configuration parameters
│   ├── requirements.txt          # Python dependencies
│   ├── data_loader/              # Custom data loaders
│   │   ├── dataloader.py         # Cache-aware dataloader
│   │   └── datasampler.py        # Importance-based sampler
│   ├── models/                   # DNN model architectures
│   │   ├── resnet.py
│   │   ├── vgg.py
│   │   └── alexnet.py
│   ├── utils/                    # Utility functions
│   │   └── redis_wrapper.py      # Redis cache interface
│   ├── examples/                 # Example training scripts
│   │   └── train_cifar.py
│   └── tests/                    # Unit tests
│
├── logs/                         # Runtime logs directory
├── etcd_name0.etcd/             # etcd data directory
├── README.md                     # Original project README
└── .gitignore                    # Git ignore patterns
```

## Key Architectural Components

### 1. Cache Server (dcserver-go)

**Entry Point:** `deepcache-go/dcserver-go/dcserver.go`

The server initializes in this order:
1. **Common** - Parse config and setup logging
2. **Services** - Initialize cache managers
3. **RESTful** - Start REST API server (port 18283)
4. **RPC** - Start gRPC server (port 18284)
5. **DistKV** - Connect to etcd cluster (port 2379)

**Supported Cache Strategies:**
- **LRU** (Least Recently Used) - Traditional cache replacement
- **LFU** (Least Frequently Used) - Frequency-based eviction
- **ISA** (Importance Sampling Aware) - Maintains cache based on sample importance values
- **Quiver** - Package-based cache with prefetching
- **Semantic** - Semantic similarity-based caching using HNSW (Hierarchical Navigable Small World)
- **CoordL** - Coordinated learning cache strategy

### 2. Cache Management Interface

All cache managers implement the `cache_mng_interface`:
```go
type cache_mng_interface interface {
    Exists(int64) bool                      // Check if sample is cached
    Access(int64) ([]byte, error)           // Retrieve cached sample
    Insert(int64, []byte) error             // Insert sample into cache
    Get_type() string                       // Return cache type
    AccessAtOnce(int64) ([]byte, int64, bool) // Atomic access operation
    GetRmoteHit() int64                     // Get remote hit statistics
}
```

### 3. ISA Cache Design

**Key Innovation:** Dual-heap architecture for dynamic importance updates

- **Main Heap**: Active heap used for cache decisions (current epoch)
- **Shadow Heap**: Background heap being built (next epoch)
- Heaps switch between epochs to enable online importance updates
- Building channel with configurable worker pool size (async_building_task)
- Optional dynamic packaging for unimportant samples

### 4. Semantic Cache Design

**Features:**
- Multi-level caching (Importance Cache + Homophily Cache)
- Uses HNSW algorithm for similarity-based sample selection
- Redis-based distributed cache (ports 6379, 6380, 6381)
- Supports both importance and frequency dimensions for eviction

### 5. Python Client Integration

**Modified PyTorch Components:**
- Custom `DataLoader` with cache-aware fetching
- `ISASampler` and `SBPSampler` for importance-based sampling
- gRPC client for communicating with cache servers
- Integration script (`prep-1.8.py`) that symlinks modified files into PyTorch installation

### 6. Distributed Coordination

**etcd** provides:
- Cluster node discovery
- Distributed cache metadata synchronization
- Configuration sharing across nodes

**Gossip Protocol** (memberlist):
- Node health monitoring
- Cluster membership updates
- Configuration broadcast

## Configuration

### Server Configuration (`deepcache-go/dcserver-go/conf/dcserver.yaml`)

```yaml
# Node configuration
node: "10.0.16.20"              # Server IP address
uuid: ""                         # Optional UUID

# Port settings
restport: 18283                  # REST API port
restadminport: 18285             # Admin REST port
rpcport: 18284                   # gRPC port
gossipport: 18282                # Cluster gossip port

# Cache configuration
cache_ratio: 0.2                 # Percentage of dataset to cache (0.0-1.0)
cache_type: "lru"                # Cache strategy: lru|lfu|isa|quiver|semantic|coordl

# Data configuration
dcserver_data_path: "/path/to/training/data"  # Path to training dataset

# ISA-specific parameters
async_building_task: 300         # Number of concurrent heap building tasks
package_design: true             # Enable dynamic packaging for unimportant samples
loading_cache_size: 450          # Size of loading cache for prefetching

# Distributed setup
etcd_nodes: "10.0.16.20"        # Comma-separated list of etcd nodes

# Development
debug: false
enableadmin: false
```

### Semantic Cache Configuration (`semanticCache/config.py`)

Key parameters:
- `working_set_size`: Cache size ratio (e.g., 0.1 = 10% of dataset)
- `replication_factor`: Data replication factor for distributed cache
- `hnsw_ef_construction`: HNSW index construction parameter
- `hnsw_M`: HNSW index connectivity parameter
- `similarity_threshold`: Threshold for semantic similarity matching

## Build, Test, and Development Commands

### Prerequisites

**Go Environment:**
```bash
# Install Go 1.18+
wget https://go.dev/dl/go1.18.2.linux-amd64.tar.gz
rm -rf /usr/local/go && tar -C /usr/local -xzf go1.18.2.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
go env -w GO111MODULE=on
```

**Python Environment:**
```bash
# Create conda environment
conda create -n icache python==3.9
conda activate icache

# Install PyTorch
pip3 install torch==1.8.0+cu111 torchvision==0.9.0+cu111 -f https://download.pytorch.org/whl/torch_stable.html

# Install dependencies
pip3 install six grpcio==1.46.1 grpcio-tools==1.46.1 requests==2.27.1
```

**etcd Installation:**
```bash
# Download and install etcd 3.4.4
wget https://github.com/etcd-io/etcd/releases/download/v3.4.4/etcd-v3.4.4-linux-amd64.tar.gz
tar -xzf etcd-v3.4.4-linux-amd64.tar.gz
export PATH=$PATH:/path/to/etcd-v3.4.4-linux-amd64
```

### Setup Commands

**1. Modify PyTorch Framework:**
```bash
cd deepcache-go/dcclient-python/
python3 prep-1.8.py
```

**2. Generate gRPC Code:**
```bash
cd deepcache-go/dcclient-python/dcrpc

# Generate Go code
protoc \
  --go_out=Mgrpc/service_config/service_config.proto=/internal/proto/grpc_service_config:. \
  --go-grpc_out=Mgrpc/service_config/service_config.proto=/internal/proto/grpc_service_config:. \
  --go_opt=paths=source_relative \
  --go-grpc_opt=paths=source_relative \
  deepcache.proto

# Generate Python code
python -m grpc_tools.protoc --python_out=. --grpc_python_out=. -I. deepcache.proto

# Repeat for server-side
cd deepcache-go/dcserver-go/rpc/cache/
# Run same protoc commands
```

**3. Configure Server:**
```bash
cd deepcache-go/dcserver-go/conf
vim dcserver.yaml  # Edit configuration
```

### Running the System

**Single Node Setup:**

```bash
# Terminal 1: Start etcd
rm -rf etcd_name0.etcd && \
etcd --name etcd_name0 \
  --listen-peer-urls http://{node_ip}:2380 \
  --initial-advertise-peer-urls http://{node_ip}:2380 \
  --listen-client-urls http://{node_ip}:2379 \
  --advertise-client-urls http://{node_ip}:2379 \
  --initial-cluster etcd_name0=http://{node_ip}:2380,

# Terminal 2: Start cache server
cd deepcache-go/dcserver-go
go run dcserver.go
# Or with custom config: go run dcserver.go -config /path/to/config.yaml

# Terminal 3: Start training
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

# For ISA cache, add these flags:
# --sbp --reuse-factor 3 --warm-up 5
```

**Distributed Setup (2 Nodes):**

```bash
# Node 0: Start etcd
rm -rf etcd_name0.etcd && \
etcd --name etcd_name0 \
  --listen-peer-urls http://{ip0}:2380 \
  --initial-advertise-peer-urls http://{ip0}:2380 \
  --listen-client-urls http://{ip0}:2379 \
  --advertise-client-urls http://{ip0}:2379 \
  --initial-cluster etcd_name0=http://{ip0}:2380,etcd_name1=http://{ip1}:2380,

# Node 1: Start etcd
rm -rf etcd_name1.etcd && \
etcd --name etcd_name1 \
  --listen-peer-urls http://{ip1}:2380 \
  --initial-advertise-peer-urls http://{ip1}:2380 \
  --listen-client-urls http://{ip1}:2379 \
  --advertise-client-urls http://{ip1}:2379 \
  --initial-cluster etcd_name0=http://{ip0}:2380,etcd_name1=http://{ip1}:2380,

# Start cache servers on each node
# Make sure yaml files are identical

# Start training clients on each node with different ranks
# Node 0:
CUDA_VISIBLE_DEVICES=0 python3 -u cached_image_classification.py \
  --multiprocessing-distributed \
  --world-size 2 --rank 0 --ngpus 1 \
  --dist-url 'tcp://{ip0}:1234' \
  --req-addr http://127.0.0.1:18283/ \
  --grpc-port {ip0}:18284 \
  # ... other args

# Node 1:
CUDA_VISIBLE_DEVICES=1 python3 -u cached_image_classification.py \
  --multiprocessing-distributed \
  --world-size 2 --rank 1 --ngpus 1 \
  --dist-url 'tcp://{ip0}:1234' \
  --req-addr http://127.0.0.1:18283/ \
  --grpc-port {ip1}:18284 \
  # ... other args
```

**Semantic Cache Setup:**

```bash
# Start Redis servers
redis-server --port 6379  # Important samples cache
redis-server --port 6380  # Unimportant samples cache
redis-server --port 6381  # Ratio information sharing

# Run training
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

### Development Commands

**Build Server:**
```bash
cd deepcache-go/dcserver-go
go build -o dcserver dcserver.go
```

**Run Tests:**
```bash
# Go tests (if available)
cd deepcache-go/dcserver-go
go test ./...

# Python tests
cd semanticCache
pytest tests/
```

**View Server Status:**
```bash
# REST API endpoints
curl http://localhost:18283/cache      # Cache statistics
curl http://localhost:18283/node       # Node information
curl http://localhost:18283/cluster    # Cluster information
curl http://localhost:18283/statistic  # Performance statistics
curl http://localhost:18283/mjob       # Multi-job information
```

## Important Architectural Patterns & Design Decisions

### 1. Client-Server Separation
- **Rationale**: Decouples training logic from cache management
- **Benefit**: Multiple training nodes can share cache servers
- **Trade-off**: Network latency for cache misses

### 2. Modified PyTorch Integration
- **Pattern**: Monkey-patching PyTorch internals via symlinks
- **Location**: `prep-1.8.py` modifies torch.utils.data and torchvision.datasets
- **Implication**: Tightly coupled to PyTorch 1.8.0 version
- **Risk**: Fragile - breaks with PyTorch updates

### 3. Dual-Heap Architecture (ISA Cache)
- **Purpose**: Enable online importance updates without stopping training
- **Pattern**: Shadow heap pattern (double buffering)
- **Switching**: Controlled via channel communication (`Switching_shadow_channel`)
- **Concurrency**: Async building tasks via goroutine pool

### 4. Interface-Based Cache Managers
- **Pattern**: Strategy pattern for cache replacement algorithms
- **Benefit**: Easy to add new cache strategies
- **Location**: `cache_mng_interface` in `data-structure.go`
- **Implementation**: Reflection-based factory pattern in `cachemodel.go`

### 5. gRPC for Data Transfer
- **Protocol**: Protocol Buffers (deepcache.proto)
- **Operations**:
  - `get_cache_info`: Get cache metadata
  - `readimg_byidx`: Fetch image by index
  - `update_ivpersample`: Update importance values
  - `refresh_server_ivpsample`: Refresh all importance values
- **Constraint**: Data size limited to 4GB per message

### 6. etcd for Distributed Coordination
- **Usage**: Distributed KV store for cluster state
- **Pattern**: Service discovery and configuration sharing
- **Alternative**: Gossip protocol (memberlist) for node membership
- **Trade-off**: Strong consistency (etcd) vs eventual consistency (gossip)

### 7. Importance Sampling Integration
- **Selective BackPropagation (SBP)**: Only backprop on important samples
- **ISA**: Importance values guide cache replacement
- **Update Flow**: Client → gRPC → Server → Update heap
- **Frequency**: Configurable importance update interval

### 8. Multi-Dimensional Eviction (Semantic Cache)
- **Dimensions**: Importance value + Access frequency
- **Data Structure**: Min-heap with custom comparator
- **Pattern**: Multi-criteria decision making
- **Implementation**: `indexheap` in `data-structure.go`

### 9. Packaging for Cold Samples
- **Feature**: `package_design` flag enables unimportant sample packaging
- **Purpose**: Reduce memory overhead for low-importance samples
- **Pattern**: Tiered caching (important vs unimportant)
- **Ratio**: Configurable via `us_ratio` (unimportant sample ratio)

### 10. Logging and Monitoring
- **Library**: logrus (Go), standard logging (Python)
- **Location**: `logs/` directory
- **REST Endpoints**: Real-time statistics via HTTP
- **Admin UI**: Optional Beego admin dashboard

## Critical Paths & Data Flow

### Training Data Access Flow

```
1. PyTorch DataLoader requests sample
   ↓
2. Custom Dataset checks cache via gRPC
   ↓
3. Cache Server (dcserver-go)
   ├─ Cache Hit → Return cached data
   └─ Cache Miss → Read from storage, cache, return
   ↓
4. Data returned to DataLoader
   ↓
5. Training proceeds
   ↓
6. After epoch: Update importance values via gRPC
```

### Importance Update Flow (ISA)

```
1. Training client computes importance values
   ↓
2. Send update_ivpersample RPC with JSON map
   ↓
3. Server receives and parses updates
   ↓
4. Build shadow heap asynchronously
   ↓
5. At epoch boundary: Switch main ↔ shadow heaps
   ↓
6. Cache eviction uses new importance values
```

## Git Status & Recent Changes

**Current Branch:** main

**Modified Files:**
- `.gitignore` - Updated ignore patterns
- `deepcache-go/dcserver-go/rpc/deepcache_rpc_imple.go` - RPC handler updates
- `deepcache-go/dcserver-go/services/cachemodel.go` - Cache model changes
- `deepcache-go/dcserver-go/services/data-structure.go` - Data structure updates
- `deepcache-go/dcserver-go/services/isa-design1-cachemng.go` - ISA cache updates

**Added Files:**
- `deepcache-go/dcserver-go/services/semantic-cachemng.go` - New semantic cache base
- `deepcache-go/dcserver-go/services/semantic-design2-cachemng.go` - Semantic cache variant
- `semanticCache/` - Complete semantic cache Python implementation

**Untracked:**
- `deepcache-go/dcserver-go/services/semantic-design1-cachemng.go` - Another variant

**Recent Commits:**
- e0b312a - modify gitignore
- 67f7cb5 - latency monitor feature
- 20920d2 - normal running fix
- c3414d6 - upload code

## Working with This Codebase

### Common Development Tasks

**Adding a New Cache Strategy:**
1. Implement `cache_mng_interface` in new file (e.g., `newcache-cachemng.go`)
2. Add initialization function (e.g., `init_newcache_cache_mng`)
3. Register in `all_cache_mng_init_funcs` map in `cachemodel.go`
4. Update `dcserver.yaml` to use new cache type

**Modifying Cache Behavior:**
- LRU/LFU: Edit `lru-cachemng.go` or `lfu-cachemng.go`
- ISA: Edit `isa-design1-cachemng.go`
- Semantic: Edit `semantic-design1-cachemng.go` or `semantic-design2-cachemng.go`

**Debugging:**
- Set `debug: true` in `dcserver.yaml`
- Check logs in `logs/` directory
- Use REST endpoints for runtime inspection
- Monitor etcd: `etcdctl get "" --prefix` (if etcdctl installed)

**Performance Tuning:**
- `cache_ratio`: Increase for more caching (memory trade-off)
- `async_building_task`: More workers = faster heap building (CPU trade-off)
- `loading_cache_size`: Prefetch buffer size
- gRPC message size: Check protobuf limits for large images

### Testing Recommendations

**Unit Testing:**
- Go: Create `*_test.go` files alongside source
- Python: Use pytest in `semanticCache/tests/`

**Integration Testing:**
- Start server with test configuration
- Run small dataset training
- Verify cache hit rates via REST API

**Performance Testing:**
- Monitor cache hit/miss ratios
- Track I/O latency reduction
- Compare training time with/without cache

## Dependencies

### Go Dependencies (go.mod)
- `github.com/beego/beego/v2` - Web framework
- `github.com/sirupsen/logrus` - Logging
- `go.etcd.io/etcd/client/v3` - etcd client
- `google.golang.org/grpc` - gRPC framework
- `google.golang.org/protobuf` - Protocol buffers
- `github.com/hashicorp/memberlist` - Gossip protocol
- `github.com/jinzhu/configor` - YAML config parsing

### Python Dependencies (semanticCache/requirements.txt)
- `torch>=1.7.0` - PyTorch framework
- `redis>=5.0.0` - Redis client
- `hnswlib>=0.6.0` - HNSW similarity search
- `wandb>=0.12.0` - Experiment tracking
- `grpcio==1.46.1` - gRPC framework
- `pytest>=6.0.0` - Testing framework

## Known Issues & Considerations

1. **PyTorch Version Lock**: System is tightly coupled to PyTorch 1.8.0
2. **Storage System**: Requires parallel file system (OrangeFS recommended)
3. **Network Requirements**: Low-latency network for distributed setup
4. **Memory Usage**: Cache size must fit in RAM
5. **Cold Start**: First epoch has poor cache performance (warming up)
6. **etcd Dependency**: Must run etcd even for single-node setup

## Additional Resources

- **Original Paper**: IEEE HPCA 2023
- **OrangeFS Setup**: https://docs.orangefs.com/quickstart/quickstart-build/
- **etcd Documentation**: https://etcd.io/docs/
- **gRPC Go Tutorial**: https://grpc.io/docs/languages/go/
- **PyTorch Custom Datasets**: https://pytorch.org/tutorials/beginner/data_loading_tutorial.html

---

**Last Updated**: 2025-11-13  
**Project Repository**: https://github.com/ISCS-ZJU/iCache
