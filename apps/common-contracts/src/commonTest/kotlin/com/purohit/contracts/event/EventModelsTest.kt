package com.purohit.contracts.event

import com.purohit.contracts.i18n.AppLanguage
import com.purohit.contracts.i18n.LocalizationManager
import kotlinx.serialization.json.Json
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertNull
import kotlin.test.assertTrue

class EventModelsTest {
    private val json = Json { ignoreUnknownKeys = true }

    private fun config(status: String, booking: Boolean) = json.decodeFromString<EventConfigResponse>(
        """{"event":{"id":"e1","slug":"godavari-2027","name":"Godavari Pushkaralu","name_te":"గోదావరి పుష్కరాలు","name_hi":"",
        "status":"$status","booking_enabled":$booking,"card_issuance_enabled":false,"govt_cards_enabled":false,
        "starts_at":"2027-07-01T00:00:00Z","ends_at":"2027-07-12T23:59:00Z","config_version":4,"updated_at":"x",
        "ghats":[{"id":"g1","event_id":"e1","name":"Pushkar Ghat","name_te":"పుష్కర ఘాట్","name_hi":"","latitude":17.0,"longitude":81.8,"sort_order":1,"is_active":true}]}}"""
    ).event!!

    @Test
    fun nullEventMeansNoEventMode() {
        assertNull(json.decodeFromString<EventConfigResponse>("""{"event":null}""").event)
    }

    @Test
    fun bookingsNeedActiveAndSwitchOn() {
        assertTrue(config("ACTIVE", true).acceptsBookings)
        assertFalse(config("ACTIVE", false).acceptsBookings)
        assertFalse(config("PAUSED", true).acceptsBookings)
        assertTrue(config("PAUSED", true).isLive)
    }

    @Test
    fun unknownStatusFromNewerServerDoesNotCrash() {
        val e = config("SOMETHING_NEW", true)
        assertEquals(EventStatus.UNKNOWN, e.status)
        assertFalse(e.isLive)
        assertFalse(e.acceptsBookings)
    }

    @Test
    fun localizedNamesFallBackToEnglish() {
        val e = config("ACTIVE", true)
        assertEquals("గోదావరి పుష్కరాలు", e.displayName(AppLanguage.TELUGU))
        assertEquals("Godavari Pushkaralu", e.displayName(AppLanguage.HINDI))
        assertEquals("పుష్కర ఘాట్", e.ghats.single().displayName(AppLanguage.TELUGU))
    }

    @Test
    fun eventStringsArePresentInEveryLanguage() {
        AppLanguage.entries.forEach { lang ->
            val s = LocalizationManager.getStrings(lang).event
            listOf(s.tabEvent, s.liveDayTemplate, s.bannerDescription, s.bookPriestAtGhat, s.bookingsPaused,
                s.chooseGhatTitle, s.noGhatsConfigured, s.ghatBookingOpensSoon).forEach { assertTrue(it.isNotBlank(), "$lang") }
            assertTrue(s.liveDayTemplate.contains("{day}") && s.liveDayTemplate.contains("{total}"), "$lang template")
        }
    }
}
