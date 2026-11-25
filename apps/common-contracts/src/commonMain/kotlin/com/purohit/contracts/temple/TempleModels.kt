package com.purohit.contracts.temple

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class Temple(
    val id: String,
    val name: String,
    val slug: String,
    val deity: String,
    val description: String = "",
    val city: String,
    val state: String = "Telangana",
    val latitude: Double = 17.3850,
    val longitude: Double = 78.4867,
    @SerialName("banner_image_url") val bannerImageUrl: String = "",
    val timings: String = "06:00 AM - 08:30 PM",
    @SerialName("contact_phone") val contactPhone: String = "",
    @SerialName("is_active") val isActive: Boolean = true
)

@Serializable
data class TempleVipPuja(
    val id: String,
    @SerialName("temple_id") val templeId: String,
    val title: String,
    val description: String = "",
    val price: Double,
    @SerialName("price_minor") val priceMinor: Long = 0L,
    val currency: String = "INR",
    @SerialName("max_daily_slots") val maxDailySlots: Int = 20,
    @SerialName("duration_minutes") val durationMinutes: Int = 45,
    @SerialName("is_online_video_call") val isOnlineVideoCall: Boolean = false,
    @SerialName("is_active") val isActive: Boolean = true
)

@Serializable
data class TempleVipTicket(
    val id: String,
    @SerialName("temple_id") val templeId: String,
    @SerialName("vip_puja_id") val vipPujaId: String,
    @SerialName("user_id") val userId: String?,
    @SerialName("devotee_name") val devoteeName: String,
    @SerialName("devotee_gotram") val devoteeGotram: String? = null,
    @SerialName("devotee_nakshatram") val devoteeNakshatram: String? = null,
    @SerialName("devotee_phone") val devoteePhone: String,
    @SerialName("visit_date") val visitDate: String, // YYYY-MM-DD
    @SerialName("slot_time") val slotTime: String,
    @SerialName("ticket_type") val ticketType: String = "VIP_IN_PERSON", // 'VIP_IN_PERSON' or 'ONLINE_E_POOJA_VIDEO'
    @SerialName("qr_token") val qrToken: String,
    val status: String = "CONFIRMED", // 'CONFIRMED', 'CHECKED_IN', 'COMPLETED', 'CANCELLED'
    @SerialName("amount_paid") val amountPaid: Double,
    @SerialName("amount_paid_minor") val amountPaidMinor: Long = 0L,
    val currency: String = "INR",
    @SerialName("payment_mode") val paymentMode: String = "ONLINE_ESCROW",
    @SerialName("check_in_timestamp") val checkInTimestamp: String? = null,
    @SerialName("created_at") val createdAt: String = ""
)

@Serializable
data class BookTempleTicketRequest(
    @SerialName("temple_id") val templeId: String,
    @SerialName("vip_puja_id") val vipPujaId: String,
    @SerialName("user_id") val userId: String,
    @SerialName("devotee_name") val devoteeName: String,
    @SerialName("devotee_gotram") val devoteeGotram: String? = null,
    @SerialName("devotee_nakshatram") val devoteeNakshatram: String? = null,
    @SerialName("devotee_phone") val devoteePhone: String,
    @SerialName("visit_date") val visitDate: String,
    @SerialName("slot_time") val slotTime: String,
    @SerialName("ticket_type") val ticketType: String = "VIP_IN_PERSON",
    @SerialName("amount_paid") val amountPaid: Double,
    @SerialName("amount_paid_minor") val amountPaidMinor: Long = 0L,
    val currency: String = "INR",
    @SerialName("payment_mode") val paymentMode: String = "ONLINE_ESCROW"
)

@Serializable
data class VerifyTicketResponse(
    @SerialName("is_valid") val isValid: Boolean,
    @SerialName("ticket_id") val ticketId: String,
    @SerialName("devotee_name") val devoteeName: String,
    val status: String,
    @SerialName("check_in_timestamp") val checkInTimestamp: String,
    val message: String
)
