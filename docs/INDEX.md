# Puja Vaahini Architecture & Specification Documentation Suite

This directory contains the complete set of architectural blueprints, use case specifications, database migrations, and 200M-user hyperscale capacity models for the **Puja Vaahini** ecosystem.

---

## 📚 Document Index

| Document | File Path | Scope & Description |
| :--- | :--- | :--- |
| **1. Master Use Cases Specification** | [`usecase.txt`](file:///E:/New%20folder%20(5)/docs/usecase.txt) *(also at root: [`usecase.txt`](file:///E:/New%20folder%20(5)/usecase.txt))* | Comprehensive breakdown of all 10 modules (A through J): Priest slots, custom services, editable samagri, Puja Store with Vendor App, In-App Chat, Instant Colleague Referral, Temple CMS with VIP & Live E-Pooja, Profile Pictures, Swiggy-style Cancellation Modal, In-App Navigation Trees, Mobile UX Resilience, and 3-Language i18n Architecture. |
| **2. Master Expansion Plan & Roadmap** | [`puja_vaahini_ecosystem_expansion_plan.md`](file:///E:/New%20folder%20(5)/docs/puja_vaahini_ecosystem_expansion_plan.md) | High-level system architecture, Sequence diagrams, ER diagrams, Modular i18n clean folder tree (English, Telugu, Hindi), 5-tab Devotee/Priest navigation graphs, Cancellation state machine, and phased implementation milestones. |
| **3. 200 Million User Hyperscale Blueprint** | [`puja_vaahini_200m_hyperscale_architecture.md`](file:///E:/New%20folder%20(5)/docs/puja_vaahini_200m_hyperscale_architecture.md) | Hyperscale engineering for 2,000 Lakh active users over a 12-hour continuous festival peak: 8M RPS, 40M CCU, Multi-AZ Replica Sets (PostgreSQL Patroni, 128-node Redis Enterprise, Kafka 3x partition replication), Citus 16-shard distributed DB, 92% Edge Shield, and Hybrid WebRTC / LL-HLS 3.0 Tbps Video CDN. |
| **4. Comprehensive Database Migration** | [`000003_comprehensive_ecosystem.sql`](file:///E:/New%20folder%20(5)/backend/migrations/000003_comprehensive_ecosystem.sql) | Production SQL migration (applied and active in PostgreSQL `purohit_db`): 11 new tables, foreign keys, indexes, and initial seeds for Dhapurmari Temple and local pooja vendors. |

---

## 🏛️ Ecosystem High-Level Summary

1. **Client Applications**:
   - **Devotee App** (Compose Multiplatform: Android + iOS)
   - **Priest / Ayagaryllu App** (Compose Multiplatform: Android + iOS)
   - **Puja Store Vendor App** (Dedicated Compose Multiplatform Android)
   - **Temple CMS Admin Portal** (Next.js / React Web)
2. **Backend & Edge**:
   - Unified Go API Gateway & Epoll WebSocket Hub (Port 8080)
   - Coturn WebRTC STUN/TURN & LiveKit SFU Cluster
   - Cloudflare / Akamai Anycast CDN Edge Caching (92% offload)
3. **In-Memory & Storage**:
   - 128-Node Sharded Redis Enterprise Cluster (GEO, Redlock, Pub/Sub, Cache)
   - 50-Broker Apache Kafka Event Streaming Backbone
   - Citus Distributed PostgreSQL 16 with Multi-AZ Sync Replica Sets
