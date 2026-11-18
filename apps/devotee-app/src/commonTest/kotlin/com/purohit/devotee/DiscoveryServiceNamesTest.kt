package com.purohit.devotee

import com.purohit.devotee.network.PoojaCatalogItem
import com.purohit.devotee.ui.catalog.discoveryServiceNames
import kotlin.test.Test
import kotlin.test.assertEquals

class DiscoveryServiceNamesTest {
    @Test fun translatedCatalogEntriesUseCanonicalNamesForMatching() {
        val selected = listOf(
            PoojaCatalogItem(id = "1", title = "हिन्दी शीर्षक", title_en = "Namakaranam & Ayushya Homa", category = "test", base_price = 0.0, samagri_kit_price = 0.0, duration_minutes = 60, is_online_eligible = false),
            PoojaCatalogItem(id = "2", title = "తెలుగు", title_en = "Ganapathi Homa", category = "test", base_price = 0.0, samagri_kit_price = 0.0, duration_minutes = 60, is_online_eligible = false)
        )
        assertEquals("Namakaranam & Ayushya Homa + Ganapathi Homa", discoveryServiceNames(selected))
    }
}
