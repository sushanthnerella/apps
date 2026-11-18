package com.purohit.priest.location
import androidx.compose.runtime.Composable

data class DeviceCoordinates(val latitude: Double, val longitude: Double, val capturedAt: String, val accuracyM: Double?)
class LocationAccess(val request: ((Result<DeviceCoordinates>) -> Unit) -> Unit)
@Composable expect fun rememberLocationAccess(): LocationAccess
