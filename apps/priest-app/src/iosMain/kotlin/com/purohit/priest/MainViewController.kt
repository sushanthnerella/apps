package com.purohit.priest
import androidx.compose.ui.window.ComposeUIViewController
fun MainViewController(): platform.UIKit.UIViewController {
    com.purohit.priest.network.PurohitApiClient.initializeSession(com.purohit.priest.session.IosKeychainSessionVault())
    return ComposeUIViewController { PriestAppRoot() }
}
