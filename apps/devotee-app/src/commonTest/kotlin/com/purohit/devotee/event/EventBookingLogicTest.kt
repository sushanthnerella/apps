package com.purohit.devotee.event

import com.purohit.contracts.event.EventConfig
import com.purohit.devotee.network.eventErrorCode
import com.purohit.devotee.ui.pushkaralu.EventBookingPhase
import com.purohit.devotee.ui.pushkaralu.IST
import com.purohit.devotee.ui.pushkaralu.eventBookingPhase
import com.purohit.devotee.ui.pushkaralu.scheduleDays
import com.purohit.devotee.ui.pushkaralu.scheduleSlots
import kotlinx.datetime.Instant
import kotlinx.datetime.LocalDate
import kotlinx.datetime.toLocalDateTime
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue

class EventBookingLogicTest {
    private val event = EventConfig(
        id = "e1", slug = "s", name = "Godavari Pushkaralu", statusRaw = "ACTIVE", bookingEnabled = true,
        startsAt = "2027-07-01T00:00:00Z", endsAt = "2027-07-03T12:00:00Z", configVersion = 1
    )

    @Test
    fun slotsAreAtLeast30MinutesAheadAndInsideEvent() {
        val now = Instant.parse("2027-07-01T04:10:00Z") // 09:40 IST
        val slots = scheduleSlots(event, LocalDate(2027, 7, 1), now)
        assertEquals(11, slots.first().toLocalDateTime(IST).hour) // 10:00 is only 20 min away
        assertTrue(slots.all { it >= Instant.parse("2027-07-01T04:40:00Z") })
        val lastDay = scheduleSlots(event, LocalDate(2027, 7, 3), now)
        assertTrue(lastDay.all { it <= Instant.parse(event.endsAt) }) // ends 17:30 IST
        assertEquals(17, lastDay.last().toLocalDateTime(IST).hour)
    }

    @Test
    fun daysSkipFullyPastDates() {
        val lateEvening = Instant.parse("2027-07-01T15:00:00Z") // 20:30 IST, no slots left today
        assertEquals(listOf(LocalDate(2027, 7, 2), LocalDate(2027, 7, 3)), scheduleDays(event, lateEvening))
    }

    @Test
    fun phasesFollowServerStatusAndExpiry() {
        val exp = Instant.parse("2027-07-01T05:00:00Z")
        assertEquals(EventBookingPhase.WAITING, eventBookingPhase("REQUESTED", exp, Instant.parse("2027-07-01T04:59:00Z")))
        assertEquals(EventBookingPhase.EXPIRED, eventBookingPhase("REQUESTED", exp, Instant.parse("2027-07-01T05:01:00Z")))
        assertEquals(EventBookingPhase.CONFIRMED, eventBookingPhase("ARRIVED", exp, exp))
        assertEquals(EventBookingPhase.STARTED, eventBookingPhase("POOJA_STARTED", exp, exp))
        assertEquals(EventBookingPhase.COMPLETED, eventBookingPhase("COMPLETED", exp, exp))
        assertEquals(EventBookingPhase.CANCELLED, eventBookingPhase("CANCELLED", exp, exp))
        // The server's expiry sweep marks unanswered requests CANCELLED after the deadline.
        assertEquals(EventBookingPhase.EXPIRED, eventBookingPhase("CANCELLED", exp, Instant.parse("2027-07-01T05:01:00Z")))
    }

    @Test
    fun errorCodeIsReadFromServerBody() {
        assertEquals("priest_unavailable", eventErrorCode("""{"error":"priest_unavailable"}"""))
        assertEquals("", eventErrorCode("<html>bad gateway</html>"))
    }
}
