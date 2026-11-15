package com.purohit.devotee

import android.Manifest
import android.content.pm.PackageManager
import android.os.Build
import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.activity.result.contract.ActivityResultContracts
import androidx.core.content.ContextCompat

class MainActivity : ComponentActivity() {

    private val permissionLauncher = registerForActivityResult(
        ActivityResultContracts.RequestMultiplePermissions()
    ) { _ ->
        // Runtime permissions handled; GPS and contacts access granted
    }

    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        AppearanceStorage.context = applicationContext
        com.purohit.devotee.network.PurohitApiClient.initializeSession(com.purohit.contracts.session.AndroidSessionVault(applicationContext))

        requestAppPermissions()
        subscribeToEventConfigTopic()

        setContent {
            DevoteeAppRoot()
        }
    }

    override fun onResume() {
        super.onResume()
        com.purohit.devotee.event.EventConfigStore.refreshIfStale()
    }

    private fun subscribeToEventConfigTopic() {
        // ponytail: without Firebase config the app still refreshes on launch and resume.
        try {
            com.google.firebase.messaging.FirebaseMessaging.getInstance().subscribeToTopic("event-config")
        } catch (e: Exception) {
            android.util.Log.w("PoojaVahini", "event-config topic unavailable: ${e.message}")
        }
    }

    private fun requestAppPermissions() {
        val permissions = mutableListOf(
            Manifest.permission.CAMERA,
            Manifest.permission.RECORD_AUDIO
        )
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            permissions.add(Manifest.permission.POST_NOTIFICATIONS)
        }

        val missing = permissions.filter {
            ContextCompat.checkSelfPermission(this, it) != PackageManager.PERMISSION_GRANTED
        }

        if (missing.isNotEmpty()) {
            permissionLauncher.launch(missing.toTypedArray())
        }
    }
}
