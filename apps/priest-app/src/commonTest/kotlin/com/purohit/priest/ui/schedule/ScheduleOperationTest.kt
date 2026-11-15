package com.purohit.priest.ui.schedule

import kotlinx.coroutines.CompletableDeferred
import kotlinx.coroutines.runBlocking
import kotlinx.coroutines.yield
import kotlin.test.Test
import kotlin.test.assertFalse
import kotlin.test.assertTrue

class ScheduleOperationTest {
    @Test fun remainsBusyUntilPublishedAvailabilityHasRefreshed() = runBlocking {
        val refreshGate = CompletableDeferred<Unit>()
        val refreshStarted = CompletableDeferred<Unit>()
        var loading = false
        val operation = publishThenRefresh(
            publish = { Result.success(Unit) },
            refresh = { refreshStarted.complete(Unit); refreshGate.await() },
            setLoading = { loading = it },
            onFailure = { error("Unexpected failure") }
        )
        refreshStarted.await()
        try { assertTrue(loading, "The controls must remain disabled while refresh is in progress") }
        finally { refreshGate.complete(Unit); operation.join() }
        yield()
        assertFalse(loading)
    }
    @Test fun failedPublishReleasesControlsWithoutRefreshing() = runBlocking {
        var loading = false
        var refreshed = false
        var failureShown = false
        val operation = publishThenRefresh(
            publish = { Result.failure(IllegalStateException("unavailable")) },
            refresh = { refreshed = true },
            setLoading = { loading = it },
            onFailure = { failureShown = true }
        )
        operation.join()
        assertFalse(loading)
        assertFalse(refreshed)
        assertTrue(failureShown)
    }
}
