package com.purohit.devotee

import platform.UIKit.UIAccessibilityIsReduceMotionEnabled
internal actual fun motionAllowed(): Boolean = !UIAccessibilityIsReduceMotionEnabled()
