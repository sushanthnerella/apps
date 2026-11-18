package com.purohit.devotee

import android.content.Context

internal object AppearanceStorage {
    lateinit var context: Context
}
internal actual fun readAppearance(): String? = AppearanceStorage.context.getSharedPreferences("appearance", Context.MODE_PRIVATE).getString("theme", null)
internal actual fun writeAppearance(value: String) { AppearanceStorage.context.getSharedPreferences("appearance", Context.MODE_PRIVATE).edit().putString("theme", value).apply() }
