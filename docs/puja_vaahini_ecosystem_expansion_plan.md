# Puja Vaahini Ecosystem: Master Expansion Plan (Architected for 200 Million Active Users)

## 1. Executive Summary & Hyperscale Target
This document establishes the master implementation plan and architectural roadmap for the **Puja Vaahini** ecosystem. The platform is designed to scale reliably to:
- **Target Scale**: **2,000 Lakh (200,000,000 / 200 Million / 20 Crore)** active users.
- **Peak Window**: **12-Hour Continuous Festival Rush** (Maha Shivaratri, Ganesh Chaturthi, Varalakshmi, Pushkaralu).
- **Peak Load**: **8,000,000 Ingress RPS**, **40,000,000 Concurrent Connected Users (CCU)**, and **120,000 Database Write TPS**.

All modules and specifications are established in pure planning mode (**zero code modified**):
1. **3-Language i18n Architecture**: English (`en`), Telugu (`te` - తెలుగు), and Hindi (`hi` - हिन्दी).
2. **Priest Slots & Dynamic Dakshina Pricing**: Calendar workbench and custom ritual creation.
3. **Editable Samagri Checklist & Puja Store**: Local vendor delivery when priest kit is unchecked.
4. **Dedicated Vendor App**: Compose Multiplatform Android.
5. **Temple CMS, VIP Pujas & Zero-Lag Offline Gate Scans**: Next.js portal + Ed25519 cryptographic tokens.
6. **Real-Time In-App Chat & Priest Peer Referral**: Epoll WebSocket hubs + instant auto-reassignment.
7. **Swiggy/Rapido-Style Cancellation Modal**: Multi-tier refund breakdown, retention hooks, and reason selector.
8. **Mobile UX Resilience**: Hardware back-button guards (`BackHandler`), logout confirmation dialog, and shimmer skeletons.
9. **Hyperscale Infrastructure Tier**: Multi-CDN edge termination (92% offload), Citus distributed PostgreSQL (16 shards), Kafka write-behind buffer, and 128-node Redis cluster.

---

## 2. 200M-User Hyperscale System Architecture

```mermaid
flowchart TD
    subgraph Clients [200 Million Active Users]
        DA["📱 195M Devotees (Mobile App)"]
        PA["📱 2M Priests (GPS & Live Video)"]
        VA["🏪 1M Store Vendors (Orders)"]
        TCMS["🖥️ 2M Pilgrims & Temple CMS"]
    end

    subgraph GlobalEdge [Global Edge & Anycast CDN Tier - 92% Offload]
        ANYCAST["🌐 Anycast BGP DNS Routing"]
        CDN_EDGE["⚡ Multi-CDN Edge Network (Cloudflare + Akamai + CloudFront)<br/>Panchangam, Catalog, Temples, i18n Strings (Cached at Edge)"]
        WAF_SHIELD["🛡️ Volumetric DDoS Shield & Layer 7 Rate Limiter (Token Bucket)"]
        QUEUE_IT["🚪 Virtual Waiting Room (Cloudflare Waiting Room / Redis Token Queue)<br/>Engages during sudden 10:00 AM VIP Flash Ticketing bursts"]
    end

    subgraph L4L7Ingress [Multi-Region Load Balancing Tier]
        NLB["AWS Network Load Balancers (L4 NLB)<br/>TCP: 8080/443 | UDP: 3478 (WebRTC TURN)"]
        ENVOY["Envoy Proxy Fleet (Layer 7)<br/>mTLS, gRPC, WebSocket Multiplexing, HTTP/3 QUIC"]
    end

    subgraph MicroservicesCluster [EKS / GKE Hyperscale Compute Fleet]
        direction TB
        GW_PODS["🚪 800x API Gateway Pods (Go 1.24 Epoll)<br/>Stateless JWT verification, 50k conns/pod"]
        DISPATCH_PODS["⚡ 150x Spatial Dispatch Pods (Go)<br/>Sub-5ms Rapido scoring against Redis GEO"]
        TICKET_PODS["🎟️ 100x Flash Ticketing Pods (Go)<br/>Zero-lock atomic Lua reservation"]
        CHAT_PODS["💬 300x Distributed Chat Gateway Pods<br/>WebSocket routing over Redis Pub/Sub"]
        MEDIA_WORKERS["📹 100x WebRTC Ingestion & Transcoding Pods<br/>H.264/AV1 transcode -> HLS Packager"]
    end

    subgraph ShardedMemoryTier [128-Node Redis Enterprise Hyperscale Tier (1.8 TB RAM)]
        R_GEO["📍 Redis GEO Cluster (16 Nodes)<br/>Priest GPS telemetry & spatial radius"]
        R_SLOTS["🔒 Redis Redlock & Lua Quota Cluster (32 Nodes)<br/>Flash VIP tickets & Priest availability slots"]
        R_PUBSUB["📢 Redis Pub/Sub Cross-Pod Fabric (32 Nodes)<br/>Inter-gateway WebSocket message routing"]
        R_CACHE["🗄️ Redis High-Hit LRU Cache (48 Nodes)<br/>Active bookings, user profiles, gotram records"]
    end

    subgraph StreamingBackbone [Distributed Streaming Backbone]
        KAFKA{{"🐘 Apache Kafka 50-Broker Cluster<br/>(3x Replication, 100k writes/sec buffer)"}}
        CONSUMERS["⚙️ Asynchronous Batch Ingestion Workers<br/>(Bulk 5,000-row micro-batching into DB)"]
    end

    subgraph DistributedDatabaseTier [Citus Distributed PostgreSQL 16 Multi-AZ Replica Sets]
        CITUS_COORD["🐘 Citus Coordinator Replica Set (Active-Passive HA with Sync Standby)"]
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
    ENVOY --> GW_PODS

    GW_PODS --> DISPATCH_PODS & TICKET_PODS & CHAT_PODS & MEDIA_WORKERS

    DISPATCH_PODS <--> R_GEO
    TICKET_PODS <--> R_SLOTS
    CHAT_PODS <--> R_PUBSUB
    GW_PODS <--> R_CACHE

    TICKET_PODS --> KAFKA
    GW_PODS --> KAFKA
    KAFKA --> CONSUMERS
    CONSUMERS --> PGBOUNCER

    PGBOUNCER --> CITUS_COORD
    CITUS_COORD --> SHARD1 & SHARD2 & SHARD3 & SHARD4
    SHARD1 & SHARD2 & SHARD3 & SHARD4 -.->|Streaming Replication| READ_POOL

    MEDIA_WORKERS --> SFU
    SFU --> HLS_ORIGIN
    HLS_ORIGIN --> VIDEO_CDN
    VIDEO_CDN --> DA & TCMS
```

---

## 3. Mathematical Capacity Dimensions for 200M Scale

| Parameter | 200M (2000 Lakh) Baseline | Production Engineering Strategy |
| :--- | :--- | :--- |
| **Gross Ingress Traffic** | 8,000,000 Peak Requests/Sec | **92% Edge Shield**: Cloudflare/Akamai Anycast PoPs terminate 7.36M RPS; only 640k RPS reach origin. |
| **Concurrent Connections** | 40,000,000 Active Peers | **800 Gateway Pods**: Go epoll `netpoll` runtime handles 50,000 idle sockets per pod with zero thread overhead. |
| **Database Write Volume** | 120,000 Transactions/Sec | **Kafka Decoupling**: Writes append to a 50-node Kafka cluster in 1.2ms; consumers micro-batch 5,000 rows into Citus DB. |
| **Atomic Quotas & Locks** | 250,000 Flash Reservations/Sec | **Sharded Redis Lua**: In-memory decrements (`DECRBY`) evaluate in 0.4ms across 128 Redis Enterprise nodes. |
| **Live Sanctum Video Egress** | 3.0 Terabits/Sec (2M viewers) | **Hybrid Video Architecture**: Priest & booked devotee use 2-way WebRTC; 2M passive viewers consume Low-Latency HLS from CDN edge. |
| **Temple Gate Scans** | 50ms verification in stone walls | **Ed25519 Cryptography**: Scanners verify tickets offline in < 5ms with zero cloud roundtrips. |

---

## 4. 🌐 3-Language Internationalization (i18n) Architecture

Localized in shared `common-contracts` for 100% code reuse across Devotee, Priest, and Vendor apps:
- **English (`en`)**: Urban devotees & NRI diaspora.
- **Telugu (`te` - తెలుగు)**: Core regional Vedic Purohit community (AP, Telangana, Dhapurmari, Kotipalli, Vemulawada).
- **Hindi (`hi` - हिन्दी)**: National Vedic ritual language.

### Modular Clean Folder Structure:
```text
apps/common-contracts/src/commonMain/kotlin/com/purohit/contracts/i18n/
├── AppLanguage.kt                   # Enum: EN, TE, HI with native display names
├── LocalizationManager.kt           # Singleton & CompositionLocal reactive live switcher
├── StringContract.kt                # Aggregator interface matching all sub-domain contracts
├── models/                          # Pure Domain String Contract Interfaces
│   ├── AuthStrings.kt               # Login, OTP, Gotram, Nakshatram, Consent
│   ├── CatalogStrings.kt            # Pooja titles, categories, durations, Muhurtham labels
│   ├── SamagriStrings.kt            # Checklist items, units, dry/fresh categories, Puja Store
│   ├── DispatchStrings.kt           # Radar, Nearby Purohits, ETA, live transit, OTP release
│   ├── TempleStrings.kt             # Sanctum, VIP passes, E-Pooja video call, deities
│   ├── CancellationStrings.kt       # Swiggy-style reasons, refund warnings, retention
│   ├── CommonStrings.kt             # Navigation tabs, buttons, dialogs, logout modal
│   └── ErrorStrings.kt              # Offline banners, retry buttons, 500 maintenance states
└── locales/                         # Pure Language Implementations
    ├── en/                          # 🇬🇧 English Modular Classes & EnStrings.kt
    ├── te/                          # 🇮🇳 Telugu Modular Classes & TeStrings.kt (తెలుగు)
    └── hi/                          # 🇮🇳 Hindi Modular Classes & HiStrings.kt (हिन्दी)
```

---

## 5. Mobile UX Resilience, Navigation Guards & Error Handling

1. **Hardware Back-Navigation Guards (`BackHandler`)**:
   - **Root Screens**: Double-tap within 2000ms with snackbar *"Press back again to exit Puja Vaahini"*.
   - **Checkout / Muhurtham Reserve Guard**: Alert popup: *"Leaving now will release your reserved Muhurtham slot. Discard? [Stay / Discard]"*.
   - **Live WebRTC Sankalpam Guard**: Intercepted with options: `[Minimize to Background Audio (PiP)]` or `[End Video Call]`.
   - **Live Tracking**: Back press returns to Home tab while keeping a persistent floating banner showing active ETA.
2. **Logout Confirmation Dialog & Session Purge**:
   - Styled confirmation modal with active booking warnings (*"⚠️ Active pooja scheduled with Pandit Sharma. Logging out will NOT cancel the ritual."*).
   - Confirming purges JWT tokens, disconnects WebSockets/WebRTC sessions, clears backstack (`popUpTo(0)`), and routes to Login.
3. **Loading Skeletons, Shimmers & Network Error Views**:
   - Shimmer placeholder cards for Catalog grid, Nearby Priests radar, Temple directory, and Puja Store.
   - Global offline network monitor with top sticky amber banner (*"⚠️ No Internet - Showing cached details"*).
   - Full-screen `NetworkErrorScreen` with sacred disconnected illustration and exponential backoff retry.
   - 500 Server Maintenance View (*"Our servers are undergoing routine Vedic ephemeris sync. We will be back online shortly."*).

---

## 6. Swiggy / Rapido Style Cancellation Modal & Refund Flow

When a Devotee or Priest taps "Cancel Booking", an interactive bottom sheet displays:
1. **Dynamic Refund Status**:
   - `REQUESTED` / `ACCEPTED` (within 5 mins): 100% Instant Refund (Free cancellation).
   - `IN_TRANSIT`: Nominal Rs. 150 travel fee credited to priest, remainder refunded to devotee.
   - `ARRIVED`: 25% Dakshina retained for priest, 75% refunded.
2. **Retention Hook**: *"Need to change timing? You can reassign to another nearby Purohit for free! [Reassign Instead]"*.
3. **Mandatory Reason Selector**: Radio selection (Priest delayed, change of plans, arranged alternate locally, wrong date).
4. **Action Slider**: *"Swipe to Cancel"* to confirm cancellation and trigger escrow refund.

---

## 7. Phased Hyperscale Implementation Roadmap

### Phase 1: Shared Modular i18n Contracts & Core DTOs
- [x] Complete domain-partitioned i18n architecture designed in plan.
- [ ] Scaffold `apps/common-contracts/.../i18n/` with `AppLanguage`, `LocalizationManager`, and domain string interfaces.
- [ ] Add `en/`, `te/`, `hi/` translation implementations for all 8 domains.
- [ ] Configure Compose `CompositionLocalProvider(LocalAppStrings provides ...)` for live reactive switching.
- [ ] Backend: Inspect `Accept-Language` header to return localized SMS and push notifications.

### Phase 2: Edge Acceleration & 92% Caching Tier
- [ ] Configure Cloudflare / Akamai Anycast CDN caching rules for Panchangam, Catalog, and static assets.
- [ ] Setup Cloudflare Virtual Waiting Room for 10:00 AM flash VIP temple openings.
- [ ] Setup L4 NLB and L7 Envoy proxies with token-bucket rate limiting.

### Phase 3: Sharded Redis 128-Node Replica Sets & Kafka Streaming Replica Sets
- [ ] Deploy 128-node Redis Enterprise cluster where each shard is a **3-Node Multi-AZ Replica Set** (1 Primary + 2 Replicas, Sentinel/Raft failover < 1.2s).
- [ ] Implement atomic Lua reservation scripts for flash VIP tickets and calendar slots.
- [ ] Deploy 50-broker Apache Kafka cluster configured with **Partition Replication Factor = 3** (`min.insync.replicas = 2`, `acks = all`).
- [ ] Implement Go Kafka producer in API Gateway and micro-batch consumer workers for bulk database writes.

### Phase 4: Citus Distributed PostgreSQL 16 Multi-AZ Replica Sets (Patroni + etcd)
- [ ] Configure 16 Citus worker nodes partitioned by `hash(user_id)`, each backed by a **Synchronous Multi-AZ Standby Replica Set** (Zero RPO, sub-10s failover).
- [ ] Deploy an **8-node Read Replica Pool** with asynchronous streaming replication for Temple CMS analytics, history, and search.
- [ ] Configure PgBouncer transaction-level connection pooler fleet (50,000 client conns $\rightarrow$ 200 DB conns).
- [ ] Configure Kubernetes **Stateless ReplicaSets** for API Gateway (HPA autoscaling up to 800 pods).

### Phase 5: Devotee & Priest Mobile Apps UI Extensions
- [ ] Devotee App: 5-tab structure, localized strings, back-navigation guards, logout modal, shimmer skeletons, and Swiggy-style cancellation modal.
- [ ] Priest App: 5-tab structure, schedule workbench, custom services & pricing editor, colleague referral sheet, and duty toggle.

### Phase 6: Dedicated Vendor App (KMP Android) & Temple CMS (Next.js)
- [ ] Scaffold `:apps:vendor-app` with inventory management and order dispatch stepper.
- [ ] Scaffold `temple-cms/` Next.js portal with role-based access and Ed25519 zero-lag offline gate scanner.

### Phase 7: Live Video Streaming (WebRTC 2-Way + LL-HLS CDN Broadcast)
- [ ] Deploy LiveKit / Pion SFU cluster for 1-to-1 WebRTC e-poojas.
- [ ] Transcode priest sanctum video into Low-Latency HLS (LL-HLS) distributed via Global Video CDN for 2M concurrent pilgrims.

### Phase 8: Hyperscale Load Injection Simulation
- [ ] Execute distributed Locust / k6 load test simulating 8,000,000 peak RPS and 40,000,000 concurrent socket connections over a 12-hour synthetic festival cycle.
- [ ] Validate p99 latency $< 15\text{ms}$ and zero database lock timeouts.
