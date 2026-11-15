package com.purohit.devotee

import com.purohit.devotee.network.ChatSendPayload
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import kotlinx.serialization.json.jsonObject
import kotlin.test.Test
import kotlin.test.assertFalse
import kotlin.test.assertTrue

class ChatRequestTest {
    @Test fun outgoingMessageOmitsServerManagedTimestamp() {
        val json = Json { encodeDefaults = true }
        val payload = ChatSendPayload(room_id = "room", sender_id = "sender", sender_role = "DEVOTEE", content = "Hello")
        val fields = json.parseToJsonElement(json.encodeToString(payload)).jsonObject
        assertTrue("room_id" in fields && "content" in fields)
        assertFalse("created_at" in fields, "An empty timestamp is rejected by the server")
    }
}
