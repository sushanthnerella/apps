package com.purohit.contracts.session

import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertNull

class SessionVaultTest {
    @Test
    fun testInMemoryVaultLifecycle() {
        val vault = InMemorySessionVault()
        assertNull(vault.getToken())
        
        vault.saveToken("test.jwt.token")
        assertEquals("test.jwt.token", vault.getToken())
        
        vault.clear()
        assertNull(vault.getToken())
    }
}
