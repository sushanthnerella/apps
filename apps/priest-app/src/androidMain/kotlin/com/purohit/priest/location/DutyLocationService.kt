package com.purohit.priest.location

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.Service
import android.content.Context
import android.content.Intent
import android.content.pm.ServiceInfo
import android.location.Location
import android.location.LocationListener
import android.location.LocationManager
import android.os.Build
import android.os.Bundle
import android.os.IBinder
import com.purohit.priest.network.PurohitApiClient
import kotlinx.coroutines.*
import java.time.Instant
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale
import java.util.TimeZone

class DutyLocationService : Service() {

    private val serviceScope = CoroutineScope(Dispatchers.IO + SupervisorJob())
    private var locationManager: LocationManager? = null
    private var locationListener: LocationListener? = null
    @Volatile private var lastLocation: Location? = null
    private var updateJob: Job? = null

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onCreate() {
        super.onCreate()
        createNotificationChannel()
        val notification = buildForegroundNotification()
        try {
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
                startForeground(NOTIFICATION_ID, notification, ServiceInfo.FOREGROUND_SERVICE_TYPE_LOCATION)
            } else {
                startForeground(NOTIFICATION_ID, notification)
            }
        } catch (e: Exception) {
            // Android refuses a location foreground service started from the background
            // (e.g. a system restart after the app was killed). Stop quietly; the app
            // restarts the service next time it is opened while on duty.
            stopSelf()
            return
        }
        startLocationTracking()
    }

    // Never let the system restart the service from the background: that restart is
    // what Android blocks for location services and it crashed the app in a loop.
    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int = START_NOT_STICKY

    private fun createNotificationChannel() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val channel = NotificationChannel(
                CHANNEL_ID,
                "Duty Location Tracking",
                NotificationManager.IMPORTANCE_LOW
            ).apply {
                description = "Shows persistent status while Ayagaryllu / Purohit is on active duty"
            }
            val manager = getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
            manager.createNotificationChannel(channel)
        }
    }

    private fun buildForegroundNotification(): Notification {
        val builder = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            Notification.Builder(this, CHANNEL_ID)
        } else {
            @Suppress("DEPRECATION")
            Notification.Builder(this)
        }

        return builder
            .setContentTitle("On duty – sharing location")
            .setContentText("Puja Vaahini dispatch active. Transmitting periodic GPS coordinates.")
            .setSmallIcon(android.R.drawable.ic_menu_mylocation)
            .setOngoing(true)
            .build()
    }

    private fun startLocationTracking() {
        locationManager = getSystemService(Context.LOCATION_SERVICE) as? LocationManager ?: return

        locationListener = object : LocationListener {
            override fun onLocationChanged(location: Location) {
                lastLocation = location
            }
            @Deprecated("Deprecated in Java")
            override fun onStatusChanged(provider: String?, status: Int, extras: Bundle?) {}
            override fun onProviderEnabled(provider: String) {}
            override fun onProviderDisabled(provider: String) {}
        }

        try {
            // Request both providers, including updates while stationary. Discovery expires
            // after 15 minutes even if the priest has not moved.
            for (provider in listOf(LocationManager.GPS_PROVIDER, LocationManager.NETWORK_PROVIDER)) {
                if (locationManager?.isProviderEnabled(provider) == true) {
                    try {
                        locationManager?.requestLocationUpdates(provider, 60_000L, 0f, locationListener!!)
                    } catch (_: SecurityException) {
                        // Coarse-only permission may allow network but not GPS.
                    }
                }
            }
        } catch (_: SecurityException) {
            // Permission verified prior to service start
        }

        // Periodic upload every 120s
        updateJob = serviceScope.launch {
            while (isActive) {
                val loc = lastLocation ?: try {
                    locationManager?.getLastKnownLocation(LocationManager.GPS_PROVIDER)
                        ?: locationManager?.getLastKnownLocation(LocationManager.NETWORK_PROVIDER)
                } catch (_: SecurityException) {
                    null
                }

                if (loc != null && System.currentTimeMillis() - loc.time in 0..300_000L) {
                    val capturedAt = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                        Instant.ofEpochMilli(loc.time).toString()
                    } else {
                        SimpleDateFormat("yyyy-MM-dd'T'HH:mm:ss.SSS'Z'", Locale.US).apply {
                            timeZone = TimeZone.getTimeZone("UTC")
                        }.format(Date(loc.time))
                    }
                    val accuracyM = loc.accuracy.toDouble()

                    val result = PurohitApiClient.updateLocation(
                        latitude = loc.latitude,
                        longitude = loc.longitude,
                        capturedAt = capturedAt,
                        accuracyM = accuracyM
                    )
                    updateLocationNotification(result.isSuccess)
                } else {
                    updateLocationNotification(false)
                }

                delay(120_000L) // 120 seconds
            }
        }
    }

    private fun updateLocationNotification(sharing: Boolean) {
        val notification = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            Notification.Builder(this, CHANNEL_ID)
        } else {
            @Suppress("DEPRECATION")
            Notification.Builder(this)
        }.setContentTitle(if (sharing) "On duty – sharing location" else "Location needs updating")
            .setContentText(if (sharing) "Your location is being shared for bookings." else "Open the app and update your location so devotees can find you.")
            .setSmallIcon(android.R.drawable.ic_menu_mylocation)
            .setOngoing(true)
            .build()
        (getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager).notify(NOTIFICATION_ID, notification)
    }

    override fun onDestroy() {
        super.onDestroy()
        updateJob?.cancel()
        serviceScope.cancel()
        locationListener?.let {
            locationManager?.removeUpdates(it)
        }
    }

    companion object {
        const val CHANNEL_ID = "duty_location_channel"
        const val NOTIFICATION_ID = 1001

        fun startService(context: Context) {
            val intent = Intent(context, DutyLocationService::class.java)
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                context.startForegroundService(intent)
            } else {
                context.startService(intent)
            }
        }

        fun stopService(context: Context) {
            val intent = Intent(context, DutyLocationService::class.java)
            context.stopService(intent)
        }
    }
}
