package com.purohit.priest.ui.duty

/** The location endpoint accepts updates only after duty is enabled. */
internal suspend fun goOnlineWithLocation(
    setOnline: suspend (Boolean) -> Result<Boolean>,
    saveLocation: suspend () -> Result<Unit>
): Result<Unit> {
    val enabled = setOnline(true)
    if (enabled.isFailure || enabled.getOrNull() != true) return Result.failure(enabled.exceptionOrNull() ?: IllegalStateException("Unable to go online"))
    val location = saveLocation()
    if (location.isFailure) {
        val rollback = setOnline(false)
        if (rollback.isFailure || rollback.getOrNull() != false) return Result.failure(IllegalStateException("Location failed and duty status could not be restored"))
    }
    return location
}
