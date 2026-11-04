package com.purohit.devotee.ui.temple

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
import com.purohit.contracts.temple.Temple
import com.purohit.contracts.temple.TempleVipPuja

import com.purohit.devotee.network.PurohitApiClient
import kotlinx.coroutines.launch

@Composable
fun TempleDirectoryScreen(
    strings: StringContract,
    onBookVipPuja: (Temple, TempleVipPuja) -> Unit
) {
    var temples by remember { mutableStateOf<List<Temple>>(emptyList()) }
    var selectedTemple by remember { mutableStateOf<Temple?>(null) }
    var vipPujas by remember { mutableStateOf<List<TempleVipPuja>>(emptyList()) }
    var showBookingModal by remember { mutableStateOf(false) }
    var selectedPujaForBooking by remember { mutableStateOf<TempleVipPuja?>(null) }
    var isBooking by remember { mutableStateOf(false) }
    val coroutineScope = rememberCoroutineScope()

    LaunchedEffect(Unit) {
        val res = PurohitApiClient.fetchTemples()
        res.onSuccess {
            temples = it
            if (it.isNotEmpty()) selectedTemple = it[0]
        }
    }

    LaunchedEffect(selectedTemple) {
        selectedTemple?.let {
            val res = PurohitApiClient.fetchTempleVipPujas(it.id)
            res.onSuccess { pujas ->
                vipPujas = pujas
            }
        }
    }

    Column(modifier = Modifier.fillMaxSize().padding(16.dp)) {
        Text(
            text = "🏛️ " + strings.temple.templesTitle,
            style = MaterialTheme.typography.headlineMedium,
            fontWeight = FontWeight.Bold,
            color = MaterialTheme.colorScheme.primary
        )
        Text(
            text = "Book official VIP darshan passes and live sanctum E-Poojas directly with temple boards",
            style = MaterialTheme.typography.bodySmall,
            color = Color.Gray
        )

        Spacer(modifier = Modifier.height(14.dp))

        LazyColumn(
            verticalArrangement = Arrangement.spacedBy(12.dp),
            modifier = Modifier.weight(1f)
        ) {
            items(temples) { temple ->
                Card(
                    modifier = Modifier.fillMaxWidth().clickable {
                        selectedTemple = temple
                        showBookingModal = true
                    },
                    shape = RoundedCornerShape(16.dp),
                    colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant)
                ) {
                    Column(modifier = Modifier.padding(16.dp)) {
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.SpaceBetween,
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            Text(
                                text = temple.name,
                                style = MaterialTheme.typography.titleMedium,
                                fontWeight = FontWeight.Bold,
                                modifier = Modifier.weight(1f)
                            )
                            Surface(
                                color = MaterialTheme.colorScheme.primary.copy(alpha = 0.15f),
                                shape = RoundedCornerShape(8.dp)
                            ) {
                                Text(
                                    text = temple.city,
                                    modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
                                    style = MaterialTheme.typography.labelSmall,
                                    fontWeight = FontWeight.Bold,
                                    color = MaterialTheme.colorScheme.primary
                                )
                            }
                        }

                        Spacer(modifier = Modifier.height(6.dp))

                        Text(
                            text = "🕉️ ${strings.temple.sanctumDeity}: ${temple.deity}",
                            style = MaterialTheme.typography.bodyMedium,
                            fontWeight = FontWeight.SemiBold
                        )

                        Spacer(modifier = Modifier.height(4.dp))

                        Text(
                            text = "⏰ ${strings.temple.templeTimings}: ${temple.timings}",
                            style = MaterialTheme.typography.bodySmall,
                            color = Color.DarkGray
                        )

                        Spacer(modifier = Modifier.height(10.dp))

                        Button(
                            onClick = {
                                selectedTemple = temple
                                showBookingModal = true
                            },
                            modifier = Modifier.fillMaxWidth(),
                            colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.primary)
                        ) {
                            Text(strings.temple.vipPassesTitle)
                        }
                    }
                }
            }
        }
    }

    // VIP Puja Selection Modal
    if (showBookingModal && selectedTemple != null) {
        AlertDialog(
            onDismissRequest = { showBookingModal = false },
            title = {
                Text(
                    text = selectedTemple!!.name,
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.titleMedium
                )
            },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                    Text(
                        text = strings.temple.vipPassesTitle,
                        fontWeight = FontWeight.Bold,
                        color = MaterialTheme.colorScheme.primary
                    )
                    vipPujas.forEach { vp ->
                        Card(
                            modifier = Modifier.fillMaxWidth().clickable {
                                coroutineScope.launch {
                                    PurohitApiClient.bookTempleTicket(selectedTemple!!.id, vp.id, vp.price)
                                }
                                onBookVipPuja(selectedTemple!!, vp)
                                showBookingModal = false
                            },
                            shape = RoundedCornerShape(10.dp),
                            colors = CardDefaults.cardColors(containerColor = Color(0xFFFEF3C7))
                        ) {
                            Column(modifier = Modifier.padding(10.dp)) {
                                Row(
                                    modifier = Modifier.fillMaxWidth(),
                                    horizontalArrangement = Arrangement.SpaceBetween
                                ) {
                                    Text(vp.title, fontWeight = FontWeight.Bold, modifier = Modifier.weight(1f))
                                    Text("₹${vp.price.toInt()}", fontWeight = FontWeight.Bold, color = Color(0xFFB45309))
                                }
                                Text(vp.description, style = MaterialTheme.typography.bodySmall)
                            }
                        }
                    }
                }
            },
            confirmButton = {
                OutlinedButton(onClick = { showBookingModal = false }) {
                    Text(strings.common.closeAction)
                }
            }
        )
    }
}
