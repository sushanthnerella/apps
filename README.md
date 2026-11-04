# Puja Vaahini (Ayagaryllu On-Demand & E-Pooja)
> An on-demand spiritual services ecosystem connecting Vedic Priests with Devotees for in-person rituals and live WebRTC e-poojas.

---

## 🏛️ System Architecture Overview

```text
puja-vaahini/
│
├── apps/
│   ├── common-contracts/             # Lightweight sync-critical wire protocol & DTOs
│   │   ├── webrtc/                   # WebRTC signaling models (Offer, Answer, ICE)
│   │   ├── booking/                  # State machine enums (REQUESTED -> IN_TRANSIT -> COMPLETED)
│   │   ├── panchangam/               # Muhurtham calculation & horoscope contracts
│   │   └── location/                 # Geolocation payload contracts
│   │
│   ├── devotee-app/                  # Devotee KMP App (Android + iOS Compose Multiplatform)
│   │   ├── Pooja Catalog & Samagri Kit add-on selection
│   │   ├── Priest-calculated Muhurtham intake (Gotram, Nakshatram, Rashi)
│   │   ├── Live Priest Arrival Tracking & OTP Completion Escrow Release
│   │   └── Live E-Pooja WebRTC viewer with two-way Sankalpam audio
│   │
│   └── priest-app/                   # Priest / Ayagaryllu KMP App (Android + iOS Compose Multiplatform)
│       ├── Online / Offline Duty Toggle with background GPS pinging
│       ├── Rapido-Style Incoming Dispatch Alert (45s countdown timer)
│       ├── Muhurtham Workbench with Panchangam guidance & Lagna picker
│       └── Live Broadcaster Studio (Front/Rear camera, Lapel mic Opus chanting)
│
├── backend/                          # High-Scale Go (Golang) Microservices (1M+ concurrent)
│   ├── cmd/
│   │   ├── api-gateway/              # REST API (Catalog, Panchangam, Escrow initiation)
│   │   ├── dispatch-service/         # Rapido-style radius search & composite scoring
│   │   └── webrtc-signaling/         # High-concurrency WebSocket signaling server
│   ├── internal/
│   │   ├── ranking/                  # Scoring: Distance + Rating + Repeats + Acceptance + Sampradaya
│   │   ├── geo/                      # Redis GEO & Haversine spatial dispatch engine
│   │   ├── panchangam/               # Vedic ephemeris (Tithi, Nakshatra, Rahu Kalam, Yamagandam)
│   │   └── models/                   # Go domain entities
│   ├── migrations/                   # PostgreSQL 16 + PostGIS schema
│   └── docker-compose.yml            # Postgres+PostGIS, Redis Stack, Coturn (STUN/TURN)
│
└── README.md
```

---

## 🚀 Getting Started

### 1. Start Infrastructure Services (Backend)
Run PostgreSQL with PostGIS, Redis Stack for GEO indexing, and Coturn for WebRTC NAT traversal:
```bash
cd backend
docker-compose up -d
```

### 2. Run Backend Go Microservices
In separate terminals:
```bash
# 1. API Gateway (Port 8080)
go run cmd/api-gateway/main.go

# 2. Rapido Dispatch Engine (Port 8081)
go run cmd/dispatch-service/main.go

# 3. WebRTC Signaling Server (Port 8082)
go run cmd/webrtc-signaling/main.go
```

### 3. Build & Run the KMP Mobile Apps
Open the project root in **Android Studio** or **IntelliJ IDEA**:
- Run **`:apps:devotee-app`** on an Android device/emulator or iOS simulator.
- Run **`:apps:priest-app`** on a separate device/emulator to simulate live dispatch alerts and WebRTC calls.

---

## 📐 Rapido-Style Dynamic Scoring Formula

Priests discovered within the radial search ring ($5\text{ km} \rightarrow 10\text{ km} \rightarrow 20\text{ km}$) are ranked by:

$$Score = 0.35 \cdot \left(1 - \frac{D}{D_{\max}}\right) + 0.25 \cdot \left(\frac{\text{Rating}}{5.0}\right) + 0.20 \cdot \min\left(\frac{N_{\text{repeat}}}{50}, 1.0\right) + 0.10 \cdot A_{\text{rate}} + 0.10 \cdot S_{\text{match}}$$

1. **Distance ($35\%$)**: Favors the closest available priest to minimize transit time.
2. **Rating ($25\%$)**: Devotee feedback and Vedic proficiency rating.
3. **Repeat Bookings ($20\%$)**: Devotees who repeatedly book this specific priest for annual/monthly poojas.
4. **Acceptance Rate ($10\%$)**: Reliability in accepting dispatch requests without expiring.
5. **Sampradaya Match ($10\%$)**: Smartha, Vaishnava, Madhwa, or specific Vedic tradition alignment.
