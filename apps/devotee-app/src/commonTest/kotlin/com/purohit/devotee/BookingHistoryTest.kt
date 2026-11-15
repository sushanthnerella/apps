package com.purohit.devotee

import com.purohit.devotee.network.BookingDetailPayload
import com.purohit.devotee.ui.history.isActiveBooking
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.json.Json
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertTrue

class BookingHistoryTest {
    @Test fun historyDecodesTerminalAndScheduledBookings() {
        val rows = Json { ignoreUnknownKeys = true }.decodeFromString<List<BookingDetailPayload>>(
            """[{"id":"b1","status":"COMPLETED","pooja_title":"Puja","scheduled_date":"2026-11-15","slot_time":"17:30 - 19:30"},{"id":"b2","status":"CANCELLED"},{"id":"b3","status":"REQUESTED"}]"""
        )
        assertEquals(listOf("COMPLETED", "CANCELLED", "REQUESTED"), rows.map { it.status })
        assertEquals("2026-11-15", rows[0].scheduled_date)
        assertEquals("17:30 - 19:30", rows[0].slot_time)
        assertFalse(isActiveBooking(rows[0].status))
        assertFalse(isActiveBooking(rows[1].status))
        assertTrue(isActiveBooking(rows[2].status))
    }
}
