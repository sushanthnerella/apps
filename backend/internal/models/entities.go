package models

import "time"

type BookingState string

const (
	StateRequested    BookingState = "REQUESTED"
	StateDispatched   BookingState = "DISPATCHED"
	StateAccepted     BookingState = "ACCEPTED"
	StateInTransit    BookingState = "IN_TRANSIT"
	StateArrived      BookingState = "ARRIVED"
	StatePoojaStarted BookingState = "POOJA_STARTED"
	StateCompleted    BookingState = "COMPLETED"
	StateCancelled    BookingState = "CANCELLED"
)

type LoginRequest struct {
	Phone string `json:"phone"`
	OTP   string `json:"otp"`
	Role  string `json:"role"` // "devotee" or "priest"
}

type AuthResponse struct {
	Status      string      `json:"status"`
	AccessToken string      `json:"access_token"`
	TokenType   string      `json:"token_type"`
	User        UserProfile `json:"user"`
	Message     string      `json:"message"`
}

type UserProfile struct {
	ID            string    `json:"id"`
	Phone         string    `json:"phone"`
	FullName      string    `json:"full_name"`
	Gotram        string    `json:"gotram,omitempty"`
	Nakshatram    string    `json:"nakshatram,omitempty"`
	Rashi         string    `json:"rashi,omitempty"`
	City          string    `json:"city"`
	Role          string    `json:"role"`
	ConsentAgreed bool      `json:"consent_agreed"`
	CreatedAt     time.Time `json:"created_at"`
}

type DevoteeRegistrationRequest struct {
	FullName      string `json:"full_name"`
	Phone         string `json:"phone"`
	Gotram        string `json:"gotram"`
	Nakshatram    string `json:"nakshatram"`
	Rashi         string `json:"rashi"`
	City          string `json:"city"`
	ConsentAgreed bool   `json:"consent_agreed"`
}

type PriestRegistrationRequest struct {
	FullName        string  `json:"full_name"`
	Phone           string  `json:"phone"`
	Sampradaya      string  `json:"sampradaya"`
	VedaShakha      string  `json:"veda_shakha"`
	ExperienceYears int     `json:"experience_years"`
	City            string  `json:"city"`
	Latitude        float64 `json:"latitude"`
	Longitude       float64 `json:"longitude"`
	ConsentAgreed   bool    `json:"consent_agreed"`
}

type NearbyPriest struct {
	ID                   string   `json:"id"`
	FullName             string   `json:"full_name"`
	Phone                string   `json:"phone"`
	Sampradaya           string   `json:"sampradaya"`
	VedaShakha           string   `json:"veda_shakha"`
	RatingAvg            float64  `json:"rating_avg"`
	TotalCompletedPoojas int      `json:"total_completed_poojas"`
	RepeatBookingCount   int      `json:"repeat_booking_count"`
	DistanceMeters       float64  `json:"distance_meters"`
	DistanceKm           float64  `json:"distance_km"`
	ETAMinutes           int      `json:"eta_minutes"`
	VehicleType          string   `json:"vehicle_type"`
	Latitude             float64  `json:"latitude"`
	Longitude            float64  `json:"longitude"`
	ServicesOffered      []string `json:"services_offered"`
	Score                float64  `json:"score"`
}

type PriestCandidate struct {
	ID                   string  `json:"id"`
	FullName             string  `json:"full_name"`
	Phone                string  `json:"phone"`
	Sampradaya           string  `json:"sampradaya"`
	VedaShakha           string  `json:"veda_shakha,omitempty"`
	RatingAvg            float64 `json:"rating_avg"`
	TotalCompletedPoojas int     `json:"total_completed_poojas"`
	RepeatBookingCount   int     `json:"repeat_booking_count"`
	AcceptanceRate       float64 `json:"acceptance_rate"`
	DistanceMeters       float64 `json:"distance_meters"`
	Score                float64 `json:"score"`
}

type DispatchRequest struct {
	BookingID           string  `json:"booking_id"`
	PoojaTitle          string  `json:"pooja_title"`
	DevoteeLatitude     float64 `json:"devotee_latitude"`
	DevoteeLongitude    float64 `json:"devotee_longitude"`
	RequiredSampradaya  string  `json:"required_sampradaya"`
	SearchRadiusMeters  float64 `json:"search_radius_meters"`
	EstimatedTotalOffer float64 `json:"estimated_total_offer"`
}

type BookingCreateRequest struct {
	UserID            string  `json:"user_id"`
	PriestID          string  `json:"priest_id"`
	PoojaID           string  `json:"pooja_id"`
	BookingType       string  `json:"booking_type"`
	IncludeSamagriKit bool    `json:"include_samagri_kit"`
	TotalAmount       float64 `json:"total_amount"`
	EventAddress      string  `json:"event_address"`
	Latitude          float64 `json:"latitude"`
	Longitude         float64 `json:"longitude"`
	ScheduledDate     string  `json:"scheduled_date"`
	SlotTime          string  `json:"slot_time"`
	DevoteeGotram     string  `json:"devotee_gotram"`
	DevoteeNakshatram string  `json:"devotee_nakshatram"`
	DevoteeRashi      string  `json:"devotee_rashi"`
	SankalpamPurpose  string  `json:"sankalpam_purpose"`
}

type BookingDetailResponse struct {
	ID                string    `json:"id"`
	UserID            string    `json:"user_id"`
	PriestID          string    `json:"priest_id"`
	DevoteeName       string    `json:"devotee_name"`
	DevoteePhone      string    `json:"devotee_phone"`
	PriestName        string    `json:"priest_name"`
	PriestPhone       string    `json:"priest_phone"`
	PoojaTitle        string    `json:"pooja_title"`
	BookingType       string    `json:"booking_type"`
	Status            string    `json:"status"`
	EscrowStatus      string    `json:"escrow_status"`
	TotalAmount       float64   `json:"total_amount"`
	IncludeSamagriKit bool      `json:"include_samagri_kit"`
	CompletionOTP     string    `json:"completion_otp"`
	EventAddress      string    `json:"event_address"`
	Latitude          float64   `json:"latitude"`
	Longitude         float64   `json:"longitude"`
	ScheduledDate     string    `json:"scheduled_date"`
	SlotTime          string    `json:"slot_time"`
	DevoteeGotram     string    `json:"devotee_gotram"`
	DevoteeNakshatram string    `json:"devotee_nakshatram"`
	DevoteeRashi      string    `json:"devotee_rashi"`
	SankalpamPurpose  string    `json:"sankalpam_purpose"`
	ChatRoomID        string    `json:"chat_room_id"`
	CreatedAt         time.Time `json:"created_at"`
}

type BookingStatusUpdateRequest struct {
	BookingID      string  `json:"booking_id"`
	Status         string  `json:"status"`
	PriestID       string  `json:"priest_id,omitempty"`
	CompletionOTP  string  `json:"completion_otp,omitempty"`
	DakshinaAmount float64 `json:"dakshina_amount,omitempty"`
}

type WebRTCMessage struct {
	Type        string `json:"type"` // OFFER, ANSWER, ICE_CANDIDATE, JOIN_ROOM, LEAVE_ROOM
	RoomID      string `json:"room_id"`
	SenderID    string `json:"sender_id"`
	RecipientID string `json:"recipient_id,omitempty"`
	SDP         string `json:"sdp,omitempty"`
	Candidate   string `json:"candidate,omitempty"`
	StreamMode  string `json:"stream_mode"` // ONE_TO_ONE_PRIVATE, ONE_TO_MANY_BROADCAST
}

type PanchangamDay struct {
	Date          time.Time `json:"date"`
	Tithi         string    `json:"tithi"`
	Nakshatra     string    `json:"nakshatra"`
	Yoga          string    `json:"yoga"`
	Karana        string    `json:"karana"`
	RahuKalam     string    `json:"rahu_kalam"`
	Yamagandam    string    `json:"yamagandam"`
	GulikaKalam   string    `json:"gulika_kalam"`
	AuspiciousDay bool      `json:"is_auspicious"`
}

// -------------------------------------------------------------
// Module A: Slots & Custom Services
// -------------------------------------------------------------
type PriestSlot struct {
	ID        string    `json:"id"`
	PriestID  string    `json:"priest_id"`
	SlotDate  string    `json:"slot_date"`
	StartTime string    `json:"start_time"`
	EndTime   string    `json:"end_time"`
	IsBooked  bool      `json:"is_booked"`
	IsHeld    bool      `json:"is_held"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateSlotRequest struct {
	PriestID  string `json:"priest_id"`
	SlotDate  string `json:"slot_date"`
	StartTime string `json:"start_time"`
	EndTime   string `json:"end_time"`
}

type HoldSlotRequest struct {
	SlotID              string `json:"slot_id"`
	DevoteeID           string `json:"devotee_id"`
	HoldDurationSeconds int    `json:"hold_duration_seconds"`
}

type PriestCustomService struct {
	ID               string    `json:"id"`
	PriestID         string    `json:"priest_id"`
	Title            string    `json:"title"`
	TitleEn          string    `json:"title_en"`
	TitleTe          string    `json:"title_te"`
	TitleHi          string    `json:"title_hi"`
	Category         string    `json:"category"`
	DakshinaFee      float64   `json:"dakshina_fee"`
	SamagriKitPrice  float64   `json:"samagri_kit_price"`
	DurationMinutes  int       `json:"duration_minutes"`
	IsOnlineEligible bool      `json:"is_online_eligible"`
	SamagriChecklist []string  `json:"samagri_checklist"`
	Description      string    `json:"description"`
	DescriptionEn    string    `json:"description_en"`
	DescriptionTe    string    `json:"description_te"`
	DescriptionHi    string    `json:"description_hi"`
	IsActive         bool      `json:"is_active"`
	CreatedAt        time.Time `json:"created_at"`
}

type CreateCustomServiceRequest struct {
	PriestID         string   `json:"priest_id"`
	Title            string   `json:"title"`
	TitleEn          string   `json:"title_en,omitempty"`
	TitleTe          string   `json:"title_te,omitempty"`
	TitleHi          string   `json:"title_hi,omitempty"`
	Category         string   `json:"category"`
	DakshinaFee      float64  `json:"dakshina_fee"`
	SamagriKitPrice  float64  `json:"samagri_kit_price"`
	DurationMinutes  int      `json:"duration_minutes"`
	IsOnlineEligible bool     `json:"is_online_eligible"`
	SamagriChecklist []string `json:"samagri_checklist"`
	Description      string   `json:"description"`
	DescriptionEn    string   `json:"description_en,omitempty"`
	DescriptionTe    string   `json:"description_te,omitempty"`
	DescriptionHi    string   `json:"description_hi,omitempty"`
}

type ServiceTranslationItem struct {
	ID            string  `json:"id"`
	Type          string  `json:"type"` // "CATALOG" or "PRIEST_CUSTOM"
	PriestID      string  `json:"priest_id,omitempty"`
	PriestName    string  `json:"priest_name,omitempty"`
	OriginalTitle string  `json:"original_title"`
	TitleEn       string  `json:"title_en"`
	TitleTe       string  `json:"title_te"`
	TitleHi       string  `json:"title_hi"`
	DescriptionEn string  `json:"description_en"`
	DescriptionTe string  `json:"description_te"`
	DescriptionHi string  `json:"description_hi"`
	Category      string  `json:"category"`
	BasePrice     float64 `json:"base_price"`
}

type UpdateServiceTranslationRequest struct {
	ID            string `json:"id"`
	Type          string `json:"type"` // "CATALOG" or "PRIEST_CUSTOM"
	TitleEn       string `json:"title_en"`
	TitleTe       string `json:"title_te"`
	TitleHi       string `json:"title_hi"`
	DescriptionEn string `json:"description_en"`
	DescriptionTe string `json:"description_te"`
	DescriptionHi string `json:"description_hi"`
}

// -------------------------------------------------------------
// Module B: Samagri & Puja Store
// -------------------------------------------------------------
type PoojaSamagriItem struct {
	ID             string  `json:"id"`
	PoojaID        string  `json:"pooja_id"`
	ItemName       string  `json:"item_name"`
	Category       string  `json:"category"`
	Quantity       string  `json:"quantity"`
	IsMandatory    bool    `json:"is_mandatory"`
	EstimatedPrice float64 `json:"estimated_price"`
}

type BookingMaterial struct {
	ID                  string `json:"id"`
	BookingID           string `json:"booking_id"`
	ItemName            string `json:"item_name"`
	Category            string `json:"category"`
	Quantity            string `json:"quantity"`
	IsIncludedByPriest  bool   `json:"is_included_by_priest"`
	IsFulfilledByVendor bool   `json:"is_fulfilled_by_vendor"`
	VendorOrderID       string `json:"vendor_order_id,omitempty"`
}

type Vendor struct {
	ID                string  `json:"id"`
	ShopName          string  `json:"shop_name"`
	OwnerName         string  `json:"owner_name"`
	Phone             string  `json:"phone"`
	ProfilePictureURL string  `json:"profile_picture_url"`
	Address           string  `json:"address"`
	City              string  `json:"city"`
	Latitude          float64 `json:"latitude"`
	Longitude         float64 `json:"longitude"`
	RatingAvg         float64 `json:"rating_avg"`
	IsOpen            bool    `json:"is_open"`
	DistanceKm        float64 `json:"distance_km,omitempty"`
}

type VendorInventoryItem struct {
	ID            string  `json:"id"`
	VendorID      string  `json:"vendor_id"`
	ItemName      string  `json:"item_name"`
	Category      string  `json:"category"`
	Price         float64 `json:"price"`
	StockQuantity int     `json:"stock_quantity"`
	Unit          string  `json:"unit"`
	ImageURL      string  `json:"image_url"`
	IsAvailable   bool    `json:"is_available"`
}

type StoreOrderItem struct {
	ID              string  `json:"id,omitempty"`
	OrderID         string  `json:"order_id,omitempty"`
	InventoryItemID string  `json:"inventory_item_id"`
	ItemName        string  `json:"item_name"`
	Quantity        int     `json:"quantity"`
	UnitPrice       float64 `json:"unit_price"`
	Subtotal        float64 `json:"subtotal"`
}

type StoreOrder struct {
	ID              string           `json:"id"`
	UserID          string           `json:"user_id"`
	VendorID        string           `json:"vendor_id"`
	BookingID       string           `json:"booking_id,omitempty"`
	Status          string           `json:"status"`
	TotalAmount     float64          `json:"total_amount"`
	DeliveryAddress string           `json:"delivery_address"`
	DeliverySlot    string           `json:"delivery_slot,omitempty"`
	Items           []StoreOrderItem `json:"items,omitempty"`
	CreatedAt       time.Time        `json:"created_at"`
}

type CreateStoreOrderRequest struct {
	UserID          string           `json:"user_id"`
	VendorID        string           `json:"vendor_id"`
	BookingID       string           `json:"booking_id,omitempty"`
	DeliveryAddress string           `json:"delivery_address"`
	DeliverySlot    string           `json:"delivery_slot,omitempty"`
	Items           []StoreOrderItem `json:"items"`
}

// -------------------------------------------------------------
// Module C & D: Chat & Priest Referrals
// -------------------------------------------------------------
type ChatRoom struct {
	ID        string `json:"id"`
	BookingID string `json:"booking_id"`
	UserID    string `json:"user_id"`
	PriestID  string `json:"priest_id"`
	IsActive  bool   `json:"is_active"`
}

type ChatMessage struct {
	ID            string    `json:"id"`
	RoomID        string    `json:"room_id"`
	SenderID      string    `json:"sender_id"`
	SenderRole    string    `json:"sender_role"`
	MessageType   string    `json:"message_type"`
	Content       string    `json:"content"`
	AttachmentURL string    `json:"attachment_url"`
	IsRead        bool      `json:"is_read"`
	CreatedAt     time.Time `json:"created_at"`
}

type PriestReferralRequest struct {
	BookingID         string `json:"booking_id"`
	ReferringPriestID string `json:"referring_priest_id"`
	ReferredPriestID  string `json:"referred_priest_id"`
	Reason            string `json:"reason"`
}

// -------------------------------------------------------------
// Module E: Temples & VIP Ticketing
// -------------------------------------------------------------
type Temple struct {
	ID             string  `json:"id"`
	Name           string  `json:"name"`
	Slug           string  `json:"slug"`
	Deity          string  `json:"deity"`
	Description    string  `json:"description"`
	City           string  `json:"city"`
	State          string  `json:"state"`
	Latitude       float64 `json:"latitude"`
	Longitude      float64 `json:"longitude"`
	BannerImageURL string  `json:"banner_image_url"`
	Timings        string  `json:"timings"`
	ContactPhone   string  `json:"contact_phone"`
	IsActive       bool    `json:"is_active"`
}

type TempleVipPuja struct {
	ID                string  `json:"id"`
	TempleID          string  `json:"temple_id"`
	Title             string  `json:"title"`
	Description       string  `json:"description"`
	Price             float64 `json:"price"`
	MaxDailySlots     int     `json:"max_daily_slots"`
	DurationMinutes   int     `json:"duration_minutes"`
	IsOnlineVideoCall bool    `json:"is_online_video_call"`
	IsActive          bool    `json:"is_active"`
}

type TempleVipTicket struct {
	ID                string     `json:"id"`
	TempleID          string     `json:"temple_id"`
	VipPujaID         string     `json:"vip_puja_id"`
	UserID            string     `json:"user_id,omitempty"`
	DevoteeName       string     `json:"devotee_name"`
	DevoteeGotram     string     `json:"devotee_gotram,omitempty"`
	DevoteeNakshatram string     `json:"devotee_nakshatram,omitempty"`
	DevoteePhone      string     `json:"devotee_phone"`
	VisitDate         string     `json:"visit_date"`
	SlotTime          string     `json:"slot_time"`
	TicketType        string     `json:"ticket_type"`
	QRToken           string     `json:"qr_token"`
	Status            string     `json:"status"`
	AmountPaid        float64    `json:"amount_paid"`
	PaymentMode       string     `json:"payment_mode"`
	CheckInTimestamp  *time.Time `json:"check_in_timestamp,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

type BookTempleTicketRequest struct {
	TempleID          string  `json:"temple_id"`
	VipPujaID         string  `json:"vip_puja_id"`
	UserID            string  `json:"user_id,omitempty"`
	DevoteeName       string  `json:"devotee_name"`
	DevoteeGotram     string  `json:"devotee_gotram,omitempty"`
	DevoteeNakshatram string  `json:"devotee_nakshatram,omitempty"`
	DevoteePhone      string  `json:"devotee_phone"`
	VisitDate         string  `json:"visit_date"`
	SlotTime          string  `json:"slot_time"`
	TicketType        string  `json:"ticket_type"`
	AmountPaid        float64 `json:"amount_paid"`
	PaymentMode       string  `json:"payment_mode"`
}

// -------------------------------------------------------------
// Module G: Swiggy/Rapido Cancellation & Refund Quotes
// -------------------------------------------------------------
type CancellationQuote struct {
	BookingID          string  `json:"booking_id"`
	CurrentStatus      string  `json:"current_status"`
	AdvancePaid        float64 `json:"advance_paid"`
	CancellationFee    float64 `json:"cancellation_fee"`
	TravelFee          float64 `json:"travel_fee"`
	PriestCompensation float64 `json:"priest_compensation"`
	RefundAmount       float64 `json:"refund_amount"`
	IsFreeCancellation bool    `json:"is_free_cancellation"`
	PolicyNotice       string  `json:"policy_notice"`
}

type CancelBookingRequest struct {
	BookingID       string `json:"booking_id"`
	CancelledByRole string `json:"cancelled_by_role"`
	ReasonCode      string `json:"reason_code"`
	ReasonText      string `json:"reason_text"`
}

// -------------------------------------------------------------
// Module H: Live Dispatch & Geo-Tracking Radar
// -------------------------------------------------------------
type DispatchRadarPriest struct {
	ID              string   `json:"id"`
	FullName        string   `json:"full_name"`
	Phone           string   `json:"phone"`
	Sampradaya      string   `json:"sampradaya"`
	VedaShakha      string   `json:"veda_shakha"`
	RatingAvg       float64  `json:"rating_avg"`
	IsOnline        bool     `json:"is_online"`
	Latitude        float64  `json:"latitude"`
	Longitude       float64  `json:"longitude"`
	VehicleType     string   `json:"vehicle_type"`
	ServicesOffered []string `json:"services_offered"`
	ActiveBookingID string   `json:"active_booking_id,omitempty"`
	Status          string   `json:"status"`
}

type DispatchRadarDevotee struct {
	ID               string  `json:"id"`
	FullName         string  `json:"full_name"`
	Phone            string  `json:"phone"`
	Gotram           string  `json:"gotram"`
	Nakshatram       string  `json:"nakshatram"`
	Rashi            string  `json:"rashi"`
	Latitude         float64 `json:"latitude"`
	Longitude        float64 `json:"longitude"`
	City             string  `json:"city"`
	HasActiveBooking bool    `json:"has_active_booking"`
	ActiveBookingID  string  `json:"active_booking_id,omitempty"`
	ActivePoojaTitle string  `json:"active_pooja_title,omitempty"`
	Status           string  `json:"status,omitempty"`
	ScheduledDate    string  `json:"scheduled_date,omitempty"`
	SlotTime         string  `json:"slot_time,omitempty"`
	AssignedPriest   string  `json:"assigned_priest,omitempty"`
	EventAddress     string  `json:"event_address,omitempty"`
}

type DispatchRadarResponse struct {
	TotalOnlinePriests   int                    `json:"total_online_priests"`
	TotalActiveDevotees  int                    `json:"total_active_devotees"`
	PendingRequestsCount int                    `json:"pending_requests_count"`
	OnlinePriests        []DispatchRadarPriest  `json:"online_priests"`
	ActiveDevotees       []DispatchRadarDevotee `json:"active_devotees"`
}
