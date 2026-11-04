package com.purohit.contracts.panchangam

import kotlinx.serialization.Serializable

@Serializable
data class MuhurthamRequest(
    val bookingId: String,
    val userId: String,
    val poojaType: String,
    val familyGotram: String,
    val headOfFamilyNakshatra: String,
    val headOfFamilyRashi: String,
    val preferredDateRangeStartEpoch: Long,
    val preferredDateRangeEndEpoch: Long,
    val notes: String? = null
)

@Serializable
data class MuhurthamProposal(
    val bookingId: String,
    val priestId: String,
    val fixedDateTimeEpoch: Long,
    val auspiciousLagna: String,
    val tithi: String,
    val nakshatra: String,
    val rahuKalamWarning: String? = null,
    val explanationNote: String
)

@Serializable
data class PanchangamSummary(
    val dateString: String,
    val sunrise: String,
    val sunset: String,
    val tithi: String,
    val nakshatra: String,
    val yoga: String,
    val karana: String,
    val rahuKalam: String,
    val yamagandam: String,
    val gulikaKalam: String,
    val isAuspiciousDay: Boolean
)
