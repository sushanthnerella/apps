package com.purohit.contracts.event

import com.purohit.contracts.i18n.AppLanguage
import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

/** Event lifecycle from PRD v2 EVT-002. Unknown future values map to [UNKNOWN] so old apps never crash. */
enum class EventStatus { DRAFT, SCHEDULED, ACTIVE, PAUSED, ENDED, ARCHIVED, UNKNOWN }

@Serializable
data class Ghat(
    val id: String,
    @SerialName("event_id") val eventId: String = "",
    val name: String,
    @SerialName("name_te") val nameTe: String = "",
    @SerialName("name_hi") val nameHi: String = "",
    val latitude: Double? = null,
    val longitude: Double? = null,
    @SerialName("sort_order") val sortOrder: Int = 0,
    @SerialName("is_active") val isActive: Boolean = true
) {
    fun displayName(language: AppLanguage): String = localized(language, name, nameTe, nameHi)
}

@Serializable
data class EventConfig(
    val id: String,
    val slug: String,
    val name: String,
    @SerialName("name_te") val nameTe: String = "",
    @SerialName("name_hi") val nameHi: String = "",
    val river: String = "",
    @SerialName("status") val statusRaw: String = "",
    @SerialName("booking_enabled") val bookingEnabled: Boolean = false,
    @SerialName("card_issuance_enabled") val cardIssuanceEnabled: Boolean = false,
    @SerialName("govt_cards_enabled") val govtCardsEnabled: Boolean = false,
    @SerialName("starts_at") val startsAt: String,
    @SerialName("ends_at") val endsAt: String,
    @SerialName("config_version") val configVersion: Long,
    val ghats: List<Ghat> = emptyList()
) {
    val status: EventStatus get() = EventStatus.entries.firstOrNull { it.name == statusRaw } ?: EventStatus.UNKNOWN

    /** The app shows event mode only while the server says the event is live. */
    val isLive: Boolean get() = status == EventStatus.ACTIVE || status == EventStatus.PAUSED

    /** New requests are allowed only when ACTIVE and the CMS booking switch is on. The server enforces this too. */
    val acceptsBookings: Boolean get() = status == EventStatus.ACTIVE && bookingEnabled

    fun displayName(language: AppLanguage): String = localized(language, name, nameTe, nameHi)
}

/** Wire shape of GET /api/v1/events/config. A null event means no event mode. */
@Serializable
data class EventConfigResponse(val event: EventConfig? = null)

private fun localized(language: AppLanguage, en: String, te: String, hi: String): String = when (language) {
    AppLanguage.TELUGU -> te.ifBlank { en }
    AppLanguage.HINDI -> hi.ifBlank { en }
    else -> en
}
