package com.purohit.contracts.resilience

import kotlin.random.Random
import kotlinx.coroutines.delay

/**
 * JitteredBackoff provides resilient polling with exponential backoff on error and random jitter
 * to prevent thundering herd and battery drain.
 *
 * @param baseDelayMs Minimum delay between successful polls (default 5,000ms = 5s)
 * @param maxDelayMs Maximum ceiling for backoff delay (default 30,000ms = 30s)
 * @param maxJitterMs Maximum random jitter added to delay (default 1,000ms = 1s)
 */
class JitteredBackoff(
    val baseDelayMs: Long = 5000L,
    val maxDelayMs: Long = 30000L,
    val maxJitterMs: Long = 1000L
) {
    var currentDelayMs: Long = baseDelayMs
        private set

    /**
     * Resets delay back to base value after a successful fetch/poll.
     */
    fun recordSuccess() {
        currentDelayMs = baseDelayMs
    }

    /**
     * Doubles delay up to max ceiling upon error or transient network failure.
     */
    fun recordError() {
        currentDelayMs = (currentDelayMs * 2).coerceAtMost(maxDelayMs)
    }

    /**
     * Computes the next delay in milliseconds with random jitter.
     */
    fun computeNextDelay(): Long {
        val jitter = if (maxJitterMs > 0) Random.nextLong(maxJitterMs) else 0L
        return currentDelayMs + jitter
    }

    /**
     * Suspends for the computed delay with jitter.
     */
    suspend fun awaitNext() {
        delay(computeNextDelay())
    }
}
