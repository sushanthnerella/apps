package com.purohit.priest

import android.provider.Settings
internal actual fun motionAllowed(): Boolean = Settings.Global.getFloat(AppearanceStorage.context.contentResolver, Settings.Global.ANIMATOR_DURATION_SCALE, 1f) != 0f
