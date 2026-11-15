package com.purohit.devotee

import androidx.compose.ui.window.ComposeUIViewController
import com.purohit.devotee.event.EventConfigStore
import platform.Foundation.NSNotificationCenter
import platform.Foundation.NSOperationQueue
import platform.UIKit.UIApplicationDidBecomeActiveNotification
import kotlin.time.Duration.Companion.minutes

fun MainViewController(): platform.UIKit.UIViewController {
    com.purohit.devotee.network.PurohitApiClient.initializeSession(com.purohit.devotee.session.IosKeychainSessionVault())
    // iOS has no config push yet (needs APNs), so recheck whenever the app comes to the foreground.
    NSNotificationCenter.defaultCenter.addObserverForName(UIApplicationDidBecomeActiveNotification, null, NSOperationQueue.mainQueue) { _ ->
        EventConfigStore.refreshIfOlderThan(5.minutes)
    }
    return ComposeUIViewController { DevoteeAppRoot() }
}
