package com.purohit.priest.location

import android.content.Context

actual object DutyServiceController {
    private var appContext: Context? = null

    fun initialize(context: Context) {
        appContext = context.applicationContext
    }

    actual fun start() {
        appContext?.let { DutyLocationService.startService(it) }
    }

    actual fun stop() {
        appContext?.let { DutyLocationService.stopService(it) }
    }
}
