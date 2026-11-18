package com.purohit.priest

import platform.UIKit.UIAccessibilityIsReduceMotionEnabled
internal actual fun motionAllowed(): Boolean = !UIAccessibilityIsReduceMotionEnabled()
