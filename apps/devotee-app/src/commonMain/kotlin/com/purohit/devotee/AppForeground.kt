package com.purohit.devotee

import kotlinx.coroutines.flow.MutableStateFlow

// Set by the platform activity (onResume/onPause). Polling screens pause while in the background
// so failures from a sleeping network are not counted against the user on return.
// ponytail: iOS never flips it (stays true), add when the iOS app polls.
object AppForeground {
    val resumed = MutableStateFlow(true)
}
