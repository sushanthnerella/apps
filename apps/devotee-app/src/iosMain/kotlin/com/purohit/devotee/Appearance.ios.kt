package com.purohit.devotee

import platform.Foundation.NSUserDefaults

internal actual fun readAppearance(): String? = NSUserDefaults.standardUserDefaults.stringForKey("appearance_theme")
internal actual fun writeAppearance(value: String) { NSUserDefaults.standardUserDefaults.setObject(value, forKey = "appearance_theme") }
