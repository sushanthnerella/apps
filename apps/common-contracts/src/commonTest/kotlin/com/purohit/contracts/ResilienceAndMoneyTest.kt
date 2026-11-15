package com.purohit.contracts

import com.purohit.contracts.idempotency.IdempotencyKeyGenerator
import com.purohit.contracts.money.formatAmountToRupees
import com.purohit.contracts.money.formatMinorToRupees
import com.purohit.contracts.resilience.ConflictRetryException
import com.purohit.contracts.resilience.JitteredBackoff
import com.purohit.contracts.resilience.executeResultWithIdempotentRetry
import kotlinx.coroutines.runBlocking
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNotEquals
import kotlin.test.assertTrue

class ResilienceAndMoneyTest {

    @Test
    fun testJitteredBackoffProgressionAndReset() {
        val backoff = JitteredBackoff(baseDelayMs = 5000L, maxDelayMs = 30000L, maxJitterMs = 1000L)
        assertEquals(5000L, backoff.currentDelayMs)

        // Error 1: 5s -> 10s
        backoff.recordError()
        assertEquals(10000L, backoff.currentDelayMs)

        // Error 2: 10s -> 20s
        backoff.recordError()
        assertEquals(20000L, backoff.currentDelayMs)

        // Error 3: 20s -> 30s (capped at max 30s)
        backoff.recordError()
        assertEquals(30000L, backoff.currentDelayMs)

        // Error 4: remains capped at 30s
        backoff.recordError()
        assertEquals(30000L, backoff.currentDelayMs)

        // Success resets back to 5s
        backoff.recordSuccess()
        assertEquals(5000L, backoff.currentDelayMs)

        // Jitter check
        val nextDelay = backoff.computeNextDelay()
        assertTrue(nextDelay in 5000L..6000L, "nextDelay with jitter should be between 5000 and 6000")
    }

    @Test
    fun testExactPaiseFormattingToRupees() {
        assertEquals("₹3716", formatMinorToRupees(371600L))
        assertEquals("₹150", formatMinorToRupees(15000L))
        assertEquals("₹123.45", formatMinorToRupees(12345L))
        assertEquals("₹0.50", formatMinorToRupees(50L))
        assertEquals("₹0.05", formatMinorToRupees(5L))
        assertEquals("₹0", formatMinorToRupees(0L))

        assertEquals("₹3716", formatAmountToRupees(371600L, 3716.0))
        assertEquals("₹150", formatAmountToRupees(0L, 150.0))
    }

    @Test
    fun testIdempotencyKeyV4FormatAndUniqueness() {
        val key1 = IdempotencyKeyGenerator.generate()
        val key2 = IdempotencyKeyGenerator.generate()

        assertEquals(36, key1.length)
        assertEquals(36, key2.length)
        assertNotEquals(key1, key2)

        val uuidRegex = Regex("^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$")
        assertTrue(uuidRegex.matches(key1), "key1 should be valid UUID v4: $key1")
        assertTrue(uuidRegex.matches(key2), "key2 should be valid UUID v4: $key2")
    }

    @Test
    fun testIdempotentRetryHelperReusesKeyAcrossAttempts() = runBlocking {
        val key = IdempotencyKeyGenerator.generate()
        val capturedKeys = mutableListOf<String>()
        var attempts = 0

        val result = executeResultWithIdempotentRetry<String>(maxRetries = 3, defaultWaitSeconds = 0L) { attempt ->
            attempts++
            capturedKeys.add(key)
            if (attempt < 3) {
                Result.failure(ConflictRetryException(retryAfterSeconds = 0L))
            } else {
                Result.success("SuccessOnAttempt3")
            }
        }

        assertTrue(result.isSuccess)
        assertEquals("SuccessOnAttempt3", result.getOrNull())
        assertEquals(3, attempts)
        assertEquals(listOf(key, key, key), capturedKeys)
    }
}
