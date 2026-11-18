package com.purohit.priest.location

import com.purohit.priest.network.PurohitApiClient
import kotlinx.cinterop.ExperimentalForeignApi
import kotlinx.cinterop.useContents
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.delay
import kotlinx.coroutines.isActive
import kotlinx.coroutines.launch
import kotlinx.datetime.Instant
import platform.CoreLocation.CLLocation
import platform.CoreLocation.CLLocationManager
import platform.CoreLocation.CLLocationManagerDelegateProtocol
import platform.CoreLocation.kCLDistanceFilterNone
import platform.CoreLocation.kCLLocationAccuracyNearestTenMeters
import platform.Foundation.NSDate
import platform.Foundation.NSError
import platform.Foundation.timeIntervalSince1970
import platform.darwin.NSObject

/** Posts the newest fix the system has given us. Kept separate so the posting cadence is easy to reason about. */
private class DutyLocationDelegate(private val onLocation: (CLLocation) -> Unit) : NSObject(), CLLocationManagerDelegateProtocol {
    override fun locationManager(manager: CLLocationManager, didUpdateLocations: List<*>) {
        (didUpdateLocations.lastOrNull() as? CLLocation)?.let(onLocation)
    }

    override fun locationManager(manager: CLLocationManager, didFailWithError: NSError) {
        // Transient failures (no fix yet, tunnel) are expected; CoreLocation keeps trying while updates are on.
    }
}

/**
 * iOS counterpart of the Android foreground service (PRD v2 LOC-001..005).
 * While on duty it keeps continuous location updates running, which also keeps the app alive in the background
 * (requires UIBackgroundModes = location and "Always" permission), and posts one point every 2 minutes.
 * The blue status-bar indicator stays visible so the priest always knows tracking is on.
 */
@OptIn(ExperimentalForeignApi::class)
actual object DutyServiceController {
    private const val POST_INTERVAL_SECONDS = 120.0

    private var manager: CLLocationManager? = null
    private var delegate: DutyLocationDelegate? = null // CoreLocation holds the delegate weakly; keep a strong reference.
    private var latest: CLLocation? = null
    private var lastPostedAt: Double = 0.0
    private var timer: Job? = null
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main)

    actual fun start() {
        if (manager != null) return
        val d = DutyLocationDelegate { location ->
            latest = location
            if (NSDate().timeIntervalSince1970 - lastPostedAt >= POST_INTERVAL_SECONDS) post(location)
        }
        val m = CLLocationManager()
        m.delegate = d
        m.desiredAccuracy = kCLLocationAccuracyNearestTenMeters
        m.distanceFilter = kCLDistanceFilterNone
        m.pausesLocationUpdatesAutomatically = false
        m.allowsBackgroundLocationUpdates = true
        m.showsBackgroundLocationIndicator = true
        m.requestAlwaysAuthorization()
        m.startUpdatingLocation()
        manager = m
        delegate = d
        // Backstop: a priest standing still may get few callbacks, so also post the last fix on a timer.
        timer = scope.launch {
            while (isActive) {
                delay(POST_INTERVAL_SECONDS.toLong() * 1000)
                latest?.let { if (NSDate().timeIntervalSince1970 - lastPostedAt >= POST_INTERVAL_SECONDS - 5) post(it) }
            }
        }
    }

    actual fun stop() {
        timer?.cancel()
        timer = null
        manager?.let {
            it.stopUpdatingLocation()
            it.allowsBackgroundLocationUpdates = false
            it.delegate = null
        }
        manager = null
        delegate = null
        latest = null
        lastPostedAt = 0.0
    }

    private fun post(location: CLLocation) {
        lastPostedAt = NSDate().timeIntervalSince1970
        val (lat, lng) = location.coordinate.useContents { latitude to longitude }
        val capturedAt = Instant.fromEpochMilliseconds((location.timestamp.timeIntervalSince1970 * 1000).toLong()).toString()
        val accuracy = location.horizontalAccuracy.takeIf { it >= 0 }
        scope.launch {
            // Failures (offline, 409 when duty ended elsewhere) are retried naturally on the next 2-minute tick.
            PurohitApiClient.updateLocation(lat, lng, capturedAt, accuracy)
        }
    }
}
