package com.purohit.devotee.ui.webrtc

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp

@Composable
fun EPoojaViewerScreen() {
    var isSankalpamAudioActive by remember { mutableStateOf(false) }
    var activeViewers by remember { mutableStateOf(142) }

    Column(modifier = Modifier.fillMaxSize().padding(16.dp)) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            Column {
                Text("E-Pooja Live Telecast", style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.Bold)
                Text("Ganapathi Homa & Sankalpam", style = MaterialTheme.typography.bodySmall)
            }
            Surface(
                color = Color.Red,
                shape = RoundedCornerShape(16.dp)
            ) {
                Text(
                    text = "● LIVE ($activeViewers watching)",
                    color = Color.White,
                    modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
                    style = MaterialTheme.typography.labelSmall,
                    fontWeight = FontWeight.Bold
                )
            }
        }

        Spacer(modifier = Modifier.height(12.dp))

        // Simulated WebRTC Media Stream Canvas
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .weight(1f)
                .background(Color.Black, shape = RoundedCornerShape(16.dp)),
            contentAlignment = Alignment.Center
        ) {
            Column(horizontalAlignment = Alignment.CenterHorizontally) {
                Text(
                    "🕉️ High-Definition Video Feed",
                    color = Color.White,
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.titleMedium
                )
                Text(
                    "WebRTC SFU Stream (1080p 60fps | Opus Audio)",
                    color = Color.LightGray,
                    style = MaterialTheme.typography.bodySmall
                )
            }

            // Two-Way Interactive Sankalpam Floating Button
            Surface(
                modifier = Modifier
                    .align(Alignment.BottomEnd)
                    .padding(16.dp),
                shape = CircleShape,
                color = if (isSankalpamAudioActive) Color.Green else Color(0xFFD97706)
            ) {
                IconButton(onClick = { isSankalpamAudioActive = !isSankalpamAudioActive }) {
                    Text(
                        if (isSankalpamAudioActive) "🎤 On" else "🎤 Sankalpam",
                        color = Color.White,
                        style = MaterialTheme.typography.labelSmall
                    )
                }
            }
        }

        Spacer(modifier = Modifier.height(12.dp))

        Card(modifier = Modifier.fillMaxWidth()) {
            Column(modifier = Modifier.padding(12.dp)) {
                Text("Gotram & Nakshatram for Sankalpam:", fontWeight = FontWeight.Bold)
                Text("Kashyapa Gotram, Rohini Nakshatram (Devotee family queued for chanting)")
            }
        }
    }
}
