package com.purohit.contracts.i18n

import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNotNull
import kotlin.test.assertTrue

class I18nCompletenessTest {

    @Test
    fun testAllLanguagesAreRegistered() {
        val en = LocalizationManager.getStrings(AppLanguage.ENGLISH)
        val te = LocalizationManager.getStrings(AppLanguage.TELUGU)
        val hi = LocalizationManager.getStrings(AppLanguage.HINDI)

        assertNotNull(en)
        assertNotNull(te)
        assertNotNull(hi)

        assertEquals(AppLanguage.ENGLISH, en.language)
        assertEquals(AppLanguage.TELUGU, te.language)
        assertEquals(AppLanguage.HINDI, hi.language)
    }

    @Test
    fun testIsoCodeResolutions() {
        assertEquals(AppLanguage.ENGLISH, AppLanguage.fromIsoCode("en"))
        assertEquals(AppLanguage.TELUGU, AppLanguage.fromIsoCode("te"))
        assertEquals(AppLanguage.HINDI, AppLanguage.fromIsoCode("hi"))
        // Fallback
        assertEquals(AppLanguage.ENGLISH, AppLanguage.fromIsoCode("fr"))
    }

    @Test
    fun testTeluguStringsHaveAuthenticContent() {
        val te = LocalizationManager.getStrings("te")
        assertTrue(te.auth.loginTitle.contains("పూజా వాహిని"))
        assertTrue(te.auth.devoteeRegistrationTitle.contains("యజమాని"))
        assertTrue(te.auth.priestRegistrationTitle.contains("అయ్యాగార్లు"))
        assertTrue(te.catalog.catalogTitle.contains("పూజలు"))
        assertTrue(te.samagri.checklistTitle.contains("సామాగ్రి"))
        assertTrue(te.cancellation.cancelBookingTitle.contains("రద్దు"))
    }

    @Test
    fun testHindiStringsHaveAuthenticContent() {
        val hi = LocalizationManager.getStrings("hi")
        assertTrue(hi.auth.loginTitle.contains("पूजा वाहिनी"))
        assertTrue(hi.auth.devoteeRegistrationTitle.contains("यजमान"))
        assertTrue(hi.auth.priestRegistrationTitle.contains("पुरोहित"))
        assertTrue(hi.catalog.catalogTitle.contains("अनुष्ठान"))
        assertTrue(hi.cancellation.cancelBookingTitle.contains("रद्द"))
    }
}
