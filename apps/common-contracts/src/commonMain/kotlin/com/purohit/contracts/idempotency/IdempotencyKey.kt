package com.purohit.contracts.idempotency

object IdempotencyKeyGenerator {
    private val hexDigits = "0123456789abcdef".toCharArray()

    fun generate(): String {
        val bytes = ByteArray(16)
        kotlin.random.Random.Default.nextBytes(bytes)
        // RFC 4122 v4 UUID format
        bytes[6] = ((bytes[6].toInt() and 0x0f) or 0x40).toByte()
        bytes[8] = ((bytes[8].toInt() and 0x3f) or 0x80).toByte()

        val chars = CharArray(36)
        var charIdx = 0
        for (i in 0 until 16) {
            if (i == 4 || i == 6 || i == 8 || i == 10) {
                chars[charIdx++] = '-'
            }
            val b = bytes[i].toInt() and 0xff
            chars[charIdx++] = hexDigits[b ushr 4]
            chars[charIdx++] = hexDigits[b and 0x0f]
        }
        return chars.concatToString()
    }
}
