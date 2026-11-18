package com.purohit.priest.ui.duty

import kotlinx.coroutines.runBlocking
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertTrue

class DutyOperationTest {
    @Test fun goesOnlineBeforeSavingLocation() = runBlocking {
        var online = false
        val events = mutableListOf<String>()
        val result = goOnlineWithLocation(
            setOnline = { value -> events.add("online=$value"); online = value; Result.success(value) },
            saveLocation = { events.add("location"); if (online) Result.success(Unit) else Result.failure(IllegalStateException("off duty")) }
        )
        assertTrue(result.isSuccess)
        assertEquals(listOf("online=true", "location"), events)
    }
    @Test fun failedLocationRestoresOfflineState() = runBlocking {
        val events = mutableListOf<String>()
        val result = goOnlineWithLocation(
            setOnline = { value -> events.add("online=$value"); Result.success(value) },
            saveLocation = { events.add("location"); Result.failure(IllegalStateException("unavailable")) }
        )
        assertTrue(result.isFailure)
        assertEquals(listOf("online=true", "location", "online=false"), events)
    }
}
