package com.purohit.contracts.i18n

import kotlinx.serialization.Serializable

@Serializable
enum class AppLanguage(
    val isoCode: String,
    val nativeName: String,
    val englishName: String
) {
    ENGLISH("en", "English", "English"),
    TELUGU("te", "తెలుగు", "Telugu"),
    HINDI("hi", "हिन्दी", "Hindi");

    companion object {
        fun fromIsoCode(code: String): AppLanguage {
            return entries.firstOrNull { it.isoCode.equals(code, ignoreCase = true) } ?: ENGLISH
        }
    }
}
