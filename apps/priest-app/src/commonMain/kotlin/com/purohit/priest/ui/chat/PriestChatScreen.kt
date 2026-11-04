package com.purohit.priest.ui.chat

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
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.purohit.priest.network.ChatMessagePayload
import com.purohit.priest.network.PurohitApiClient
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

@Composable
fun PriestChatScreen(
    roomId: String,
    priestId: String = "6e7d277c-4ee2-469f-96ac-331517389762",
    devoteeName: String = "Sushanth2",
    devoteePhone: String = "9177961089",
    eventAddress: String = "Plot 42, Sri Rama Nagar, Madhav Nagar, Hyderabad",
    initialPrefillText: String = "",
    onBack: () -> Unit
) {
    var messages by remember { mutableStateOf<List<ChatMessagePayload>>(emptyList()) }
    var inputText by remember { mutableStateOf(initialPrefillText) }
    var isSending by remember { mutableStateOf(false) }
    val coroutineScope = rememberCoroutineScope()
    val listState = rememberLazyListState()

    // Sync messages every 2.5s
    LaunchedEffect(roomId) {
        while (true) {
            val res = PurohitApiClient.fetchChatMessages(roomId)
            res.onSuccess { list ->
                messages = list
            }
            delay(2500L)
        }
    }

    LaunchedEffect(messages.size) {
        if (messages.isNotEmpty()) {
            listState.animateScrollToItem(messages.size - 1)
        }
    }

    fun sendMessage(content: String) {
        if (content.isBlank() || isSending) return
        isSending = true
        coroutineScope.launch {
            val res = PurohitApiClient.sendChatMessage(
                roomId = roomId,
                senderId = priestId,
                senderRole = "PRIEST",
                content = content.trim()
            )
            isSending = false
            res.onSuccess { newMsg ->
                inputText = ""
                messages = messages + newMsg
            }
        }
    }

    val quickPriestChips = listOf(
        "🙏 Subhamastu! Started on my two-wheeler.",
        "🛵 Reaching your home in ~5 minutes.",
        "🪔 Sacred Samagri Kit packed. Please ensure ghee & 3 plates ready.",
        "📍 Reached your building entrance/gate."
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
                    IconButton(onClick = onBack) {
                        Text("←", color = Color.White, style = MaterialTheme.typography.titleLarge)
                    }
                    Spacer(modifier = Modifier.width(4.dp))
                    Column(modifier = Modifier.weight(1f)) {
                        Row(verticalAlignment = Alignment.CenterVertically) {
                            Text(
                                text = "👤 Devotee: $devoteeName",
                                color = Color.White,
                                fontWeight = FontWeight.Bold,
                                style = MaterialTheme.typography.titleMedium
                            )
                            Spacer(modifier = Modifier.width(6.dp))
                            Surface(
                                shape = CircleShape,
                                color = Color(0xFF4ADE80),
                                modifier = Modifier.size(8.dp)
                            ) {}
                        }
                        Text(
                            text = "📍 $eventAddress • 📞 $devoteePhone",
                            color = Color.White.copy(alpha = 0.85f),
                            style = MaterialTheme.typography.labelSmall
                        )
                    }
                }
            }
        }
    ) { paddingValues ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(paddingValues)
                .background(Color(0xFFFFFBEB))
        ) {
            // Devotee Address Guidance Strip
            Surface(
                color = Color(0xFFFEF3C7),
                modifier = Modifier.fillMaxWidth()
            ) {
                Row(
                    modifier = Modifier.padding(horizontal = 12.dp, vertical = 6.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text("📍", style = MaterialTheme.typography.bodySmall)
                    Spacer(modifier = Modifier.width(6.dp))
                    Text(
                        text = "Destination: $eventAddress",
                        style = MaterialTheme.typography.labelSmall,
                        color = Color(0xFF78350F),
                        fontWeight = FontWeight.SemiBold
                    )
                }
            }

            // Message Stream
            LazyColumn(
                state = listState,
                modifier = Modifier
                    .weight(1f)
                    .fillMaxWidth()
                    .padding(horizontal = 12.dp, vertical = 8.dp),
                verticalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                items(messages) { msg ->
                    val isFromMe = msg.sender_role.equals("PRIEST", ignoreCase = true)
                    val isSystem = msg.sender_role.equals("SYSTEM", ignoreCase = true)

                    if (isSystem) {
                        Box(modifier = Modifier.fillMaxWidth(), contentAlignment = Alignment.Center) {
                            Surface(
                                shape = RoundedCornerShape(8.dp),
                                color = Color(0xFFE2E8F0)
                            ) {
                                Text(
                                    text = "🔔 ${msg.content}",
                                    style = MaterialTheme.typography.labelSmall,
                                    color = Color(0xFF334155),
                                    modifier = Modifier.padding(horizontal = 10.dp, vertical = 4.dp)
                                )
                            }
                        }
                    } else {
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = if (isFromMe) Arrangement.End else Arrangement.Start
                        ) {
                            Surface(
                                shape = RoundedCornerShape(
                                    topStart = 14.dp,
                                    topEnd = 14.dp,
                                    bottomStart = if (isFromMe) 14.dp else 2.dp,
                                    bottomEnd = if (isFromMe) 2.dp else 14.dp
                                ),
                                color = if (isFromMe) MaterialTheme.colorScheme.primary else Color.White,
                                shadowElevation = 1.dp,
                                modifier = Modifier.widthIn(max = 290.dp)
                            ) {
                                Column(modifier = Modifier.padding(10.dp)) {
                                    Text(
                                        text = if (isFromMe) "🕉️ You (Purohit)" else "👤 $devoteeName",
                                        style = MaterialTheme.typography.labelSmall,
                                        fontWeight = FontWeight.Bold,
                                        color = if (isFromMe) Color.White.copy(alpha = 0.85f) else MaterialTheme.colorScheme.primary
                                    )
                                    Spacer(modifier = Modifier.height(2.dp))
                                    Text(
                                        text = msg.content,
                                        style = MaterialTheme.typography.bodyMedium,
                                        color = if (isFromMe) Color.White else Color.Black
                                    )
                                    if (msg.created_at.isNotBlank()) {
                                        Spacer(modifier = Modifier.height(2.dp))
                                        Text(
                                            text = if (msg.created_at.contains("T")) {
                                                msg.created_at.substringAfter("T").take(5)
                                            } else "",
                                            style = MaterialTheme.typography.labelSmall,
                                            color = if (isFromMe) Color.White.copy(alpha = 0.7f) else Color.Gray,
                                            modifier = Modifier.align(Alignment.End)
                                        )
                                    }
                                }
                            }
                        }
                    }
                }
            }

            // Quick Chips
            LazyRow(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 8.dp, vertical = 4.dp),
                horizontalArrangement = Arrangement.spacedBy(6.dp)
            ) {
                items(quickPriestChips) { chip ->
                    SuggestionChip(
                        onClick = { sendMessage(chip) },
                        label = { Text(chip, style = MaterialTheme.typography.labelSmall) }
                    )
                }
            }

            // Text Input & Send
            Surface(
                color = Color.White,
                tonalElevation = 3.dp,
                modifier = Modifier.fillMaxWidth()
            ) {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(horizontal = 8.dp, vertical = 6.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    OutlinedTextField(
                        value = inputText,
                        onValueChange = { inputText = it },
                        placeholder = { Text("Message devotee regarding arrival or samagri...") },
                        modifier = Modifier.weight(1f),
                        shape = RoundedCornerShape(24.dp),
                        singleLine = false,
                        maxLines = 3
                    )
                    Spacer(modifier = Modifier.width(6.dp))
                    IconButton(
                        onClick = { sendMessage(inputText) },
                        enabled = inputText.isNotBlank() && !isSending,
                        modifier = Modifier
                            .size(46.dp)
                            .clip(CircleShape)
                            .background(if (inputText.isNotBlank()) MaterialTheme.colorScheme.primary else Color.LightGray)
                    ) {
                        Text("➤", color = Color.White, style = MaterialTheme.typography.titleMedium)
                    }
                }
            }
        }
    }
}
