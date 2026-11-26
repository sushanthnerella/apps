package com.purohit.priest

import platform.Foundation.NSUserDefaults

internal actual fun readAppearance(): String? = NSUserDefaults.standardUserDefaults.stringForKey("appearance_theme")
internal actual fun writeAppearance(value: String) { NSUserDefaults.standardUserDefaults.setObject(value, forKey = "appearance_theme") }
internal actual fun readPref(key: String): String? = NSUserDefaults.standardUserDefaults.stringForKey(key)
internal actual fun writePref(key: String, value: String) { NSUserDefaults.standardUserDefaults.setObject(value, forKey = key) }
