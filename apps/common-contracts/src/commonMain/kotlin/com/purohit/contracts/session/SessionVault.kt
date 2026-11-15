package com.purohit.contracts.session

/**
 * Minimal cross-platform secure credential store for authentication session recovery.
 * On Android, implemented via secure SharedPreferences / KeyStore.
 * On iOS, implemented via Apple Keychain Services (kSecClassGenericPassword).
 */
interface SessionVault {
    fun saveToken(token: String)
    fun getToken(): String?
    fun clear()
}

/**
 * Common in-memory fallback or testing vault
 */
class InMemorySessionVault : SessionVault {
    private var token: String? = null
    override fun saveToken(token: String) { this.token = token }
    override fun getToken(): String? = token
    override fun clear() { token = null }
}
