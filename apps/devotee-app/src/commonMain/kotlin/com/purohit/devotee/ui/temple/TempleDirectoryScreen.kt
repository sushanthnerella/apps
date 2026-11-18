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
import com.purohit.contracts.idempotency.IdempotencyKeyGenerator
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
    var templeLoading by remember { mutableStateOf(true) }
    var templeError by remember { mutableStateOf<String?>(null) }
    var temples by remember { mutableStateOf<List<Temple>>(emptyList()) }
    var selectedTemple by remember { mutableStateOf<Temple?>(null) }
    var vipLoading by remember { mutableStateOf(false) }
    var vipError by remember { mutableStateOf<String?>(null) }
    var vipPujas by remember { mutableStateOf<List<TempleVipPuja>>(emptyList()) }
    var showBookingModal by remember { mutableStateOf(false) }
    var ticketIntentKey by remember { mutableStateOf(IdempotencyKeyGenerator.generate()) }
    var selectedPujaForBooking by remember { mutableStateOf<TempleVipPuja?>(null) }
    var isBooking by remember { mutableStateOf(false) }
    val coroutineScope = rememberCoroutineScope()

    LaunchedEffect(Unit) {
        val res = PurohitApiClient.fetchTemples()
        res.onSuccess {
            temples = it
            if (it.isNotEmpty()) selectedTemple = it[0]
        }.onFailure { templeError = "Could not load temples. Please reopen this screen to retry." }
        templeLoading = false
    }

    LaunchedEffect(selectedTemple) {
        vipPujas = emptyList()
        vipError = null
        selectedTemple?.let {
            vipLoading = true
            val res = PurohitApiClient.fetchTempleVipPujas(it.id)
            res.onSuccess { pujas -> vipPujas = pujas }
                .onFailure { vipError = "Could not load temple services. Please try again." }
            vipLoading = false
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
            text = "Explore temples and available services",
            style = MaterialTheme.typography.bodySmall,
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )

        Spacer(modifier = Modifier.height(14.dp))

        if (templeLoading) LinearProgressIndicator(modifier = Modifier.fillMaxWidth())
        else if (templeError != null) Text(templeError!!, color = MaterialTheme.colorScheme.error)
        else if (temples.isEmpty()) Text("No temples available", color = MaterialTheme.colorScheme.onSurfaceVariant)

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
                            color = MaterialTheme.colorScheme.onSurfaceVariant
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
                    if (vipLoading) LinearProgressIndicator(modifier = Modifier.fillMaxWidth())
                    else if (vipError != null) Text(vipError!!, color = MaterialTheme.colorScheme.error)
                    else if (vipPujas.isEmpty()) Text("No services available", color = MaterialTheme.colorScheme.onSurfaceVariant)
                    if (isBooking) LinearProgressIndicator(modifier = Modifier.fillMaxWidth())
                    vipPujas.forEach { vp ->
                        Card(
                            modifier = Modifier.fillMaxWidth().clickable(enabled = !isBooking) {
                                val temple = selectedTemple ?: return@clickable
                                isBooking = true
                                vipError = null
                                coroutineScope.launch {
                                    PurohitApiClient.bookTempleTicket(temple.id, vp.id, vp.price, idempotencyKey = ticketIntentKey)
                                        .onSuccess {
                                            ticketIntentKey = IdempotencyKeyGenerator.generate()
                                            showBookingModal = false
                                            onBookVipPuja(temple, vp)
                                        }.onFailure {
                                            vipError = "Could not book this service. Please try again."
                                        }
                                    isBooking = false
                                }
                            },
                            shape = RoundedCornerShape(10.dp),
                            colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant)
                        ) {
                            Column(modifier = Modifier.padding(10.dp)) {
                                Row(
                                    modifier = Modifier.fillMaxWidth(),
                                    horizontalArrangement = Arrangement.SpaceBetween
                                ) {
                                    Text(vp.title, fontWeight = FontWeight.Bold, modifier = Modifier.weight(1f))
                                    Text("₹${vp.price.toInt()}", fontWeight = FontWeight.Bold, color = MaterialTheme.colorScheme.primary)
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
