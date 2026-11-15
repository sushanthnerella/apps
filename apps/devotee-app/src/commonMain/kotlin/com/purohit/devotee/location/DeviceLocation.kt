package com.purohit.devotee.location
import androidx.compose.runtime.Composable

data class DeviceCoordinates(val latitude: Double, val longitude: Double)
class LocationAccess(val request: ((Result<DeviceCoordinates>) -> Unit) -> Unit)
@Composable expect fun rememberLocationAccess(): LocationAccess
