package com.purohit.devotee

import com.purohit.devotee.network.NearbyPriestsResponse
import kotlinx.serialization.decodeFromString
import kotlinx.serialization.json.Json
import kotlin.test.Test
import kotlin.test.assertEquals

class NearbyPriestTest {
    @Test fun nearbyPriestsMayOmitPrivatePhoneNumber() {
        val response = """{"total_priests_online":1,"priests":[{"id":"p1","full_name":"Priest","sampradaya":"Test","rating_avg":5.0,"total_completed_poojas":0,"repeat_booking_count":0,"distance_meters":0.0,"distance_km":0.0,"eta_minutes":4}]}"""
        val result = Json { ignoreUnknownKeys = true }.decodeFromString<NearbyPriestsResponse>(response)
        assertEquals("", result.priests.single().phone)
    }
}
