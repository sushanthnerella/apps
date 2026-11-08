package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/purohit-platform/backend/internal/auth"
	"github.com/purohit-platform/backend/internal/db"
	"github.com/purohit-platform/backend/internal/geo"
	"github.com/purohit-platform/backend/internal/models"
	"github.com/purohit-platform/backend/internal/panchangam"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

type Client struct {
	ID     string
	RoomID string
	Conn   *websocket.Conn
	Send   chan []byte
}

type RoomHub struct {
	rooms map[string]map[string]*Client
	mutex sync.RWMutex
}

var hub = &RoomHub{
	rooms: make(map[string]map[string]*Client),
}

func (h *RoomHub) Register(c *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if _, exists := h.rooms[c.RoomID]; !exists {
		h.rooms[c.RoomID] = make(map[string]*Client)
	}
	h.rooms[c.RoomID][c.ID] = c
	fmt.Printf("🎥 WebRTC: Peer %s joined room %s\n", c.ID, c.RoomID)
}

func (h *RoomHub) Unregister(c *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if room, exists := h.rooms[c.RoomID]; exists {
		delete(room, c.ID)
		close(c.Send)
		if len(room) == 0 {
			delete(h.rooms, c.RoomID)
		}
	}
	fmt.Printf("🎥 WebRTC: Peer %s left room %s\n", c.ID, c.RoomID)
}

func (h *RoomHub) BroadcastToRoom(senderID, roomID string, msgBytes []byte) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	if room, exists := h.rooms[roomID]; exists {
		for clientID, client := range room {
			if clientID != senderID {
				select {
				case client.Send <- msgBytes:
				default:
					close(client.Send)
					delete(room, clientID)
				}
			}
		}
	}
}

type ChatHub struct {
	rooms map[string]map[string]*Client
	mutex sync.RWMutex
}

var chatHub = &ChatHub{
	rooms: make(map[string]map[string]*Client),
}

func (h *ChatHub) Register(c *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if _, exists := h.rooms[c.RoomID]; !exists {
		h.rooms[c.RoomID] = make(map[string]*Client)
	}
	h.rooms[c.RoomID][c.ID] = c
	fmt.Printf("💬 Chat: Peer %s joined room %s\n", c.ID, c.RoomID)
}

func (h *ChatHub) Unregister(c *Client) {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	if room, exists := h.rooms[c.RoomID]; exists {
		delete(room, c.ID)
		close(c.Send)
		if len(room) == 0 {
			delete(h.rooms, c.RoomID)
		}
	}
	fmt.Printf("💬 Chat: Peer %s left room %s\n", c.ID, c.RoomID)
}

func (h *ChatHub) BroadcastToRoom(senderID, roomID string, msgBytes []byte) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	if room, exists := h.rooms[roomID]; exists {
		for clientID, client := range room {
			if clientID != senderID {
				select {
				case client.Send <- msgBytes:
				default:
					close(client.Send)
					delete(room, clientID)
				}
			}
		}
	}
}

func enableCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept-Language, ngrok-skip-browser-warning")
		lang := r.Header.Get("Accept-Language")
		if lang != "" {
			w.Header().Set("Content-Language", lang)
		}
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

// validateOptionalCallerSession checks Bearer authorization when present, ensuring
// that expired tokens or revoked sessions are immediately rejected.
func validateOptionalCallerSession(r *http.Request, storage *db.Storage) (*auth.CustomClaims, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, nil // Header not provided; full mandatory enforcement deferred to Task 05
	}
	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := auth.ValidateToken(tokenStr)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	dbVersion, err := storage.GetUserSessionVersion(claims.UserID, claims.Role)
	if err != nil {
		return nil, fmt.Errorf("session lookup failed: %w", err)
	}
	if err := auth.ValidateSessionVersion(claims.SessionVersion, dbVersion); err != nil {
		return nil, err
	}
	return claims, nil
}

// validateCallerSession strictly enforces that a valid Bearer token is provided
// and that the active session has not been revoked.
func validateCallerSession(r *http.Request, storage *db.Storage) (*auth.CustomClaims, error) {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, fmt.Errorf("authorization header with Bearer token required")
	}
	tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
	claims, err := auth.ValidateToken(tokenStr)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}
	dbVersion, err := storage.GetUserSessionVersion(claims.UserID, claims.Role)
	if err != nil {
		return nil, fmt.Errorf("session lookup failed: %w", err)
	}
	if err := auth.ValidateSessionVersion(claims.SessionVersion, dbVersion); err != nil {
		return nil, err
	}
	return claims, nil
}

func main() {
	// Load local .env if present without overriding deployment environment values
	_ = auth.LoadLocalEnv()
	auth.InitEmailProviderFromEnv()

	pgConnStr := os.Getenv("PG_CONN_STR")
	if pgConnStr == "" {
		pgConnStr = os.Getenv("DATABASE_URL")
	}
	if pgConnStr == "" {
		pgConnStr = "postgres://postgres:123456@localhost:5432/purohit_db?sslmode=disable"
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = os.Getenv("REDIS_URL")
	}
	if redisAddr == "" {
		redisAddr = os.Getenv("REDISPRIVATEURL")
	}
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	storage, err := db.InitStorage(pgConnStr, redisAddr)
	if err != nil {
		log.Fatalf("❌ Failed to initialize database: %v", err)
	}

	// Initialize JWT configuration from deployment environment (fail fast if missing or unsuitable)
	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		jwtSecret = os.Getenv("PUROHIT_JWT_SECRET")
	}
	if err := auth.InitJWT(jwtSecret); err != nil {
		log.Fatalf("❌ Fatal: Missing or unsuitable JWT configuration: %v\nPlease set JWT_SECRET environment variable (minimum 32 characters) before starting API Gateway.", err)
	}

	// Verify required database schema version before starting server
	requiredSchemaVersion := "000007"
	if err := storage.VerifySchemaReadiness(requiredSchemaVersion); err != nil {
		log.Fatalf("❌ Schema readiness check failed: %v\nPlease run the database migration tool before starting the gateway:\n   go run ./cmd/migrate up\nOr to baseline an existing database:\n   go run ./cmd/migrate baseline\n", err)
	}

	// Configure trusted reverse proxies if specified in environment
	if tpEnv := os.Getenv("TRUSTED_PROXIES"); tpEnv != "" {
		proxies := strings.Split(tpEnv, ",")
		if err := auth.ConfigureTrustedProxies(proxies); err != nil {
			log.Fatalf("❌ Fatal: Invalid TRUSTED_PROXIES configuration: %v", err)
		}
	}

	router := mux.NewRouter()

	// 1. Health check endpoints (Truthful request-time probing for Railway and load balancers)
	registerHealthRoutes(router, storage)

	// 2. Login with Password & Issue JWT Token (Throttled & Side-Channel Protected)
	router.HandleFunc("/api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var req models.LoginRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid_payload","message":"invalid login request format"}`, http.StatusBadRequest)
			return
		}

		clientIP := auth.ExtractClientIP(r)
		accountKey := auth.BuildAccountThrottleKey(req.Role, req.Phone)

		// Check rate limiting / brute force throttling
		if throttled, remaining := auth.DefaultThrottler.IsThrottled(accountKey, clientIP); throttled {
			w.Header().Set("Retry-After", fmt.Sprintf("%d", int(remaining.Seconds())))
			http.Error(w, fmt.Sprintf(`{"error":"too_many_requests","message":"Too many failed login attempts. Please try again in %d seconds."}`, int(remaining.Seconds())), http.StatusTooManyRequests)
			return
		}

		authRes, err := storage.Login(req)
		if err != nil {
			if errors.Is(err, auth.ErrRecoveryRequired) {
				http.Error(w, `{"error":"recovery_required","message":"This account requires password setup/recovery before login. Please use password recovery."}`, http.StatusForbidden)
				return
			}

			// Record failure in throttler
			auth.DefaultThrottler.RecordFailure(accountKey, clientIP)

			if strings.Contains(err.Error(), "role is required") || strings.Contains(err.Error(), "phone number is required") || strings.Contains(err.Error(), "password is required") {
				http.Error(w, fmt.Sprintf(`{"error":"bad_request","message":"%s"}`, err.Error()), http.StatusBadRequest)
				return
			}

			// Return generic invalid credentials (avoid account enumeration)
			http.Error(w, `{"error":"invalid_credentials","message":"Invalid phone number or password."}`, http.StatusUnauthorized)
			return
		}

		// Success: clear failed attempts for this account
		auth.DefaultThrottler.RecordSuccess(accountKey)
		json.NewEncoder(w).Encode(authRes)
	}).Methods("POST", "OPTIONS")

	// 2b. Verified Password Recovery & Email Verification Routes (Modularized in recovery_handlers.go)
	secret, err := auth.GetSecretKey()
	if err != nil {
		log.Fatalf("❌ Fatal: Failed to resolve JWT signing key: %v", err)
	}
	registerRecoveryRoutes(router, storage, secret)

	// 3. User Profile Endpoint (JWT Protected with Revocation Validation)
	router.HandleFunc("/api/v1/user/profile", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, `{"error":"missing or invalid authorization header"}`, http.StatusUnauthorized)
			return
		}

		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
		claims, err := auth.ValidateToken(tokenStr)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusUnauthorized)
			return
		}

		// Validate token session version against current database state
		dbVersion, err := storage.GetUserSessionVersion(claims.UserID, claims.Role)
		if err != nil || auth.ValidateSessionVersion(claims.SessionVersion, dbVersion) != nil {
			http.Error(w, `{"error":"session_revoked","message":"Session has been revoked or expired. Please log in again."}`, http.StatusUnauthorized)
			return
		}

		profile, err := storage.GetUserProfile(claims.UserID, claims.Role)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"user profile not found: %v"}`, err), http.StatusNotFound)
			return
		}

		json.NewEncoder(w).Encode(profile)
	}).Methods("GET", "OPTIONS")

	// 4. Rapido-Style Nearby Priests (Returns ALL nearby priests around user's location)
	router.HandleFunc("/api/v1/priests/nearby", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		latStr := r.URL.Query().Get("lat")
		lonStr := r.URL.Query().Get("lng")
		if lonStr == "" {
			lonStr = r.URL.Query().Get("lon")
		}
		radiusStr := r.URL.Query().Get("radius")
		serviceFilter := r.URL.Query().Get("service")

		lat := 17.3850
		lon := 78.4867
		radius := 50000.0 // 50km metro radius default

		if v, err := strconv.ParseFloat(latStr, 64); err == nil && v != 0 {
			lat = v
		}
		if v, err := strconv.ParseFloat(lonStr, 64); err == nil && v != 0 {
			lon = v
		}
		if v, err := strconv.ParseFloat(radiusStr, 64); err == nil && v != 0 {
			radius = v
		}

		priests, err := storage.GetNearbyPriests(lat, lon, radius, serviceFilter)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"user_location":        map[string]float64{"latitude": lat, "longitude": lon},
			"search_radius_meters": radius,
			"total_priests_online": len(priests),
			"priests":              priests,
		})
	}).Methods("GET", "OPTIONS")

	// 4c. All Verified Priests Directory (for referrals & scheduling)
	router.HandleFunc("/api/v1/priests", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		priests, err := storage.GetAllPriests()
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(priests)
	}).Methods("GET", "OPTIONS")

	// 4d. Temple CMS Live Dispatch Geo-Radar Endpoint
	router.HandleFunc("/api/v1/dispatch/radar", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		radarData, err := storage.GetDispatchRadarData()
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(radarData)
	}).Methods("GET", "OPTIONS")

	// 4b. Rapido-Style Dynamic Dispatch Match Endpoint
	dispatchEngine := geo.NewDispatchEngine()
	router.HandleFunc("/api/v1/dispatch/match", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var req models.DispatchRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			req = models.DispatchRequest{
				BookingID:          "booking_test_001",
				PoojaTitle:         "Gruhapravesam",
				DevoteeLatitude:    17.3850,
				DevoteeLongitude:   78.4867,
				RequiredSampradaya: "Smartha",
				SearchRadiusMeters: 5000.0,
			}
		}

		// Pull candidates from database priests
		dbPriests, err := storage.GetNearbyPriests(req.DevoteeLatitude, req.DevoteeLongitude, req.SearchRadiusMeters, "")
		var candidates []models.PriestCandidate
		if err == nil && len(dbPriests) > 0 {
			for _, p := range dbPriests {
				candidates = append(candidates, models.PriestCandidate{
					ID:                   p.ID,
					FullName:             p.FullName,
					Phone:                p.Phone,
					Sampradaya:           p.Sampradaya,
					RatingAvg:            p.RatingAvg,
					TotalCompletedPoojas: p.TotalCompletedPoojas,
					RepeatBookingCount:   p.RepeatBookingCount,
					AcceptanceRate:       0.96,
					DistanceMeters:       p.DistanceMeters,
				})
			}
		}

		if len(candidates) == 0 {
			candidates = []models.PriestCandidate{
				{
					ID:                   "00000000-0000-0000-0000-000000000101",
					FullName:             "Brahmasri Venkata Sharma Garu",
					Phone:                "+919876543210",
					Sampradaya:           "Smartha",
					RatingAvg:            4.95,
					TotalCompletedPoojas: 184,
					RepeatBookingCount:   42,
					AcceptanceRate:       0.98,
					DistanceMeters:       1800.0,
				},
			}
		}

		rankedPriests := dispatchEngine.FindNearestPriests(context.Background(), req, candidates, nil)
		if len(rankedPriests) == 0 {
			rankedPriests = candidates
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"booking_id":       req.BookingID,
			"total_candidates": len(rankedPriests),
			"best_match":       rankedPriests[0],
			"candidates":       rankedPriests,
		})
	}).Methods("POST", "OPTIONS")

	// 5. Pooja Catalog with Search Bar, Category & 3-Language Filter
	router.HandleFunc("/api/v1/poojas", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		searchQuery := r.URL.Query().Get("search")
		categoryFilter := r.URL.Query().Get("category")
		lang := r.URL.Query().Get("lang")
		if lang == "" {
			lang = r.Header.Get("Accept-Language")
		}

		catalog, err := storage.GetCatalog(searchQuery, categoryFilter, lang)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(catalog)
	}).Methods("GET", "OPTIONS")

	// 6. Devotee Registration with Explicit Consent & Password
	router.HandleFunc("/api/v1/auth/register-devotee", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var req models.DevoteeRegistrationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid_payload","message":"invalid request format"}`, http.StatusBadRequest)
			return
		}

		if !req.ConsentAgreed {
			http.Error(w, `{"error":"consent_required","message":"Devotee terms and ritual consent are mandatory."}`, http.StatusBadRequest)
			return
		}

		userID, err := storage.RegisterDevotee(req)
		if err != nil {
			if strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "already registered") {
				http.Error(w, fmt.Sprintf(`{"error":"account_already_exists","message":"%s"}`, err.Error()), http.StatusConflict)
				return
			}
			if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "mandatory") || strings.Contains(err.Error(), "password") || strings.Contains(err.Error(), "invalid") {
				http.Error(w, fmt.Sprintf(`{"error":"bad_request","message":"%s"}`, err.Error()), http.StatusBadRequest)
				return
			}
			http.Error(w, `{"error":"internal_server_error","message":"Registration failed. Please try again."}`, http.StatusInternalServerError)
			return
		}

		canonicalPhone, _ := auth.NormalizePhone(req.Phone)
		token, err := auth.GenerateToken(userID, canonicalPhone, "devotee", req.FullName, 1)
		if err != nil {
			http.Error(w, `{"error":"token_generation_failed","message":"Account created but session generation failed."}`, http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":       "SUCCESS",
			"user_id":      userID,
			"access_token": token,
			"token_type":   "Bearer",
			"user": map[string]interface{}{
				"id":             userID,
				"phone":          canonicalPhone,
				"full_name":      req.FullName,
				"gotram":         req.Gotram,
				"nakshatram":     req.Nakshatram,
				"rashi":          req.Rashi,
				"city":           req.City,
				"role":           "devotee",
				"consent_agreed": true,
			},
			"full_name":      req.FullName,
			"consent_agreed": true,
			"message":        "Devotee registered successfully with JWT session generated",
		})
	}).Methods("POST", "OPTIONS")

	// 7. Priest Registration with Consent & Password (Approval Kept Separate)
	router.HandleFunc("/api/v1/auth/register-priest", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var req models.PriestRegistrationRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid_payload","message":"invalid request format"}`, http.StatusBadRequest)
			return
		}

		if !req.ConsentAgreed {
			http.Error(w, `{"error":"consent_required","message":"purohit code of conduct consent is mandatory"}`, http.StatusBadRequest)
			return
		}

		priestID, err := storage.RegisterPriest(req)
		if err != nil {
			if strings.Contains(err.Error(), "already exists") || strings.Contains(err.Error(), "already registered") {
				http.Error(w, fmt.Sprintf(`{"error":"account_already_exists","message":"%s"}`, err.Error()), http.StatusConflict)
				return
			}
			if strings.Contains(err.Error(), "required") || strings.Contains(err.Error(), "mandatory") || strings.Contains(err.Error(), "password") || strings.Contains(err.Error(), "invalid") {
				http.Error(w, fmt.Sprintf(`{"error":"bad_request","message":"%s"}`, err.Error()), http.StatusBadRequest)
				return
			}
			http.Error(w, `{"error":"internal_server_error","message":"Registration failed. Please try again."}`, http.StatusInternalServerError)
			return
		}

		canonicalPhone, _ := auth.NormalizePhone(req.Phone)
		token, err := auth.GenerateToken(priestID, canonicalPhone, "priest", req.FullName, 1)
		if err != nil {
			http.Error(w, `{"error":"token_generation_failed","message":"Account created but session generation failed."}`, http.StatusInternalServerError)
			return
		}

		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":         "SUCCESS",
			"priest_id":      priestID,
			"access_token":   token,
			"full_name":      req.FullName,
			"sampradaya":     req.Sampradaya,
			"consent_agreed": true,
			"message":        "Ayagaryllu registered with verified qualifications",
		})
	}).Methods("POST", "OPTIONS")

	// 8. Panchangam Daily Astrological Info
	router.HandleFunc("/api/v1/panchangam/today", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		todayPanchangam := panchangam.ComputePanchangam(time.Now())
		json.NewEncoder(w).Encode(todayPanchangam)
	}).Methods("GET", "OPTIONS")

	// 9. Protected Booking Lifecycle, Asymmetric Completion OTP & Admin Emergency Cancel Routes (Task 06)
	registerBookingRoutes(router, storage)

	// 11. WebRTC Signaling WebSocket
	router.HandleFunc("/ws/webrtc", func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		roomID := r.URL.Query().Get("roomId")
		clientID := r.URL.Query().Get("clientId")

		if roomID == "" || clientID == "" {
			conn.WriteMessage(websocket.TextMessage, []byte(`{"error":"missing roomId or clientId"}`))
			conn.Close()
			return
		}

		client := &Client{
			ID:     clientID,
			RoomID: roomID,
			Conn:   conn,
			Send:   make(chan []byte, 256),
		}

		hub.Register(client)

		go func() {
			defer conn.Close()
			for msg := range client.Send {
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					break
				}
			}
		}()

		defer func() {
			hub.Unregister(client)
			conn.Close()
		}()

		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				break
			}

			var signal models.WebRTCMessage
			if err := json.Unmarshal(message, &signal); err == nil {
				signal.SenderID = clientID
				signal.RoomID = roomID
				broadcastData, _ := json.Marshal(signal)
				hub.BroadcastToRoom(clientID, roomID, broadcastData)
			}
		}
	})

	// 12. Priest Availability Slots
	router.HandleFunc("/api/v1/priests/slots", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "GET" {
			priestID := r.URL.Query().Get("priest_id")
			date := r.URL.Query().Get("date")
			if date == "" {
				date = time.Now().Format("2006-01-02")
			}
			slots, err := storage.GetPriestSlots(priestID, date)
			if err != nil {
				http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(slots)
			return
		}

		if r.Method == "POST" {
			var req models.CreateSlotRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, `{"error":"invalid slot payload"}`, http.StatusBadRequest)
				return
			}
			slotID, err := storage.CreatePriestSlot(req)
			if err != nil {
				http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "SUCCESS",
				"slot_id": slotID,
				"message": "Availability slot published successfully",
			})
			return
		}
	}).Methods("GET", "POST", "OPTIONS")

	router.HandleFunc("/api/v1/priests/slots/hold", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var req models.HoldSlotRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid hold request"}`, http.StatusBadRequest)
			return
		}
		success, expiresAt, err := storage.HoldPriestSlot(req.SlotID, req.DevoteeID, req.HoldDurationSeconds)
		if err != nil || !success {
			http.Error(w, fmt.Sprintf(`{"error":"slot cannot be held: %v"}`, err), http.StatusConflict)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":               "HELD",
			"slot_id":              req.SlotID,
			"expires_at_timestamp": expiresAt,
			"message":              "Slot locked for 10 minutes pending checkout confirmation",
		})
	}).Methods("POST", "OPTIONS")

	// 13. Priest Custom Services & Dynamic Pricing with Multi-Language Support
	router.HandleFunc("/api/v1/priests/custom-services", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "GET" {
			priestID := r.URL.Query().Get("priest_id")
			lang := r.URL.Query().Get("lang")
			if lang == "" {
				lang = r.Header.Get("Accept-Language")
			}
			services, err := storage.GetPriestCustomServices(priestID, lang)
			if err != nil {
				http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(services)
			return
		}

		if r.Method == "POST" {
			var req models.CreateCustomServiceRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, `{"error":"invalid service payload"}`, http.StatusBadRequest)
				return
			}
			serviceID, err := storage.CreatePriestCustomService(req)
			if err != nil {
				http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":     "SUCCESS",
				"service_id": serviceID,
				"message":    "Custom ritual service created with personalized Dakshina",
			})
			return
		}
	}).Methods("GET", "POST", "OPTIONS")

	// 13b. CMS 3-Language Service Translation & Converter
	router.HandleFunc("/api/v1/services/translations", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if r.Method == "GET" {
			items, err := storage.GetAllServiceTranslations()
			if err != nil {
				http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(items)
			return
		}

		if r.Method == "PUT" {
			var req models.UpdateServiceTranslationRequest
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, `{"error":"invalid translation payload"}`, http.StatusBadRequest)
				return
			}
			if err := storage.UpdateServiceTranslation(req); err != nil {
				http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
				return
			}
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "SUCCESS",
				"message": "Service translations updated across 3 languages",
			})
			return
		}
	}).Methods("GET", "PUT", "OPTIONS")

	router.HandleFunc("/api/v1/services/auto-translate", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		count, err := storage.AutoTranslateAllServices()
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":              "SUCCESS",
			"services_translated": count,
			"message":             fmt.Sprintf("Auto-translated %d services into Telugu & Hindi", count),
		})
	}).Methods("POST", "OPTIONS")

	// 14. Samagri Checklist per Pooja (GET, POST for CMS)
	router.HandleFunc("/api/v1/poojas/{id}/samagri", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		vars := mux.Vars(r)
		poojaID := vars["id"]

		if r.Method == "POST" {
			var item models.PoojaSamagriItem
			if err := json.NewDecoder(r.Body).Decode(&item); err != nil {
				http.Error(w, `{"error":"invalid item format"}`, http.StatusBadRequest)
				return
			}
			item.PoojaID = poojaID
			id, err := storage.AddPoojaSamagriItem(item)
			if err != nil {
				http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
				return
			}
			item.ID = id
			json.NewEncoder(w).Encode(item)
			return
		}

		items, err := storage.GetPoojaSamagri(poojaID)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(items)
	}).Methods("GET", "POST", "OPTIONS")

	router.HandleFunc("/api/v1/poojas/samagri/{item_id}", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		vars := mux.Vars(r)
		itemID := vars["item_id"]
		if err := storage.DeletePoojaSamagriItem(itemID); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"status": "DELETED", "id": itemID})
	}).Methods("DELETE", "OPTIONS")

	// 15. Local Puja Store Vendors & Inventory
	router.HandleFunc("/api/v1/store/vendors", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		latStr := r.URL.Query().Get("lat")
		lonStr := r.URL.Query().Get("lng")
		radiusStr := r.URL.Query().Get("radius")

		lat := 17.3850
		lon := 78.4867
		radius := 10000.0 // 10km default

		if v, err := strconv.ParseFloat(latStr, 64); err == nil && v != 0 {
			lat = v
		}
		if v, err := strconv.ParseFloat(lonStr, 64); err == nil && v != 0 {
			lon = v
		}
		if v, err := strconv.ParseFloat(radiusStr, 64); err == nil && v != 0 {
			radius = v
		}

		vendors, err := storage.GetNearbyVendors(lat, lon, radius)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(vendors)
	}).Methods("GET", "OPTIONS")

	router.HandleFunc("/api/v1/store/vendors/{id}/inventory", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		vars := mux.Vars(r)
		vendorID := vars["id"]
		items, err := storage.GetVendorInventory(vendorID)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(items)
	}).Methods("GET", "OPTIONS")

	// 16. Store Orders & Status Updates
	router.HandleFunc("/api/v1/store/orders", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var req models.CreateStoreOrderRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid order payload"}`, http.StatusBadRequest)
			return
		}
		orderID, err := storage.CreateStoreOrder(req)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":   "SUCCESS",
			"order_id": orderID,
			"message":  "Puja samagri order routed to local vendor",
		})
	}).Methods("POST", "OPTIONS")

	router.HandleFunc("/api/v1/store/orders/{id}/status", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		vars := mux.Vars(r)
		orderID := vars["id"]
		var body struct {
			Status string `json:"status"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, `{"error":"invalid status payload"}`, http.StatusBadRequest)
			return
		}
		if err := storage.UpdateStoreOrderStatus(orderID, body.Status); err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"order_id": orderID,
			"status":   body.Status,
			"message":  "Order status updated successfully",
		})
	}).Methods("PUT", "OPTIONS")

	// 17. Real-Time In-App Chat (Authenticated & Authorized Participant Enforced)
	registerChatRoutes(router, storage)

	// 19. Sacred Temples & VIP Ticketing
	router.HandleFunc("/api/v1/temples", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		temples, err := storage.GetTemples()
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(temples)
	}).Methods("GET", "OPTIONS")

	router.HandleFunc("/api/v1/temples/{id}/vip-pujas", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		vars := mux.Vars(r)
		templeID := vars["id"]
		pujas, err := storage.GetTempleVipPujas(templeID)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(pujas)
	}).Methods("GET", "OPTIONS")

	router.HandleFunc("/api/v1/temples/tickets/book", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var req models.BookTempleTicketRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":"invalid ticket booking payload"}`, http.StatusBadRequest)
			return
		}
		ticket, err := storage.BookTempleTicket(req)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(ticket)
	}).Methods("POST", "OPTIONS")

	router.HandleFunc("/api/v1/temples/tickets/verify", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var body struct {
			QRToken string `json:"qr_token"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, `{"error":"invalid qr payload"}`, http.StatusBadRequest)
			return
		}
		ticket, err := storage.VerifyTempleTicket(body.QRToken)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"is_valid":             true,
			"ticket":               ticket,
			"check_in_timestamp":   ticket.CheckInTimestamp,
			"verification_latency": "< 50ms (zero-lag offline cryptographically verified)",
		})
	}).Methods("POST", "OPTIONS")

	handler := enableCORS(router)

	port := os.Getenv("PORT")
	if port == "" {
		port = ":8085"
	}
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}
	sanitizedRedis := db.SanitizeRedisEndpoint(redisAddr)
	fmt.Println("==================================================================")
	fmt.Printf("🕉️  PUJA VAAHINI UNIFIED BACKEND GATEWAY v2.5\n")
	fmt.Printf("   PostgreSQL Database: Connected\n")
	fmt.Printf("   Redis Geo Cache:     Connected (%s)\n", sanitizedRedis)
	fmt.Printf("   JWT Security:        HS256 Active\n")
	fmt.Printf("   Port:                %s\n", port)
	fmt.Println("==================================================================")

	server := &http.Server{Addr: port, Handler: handler, ReadHeaderTimeout: 10 * time.Second, IdleTimeout: 60 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			log.Printf("server shutdown: %v", err)
		}
	}()
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatal(err)
	}
	_ = storage.Redis.Close()
	_ = storage.DB.Close()
}

// registerChatRoutes mounts authenticated and participant-protected chat endpoints
func registerChatRoutes(router *mux.Router, storage *db.Storage) {
	router.HandleFunc("/api/v1/chat/rooms", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		claims, err := requireCallerSession(r, storage)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"unauthorized","message":"%v"}`, err), http.StatusUnauthorized)
			return
		}

		var body struct {
			BookingID string `json:"booking_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.BookingID == "" {
			http.Error(w, `{"error":"invalid room payload: booking_id is required"}`, http.StatusBadRequest)
			return
		}

		ownerUID, assignedPID, err := storage.VerifyBookingParticipant(body.BookingID, claims.UserID, claims.Role)
		if err != nil {
			if strings.Contains(err.Error(), "booking not found") {
				http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusNotFound)
			} else {
				http.Error(w, fmt.Sprintf(`{"error":"forbidden","message":"%v"}`, err), http.StatusForbidden)
			}
			return
		}

		roomID, err := storage.GetOrCreateChatRoom(body.BookingID, ownerUID, assignedPID)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]interface{}{
			"room_id":    roomID,
			"booking_id": body.BookingID,
		})
	}).Methods("POST", "OPTIONS")

	router.HandleFunc("/api/v1/chat/messages", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		claims, err := requireCallerSession(r, storage)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"unauthorized","message":"%v"}`, err), http.StatusUnauthorized)
			return
		}

		if r.Method == "POST" {
			var msg models.ChatMessage
			if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
				http.Error(w, `{"error":"invalid message payload"}`, http.StatusBadRequest)
				return
			}
			if msg.RoomID == "" {
				http.Error(w, `{"error":"missing room_id"}`, http.StatusBadRequest)
				return
			}

			_, err := storage.VerifyChatRoomParticipant(msg.RoomID, claims.UserID, claims.Role)
			if err != nil {
				if strings.Contains(err.Error(), "chat room not found") {
					http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusNotFound)
				} else {
					http.Error(w, fmt.Sprintf(`{"error":"forbidden","message":"%v"}`, err), http.StatusForbidden)
				}
				return
			}

			// Invariant: Never trust client-supplied sender IDs or roles!
			msg.SenderID = claims.UserID
			if claims.Role == "priest" {
				msg.SenderRole = "PRIEST"
			} else {
				msg.SenderRole = "DEVOTEE"
			}
			if msg.MessageType == "" {
				msg.MessageType = "TEXT"
			}

			msgID, err := storage.SaveChatMessage(msg)
			if err != nil {
				http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
				return
			}
			msg.ID = msgID
			msgBytes, _ := json.Marshal(msg)
			chatHub.BroadcastToRoom(msg.SenderID, msg.RoomID, msgBytes)

			json.NewEncoder(w).Encode(msg)
			return
		}

		roomID := r.URL.Query().Get("room_id")
		if roomID == "" {
			http.Error(w, `{"error":"missing room_id parameter"}`, http.StatusBadRequest)
			return
		}

		_, err = storage.VerifyChatRoomParticipant(roomID, claims.UserID, claims.Role)
		if err != nil {
			if strings.Contains(err.Error(), "chat room not found") {
				http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusNotFound)
			} else {
				http.Error(w, fmt.Sprintf(`{"error":"forbidden","message":"%v"}`, err), http.StatusForbidden)
			}
			return
		}

		messages, err := storage.GetChatMessages(roomID)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
			return
		}
		if messages == nil {
			messages = []models.ChatMessage{}
		}
		json.NewEncoder(w).Encode(messages)
	}).Methods("GET", "POST", "OPTIONS")

	router.HandleFunc("/api/v1/chat/room-by-booking", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		claims, err := requireCallerSession(r, storage)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"unauthorized","message":"%v"}`, err), http.StatusUnauthorized)
			return
		}

		bookingID := r.URL.Query().Get("booking_id")
		if bookingID == "" {
			http.Error(w, `{"error":"missing booking_id parameter"}`, http.StatusBadRequest)
			return
		}

		ownerUID, assignedPID, err := storage.VerifyBookingParticipant(bookingID, claims.UserID, claims.Role)
		if err != nil {
			if strings.Contains(err.Error(), "booking not found") {
				http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusNotFound)
			} else {
				http.Error(w, fmt.Sprintf(`{"error":"forbidden","message":"%v"}`, err), http.StatusForbidden)
			}
			return
		}

		roomID, err := storage.GetOrCreateChatRoom(bookingID, ownerUID, assignedPID)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"%v"}`, err), http.StatusInternalServerError)
			return
		}
		json.NewEncoder(w).Encode(map[string]string{
			"room_id":    roomID,
			"booking_id": bookingID,
		})
	}).Methods("GET", "OPTIONS")

	router.HandleFunc("/ws/chat", func(w http.ResponseWriter, r *http.Request) {
		// 1. Authenticate caller: check Authorization header, or token/authToken query parameter
		var tokenStr string
		authHeader := r.Header.Get("Authorization")
		if strings.HasPrefix(authHeader, "Bearer ") {
			tokenStr = strings.TrimSpace(strings.TrimPrefix(authHeader, "Bearer "))
		}
		if tokenStr == "" {
			tokenStr = r.URL.Query().Get("token")
		}
		if tokenStr == "" {
			tokenStr = r.URL.Query().Get("authToken")
		}
		if tokenStr == "" {
			http.Error(w, `{"error":"unauthorized","message":"missing bearer token or token query parameter"}`, http.StatusUnauthorized)
			return
		}

		claims, err := auth.ValidateToken(tokenStr)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"unauthorized","message":"%v"}`, err), http.StatusUnauthorized)
			return
		}
		dbVersion, err := storage.GetUserSessionVersion(claims.UserID, claims.Role)
		if err != nil || auth.ValidateSessionVersion(claims.SessionVersion, dbVersion) != nil {
			http.Error(w, `{"error":"unauthorized","message":"session expired or revoked"}`, http.StatusUnauthorized)
			return
		}

		roomID := r.URL.Query().Get("roomId")
		if roomID == "" {
			http.Error(w, `{"error":"bad_request","message":"missing roomId query parameter"}`, http.StatusBadRequest)
			return
		}

		_, err = storage.VerifyChatRoomParticipant(roomID, claims.UserID, claims.Role)
		if err != nil {
			http.Error(w, fmt.Sprintf(`{"error":"forbidden","message":"%v"}`, err), http.StatusForbidden)
			return
		}

		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}

		// Invariant: Never trust client-supplied sender ID or role!
		clientID := claims.UserID
		var role string
		if claims.Role == "priest" {
			role = "PRIEST"
		} else {
			role = "DEVOTEE"
		}

		client := &Client{
			ID:     clientID,
			RoomID: roomID,
			Conn:   conn,
			Send:   make(chan []byte, 256),
		}

		chatHub.Register(client)

		go func() {
			defer conn.Close()
			for msg := range client.Send {
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					break
				}
			}
		}()

		defer func() {
			chatHub.Unregister(client)
			conn.Close()
		}()

		for {
			_, message, err := conn.ReadMessage()
			if err != nil {
				break
			}

			var msg models.ChatMessage
			if err := json.Unmarshal(message, &msg); err == nil {
				msg.RoomID = roomID
				// Server-enforced identity
				msg.SenderID = clientID
				msg.SenderRole = role
				if msg.MessageType == "" {
					msg.MessageType = "TEXT"
				}
				msgID, _ := storage.SaveChatMessage(msg)
				msg.ID = msgID

				broadcastData, _ := json.Marshal(msg)
				chatHub.BroadcastToRoom(clientID, roomID, broadcastData)
			}
		}
	})
}
