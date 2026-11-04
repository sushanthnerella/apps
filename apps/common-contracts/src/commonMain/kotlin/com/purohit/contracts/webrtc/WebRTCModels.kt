package com.purohit.contracts.webrtc

import kotlinx.serialization.Serializable

@Serializable
enum class WebRTCMessageType {
    JOIN_ROOM,
    ROOM_JOINED,
    OFFER,
    ANSWER,
    ICE_CANDIDATE,
    LEAVE_ROOM,
    STREAM_STATE_CHANGED,
    ERROR
}

@Serializable
enum class StreamMode {
    ONE_TO_ONE_PRIVATE,  // Devotee family <-> Priest bidirectional
    ONE_TO_MANY_BROADCAST // Priest broadcaster -> Many devotee viewers (SFU)
}

@Serializable
data class WebRTCMessage(
    val type: WebRTCMessageType,
    val roomId: String,
    val senderId: String,
    val recipientId: String? = null,
    val sdp: String? = null,
    val sdpMid: String? = null,
    val sdpMLineIndex: Int? = null,
    val candidate: String? = null,
    val streamMode: StreamMode = StreamMode.ONE_TO_ONE_PRIVATE,
    val errorMessage: String? = null
)

@Serializable
data class RoomState(
    val roomId: String,
    val bookingId: String,
    val priestId: String,
    val isLive: Boolean,
    val activeViewersCount: Int = 0,
    val isMuted: Boolean = false
)
