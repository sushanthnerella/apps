package com.purohit.contracts.temple

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
    val bannerImageUrl: String = "",
    val timings: String = "06:00 AM - 08:30 PM",
    val contactPhone: String = "",
    val isActive: Boolean = true
)

@Serializable
data class TempleVipPuja(
    val id: String,
    val templeId: String,
    val title: String,
    val description: String = "",
    val price: Double,
    val maxDailySlots: Int = 20,
    val durationMinutes: Int = 45,
    val isOnlineVideoCall: Boolean = false,
    val isActive: Boolean = true
)

@Serializable
data class TempleVipTicket(
    val id: String,
    val templeId: String,
    val vipPujaId: String,
    val userId: String?,
    val devoteeName: String,
    val devoteeGotram: String? = null,
    val devoteeNakshatram: String? = null,
    val devoteePhone: String,
    val visitDate: String, // YYYY-MM-DD
    val slotTime: String,
    val ticketType: String = "VIP_IN_PERSON", // 'VIP_IN_PERSON' or 'ONLINE_E_POOJA_VIDEO'
    val qrToken: String,
    val status: String = "CONFIRMED", // 'CONFIRMED', 'CHECKED_IN', 'COMPLETED', 'CANCELLED'
    val amountPaid: Double,
    val paymentMode: String = "ONLINE_ESCROW",
    val checkInTimestamp: String? = null,
    val createdAt: String = ""
)

@Serializable
data class BookTempleTicketRequest(
    val templeId: String,
    val vipPujaId: String,
    val userId: String,
    val devoteeName: String,
    val devoteeGotram: String? = null,
    val devoteeNakshatram: String? = null,
    val devoteePhone: String,
    val visitDate: String,
    val slotTime: String,
    val ticketType: String = "VIP_IN_PERSON",
    val amountPaid: Double,
    val paymentMode: String = "ONLINE_ESCROW"
)

@Serializable
data class VerifyTicketResponse(
    val isValid: Boolean,
    val ticketId: String,
    val devoteeName: String,
    val status: String,
    val checkInTimestamp: String,
    val message: String
)
