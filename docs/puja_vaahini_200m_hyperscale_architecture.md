# Puja Vaahini Hyperscale Architecture: 200 Million (2000 Lakh) Active Users / 12-Hour Peak Capacity Plan

## 1. Mathematical Capacity & Workload Modeling

### 1.1 Volume Breakdown
- **Target Active Users**: 2,000 Lakh = **200,000,000 (200 Million / 20 Crore)** active devotees, priests, and vendors.
- **Operating Window**: 12-Hour Continuous Peak Festival Day (e.g., Mahashivaratri, Ganesh Chaturthi, Varalakshmi, Pushkaralu).
- **Time Window**: 12 hours = $12 \times 3,600 = 43,200\text{ seconds}$.

### 1.2 Quantitative Traffic & Hardware Dimensioning

| Metric | Calculation & Baseline | Peak Scaling Requirement | Architectural Defense |
| :--- | :--- | :--- | :--- |
| **Concurrent Connected Users (CCU)** | 200M users distributed with 20%–25% peak concurrency | **40,000,000 – 50,000,000 simultaneous active connections** | 800+ API Gateway Pods (Go netpoll), stateless sessions, connection reuse |
| **Gross Ingress Request Rate (RPS)** | Avg 1 request per user every 10–15 seconds during active ritual booking | **3,000,000 – 5,000,000 RPS sustained** (peaking at **8,000,000 RPS**) | Multi-CDN Edge Termination (Cloudflare/Akamai) absorbing 92% of reads |
| **Net Origin Gateway Throughput** | 8M RPS gross with 92% edge cache hit ratio | **640,000 – 800,000 RPS hitting Go backend cluster** | Autoscaling Kubernetes cluster across 3 AWS/GCP Availability Zones |
| **Network Egress Bandwidth** | 8M RPS $\times$ 2.5 KB avg JSON response payload | **160 Gbps** application data (excluding video) | Anycast BGP Multi-CDN edge routing with Brotli compression |
| **Live Temple Broadcast Egress** | 1% of users (2M devotees) streaming live sanctum video at 1.5 Mbps | **3.0 Terabits per second (Tbps)** video egress | **Hybrid WebRTC + LL-HLS CDN**: Priest WebRTC feeds into global HLS CDN |
| **Spatial Radar Ingestion** | 50,000 active priests pinging telemetry every 5s | **10,000 spatial writes/sec** | Sharded Redis GEO memory ring; write-behind async PostGIS sync |
| **Database Transaction Rate (TPS)** | Writes: Bookings, Escrow deposits, OTPs, Cancellations | **80,000 – 120,000 write TPS** | **Kafka Write-Behind Buffer** + **Citus Distributed PostgreSQL (16 Shards)** |
| **Distributed Memory Footprint** | Active sessions, slot locks, GEO rings, cache | **1.8 Terabytes RAM** across Redis Cluster | 128-node Redis Enterprise Cluster (Hash-slot partitioned) |

---

## 2. Hyperscale Multi-Region Topology with High-Availability Replica Sets

```mermaid
flowchart TD
    subgraph Users [200M Active Clients]
        DA["📱 195M Devotees (Mobile Apps)"]
        PA["📱 2M Priests (GPS & Live Video)"]
        VA["🏪 1M Store Vendors (Orders)"]
        TCMS["🖥️ 2M Pilgrims & Temple CMS"]
    end

    subgraph GlobalEdge [Global Edge & Anycast CDN Tier - Absorbs 92% Load]
        ANYCAST["🌐 Anycast BGP DNS Routing"]
        CDN_EDGE["⚡ Multi-CDN Edge Network (Cloudflare + Akamai + CloudFront)<br/>Panchangam, Catalog, Temples, i18n Strings (Cached at Edge)"]
        WAF_SHIELD["🛡️ Volumetric DDoS Shield & Layer 7 Rate Limiter (Token Bucket)"]
        QUEUE_IT["🚪 Virtual Waiting Room (Cloudflare Waiting Room / Redis Token Queue)<br/>Engages during sudden 10:00 AM VIP Flash Ticketing bursts"]
    end

    subgraph L4L7Ingress [Multi-Region Load Balancing Tier]
        NLB["AWS Network Load Balancers (L4 NLB)<br/>TCP: 8080/443 | UDP: 3478 (WebRTC TURN)"]
        ENVOY["Envoy Proxy Fleet (Layer 7)<br/>mTLS, gRPC, WebSocket Multiplexing, HTTP/3 QUIC"]
    end

    subgraph K8sReplicaSets [Kubernetes Stateless ReplicaSets Fleet (800+ Pods)]
        direction TB
        GW_RS["🚪 API Gateway ReplicaSet (800 Pods)<br/>Stateless JWT verification, 50k conns/pod"]
        DISPATCH_RS["⚡ Spatial Dispatch ReplicaSet (150 Pods)<br/>Sub-5ms Rapido scoring against Redis GEO"]
        TICKET_RS["🎟️ Flash Ticketing ReplicaSet (100 Pods)<br/>Zero-lock atomic Lua reservation"]
        CHAT_RS["💬 Distributed Chat Gateway ReplicaSet (300 Pods)<br/>WebSocket routing over Redis Pub/Sub"]
        MEDIA_RS["📹 WebRTC Ingestion & Transcoding ReplicaSet (100 Pods)<br/>H.264/AV1 transcode -> HLS Packager"]
    end

    subgraph RedisReplicaSets [128-Node Sharded Redis Enterprise Replica Sets (1 Primary + 2 Replicas per Shard)]
        R_GEO_RS["📍 Redis GEO Replica Sets (16 Shards x 3 Nodes)<br/>Priest GPS telemetry & spatial radius"]
        R_SLOTS_RS["🔒 Redis Redlock & Quota Replica Sets (32 Shards x 3 Nodes)<br/>Flash VIP tickets & Priest availability slots"]
        R_PUBSUB_RS["📢 Redis Pub/Sub Fabric Replica Sets (32 Shards x 3 Nodes)<br/>Inter-gateway WebSocket message routing"]
        R_CACHE_RS["🗄️ Redis LRU Cache Replica Sets (48 Shards x 3 Nodes)<br/>Active bookings, user profiles, gotram records"]
    end

    subgraph KafkaReplicaSets [Apache Kafka Distributed Streaming Replica Sets]
        KAFKA{{"🐘 50-Broker Kafka Cluster (Topic Partition Replication Factor = 3, min.insync.replicas = 2)"}}
        CONSUMERS["⚙️ Asynchronous Batch Ingestion Workers<br/>(Bulk 5,000-row micro-batching into DB)"]
    end

    subgraph PostgresReplicaSet [PostgreSQL 16 Multi-AZ High-Availability Replica Set (Patroni + etcd)]
        CITUS_COORD["🐘 Citus Distributed Coordinator (Active-Passive HA with Sync Replica)"]
        PGBOUNCER["⚡ PgBouncer Pooler Fleet (50,000 client conns -> 200 DB conns)"]
        
        subgraph ShardReplicaSets [16 Citus Worker Shards with Multi-AZ Replication]
            direction LR
            SHARD1_P[("Shard 1 Primary (AZ-A)")] -.->|Sync Rep| SHARD1_S[("Shard 1 Standby (AZ-B)")]
            SHARD2_P[("Shard 2 Primary (AZ-B)")] -.->|Sync Rep| SHARD2_S[("Shard 2 Standby (AZ-C)")]
            SHARD3_P[("Shard 3 Primary (AZ-C)")] -.->|Sync Rep| SHARD3_S[("Shard 3 Standby (AZ-A)")]
            SHARD4_P[("Shard 4 Primary (AZ-A)")] -.->|Sync Rep| SHARD4_S[("Shard 4 Standby (AZ-B)")]
        end

        subgraph ReadPoolReplicaSet [Read Replica Pool (Asynchronous Streaming Replication)]
            READ_REP1[("Read Replica 1 (Devotee Radar)")]
            READ_REP2[("Read Replica 2 (Catalog & Temples)")]
            READ_REP3[("Read Replica 3 (Temple CMS Reports)")]
            READ_REP4[("Read Replica 4 (Audit & Analytics)")]
        end
    end

    subgraph VideoCDN [Live Video Streaming Tier]
        SFU["LiveKit / Pion SFU Cluster"]
        HLS_ORIGIN["HLS Origin Packager"]
        VIDEO_CDN["Global Video CDN (3.0 Tbps Broadcast Egress)"]
    end

    Users --> ANYCAST
    ANYCAST --> CDN_EDGE
    CDN_EDGE --> WAF_SHIELD
    WAF_SHIELD --> QUEUE_IT
    QUEUE_IT --> NLB
    NLB --> ENVOY
    ENVOY --> GW_RS

    GW_RS --> DISPATCH_RS & TICKET_RS & CHAT_RS & MEDIA_RS

    DISPATCH_RS <--> R_GEO_RS
    TICKET_RS <--> R_SLOTS_RS
    CHAT_RS <--> R_PUBSUB_RS
    GW_RS <--> R_CACHE_RS

    TICKET_RS --> KAFKA
    GW_RS --> KAFKA
    KAFKA --> CONSUMERS
    CONSUMERS --> PGBOUNCER

    PGBOUNCER --> CITUS_COORD
    CITUS_COORD --> SHARD1_P & SHARD2_P & SHARD3_P & SHARD4_P
    SHARD1_P & SHARD2_P & SHARD3_P & SHARD4_P -.->|Async Streaming Replication| ReadPoolReplicaSet

    MEDIA_RS --> SFU
    SFU --> HLS_ORIGIN
    HLS_ORIGIN --> VIDEO_CDN
    VIDEO_CDN --> DA & TCMS
```

---

## 3. High-Availability Replica Set Architecture Specifications

To survive data center failures with zero downtime and zero data loss under a 200M-user load, **Replica Sets** are engineered into every tier:

### 3.1 PostgreSQL 16 Multi-AZ Database Replica Sets (Patroni + etcd)

```mermaid
flowchart TD
    subgraph AZ_A [Availability Zone A (Primary Data Center)]
        PG_PRIMARY[("PostgreSQL 16 Primary (RW)<br/>Accepts all write transactions")]
        PATRONI_A["Patroni HA Agent A"]
        ETCD_A["etcd Leader Consensus Node 1"]
    end

    subgraph AZ_B [Availability Zone B (Synchronous Standby)]
        PG_SYNC_STANDBY[("PostgreSQL Sync Standby (RPO = 0)<br/>Guaranteed byte-for-byte WAL match")]
        PATRONI_B["Patroni HA Agent B"]
        ETCD_B["etcd Leader Consensus Node 2"]
    end

    subgraph AZ_C [Availability Zone C (Tie-Breaker & Async Standby)]
        PG_ASYNC_STANDBY[("PostgreSQL Async Standby<br/>Disaster Recovery Node")]
        PATRONI_C["Patroni HA Agent C"]
        ETCD_C["etcd Leader Consensus Node 3"]
    end

    subgraph ReadReplicaPool [Read-Only Replica Pool (Behind PgBouncer)]
        RR1[("Read Replica 1<br/>Catalog & Ephemeris")]
        RR2[("Read Replica 2<br/>Devotee Spatial Lookups")]
        RR3[("Read Replica 3<br/>Temple CMS Queries")]
        RR4[("Read Replica 4<br/>Ledgers & Reports")]
    end

    PG_PRIMARY -->|Synchronous Physical Streaming (sync_commit = ON)| PG_SYNC_STANDBY
    PG_PRIMARY -->|Asynchronous Physical Streaming| PG_ASYNC_STANDBY
    PG_PRIMARY -->|Asynchronous Physical Streaming| ReadReplicaPool

    PATRONI_A <--> ETCD_A
    PATRONI_B <--> ETCD_B
    PATRONI_C <--> ETCD_C

    ETCD_A <--> ETCD_B <--> ETCD_C
```

#### Replica Set Guarantees:
1. **Zero Data Loss ($\text{RPO} = 0$)**:
   - `synchronous_commit = on` with `synchronous_standby_names = 'FIRST 1 (standby_az_b, standby_az_c)'`.
   - Every financial booking, escrow transaction, and VIP ticket write is committed to at least **two independent cloud data centers** before acknowledging HTTP 200 to the client.
2. **Sub-10 Second Automatic Failover ($\text{RTO} < 10\text{s}$)**:
   - Patroni uses `etcd` distributed consensus.
   - If AZ-A suffers a power cut, `etcd` quorum immediately detects missing heartbeats (< 3s), elects the synchronous standby in AZ-B, promotes it to Primary, and signals PgBouncer to reroute writes without operator intervention.
3. **Read Scaling via Asynchronous Replica Set Pool**:
   - 4 to 8 dedicated read replicas handle 100% of non-transactional reads (catalog searches, temple history, public reviews), keeping the Primary node CPU under 40%.

---

### 3.2 Redis Enterprise Sharded In-Memory Replica Sets

Each of the 128 Redis shards runs as a **3-Node Replica Set** spanning three availability zones:
- **Topology**: 1 Primary Master + 1 Synchronous Replica + 1 Asynchronous Standby per shard.
- **Failover Mechanism**: Redis Sentinel / Raft consensus monitors health with 500ms pings.
- **Failover Speed**: Node death promoted in `< 1,200ms`.
- **Read Offloading**: Read-only queries for static Panchangam and pooja catalog read from replicas (`READONLY` mode), preserving Master CPU for atomic Lua locks and priest GPS writes.

---

### 3.3 Apache Kafka Streaming Replica Sets

To buffer 120,000 write operations per second:
- **Broker Topology**: 50 Kafka brokers distributed uniformly across 3 Availability Zones.
- **Topic Replication Factor**: **3** (`replication.factor = 3`).
- **In-Sync Replicas**: `min.insync.replicas = 2`.
- **Durability Guarantee**: `acks = all`. An event is acknowledged only when the partition leader and at least one follower replica commit the log to NVMe disk. Even if an entire broker rack explodes, zero booking events are lost.

---

### 3.4 Kubernetes Stateless ReplicaSets (800+ Pods with HPA)

Stateless Go microservices are managed by Kubernetes `ReplicaSet` controllers backed by Horizontal Pod Autoscalers (HPA):
- **API Gateway ReplicaSet**: Base 200 pods, autoscaling up to **800 pods** based on CPU utilization (> 65%) and active WebSocket connections (> 40,000/pod).
- **Spatial Dispatch ReplicaSet**: 50 to **150 pods** scaled based on priest GPS ingestion queue depth.
- **Flash Ticketing ReplicaSet**: Pre-warmed to **100 pods** at 09:55 AM IST before the 10:00 AM daily temple quota opens.
- **Rolling Updates**: Zero-downtime rolling deployments (`maxSurge: 25%`, `maxUnavailable: 0`) ensure users never drop an active booking during service updates.

---

## 4. The 5 Core Hyperscale Pillars for 200M Users

### Pillar 1: The 92% Edge Shield (Protecting Origin from 8M RPS)
- 280+ Anycast edge PoPs terminate 7.36M RPS without touching application servers.
- Cloudflare Edge Virtual Waiting Room meters flash VIP ticketing spikes at 25,000 checkouts/second.

### Pillar 2: Sharded In-Memory Tier (Redis Enterprise 128 Nodes)
- 128-node sharded cluster evaluated in 0.4ms across distributed hash slots.
- In-memory atomic Lua scripts guarantee zero overselling of temple VIP slots without relational database locks.

### Pillar 3: Citus Distributed PostgreSQL 16 + Kafka Write-Behind
- 16 Citus worker shards with Multi-AZ replica sets.
- Writes append to Kafka in 1.2ms; micro-batch consumer workers bulk-insert 5,000 rows at a time, transforming random I/O into sequential disk writes.

### Pillar 4: Real-Time Chat & State Signaling for 40M Connected Peers
- Go Epoll network pollers eliminate goroutine-per-connection memory bloat.
- Redis Pub/Sub sharded message fabric delivers cross-pod messages in < 4ms.

### Pillar 5: Hybrid WebRTC + Low-Latency HLS Video Distribution (3.0 Tbps)
- 1-to-1 two-way WebRTC between priest and devotee family for Sankalpam.
- 1-to-many mass pilgrim broadcast transcoded into LL-HLS and distributed via Global Video CDN at 3.0 Tbps egress.

---

## 5. Hardware & Fleet Sizing Matrix

| Component | ReplicaSet / Node Count | Specifications per Node | Total Compute / Memory |
| :--- | :--- | :--- | :--- |
| **Edge L7 Envoy Proxies** | 120 Pods ReplicaSet | 8 vCPU, 16 GB RAM | 960 vCPU, 1.9 TB RAM |
| **API Gateway Microservices** | 800 Pods ReplicaSet (Go 1.24) | 4 vCPU, 8 GB RAM | 3,200 vCPU, 6.4 TB RAM |
| **Spatial Dispatch Engines** | 150 Pods ReplicaSet (Go) | 4 vCPU, 8 GB RAM | 600 vCPU, 1.2 TB RAM |
| **Redis Enterprise Replica Sets** | 128 Shards $\times$ 3 Nodes (384 Nodes) | 8 vCPU, 32 GB RAM | 3,072 vCPU, 12.3 TB RAM |
| **Apache Kafka Replica Sets** | 50 Brokers (RF=3) | 16 vCPU, 64 GB RAM, 2TB NVMe | 800 vCPU, 3.2 TB RAM, 100TB NVMe |
| **Citus PostgreSQL Shard Replica Sets** | 16 Primaries + 16 Sync Standbys + 8 Read Replicas | 32 vCPU, 128 GB RAM, 4TB NVMe | 1,280 vCPU, 5.1 TB RAM, 160TB NVMe |
| **PgBouncer Connection Fleet** | 40 Pods ReplicaSet | 4 vCPU, 8 GB RAM | 160 vCPU, 320 GB RAM |
| **WebRTC / SFU Transcoders** | 100 GPU/CPU Pods ReplicaSet | 16 vCPU, 32 GB RAM | 1,600 vCPU, 3.2 TB RAM |
