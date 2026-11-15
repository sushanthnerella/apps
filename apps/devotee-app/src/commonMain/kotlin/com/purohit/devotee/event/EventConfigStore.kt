package com.purohit.devotee.event

import com.purohit.contracts.event.EventConfig
import com.purohit.devotee.network.EventConfigFetch
import com.purohit.devotee.network.PurohitApiClient
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.sync.Mutex
import kotlinx.coroutines.sync.withLock
import kotlin.time.Duration
import kotlin.time.Duration.Companion.hours
import kotlin.time.TimeMark
import kotlin.time.TimeSource

/**
 * Holds the server's live event config (PRD v2 EVT-005).
 * Refreshed on launch, on resume when older than 24 h, and immediately on an "event_config" push.
 * A failed refresh keeps the last known config; the server enforces every flag regardless.
 */
class EventConfigCache(
    private val fetch: suspend (etag: String?) -> Result<EventConfigFetch>,
    private val maxAge: Duration = 24.hours,
    private val clock: TimeSource = TimeSource.Monotonic
) {
    private val _config = MutableStateFlow<EventConfig?>(null)
    val config: StateFlow<EventConfig?> = _config.asStateFlow()
    private var etag: String? = null
    private var fetchedAt: TimeMark? = null
    private val lock = Mutex()

    suspend fun refresh(force: Boolean = false, olderThan: Duration = maxAge) = lock.withLock {
        val last = fetchedAt
        if (!force && last != null && last.elapsedNow() < olderThan) return@withLock
        fetch(etag).onSuccess { result ->
            if (!result.notModified) {
                _config.value = result.event
                etag = result.etag
            }
            fetchedAt = clock.markNow()
        }
    }
}

object EventConfigStore {
    private val cache = EventConfigCache(PurohitApiClient::fetchEventConfig)
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Default)

    val config: StateFlow<EventConfig?> get() = cache.config

    /** Called on launch and resume; skips the network when the cached copy is fresh. */
    fun refreshIfStale() {
        scope.launch { cache.refresh(force = false) }
    }

    /**
     * Used where no push arrives (iOS until APNs is set up) and while the app stays open:
     * an unchanged config costs one 304 response.
     */
    fun refreshIfOlderThan(age: Duration) {
        scope.launch { cache.refresh(force = false, olderThan = age) }
    }

    /** Called when the CMS changes the event (FCM data message). */
    fun refreshNow() {
        scope.launch { cache.refresh(force = true) }
    }
}
