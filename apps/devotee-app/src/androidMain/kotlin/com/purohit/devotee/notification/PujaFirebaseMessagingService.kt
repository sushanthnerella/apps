package com.purohit.devotee.notification

import com.google.firebase.messaging.FirebaseMessagingService
import com.google.firebase.messaging.RemoteMessage
import com.purohit.devotee.network.PurohitApiClient
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
        if (remoteMessage.data["type"] == "event_config") {
            com.purohit.devotee.event.EventConfigStore.refreshNow()
        }
        // Other messages are handled via system notification tray or foreground alert
    }
}
