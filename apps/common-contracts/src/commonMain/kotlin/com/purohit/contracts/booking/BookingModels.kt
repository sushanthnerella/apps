package com.purohit.contracts.booking

import kotlinx.serialization.Serializable

@Serializable
enum class BookingState {
    REQUESTED,
    DISPATCHED,
    ACCEPTED,
    IN_TRANSIT,
    ARRIVED,
    POOJA_STARTED,
    COMPLETED,
    CANCELLED
}

@Serializable
enum class BookingType {
    INSTANT_DISPATCH,
    SCHEDULED_MUHURTHAM,
    ONLINE_TELECAST
}

@Serializable
enum class PaymentEscrowStatus {
    PENDING,
    HELD_IN_ESCROW,
    RELEASED_TO_PRIEST,
    REFUNDED
}

@Serializable
data class SamagriItem(
    val id: String,
    val name: String,
    val quantity: String,
    val isPerishable: Boolean // true: flowers, fruits, milk; false: camphor, ghee, kumkum
)

@Serializable
data class PoojaPackage(
    val id: String,
    val title: String,
    val category: String, // Gruhapravesam, Satyanarayana Vratam, Ganapathi Homa, etc.
    val basePrice: Double,
    val samagriKitPrice: Double,
    val durationMinutes: Int,
    val isOnlineEligible: Boolean,
    val samagriChecklist: List<SamagriItem> = emptyList()
)

@Serializable
data class Booking(
    val id: String,
    val userId: String,
    val userName: String,
    val userPhone: String,
    val priestId: String? = null,
    val priestName: String? = null,
    val priestPhone: String? = null,
    val poojaPackage: PoojaPackage,
    val bookingType: BookingType,
    val state: BookingState,
    val includeSamagriKit: Boolean,
    val totalAmount: Double,
    val dakshinaAmount: Double = 0.0,
    val escrowStatus: PaymentEscrowStatus,
    val scheduledEpochMillis: Long? = null,
    val completionOtp: String? = null,
    val address: String,
    val latitude: Double,
    val longitude: Double
)
