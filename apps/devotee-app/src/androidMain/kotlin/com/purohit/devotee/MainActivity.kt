package com.purohit.devotee

import android.Manifest
import android.content.pm.PackageManager
import android.os.Build
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.result.contract.ActivityResultContracts
import androidx.core.content.ContextCompat
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch

class MainActivity : ComponentActivity() {

    private val permissionLauncher = registerForActivityResult(
        ActivityResultContracts.RequestMultiplePermissions()
    ) { _ ->
        // Runtime permissions handled; GPS and contacts access granted
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        AppearanceStorage.context = applicationContext
        com.purohit.devotee.network.PurohitApiClient.onAuthenticated = {
            com.google.firebase.messaging.FirebaseMessaging.getInstance().token.addOnSuccessListener { token ->
                CoroutineScope(Dispatchers.IO).launch {
                    com.purohit.devotee.network.PurohitApiClient.registerDeviceToken(token, "android")
                }
            }
        }
        com.purohit.devotee.network.PurohitApiClient.initializeSession(com.purohit.contracts.session.AndroidSessionVault(applicationContext))

        subscribeToEventConfigTopic()

        setContent {
            DevoteeAppRoot()
        }
    }

    override fun onResume() {
        super.onResume()
        com.purohit.devotee.event.EventConfigStore.refreshIfStale()
        // Ask for notifications only once the devotee is signed in, not on first launch.
        if (com.purohit.devotee.network.PurohitApiClient.authToken.isNotBlank()) requestNotificationPermission()
    }

    private fun subscribeToEventConfigTopic() {
        // ponytail: without Firebase config the app still refreshes on launch and resume.
        try {
            com.google.firebase.messaging.FirebaseMessaging.getInstance().subscribeToTopic("event-config")
        } catch (e: Exception) {
            android.util.Log.w("PoojaVahini", "event-config topic unavailable: ${e.message}")
        }
    }

    private var askedForNotifications = false

    private fun requestNotificationPermission() {
        if (askedForNotifications || Build.VERSION.SDK_INT < Build.VERSION_CODES.TIRAMISU) return
        if (ContextCompat.checkSelfPermission(this, Manifest.permission.POST_NOTIFICATIONS) == PackageManager.PERMISSION_GRANTED) return
        askedForNotifications = true
        permissionLauncher.launch(arrayOf(Manifest.permission.POST_NOTIFICATIONS))
    }
}
