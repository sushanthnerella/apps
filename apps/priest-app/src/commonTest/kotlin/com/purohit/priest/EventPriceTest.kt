package com.purohit.priest

import com.purohit.priest.ui.pushkaralu.paiseToRupeesText
import com.purohit.priest.ui.pushkaralu.rupeesToPaise
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNull

class EventPriceTest {
    @Test
    fun rupeeTextConvertsToPaise() {
        assertEquals(110000L, rupeesToPaise("1100"))
        assertEquals(110050L, rupeesToPaise("1100.5"))
        assertEquals(110055L, rupeesToPaise("1100.55"))
        assertNull(rupeesToPaise("1100.555"))
        assertNull(rupeesToPaise(""))
        assertNull(rupeesToPaise("0"))
        assertNull(rupeesToPaise("1.2.3"))
        assertNull(rupeesToPaise(".5"))
        assertNull(rupeesToPaise("200000")) // above ₹1,00,000 cap
    }

    @Test
    fun paiseRoundTrips() {
        assertEquals("1100", paiseToRupeesText(110000))
        assertEquals("1100.05", paiseToRupeesText(110005))
        assertEquals(110005L, rupeesToPaise(paiseToRupeesText(110005)))
    }
}
