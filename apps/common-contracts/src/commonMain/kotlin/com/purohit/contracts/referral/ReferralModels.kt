package com.purohit.contracts.referral

import kotlinx.serialization.Serializable

@Serializable
data class PriestReferralRequest(
    val bookingId: String,
    val referringPriestId: String,
    val referredPriestId: String,
    val reason: String = "Sudden scheduling conflict"
)

@Serializable
data class PriestReferralResponse(
    val success: Boolean,
    val bookingId: String,
    val newPriestId: String,
    val newPriestName: String,
    val message: String
)
