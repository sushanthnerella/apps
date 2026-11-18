package com.purohit.contracts.resilience

import kotlinx.coroutines.delay

/**
 * Exception indicating HTTP 409 Conflict with an in-progress idempotency lock.
 */
class ConflictRetryException(
    val retryAfterSeconds: Long = 1L,
    message: String = "Operation in progress, retry after $retryAfterSeconds s",
    cause: Throwable? = null
) : Exception(message, cause)

/**
 * Executes an operation with automatic retry on 409 Conflict.
 * Retries up to [maxRetries] times (default 3), delaying for the Retry-After interval
 * while reusing the exact same idempotency key across all attempts.
 */
suspend fun <T> executeWithIdempotentRetry(
    maxRetries: Int = 3,
    defaultWaitSeconds: Long = 1L,
    block: suspend (attempt: Int) -> T
): T {
    var attempts = 0
    while (true) {
        attempts++
        try {
            return block(attempts)
        } catch (e: ConflictRetryException) {
            if (attempts > maxRetries) {
                throw e
            }
            val waitSec = e.retryAfterSeconds.coerceAtLeast(defaultWaitSeconds)
            if (waitSec > 0L) {
                delay(waitSec * 1000L)
            }
        }
    }
}

/**
 * Executes a [Result]-returning operation with automatic retry on [ConflictRetryException].
 * Retries up to [maxRetries] times (default 3), delaying for the Retry-After interval
 * while reusing the exact same idempotency key across all attempts.
 */
suspend fun <T> executeResultWithIdempotentRetry(
    maxRetries: Int = 3,
    defaultWaitSeconds: Long = 1L,
    block: suspend (attempt: Int) -> Result<T>
): Result<T> {
    var attempts = 0
    while (true) {
        attempts++
        val res = block(attempts)
        val ex = res.exceptionOrNull()
        if (ex is ConflictRetryException && attempts <= maxRetries) {
            val waitSec = ex.retryAfterSeconds.coerceAtLeast(defaultWaitSeconds)
            if (waitSec > 0L) {
                delay(waitSec * 1000L)
            }
            continue
        }
        return res
    }
}
