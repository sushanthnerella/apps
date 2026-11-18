package com.purohit.contracts.config

import kotlin.test.AfterTest
import kotlin.test.BeforeTest
import kotlin.test.Test
import kotlin.test.assertEquals
import kotlin.test.assertFailsWith
import kotlin.test.assertTrue

class BackendConfigTest {

    @BeforeTest
    fun setUp() {
        BackendConfig.currentEnvironment = BackendConfig.ENV_EMULATOR
        BackendConfig.customOverrideUrl = null
        BackendConfig.clearStagingUrl()
    }

    @AfterTest
    fun tearDown() {
        BackendConfig.currentEnvironment = BackendConfig.ENV_EMULATOR
        BackendConfig.customOverrideUrl = null
        BackendConfig.clearStagingUrl()
    }

    @Test
    fun testDefaultEnvironmentIsEmulator() {
        assertEquals("http://10.0.2.2:8085", BackendConfig.activeBaseUrl)
    }

    @Test
    fun testLocalEnvironmentReturnsWorkstationHost() {
        BackendConfig.currentEnvironment = BackendConfig.ENV_LOCAL
        assertEquals("http://localhost:8085", BackendConfig.activeBaseUrl)
    }

    @Test
    fun testStagingWithoutUrlThrowsAndDoesNotDefaultToEmulator() {
        BackendConfig.currentEnvironment = BackendConfig.ENV_STAGING
        val ex = assertFailsWith<IllegalStateException> {
            BackendConfig.activeBaseUrl
        }
        assertTrue(ex.message!!.contains("explicit HTTPS staging URL"))
    }

    @Test
    fun testPilotWithoutUrlThrowsAndDoesNotDefaultToEmulator() {
        BackendConfig.currentEnvironment = BackendConfig.ENV_PILOT
        val ex = assertFailsWith<IllegalStateException> {
            BackendConfig.activeBaseUrl
        }
        assertTrue(ex.message!!.contains("explicit HTTPS staging URL"))
    }

    @Test
    fun testStagingRejectsNonHttpsUrl() {
        val ex = assertFailsWith<IllegalArgumentException> {
            BackendConfig.setStagingUrl("http://insecure-backend.railway.app")
        }
        assertTrue(ex.message!!.contains("HTTPS"))
    }

    @Test
    fun testStagingRejectsEmulatorIp() {
        val ex = assertFailsWith<IllegalArgumentException> {
            BackendConfig.setStagingUrl("http://10.0.2.2:8085")
        }
        assertTrue(ex.message!!.contains("HTTPS"))
    }

    @Test
    fun testValidStagingHttpsUrlAcceptedAndResolved() {
        BackendConfig.currentEnvironment = BackendConfig.ENV_STAGING
        BackendConfig.setStagingUrl("https://pooja-vahini-staging.up.railway.app")
        assertEquals("https://pooja-vahini-staging.up.railway.app", BackendConfig.activeBaseUrl)
    }

    @Test
    fun testOverrideUrlEnforcesHttpsInStaging() {
        BackendConfig.currentEnvironment = BackendConfig.ENV_STAGING
        BackendConfig.customOverrideUrl = "http://insecure-override.com"
        val ex = assertFailsWith<IllegalStateException> {
            BackendConfig.activeBaseUrl
        }
        assertTrue(ex.message!!.contains("HTTPS"))
    }
}
