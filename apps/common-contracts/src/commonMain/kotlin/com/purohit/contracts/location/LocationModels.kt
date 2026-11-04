package com.purohit.contracts.location

import kotlinx.serialization.Serializable

@Serializable
data class LocationPayload(
    val entityId: String, // priestId or userId
    val latitude: Double,
    val longitude: Double,
    val heading: Float? = null,
    val speed: Float? = null,
    val timestampEpochMillis: Long
)

@Serializable
data class PriestDispatchCandidate(
    val priestId: String,
    val name: String,
    val ratingAvg: Float,
    val totalRepeatBookings: Int,
    val distanceMeters: Double,
    val sampradaya: String,
    val score: Double
)
