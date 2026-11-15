package com.purohit.devotee.session

import com.purohit.contracts.session.SessionVault
import kotlinx.cinterop.*
import platform.CoreFoundation.*
import platform.Foundation.*
import platform.Security.*

@OptIn(ExperimentalForeignApi::class)
class IosKeychainSessionVault(
    private val serviceName: String = "com.purohit.devotee.session",
    private val accountName: String = "auth_token"
) : SessionVault {

    override fun saveToken(token: String) {
        clear() // remove existing before adding new
        val tokenData = (token as NSString).dataUsingEncoding(NSUTF8StringEncoding) ?: return

        val query = CFDictionaryCreateMutable(kCFAllocatorDefault, 4, null, null)
        CFDictionarySetValue(query, kSecClass, kSecClassGenericPassword)
        CFDictionarySetValue(query, kSecAttrService, CFBridgingRetain(serviceName as NSString))
        CFDictionarySetValue(query, kSecAttrAccount, CFBridgingRetain(accountName as NSString))
        CFDictionarySetValue(query, kSecValueData, CFBridgingRetain(tokenData))
        CFDictionarySetValue(query, kSecAttrAccessible, kSecAttrAccessibleAfterFirstUnlock)

        SecItemAdd(query, null)
        CFRelease(query)
    }

    override fun getToken(): String? {
        val query = CFDictionaryCreateMutable(kCFAllocatorDefault, 5, null, null)
        CFDictionarySetValue(query, kSecClass, kSecClassGenericPassword)
        CFDictionarySetValue(query, kSecAttrService, CFBridgingRetain(serviceName as NSString))
        CFDictionarySetValue(query, kSecAttrAccount, CFBridgingRetain(accountName as NSString))
        CFDictionarySetValue(query, kSecReturnData, kCFBooleanTrue)
        CFDictionarySetValue(query, kSecMatchLimit, kSecMatchLimitOne)

        memScoped {
            val resultRef = alloc<CFTypeRefVar>()
            val status = SecItemCopyMatching(query, resultRef.ptr)
            CFRelease(query)

            if (status == errSecSuccess) {
                val data = CFBridgingRelease(resultRef.value) as? NSData
                if (data != null) {
                    return NSString.create(data = data, encoding = NSUTF8StringEncoding) as? String
                }
            }
        }
        return null
    }

    override fun clear() {
        val query = CFDictionaryCreateMutable(kCFAllocatorDefault, 3, null, null)
        CFDictionarySetValue(query, kSecClass, kSecClassGenericPassword)
        CFDictionarySetValue(query, kSecAttrService, CFBridgingRetain(serviceName as NSString))
        CFDictionarySetValue(query, kSecAttrAccount, CFBridgingRetain(accountName as NSString))

        SecItemDelete(query)
        CFRelease(query)
    }
}
