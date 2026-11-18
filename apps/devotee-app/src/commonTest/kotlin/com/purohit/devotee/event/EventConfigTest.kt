package com.purohit.devotee.event

import com.purohit.contracts.event.EventConfig
import com.purohit.contracts.event.Ghat
import com.purohit.devotee.location.DeviceCoordinates
import com.purohit.devotee.network.EventConfigFetch
import com.purohit.devotee.ui.pushkaralu.eventDay
import com.purohit.devotee.ui.pushkaralu.visibleGhats
import kotlinx.datetime.Instant
import kotlin.coroutines.Continuation
import kotlin.coroutines.EmptyCoroutineContext
import kotlin.coroutines.startCoroutine
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNull
import kotlin.time.Duration.Companion.hours
import kotlin.time.TestTimeSource

// ponytail: the fake fetch never suspends, so the coroutine completes inline; no coroutines-test dependency needed.
private fun runSync(block: suspend () -> Unit) {
    var outcome: Result<Unit>? = null
    block.startCoroutine(Continuation(EmptyCoroutineContext) { outcome = it })
    (outcome ?: error("coroutine suspended unexpectedly")).getOrThrow()
}

class EventConfigTest {
    private fun event(version: Long, booking: Boolean = true) = EventConfig(
        id = "e1", slug = "godavari-2027", name = "Godavari Pushkaralu", statusRaw = "ACTIVE", bookingEnabled = booking,
        startsAt = "2027-07-01T00:00:00Z", endsAt = "2027-07-12T23:59:00Z", configVersion = version
    )

    @Test
    fun cacheSkipsNetworkWhenFreshAndKeepsConfigOnFailureOr304() = runSync {
        val clock = TestTimeSource()
        val responses = ArrayDeque(listOf(
            Result.success(EventConfigFetch(event(1), "\"e1-1\"", notModified = false)),
            Result.success(EventConfigFetch(null, "\"e1-1\"", notModified = true)),
            Result.failure(Exception("offline")),
            Result.success(EventConfigFetch(event(2, booking = false), "\"e1-2\"", notModified = false))
        ))
        val sentEtags = mutableListOf<String?>()
        val cache = EventConfigCache({ etag -> sentEtags += etag; responses.removeFirst() }, maxAge = 24.hours, clock = clock)

        cache.refresh()
        assertEquals(1, cache.config.value?.configVersion)
        cache.refresh() // fresh: no network call
        assertEquals(1, sentEtags.size)

        clock += 25.hours
        cache.refresh() // 304 keeps cached config
        assertEquals(1, cache.config.value?.configVersion)
        assertEquals("\"e1-1\"", sentEtags.last())

        cache.refresh(force = true) // failure keeps last known config
        assertEquals(1, cache.config.value?.configVersion)

        cache.refresh(force = true) // CMS toggle arrives via push
        assertEquals(false, cache.config.value?.acceptsBookings)
    }

    @Test
    fun dayIsClampedIntoEventWindow() {
        val e = event(1)
        assertEquals(1 to 12, eventDay(e, Instant.parse("2027-06-20T00:00:00Z")))
        assertEquals(4 to 12, eventDay(e, Instant.parse("2027-07-04T10:00:00Z")))
        assertEquals(12 to 12, eventDay(e, Instant.parse("2027-08-01T00:00:00Z")))
        assertNull(eventDay(e.copy(startsAt = "not a date")))
    }

    @Test
    fun ghatsSearchAnyLanguageAndSortByDistanceWhenKnown() {
        val ghats = listOf(
            Ghat(id = "a", name = "Pushkar Ghat", nameTe = "పుష్కర ఘాట్", latitude = 17.00, longitude = 81.80, sortOrder = 1),
            Ghat(id = "b", name = "Kotilingala Ghat", latitude = 17.02, longitude = 81.77, sortOrder = 2),
            Ghat(id = "c", name = "Closed Ghat", sortOrder = 0, isActive = false),
            Ghat(id = "d", name = "No Coordinates Ghat", sortOrder = 3)
        )
        assertEquals(listOf("a", "b", "d"), visibleGhats(ghats, "", null).map { it.first.id })
        assertEquals(listOf("a"), visibleGhats(ghats, "పుష్కర", null).map { it.first.id })
        val nearKotilingala = DeviceCoordinates(17.021, 81.771)
        assertEquals(listOf("b", "a", "d"), visibleGhats(ghats, "", nearKotilingala).map { it.first.id })
    }
}
