package com.purohit.priest.location

import androidx.compose.runtime.Composable
import androidx.compose.runtime.remember
import kotlinx.cinterop.ExperimentalForeignApi
import kotlinx.cinterop.useContents
import platform.CoreLocation.*
import platform.Foundation.NSError
import platform.darwin.NSObject
import kotlinx.datetime.Instant

@OptIn(ExperimentalForeignApi::class)
private class CoreLocationDelegate(
    private val onResult: (Result<DeviceCoordinates>) -> Unit
) : NSObject(), CLLocationManagerDelegateProtocol {

    private var completed = false

    override fun locationManager(manager: CLLocationManager, didUpdateLocations: List<*>) {
        if (completed) return
        val clLocation = didUpdateLocations.lastOrNull() as? CLLocation
        if (clLocation != null) {
            completed = true
            manager.stopUpdatingLocation()
            manager.delegate = null
            val (lat, lon) = clLocation.coordinate.useContents { latitude to longitude }
            onResult(Result.success(DeviceCoordinates(lat, lon, Instant.fromEpochMilliseconds((clLocation.timestamp.timeIntervalSince1970 * 1000).toLong()).toString(), clLocation.horizontalAccuracy.takeIf { it >= 0 })))
        }
    }

    override fun locationManager(manager: CLLocationManager, didFailWithError: NSError) {
        if (completed) return
        completed = true
        manager.stopUpdatingLocation()
        manager.delegate = null
        onResult(Result.failure(Exception("Location error: ${didFailWithError.localizedDescription}")))
    }

    override fun locationManager(manager: CLLocationManager, didChangeAuthorizationStatus: CLAuthorizationStatus) {
        when (didChangeAuthorizationStatus) {
            kCLAuthorizationStatusAuthorizedWhenInUse,
            kCLAuthorizationStatusAuthorizedAlways -> {
                manager.startUpdatingLocation()
            }
            kCLAuthorizationStatusDenied,
            kCLAuthorizationStatusRestricted -> {
                if (completed) return
                completed = true
                manager.stopUpdatingLocation()
                manager.delegate = null
                onResult(Result.failure(Exception("Location permission denied. Please enable location access in iOS Settings.")))
            }
            else -> {
                // NotDetermined or unknown, awaiting user prompt
            }
        }
    }
}

@Composable
actual fun rememberLocationAccess(): LocationAccess {
    return remember {
        LocationAccess { done ->
            if (!CLLocationManager.locationServicesEnabled()) {
                done(Result.failure(Exception("Device location services are disabled. Please enable Location Services in iOS Settings.")))
                return@LocationAccess
            }

            val manager = CLLocationManager()
            val delegate = CoreLocationDelegate(done)
            manager.delegate = delegate
            manager.desiredAccuracy = kCLLocationAccuracyNearestTenMeters

            val status = manager.authorizationStatus
            when (status) {
                kCLAuthorizationStatusNotDetermined -> {
                    manager.requestWhenInUseAuthorization()
                }
                kCLAuthorizationStatusAuthorizedWhenInUse,
                kCLAuthorizationStatusAuthorizedAlways -> {
                    manager.startUpdatingLocation()
                }
                kCLAuthorizationStatusDenied,
                kCLAuthorizationStatusRestricted -> {
                    manager.delegate = null
                    done(Result.failure(Exception("Location permission denied. Please enable location access in iOS Settings.")))
                }
                else -> {
                    manager.requestWhenInUseAuthorization()
                }
            }
        }
    }
}
