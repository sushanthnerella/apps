package com.purohit.priest.ui.studio

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp

@Composable
fun BroadcasterStudioScreen() {
    var isBroadcasting by remember { mutableStateOf(false) }
    var isCameraFront by remember { mutableStateOf(false) }
    var isMicMuted by remember { mutableStateOf(false) }

    Column(modifier = Modifier.fillMaxSize().padding(16.dp)) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            Text("Puja Vaahini Live Studio", style = MaterialTheme.typography.titleLarge, fontWeight = FontWeight.Bold)
            Surface(
                color = if (isBroadcasting) Color.Red else Color.Gray,
                shape = RoundedCornerShape(16.dp)
            ) {
                Text(
                    text = if (isBroadcasting) "● ON AIR (142 Viewers)" else "OFFLINE",
                    color = Color.White,
                    modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
                    style = MaterialTheme.typography.labelSmall,
                    fontWeight = FontWeight.Bold
                )
            }
        }

        Spacer(modifier = Modifier.height(12.dp))

        // WebRTC Camera Preview Area
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .weight(1f)
                .background(Color.DarkGray, shape = RoundedCornerShape(16.dp)),
            contentAlignment = Alignment.Center
        ) {
            Column(horizontalAlignment = Alignment.CenterHorizontally) {
                Text("📷 WebRTC High-Resolution Camera Feed", color = Color.White, fontWeight = FontWeight.Bold)
                Text(
                    if (isCameraFront) "Using Front Camera (Selfie / Face-to-Face)" else "Using Rear Camera (Yajna Kundam / Mandapam Focus)",
                    color = Color.LightGray,
                    style = MaterialTheme.typography.bodySmall
                )
                Spacer(modifier = Modifier.height(8.dp))
                Text(
                    "Audio: Lapel Mic (Opus 48kHz Chanting Mode - Noise Gate ON)",
                    color = Color(0xFF86EFAC),
                    style = MaterialTheme.typography.bodySmall
                )
            }
        }

        Spacer(modifier = Modifier.height(12.dp))

        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            Button(
                onClick = { isCameraFront = !isCameraFront },
                modifier = Modifier.weight(1f)
            ) {
                Text("Flip Camera")
            }
            Button(
                onClick = { isMicMuted = !isMicMuted },
                modifier = Modifier.weight(1f),
                colors = ButtonDefaults.buttonColors(
                    containerColor = if (isMicMuted) Color.Red else MaterialTheme.colorScheme.primary
                )
            ) {
                Text(if (isMicMuted) "Unmute Mic" else "Mute Mic")
            }
        }

        Spacer(modifier = Modifier.height(8.dp))

        Button(
            onClick = { isBroadcasting = !isBroadcasting },
            modifier = Modifier.fillMaxWidth(),
            colors = ButtonDefaults.buttonColors(
                containerColor = if (isBroadcasting) Color.Red else Color(0xFF15803D)
            )
        ) {
            Text(if (isBroadcasting) "END TELECAST" else "START LIVE BROADCAST")
        }
    }
}
