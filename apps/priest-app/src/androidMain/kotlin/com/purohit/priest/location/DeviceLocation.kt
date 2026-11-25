package com.purohit.priest.location

import android.Manifest
import android.content.Context
import android.content.pm.PackageManager
import android.location.LocationListener
import android.location.LocationManager
import android.os.Handler
import android.os.Looper
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.runtime.*
import kotlinx.datetime.Instant
import androidx.compose.ui.platform.LocalContext
import androidx.core.content.ContextCompat

@Composable
actual fun rememberLocationAccess(): LocationAccess {
    val context = LocalContext.current
    var pending by remember { mutableStateOf<((Result<DeviceCoordinates>) -> Unit)?>(null) }
    fun locate(done: (Result<DeviceCoordinates>) -> Unit) {
        val manager = context.getSystemService(Context.LOCATION_SERVICE) as LocationManager
        val fine = ContextCompat.checkSelfPermission(context, Manifest.permission.ACCESS_FINE_LOCATION) == PackageManager.PERMISSION_GRANTED
        val coarse = ContextCompat.checkSelfPermission(context, Manifest.permission.ACCESS_COARSE_LOCATION) == PackageManager.PERMISSION_GRANTED
        if (!fine && !coarse) { done(Result.failure(IllegalStateException("Location permission is required. Enable it in app settings and try again."))); return }
        // GPS alone fails indoors; ask network (Wi-Fi/cell) in parallel and take whichever answers first.
        // ponytail: no cached-fix fallback here; priest fixes carry time/accuracy for check-in, so stay fresh.
        val providers = buildList {
            if (fine && manager.isProviderEnabled(LocationManager.GPS_PROVIDER)) add(LocationManager.GPS_PROVIDER)
            if (manager.isProviderEnabled(LocationManager.NETWORK_PROVIDER)) add(LocationManager.NETWORK_PROVIDER)
        }
        if (providers.isEmpty()) { done(Result.failure(IllegalStateException("Turn on device location and try again."))); return }
        try {
            var finished = false
            val handler = Handler(Looper.getMainLooper())
            lateinit var listener: LocationListener
            val timeout = Runnable { if (!finished) { finished = true; manager.removeUpdates(listener); done(Result.failure(IllegalStateException("Could not get a current location. Try again outdoors."))) } }
            listener = LocationListener { location ->
                if (!finished) { finished = true; handler.removeCallbacks(timeout); manager.removeUpdates(listener); if (kotlin.math.abs(System.currentTimeMillis() - location.time) > 120000L) done(Result.failure(IllegalStateException("Location is out of date. Please try again."))) else done(Result.success(DeviceCoordinates(location.latitude, location.longitude, Instant.fromEpochMilliseconds(location.time).toString(), if (location.hasAccuracy()) location.accuracy.toDouble() else null))) }
            }
            providers.forEach { manager.requestSingleUpdate(it, listener, Looper.getMainLooper()) }
            handler.postDelayed(timeout, 20000)
        } catch (e: Exception) { done(Result.failure(IllegalStateException("Could not access location. Check your settings and try again."))) }
    }
    val launcher = rememberLauncherForActivityResult(ActivityResultContracts.RequestMultiplePermissions()) { result ->
        val callback = pending
        pending = null
        if (callback != null) {
            if (result.values.any { it }) locate(callback)
            else callback(Result.failure(IllegalStateException("Location permission is required to find nearby priests.")))
        }
    }
    return remember(context, launcher) { LocationAccess { done ->
        val fine = ContextCompat.checkSelfPermission(context, Manifest.permission.ACCESS_FINE_LOCATION) == PackageManager.PERMISSION_GRANTED
        val coarse = ContextCompat.checkSelfPermission(context, Manifest.permission.ACCESS_COARSE_LOCATION) == PackageManager.PERMISSION_GRANTED
        if (fine || coarse) locate(done) else { pending = done; launcher.launch(arrayOf(Manifest.permission.ACCESS_FINE_LOCATION, Manifest.permission.ACCESS_COARSE_LOCATION)) }
    } }
}
