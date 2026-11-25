package com.purohit.priest.ui.schedule

import com.purohit.priest.network.PriestSlotPayload
import kotlinx.datetime.LocalDate
import kotlinx.datetime.LocalDateTime
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFalse
import kotlin.test.assertNull
import kotlin.test.assertTrue

class SlotRulesTest {
    private val now = LocalDateTime(2026, 11, 23, 10, 15)
    private val today = now.date
    private val tomorrow = LocalDate(2026, 11, 24)
    private val existing = listOf(PriestSlotPayload(start_time = "09:00", end_time = "10:00"), PriestSlotPayload(start_time = "14:00:00", end_time = "15:30:00"))

    @Test fun formatsTwelveHourClock() {
        assertEquals("12:00 AM", format12h(0))
        assertEquals("9:00 AM", format12h(9 * 60))
        assertEquals("12:30 PM", format12h(12 * 60 + 30))
        assertEquals("11:59 PM", format12h(23 * 60 + 59))
        assertEquals("09:05", hhmm(9 * 60 + 5))
        assertEquals(14 * 60, minutesOf("14:00:00"))
    }

    @Test fun rejectsPastDateAndPastTimeToday() {
        assertEquals(SlotProblem.PAST_DATE, validateSlot(LocalDate(2026, 11, 22), 600, 660, now, emptyList()))
        assertEquals(SlotProblem.PAST_TIME, validateSlot(today, 10 * 60, 11 * 60, now, emptyList()))
        assertEquals(SlotProblem.PAST_TIME, validateSlot(today, 10 * 60 + 15, 11 * 60, now, emptyList()))
        assertNull(validateSlot(today, 10 * 60 + 30, 11 * 60, now, emptyList()))
        assertNull(validateSlot(tomorrow, 6 * 60, 7 * 60, now, emptyList()))
    }

    @Test fun rejectsEndNotAfterStart() {
        assertEquals(SlotProblem.END_NOT_AFTER_START, validateSlot(tomorrow, 600, 600, now, emptyList()))
        assertEquals(SlotProblem.END_NOT_AFTER_START, validateSlot(tomorrow, 600, 540, now, emptyList()))
    }

    @Test fun rejectsOverlapButAllowsAdjacent() {
        assertEquals(SlotProblem.OVERLAP, validateSlot(tomorrow, 9 * 60 + 30, 10 * 60 + 30, existing))
        assertEquals(SlotProblem.OVERLAP, validateSlot(tomorrow, 8 * 60, 16 * 60, existing))
        assertEquals(SlotProblem.OVERLAP, validateSlot(tomorrow, 15 * 60, 15 * 60 + 15, existing))
        assertNull(validateSlot(tomorrow, 10 * 60, 11 * 60, existing))
        assertNull(validateSlot(tomorrow, 8 * 60, 9 * 60, existing))
    }

    @Test fun onlyAvailableSlotsAreRemovable() {
        assertTrue(PriestSlotPayload().isRemovable())
        assertFalse(PriestSlotPayload(is_booked = true).isRemovable())
        assertFalse(PriestSlotPayload(is_held = true).isRemovable())
    }

    private fun validateSlot(date: LocalDate, s: Int, e: Int, existing: List<PriestSlotPayload>) = validateSlot(date, s, e, now, existing)
}
