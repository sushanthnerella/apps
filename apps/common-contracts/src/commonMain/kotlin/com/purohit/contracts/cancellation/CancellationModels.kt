package com.purohit.contracts.cancellation

import kotlinx.serialization.Serializable

@Serializable
data class CancellationQuote(
    val bookingId: String,
    val currentStatus: String,
    val advancePaid: Double,
    val cancellationFee: Double,
    val travelFee: Double,
    val priestCompensation: Double,
    val refundAmount: Double,
    val advancePaidMinor: Long = 0L,
    val cancellationFeeMinor: Long = 0L,
    val travelFeeMinor: Long = 0L,
    val priestCompensationMinor: Long = 0L,
    val refundAmountMinor: Long = 0L,
    val currency: String = "INR",
    val isFreeCancellation: Boolean,
    val policyNotice: String
)

@Serializable
data class CancelBookingRequest(
    val bookingId: String,
    val cancelledByRole: String, // 'devotee' or 'priest'
    val reasonCode: String,
    val reasonText: String = ""
)

@Serializable
data class CancelBookingResponse(
    val success: Boolean,
    val bookingId: String,
    val newStatus: String = "CANCELLED",
    val refundAmount: Double,
    val priestCompensation: Double,
    val refundAmountMinor: Long = 0L,
    val priestCompensationMinor: Long = 0L,
    val currency: String = "INR",
    val message: String
)
