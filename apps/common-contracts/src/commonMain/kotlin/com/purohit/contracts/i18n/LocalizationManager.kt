package com.purohit.contracts.i18n

import com.purohit.contracts.i18n.locales.en.EnStrings
import com.purohit.contracts.i18n.locales.hi.HiStrings
import com.purohit.contracts.i18n.locales.te.TeStrings

object LocalizationManager {
    private val locales: Map<AppLanguage, StringContract> = mapOf(
        AppLanguage.ENGLISH to EnStrings,
        AppLanguage.TELUGU to TeStrings,
        AppLanguage.HINDI to HiStrings
    )

    fun getStrings(language: AppLanguage): StringContract {
        return locales[language] ?: EnStrings
    }

    fun getStrings(isoCode: String): StringContract {
        return getStrings(AppLanguage.fromIsoCode(isoCode))
    }
}
