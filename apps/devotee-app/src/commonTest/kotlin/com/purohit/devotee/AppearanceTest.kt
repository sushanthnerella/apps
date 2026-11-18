package com.purohit.devotee

import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNotEquals
import kotlin.test.assertTrue
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.graphics.luminance

class AppearanceTest {
    @Test fun storedThemesRoundTrip() {
        Appearance.entries.forEach { assertEquals(it, appearanceFromStoredValue(it.name)) }
        assertEquals(Appearance.ROYAL, appearanceFromStoredValue(null))
        assertEquals(Appearance.ROYAL, appearanceFromStoredValue("unknown"))
    }
    @Test fun themesHaveDistinctPrimaryColors() {
        assertNotEquals(appearanceScheme(Appearance.ROYAL).primary, appearanceScheme(Appearance.CALM).primary)
        assertNotEquals(appearanceScheme(Appearance.CALM).primary, appearanceScheme(Appearance.EARTH).primary)
        assertNotEquals(appearanceScheme(Appearance.ROYAL).primary, appearanceScheme(Appearance.EARTH).primary)
    }
    @Test fun textColorsHaveReadableContrast() {
        fun contrast(a: Color, b: Color): Float {
            val light = maxOf(a.luminance(), b.luminance())
            val dark = minOf(a.luminance(), b.luminance())
            return (light + 0.05f) / (dark + 0.05f)
        }
        Appearance.entries.forEach { theme ->
            val colors = appearanceScheme(theme)
            assertTrue(contrast(colors.primary, colors.onPrimary) >= 4.5f, "$theme primary")
            assertTrue(contrast(colors.surfaceVariant, colors.onSurfaceVariant) >= 4.5f, "$theme surface")
            assertTrue(contrast(colors.secondary, colors.onSecondary) >= 4.5f, "$theme secondary")
        }
    }
}
