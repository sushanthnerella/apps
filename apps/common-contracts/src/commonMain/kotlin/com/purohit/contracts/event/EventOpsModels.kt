package com.purohit.contracts.event

import com.purohit.contracts.i18n.AppLanguage
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

@Serializable
data class EventRitual(
    val id: String,
    val name: String,
    @SerialName("name_te") val nameTe: String = "",
    @SerialName("name_hi") val nameHi: String = "",
    @SerialName("duration_minutes") val durationMinutes: Int = 0
) {
    fun displayName(language: AppLanguage) = pick(language, name, nameTe, nameHi)
}

@Serializable
data class PriestService(
    val id: String,
    @SerialName("ritual_id") val ritualId: String,
    @SerialName("ritual_name") val ritualName: String,
    @SerialName("ritual_name_te") val ritualNameTe: String = "",
    @SerialName("ritual_name_hi") val ritualNameHi: String = "",
    @SerialName("duration_minutes") val durationMinutes: Int = 0,
    @SerialName("price_minor") val priceMinor: Long
) {
    fun displayName(language: AppLanguage) = pick(language, ritualName, ritualNameTe, ritualNameHi)
}

/** Location freshness buckets from PRD v2 LOC-004. Stale priests are never returned by the server. */
enum class Freshness { FRESH, DELAYED }

@Serializable
data class GhatPriest(
    @SerialName("priest_id") val priestId: String,
    @SerialName("full_name") val fullName: String,
    @SerialName("rating_avg") val ratingAvg: Double = 0.0,
    @SerialName("completed_poojas") val completedPoojas: Int = 0,
    @SerialName("duty_status") val dutyStatus: String = "",
    @SerialName("location_age_seconds") val locationAgeSeconds: Long = 0,
    val services: List<PriestService> = emptyList()
) {
    val freshness: Freshness get() = if (locationAgeSeconds <= 300) Freshness.FRESH else Freshness.DELAYED
    val locationAgeMinutes: Long get() = (locationAgeSeconds / 60).coerceAtLeast(0)
    val lowestPriceMinor: Long? get() = services.minOfOrNull { it.priceMinor }
}

@Serializable
data class GhatAvailability(
    @SerialName("ghat_id") val ghatId: String,
    @SerialName("available_count") val availableCount: Int
)

@Serializable
data class CreateEventBookingRequest(
    @SerialName("priest_id") val priestId: String,
    @SerialName("ghat_id") val ghatId: String,
    @SerialName("service_id") val serviceId: String,
    val `when`: String,
    @SerialName("scheduled_at") val scheduledAt: String? = null,
    @SerialName("family_size") val familySize: Int = 1
)

@Serializable
data class EventBookingResponse(
    @SerialName("booking_id") val bookingId: String,
    val status: String,
    @SerialName("booking_code") val bookingCode: String,
    @SerialName("total_amount_minor") val totalAmountMinor: Long,
    @SerialName("request_expires_at") val requestExpiresAt: String,
    @SerialName("chat_room_id") val chatRoomId: String = ""
)

@Serializable
data class EventCard(
    @SerialName("card_code") val cardCode: String,
    val issuer: String,
    val status: String,
    @SerialName("event_name") val eventName: String = "",
    @SerialName("starts_at") val startsAt: String = "",
    @SerialName("ends_at") val endsAt: String = ""
)

@Serializable
data class DutyState(
    @SerialName("ghat_id") val ghatId: String,
    @SerialName("duty_status") val dutyStatus: String
)

@Serializable
data class Participation(
    @SerialName("event_id") val eventId: String,
    val status: String,
    @SerialName("reviewer_reason") val reviewerReason: String = "",
    val card: EventCard? = null,
    val rituals: List<EventRitual> = emptyList(),
    val services: List<PriestService> = emptyList(),
    val duty: DutyState? = null
) {
    val isApproved: Boolean get() = status == "APPROVED"
    val canApply: Boolean get() = status == "NOT_APPLIED" || status == "CHANGES_REQUESTED"
}

@Serializable
data class SetServicesRequest(val services: List<ServicePrice>) {
    @Serializable
    data class ServicePrice(@SerialName("ritual_id") val ritualId: String, @SerialName("price_minor") val priceMinor: Long)
}

@Serializable
data class DutyRequest(@SerialName("ghat_id") val ghatId: String, @SerialName("duty_status") val dutyStatus: String)

/** Server error codes the apps turn into user messages. */
object EventErrorCodes {
    const val BOOKING_DISABLED = "event_booking_disabled"
    const val PRIEST_UNAVAILABLE = "priest_unavailable"
    const val SERVICE_UNAVAILABLE = "service_unavailable"
    const val INVALID_SCHEDULE = "invalid_schedule"
    const val REQUEST_EXPIRED = "request_expired"
    const val PRIEST_BUSY = "priest_busy"
    const val BOOKING_CODE_REQUIRED = "booking_code_required"
    const val NOT_APPROVED = "not_approved"
}

private fun pick(language: AppLanguage, en: String, te: String, hi: String) = when (language) {
    AppLanguage.TELUGU -> te.ifBlank { en }
    AppLanguage.HINDI -> hi.ifBlank { en }
    else -> en
}
