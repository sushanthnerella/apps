package com.purohit.priest.ui.schedule

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.purohit.contracts.i18n.StringContract
import com.purohit.contracts.slots.PriestSlot
import com.purohit.priest.network.PurohitApiClient
import kotlinx.coroutines.launch

@Composable
fun PriestScheduleScreen(
    priestId: String = "6e7d277c-4ee2-469f-96ac-331517389762",
    strings: StringContract
) {
    var selectedDate by remember { mutableStateOf("Tomorrow (Nov 4)") }
    val slots = remember {
        mutableStateListOf(
            PriestSlot("s1", priestId, "2026-11-04", "05:30", "07:30", isBooked = false),
            PriestSlot("s2", priestId, "2026-11-04", "08:30", "10:30", isBooked = true),
            PriestSlot("s3", priestId, "2026-11-04", "11:45", "13:45", isBooked = true),
            PriestSlot("s4", priestId, "2026-11-04", "17:30", "19:30", isBooked = false)
        )
    }

    val coroutineScope = rememberCoroutineScope()

    LaunchedEffect(selectedDate) {
        val dateParam = if (selectedDate.contains("Today")) "2026-11-03" else if (selectedDate.contains("Tomorrow")) "2026-11-04" else "2026-11-10"
        val res = PurohitApiClient.fetchPriestSlots(priestId, dateParam)
        res.onSuccess { remoteSlots ->
            slots.clear()
            remoteSlots.forEach { r ->
                slots.add(
                    PriestSlot(
                        id = r.id,
                        priestId = r.priest_id,
                        slotDate = r.slot_date,
                        startTime = r.start_time.take(5),
                        endTime = r.end_time.take(5),
                        isBooked = r.is_booked
                    )
                )
            }
        }
    }

    var showAddSlotModal by remember { mutableStateOf(false) }
    var newStartTime by remember { mutableStateOf("07:00") }
    var newEndTime by remember { mutableStateOf("09:30") }

    Column(modifier = Modifier.fillMaxSize().padding(16.dp)) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            Column {
                Text(
                    text = "📅 " + strings.common.tabSchedule,
                    style = MaterialTheme.typography.headlineMedium,
                    fontWeight = FontWeight.Bold,
                    color = MaterialTheme.colorScheme.primary
                )
                Text(
                    text = "Vedic Muhurtham Calendar • Auto-blocks on booking acceptance",
                    style = MaterialTheme.typography.bodySmall,
                    color = Color.Gray
                )
            }
            FilledTonalButton(onClick = { showAddSlotModal = true }) {
                Text("+ Slot")
            }
        }

        Spacer(modifier = Modifier.height(16.dp))

        // Date Tabs
        Surface(
            color = MaterialTheme.colorScheme.surfaceVariant,
            shape = RoundedCornerShape(12.dp),
            modifier = Modifier.fillMaxWidth()
        ) {
            Row(
                modifier = Modifier.padding(6.dp).fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceAround
            ) {
                listOf("Today (Nov 3)", "Tomorrow (Nov 4)", "Next Week (Nov 10)").forEach { d ->
                    Surface(
                        color = if (selectedDate.contains(d.take(6))) MaterialTheme.colorScheme.primary else Color.Transparent,
                        shape = RoundedCornerShape(8.dp),
                        modifier = Modifier.clickable { selectedDate = d }
                    ) {
                        Text(
                            text = d,
                            modifier = Modifier.padding(horizontal = 10.dp, vertical = 6.dp),
                            color = if (selectedDate.contains(d.take(6))) Color.White else Color.Black,
                            fontWeight = FontWeight.Bold,
                            style = MaterialTheme.typography.labelMedium
                        )
                    }
                }
            }
        }

        Spacer(modifier = Modifier.height(16.dp))

        Text(
            text = "Hourly Muhurtham Windows ($selectedDate)",
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.titleSmall
        )

        Spacer(modifier = Modifier.height(10.dp))

        if (slots.isEmpty()) {
            Box(
                modifier = Modifier.weight(1f).fillMaxWidth(),
                contentAlignment = Alignment.Center
            ) {
                Column(horizontalAlignment = Alignment.CenterHorizontally) {
                    Text("📅", style = MaterialTheme.typography.displayMedium)
                    Spacer(modifier = Modifier.height(8.dp))
                    Text(
                        "No availability slots published for $selectedDate",
                        style = MaterialTheme.typography.bodyMedium,
                        fontWeight = FontWeight.Medium,
                        color = Color.Gray
                    )
                    Spacer(modifier = Modifier.height(4.dp))
                    Text(
                        "Tap '+ Slot' above to configure a Vedic Muhurtham window.",
                        style = MaterialTheme.typography.bodySmall,
                        color = Color.Gray
                    )
                }
            }
        } else {
            LazyColumn(
                verticalArrangement = Arrangement.spacedBy(10.dp),
                modifier = Modifier.weight(1f)
            ) {
            items(slots) { slot ->
                Card(
                    modifier = Modifier.fillMaxWidth(),
                    shape = RoundedCornerShape(12.dp),
                    colors = CardDefaults.cardColors(
                        containerColor = if (slot.isBooked) Color(0xFFFEE2E2) else Color(0xFFECFDF5)
                    )
                ) {
                    Row(
                        modifier = Modifier.padding(16.dp).fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Column {
                            Text(
                                text = "⏰ ${slot.startTime} - ${slot.endTime}",
                                style = MaterialTheme.typography.titleMedium,
                                fontWeight = FontWeight.Bold,
                                color = if (slot.isBooked) Color(0xFF991B1B) else Color(0xFF065F46)
                            )
                            Spacer(modifier = Modifier.height(2.dp))
                            Text(
                                text = if (slot.isBooked) "🔴 Booked (Sri Satyanarayana Swamy Vratam - Sushanth2) • Escrow Locked" else "🟢 Available for Instant Reservation",
                                style = MaterialTheme.typography.labelSmall,
                                color = if (slot.isBooked) Color(0xFF991B1B) else Color.DarkGray,
                                fontWeight = if (slot.isBooked) FontWeight.Bold else FontWeight.Normal
                            )
                        }

                        if (!slot.isBooked) {
                            TextButton(onClick = { slots.remove(slot) }) {
                                Text("Remove", color = Color.Gray)
                            }
                        }
                    }
                }
            }
        }
    }
    }

    if (showAddSlotModal) {
        AlertDialog(
            onDismissRequest = { showAddSlotModal = false },
            title = { Text("Add Availability Slot", fontWeight = FontWeight.Bold) },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                    OutlinedTextField(
                        value = newStartTime,
                        onValueChange = { newStartTime = it },
                        label = { Text("Start Time (HH:MM)") },
                        modifier = Modifier.fillMaxWidth()
                    )
                    OutlinedTextField(
                        value = newEndTime,
                        onValueChange = { newEndTime = it },
                        label = { Text("End Time (HH:MM)") },
                        modifier = Modifier.fillMaxWidth()
                    )
                }
            },
            confirmButton = {
                Button(
                    onClick = {
                        slots.add(PriestSlot("s_${slots.size + 1}", priestId, "2026-11-04", newStartTime, newEndTime))
                        showAddSlotModal = false
                    }
                ) {
                    Text("Add Window")
                }
            },
            dismissButton = {
                TextButton(onClick = { showAddSlotModal = false }) {
                    Text("Cancel")
                }
            }
        )
    }
}
