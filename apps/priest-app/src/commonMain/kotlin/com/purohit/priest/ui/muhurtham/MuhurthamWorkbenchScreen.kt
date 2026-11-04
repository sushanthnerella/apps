package com.purohit.priest.ui.muhurtham

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp

@Composable
fun MuhurthamWorkbenchScreen() {
    var selectedLagna by remember { mutableStateOf("Vrishabha Lagna (05:45 AM - 07:15 AM)") }
    var notesForDevotee by remember { mutableStateOf("Auspicious for Gruhapravesam. Avoid Rahu Kalam (04:30 PM - 06:00 PM).") }
    var isConfirmed by remember { mutableStateOf(false) }

    Column(modifier = Modifier.fillMaxSize().padding(16.dp)) {
        Text("Muhurtham Workbench", style = MaterialTheme.typography.headlineMedium, fontWeight = FontWeight.Bold)
        Text("Requests where devotee asked you to calculate auspicious time", style = MaterialTheme.typography.bodySmall)

        Spacer(modifier = Modifier.height(16.dp))

        Card(modifier = Modifier.fillMaxWidth()) {
            Column(modifier = Modifier.padding(16.dp)) {
                Text("Devotee Request Details", fontWeight = FontWeight.Bold, color = MaterialTheme.colorScheme.primary)
                Spacer(modifier = Modifier.height(8.dp))
                Text("Occasion: Gruhapravesam (House Warming)")
                Text("Devotee: Srikanth Sharma (Family)")
                Text("Gotram: Kasyapa | Nakshatra: Uttara Phalguni | Rashi: Kanya")
                Text("Requested Window: Nov 14, 2026 – Nov 18, 2026")
            }
        }

        Spacer(modifier = Modifier.height(16.dp))

        // Panchangam Guidance Card
        Card(
            modifier = Modifier.fillMaxWidth(),
            colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.primaryContainer)
        ) {
            Column(modifier = Modifier.padding(16.dp)) {
                Text("📖 Panchangam Guidance (Kartika Masam - Shukla Paksha):", fontWeight = FontWeight.Bold)
                Text("• Best Tithi: Panchami / Dashami")
                Text("• Recommended Day: Thursday, Nov 15")
                Text("• Rahu Kalam: 01:30 PM - 03:00 PM (Strictly avoid)")
            }
        }

        Spacer(modifier = Modifier.height(16.dp))

        Text("Select Auspicious Lagna & Slot:", fontWeight = FontWeight.Bold)
        Spacer(modifier = Modifier.height(8.dp))

        listOf(
            "Vrishabha Lagna (05:45 AM - 07:15 AM) - Highly Auspicious",
            "Mithuna Lagna (08:30 AM - 10:15 AM) - Suitable",
            "Simha Lagna (11:30 AM - 01:00 PM) - Secondary"
        ).forEach { lagna ->
            Row(modifier = Modifier.fillMaxWidth()) {
                RadioButton(
                    selected = selectedLagna.startsWith(lagna.take(15)),
                    onClick = { selectedLagna = lagna }
                )
                Text(lagna, modifier = Modifier.padding(top = 12.dp), style = MaterialTheme.typography.bodyMedium)
            }
        }

        Spacer(modifier = Modifier.height(16.dp))

        OutlinedTextField(
            value = notesForDevotee,
            onValueChange = { notesForDevotee = it },
            label = { Text("Instructions / Notes to Devotee") },
            modifier = Modifier.fillMaxWidth()
        )

        Spacer(modifier = Modifier.height(16.dp))

        Button(
            onClick = { isConfirmed = true },
            modifier = Modifier.fillMaxWidth(),
            colors = ButtonDefaults.buttonColors(containerColor = Color(0xFF15803D))
        ) {
            Text(if (isConfirmed) "✓ Muhurtham Sent to Devotee" else "Confirm & Send Muhurtham to Devotee")
        }
    }
}
