package com.purohit.contracts.chat

import kotlinx.serialization.Serializable

@Serializable
enum class MessageRole {
    DEVOTEE,
    PRIEST,
    SYSTEM
}

@Serializable
enum class MessageType {
    TEXT,
    IMAGE,
    LOCATION,
    SYSTEM_EVENT
}

@Serializable
data class ChatMessage(
    val id: String = "",
    val roomId: String,
    val senderId: String,
    val senderRole: MessageRole,
    val messageType: MessageType = MessageType.TEXT,
    val content: String,
    val attachmentUrl: String = "",
    val isRead: Boolean = false,
    val timestamp: String = ""
)

@Serializable
data class SendChatMessageRequest(
    val roomId: String,
    val senderId: String,
    val senderRole: MessageRole,
    val messageType: MessageType = MessageType.TEXT,
    val content: String,
    val attachmentUrl: String = ""
)
