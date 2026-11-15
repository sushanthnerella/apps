package com.purohit.contracts.money

/**
 * Formats monetary amounts in minor units (paise) to displayable Indian Rupees string.
 * Pure Kotlin Multiplatform implementation without JVM-specific String.format.
 */
fun formatMinorToRupees(minorUnits: Long): String {
    val rupees = minorUnits / 100
    val paise = kotlin.math.abs(minorUnits % 100)
    return if (paise == 0L) {
        "₹$rupees"
    } else {
        "₹$rupees.${paise.toString().padStart(2, '0')}"
    }
}

fun formatAmountToRupees(amountMinor: Long, amountFloat: Double = 0.0): String {
    if (amountMinor > 0) {
        return formatMinorToRupees(amountMinor)
    }
    val fallbackMinor = kotlin.math.round(amountFloat * 100).toLong()
    return formatMinorToRupees(fallbackMinor)
}
