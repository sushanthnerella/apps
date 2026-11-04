package com.purohit.contracts.slots

import kotlinx.serialization.Serializable

@Serializable
data class PriestSlot(
    val id: String,
    val priestId: String,
    val slotDate: String, // YYYY-MM-DD
    val startTime: String, // HH:MM
    val endTime: String, // HH:MM
    val isBooked: Boolean = false,
    val isHeld: Boolean = false
)

@Serializable
data class CreateSlotRequest(
    val priestId: String,
    val slotDate: String,
    val startTime: String,
    val endTime: String
)

@Serializable
data class HoldSlotRequest(
    val slotId: String,
    val devoteeId: String,
    val holdDurationSeconds: Int = 600 // 10 minutes default
)

@Serializable
data class HoldSlotResponse(
    val success: Boolean,
    val slotId: String,
    val expiresAtTimestamp: Long,
    val message: String
)
