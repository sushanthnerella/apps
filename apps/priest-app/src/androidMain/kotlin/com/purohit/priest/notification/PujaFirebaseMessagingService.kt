package com.purohit.priest.notification

import com.google.firebase.messaging.FirebaseMessagingService
import com.google.firebase.messaging.RemoteMessage
import com.purohit.priest.network.PurohitApiClient
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch

class PujaFirebaseMessagingService : FirebaseMessagingService() {
    override fun onNewToken(token: String) {
        super.onNewToken(token)
        if (PurohitApiClient.authToken.isNotBlank()) {
            CoroutineScope(Dispatchers.IO).launch {
                PurohitApiClient.registerDeviceToken(token, "android")
            }
        }
    }

    override fun onMessageReceived(remoteMessage: RemoteMessage) {
        super.onMessageReceived(remoteMessage)
        // Handled via system notification tray or foreground alert
    }
}
