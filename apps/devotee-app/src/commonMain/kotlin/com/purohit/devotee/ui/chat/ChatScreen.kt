package com.purohit.devotee.ui.chat

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.platform.LocalUriHandler
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.semantics
import com.purohit.contracts.i18n.AppLanguage
import com.purohit.devotee.LocalAppLanguage
import com.purohit.devotee.motionAllowed
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.purohit.devotee.network.ChatMessagePayload
import com.purohit.devotee.network.PurohitApiClient
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

@Composable
fun ChatScreen(
    roomId: String,
    priestName: String = "",
    priestPhone: String = "",
    eventAddress: String = "",
    onBack: () -> Unit
) {
    var messages by remember { mutableStateOf<List<ChatMessagePayload>>(emptyList()) }
    var inputText by remember { mutableStateOf("") }
    var isSending by remember { mutableStateOf(false) }
    val uriHandler = LocalUriHandler.current
    val language = LocalAppLanguage.current
    val backLabel = when (language) { AppLanguage.TELUGU -> "వెనుకకు"; AppLanguage.HINDI -> "वापस"; else -> "Back" }
    val sendLabel = when (language) { AppLanguage.TELUGU -> "సందేశం పంపండి"; AppLanguage.HINDI -> "संदेश भेजें"; else -> "Send message" }
    var sendError by remember { mutableStateOf<String?>(null) }
    val coroutineScope = rememberCoroutineScope()
    val listState = rememberLazyListState()

    val currentUserId = PurohitApiClient.currentProfile?.id ?: "devotee_direct"

    // Load and auto-refresh chat messages
    LaunchedEffect(roomId) {
        if (roomId.isNotBlank()) PurohitApiClient.chatUpdates(roomId).collect { messages = it }

    }

    // Scroll to bottom when message arrives
    LaunchedEffect(messages.size) {
        if (messages.isNotEmpty()) {
            if (motionAllowed()) listState.animateScrollToItem(messages.size - 1) else listState.scrollToItem(messages.size - 1)
        }
    }

    fun sendMessage(content: String) {
        if (content.isBlank() || isSending) return
        isSending = true
        sendError = null
        coroutineScope.launch {
            val res = PurohitApiClient.sendChatMessage(
                roomId = roomId,
                senderId = currentUserId,
                senderRole = "DEVOTEE",
                content = content.trim()
            )
            isSending = false
            res.onSuccess { newMsg ->
                inputText = ""
                messages = messages + newMsg
            }.onFailure {
                sendError = when (language) { AppLanguage.TELUGU -> "సందేశం పంపలేకపోయాము. మళ్లీ ప్రయత్నించండి."; AppLanguage.HINDI -> "संदेश नहीं भेजा गया। फिर से प्रयास करें।"; else -> "Message could not be sent. Please try again." }
            }
        }
    }

    val quickChips = listOf(
        "📍 Please share the directions.",
        "🔔 Ring calling bell upon arrival",
        "🪔 What should I prepare for the pooja?",
        "🛵 Swamiji, how many mins away are you?"
    )

    Scaffold(
        topBar = {
            Surface(
                color = MaterialTheme.colorScheme.primary,
                tonalElevation = 4.dp
            ) {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(horizontal = 8.dp, vertical = 10.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    IconButton(onClick = onBack, modifier = Modifier.semantics { contentDescription = backLabel }) {
                        Text("←", color = Color.White, style = MaterialTheme.typography.titleLarge)
                    }
                    Spacer(modifier = Modifier.width(4.dp))
                    Column(modifier = Modifier.weight(1f)) {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Text(
                                text = priestName,
                                color = Color.White,
                                fontWeight = FontWeight.Bold,
                                style = MaterialTheme.typography.titleMedium
                            )

                        }
                        Text(
                            text = "📍 $eventAddress",
                            color = Color.White.copy(alpha = 0.85f),
                            style = MaterialTheme.typography.labelSmall,
                            maxLines = 1
                        )
                    }
                    Button(
                        onClick = { runCatching { uriHandler.openUri("tel:" + priestPhone.filter { it.isDigit() || it == '+' }) }.onFailure { sendError = "Could not open the phone app." } },
                        enabled = priestPhone.any { it.isDigit() },
                        colors = ButtonDefaults.buttonColors(containerColor = Color.White.copy(alpha = 0.2f)),
                        shape = RoundedCornerShape(12.dp),
                        contentPadding = PaddingValues(horizontal = 10.dp, vertical = 4.dp)
                    ) {
                        Text("📞 Call", color = Color.White, style = MaterialTheme.typography.labelSmall)
                    }
                }
            }
        }
    ) { paddingValues ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(paddingValues)
                .background(MaterialTheme.colorScheme.background)
        ) {
            // Message List
            LazyColumn(
                state = listState,
                modifier = Modifier
                    .weight(1f)
                    .fillMaxWidth()
                    .padding(horizontal = 12.dp, vertical = 8.dp),
                verticalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                items(messages) { msg ->
                    if (msg.sender_role == "SYSTEM") {
                        // System notification banner
                        Box(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(vertical = 4.dp),
                            contentAlignment = Alignment.Center
                        ) {
                            Surface(
                                shape = RoundedCornerShape(12.dp),
                                color = MaterialTheme.colorScheme.surfaceVariant,
                                border = androidx.compose.foundation.BorderStroke(1.dp, Color(0xFFFDE68A)),
                                modifier = Modifier.fillMaxWidth(0.92f)
                            ) {
                                Text(
                                    text = msg.content,
                                    modifier = Modifier.padding(10.dp),
                                    style = MaterialTheme.typography.bodySmall,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant
                                )
                            }
                        }
                    } else {
                        val isMe = msg.sender_role == "DEVOTEE"
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = if (isMe) Arrangement.End else Arrangement.Start
                        ) {
                            Surface(
                                shape = RoundedCornerShape(
                                    topStart = 14.dp,
                                    topEnd = 14.dp,
                                    bottomStart = if (isMe) 14.dp else 2.dp,
                                    bottomEnd = if (isMe) 2.dp else 14.dp
                                ),
                                color = if (isMe) MaterialTheme.colorScheme.primary else Color.White,
                                tonalElevation = 2.dp,
                                shadowElevation = 1.dp,
                                modifier = Modifier.widthIn(max = 280.dp)
                            ) {
                                Column(modifier = Modifier.padding(10.dp)) {
                                    if (!isMe) {
                                        Text(
                                            text = "Purohit",
                                            fontWeight = FontWeight.Bold,
                                            style = MaterialTheme.typography.labelSmall,
                                            color = MaterialTheme.colorScheme.primary
                                        )
                                        Spacer(modifier = Modifier.height(2.dp))
                                    }
                                    Text(
                                        text = msg.content,
                                        style = MaterialTheme.typography.bodyMedium,
                                        color = if (isMe) Color.White else Color(0xFF0F172A)
                                    )
                                }
                            }
                        }
                    }
                }
            }

            sendError?.let { Text(it, color = MaterialTheme.colorScheme.error, modifier = Modifier.padding(horizontal = 16.dp, vertical = 4.dp)) }
            // Quick Direction Chips
            LazyRow(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 8.dp, vertical = 4.dp),
                horizontalArrangement = Arrangement.spacedBy(6.dp)
            ) {
                items(quickChips) { chip ->
                    SuggestionChip(
                        onClick = { sendMessage(chip) },
                        label = { Text(chip, style = MaterialTheme.typography.labelSmall) }
                    )
                }
            }

            // Bottom Input Bar
            Surface(
                modifier = Modifier.fillMaxWidth(),
                color = MaterialTheme.colorScheme.surface,
                tonalElevation = 6.dp
            ) {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(horizontal = 10.dp, vertical = 8.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    OutlinedTextField(
                        value = inputText,
                        onValueChange = { inputText = it },
                        placeholder = { Text(when (language) { AppLanguage.TELUGU -> "సందేశం టైప్ చేయండి..."; AppLanguage.HINDI -> "संदेश लिखें..."; else -> "Type a message..." }) },
                        modifier = Modifier.weight(1f),
                        shape = RoundedCornerShape(24.dp),
                        singleLine = true
                    )
                    Spacer(modifier = Modifier.width(8.dp))
                    Button(
                        onClick = { sendMessage(inputText) },
                        modifier = Modifier.semantics { contentDescription = sendLabel },
                        shape = CircleShape,
                        colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.primary),
                        contentPadding = PaddingValues(12.dp),
                        enabled = inputText.isNotBlank() && !isSending
                    ) {
                        Text("➤", fontWeight = FontWeight.Bold)
                    }
                }
            }
        }
    }
}
