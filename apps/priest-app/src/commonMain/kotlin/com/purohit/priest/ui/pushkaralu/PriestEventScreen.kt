package com.purohit.priest.ui.pushkaralu

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.unit.dp
import com.purohit.contracts.event.*
import com.purohit.contracts.i18n.AppLanguage
import com.purohit.contracts.i18n.models.EventStrings
import com.purohit.priest.location.DutyServiceController
import com.purohit.priest.location.rememberLocationAccess
import com.purohit.priest.network.PurohitApiClient
import kotlinx.coroutines.launch

/** Rupee text typed by the priest to paise; null when empty or invalid. */
internal fun rupeesToPaise(text: String): Long? {
    val t = text.trim()
    if (t.isEmpty()) return null
    val parts = t.split(".")
    if (parts.size > 2 || parts.any { p -> p.any { !it.isDigit() } } || parts[0].isEmpty()) return null
    val rupees = parts[0].toLongOrNull() ?: return null
    val paise = parts.getOrNull(1)?.padEnd(2, '0')?.take(2)?.toLongOrNull() ?: 0
    if (parts.getOrNull(1)?.length ?: 0 > 2) return null
    val total = rupees * 100 + paise
    return if (total in 1..10_000_000) total else null
}

internal fun paiseToRupeesText(p: Long): String = if (p % 100 == 0L) (p / 100).toString() else "${p / 100}.${(p % 100).toString().padStart(2, '0')}"

@OptIn(ExperimentalLayoutApi::class)
@Composable
fun PriestEventScreen(event: EventConfig?, language: AppLanguage, strings: EventStrings, hasActiveBooking: Boolean) {
    if (event == null || !event.isLive && event.status != EventStatus.SCHEDULED) {
        Box(Modifier.fillMaxSize().padding(24.dp), contentAlignment = Alignment.Center) { Text(strings.eventEndedNotice) }
        return
    }
    var participation by remember { mutableStateOf<Participation?>(null) }
    var message by remember { mutableStateOf<String?>(null) }
    var busy by remember { mutableStateOf(false) }
    var reload by remember { mutableStateOf(0) }
    val prices = remember { mutableStateMapOf<String, String>() }
    var selectedGhat by remember { mutableStateOf<String?>(null) }
    var confirmEnd by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()
    val location = rememberLocationAccess()

    LaunchedEffect(event.id, reload) {
        PurohitApiClient.fetchParticipation(event.id)
            .onSuccess { p ->
                participation = p
                prices.clear()
                p.services.forEach { prices[it.ritualId] = paiseToRupeesText(it.priceMinor) }
                selectedGhat = p.duty?.ghatId ?: selectedGhat ?: event.ghats.firstOrNull()?.id
            }
            .onFailure { message = strings.noPriestsAvailable }
    }

    fun run(block: suspend () -> Result<*>, ok: String? = null) {
        busy = true; message = null
        scope.launch {
            block().onSuccess { if (ok != null) message = ok; reload++ }.onFailure { e ->
                message = when (e.message) {
                    EventErrorCodes.NOT_APPROVED -> strings.dutyNeedsApproval
                    "event_not_open" -> strings.eventEndedNotice
                    else -> strings.retryAction
                }
            }
            busy = false
        }
    }

    val p = participation
    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(16.dp), verticalArrangement = Arrangement.spacedBy(16.dp)) {
        Text(strings.applyTitleTemplate.replace("{event}", event.displayName(language)), style = MaterialTheme.typography.headlineSmall,
            color = MaterialTheme.colorScheme.primary, modifier = Modifier.semantics { heading() })
        if (p == null) {
            CircularProgressIndicator()
            message?.let { Text(it, color = MaterialTheme.colorScheme.error) }
            return@Column
        }

        // Application status
        Card(colors = CardDefaults.cardColors(containerColor = Color.White)) {
            Column(Modifier.padding(16.dp).fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                val label = when (p.status) {
                    "SUBMITTED" -> strings.statusSubmitted
                    "CHANGES_REQUESTED" -> strings.statusChangesRequested
                    "APPROVED" -> strings.statusApproved
                    "REJECTED" -> strings.statusRejected
                    "SUSPENDED" -> strings.statusSuspended
                    else -> strings.statusNotApplied
                }
                Text(label, style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
                if (p.reviewerReason.isNotBlank()) {
                    Surface(color = MaterialTheme.colorScheme.secondaryContainer, shape = RoundedCornerShape(10.dp)) {
                        Column(Modifier.padding(12.dp).fillMaxWidth()) {
                            Text(strings.reviewerNoteLabel, fontWeight = FontWeight.Bold)
                            Text(p.reviewerReason)
                        }
                    }
                }
                if (p.canApply) Button(enabled = !busy, onClick = { run({ PurohitApiClient.applyForEvent(event.id) }) }, modifier = Modifier.fillMaxWidth().heightIn(min = 48.dp)) {
                    Text(if (p.status == "CHANGES_REQUESTED") strings.resubmitAction else strings.applyAction)
                }
            }
        }

        // Prices per ritual (frozen into each booking server-side)
        if (p.status in setOf("SUBMITTED", "CHANGES_REQUESTED", "APPROVED") && p.rituals.isNotEmpty()) {
            Card(colors = CardDefaults.cardColors(containerColor = Color.White)) {
                Column(Modifier.padding(16.dp).fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(10.dp)) {
                    Text(strings.yourPricesTitle, style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
                    p.rituals.forEach { r ->
                        val text = prices[r.id].orEmpty()
                        OutlinedTextField(
                            value = text, onValueChange = { prices[r.id] = it.filter { c -> c.isDigit() || c == '.' }.take(9) },
                            label = { Text("${r.displayName(language)} · ${r.durationMinutes} min") },
                            placeholder = { Text(strings.priceInRupees) }, prefix = { Text("₹") },
                            isError = text.isNotBlank() && rupeesToPaise(text) == null,
                            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal), singleLine = true, modifier = Modifier.fillMaxWidth()
                        )
                    }
                    val parsed = p.rituals.mapNotNull { r -> prices[r.id]?.takeIf { it.isNotBlank() }?.let { r.id to rupeesToPaise(it) } }
                    Button(
                        enabled = !busy && parsed.isNotEmpty() && parsed.all { it.second != null },
                        onClick = { run({ PurohitApiClient.setEventServices(event.id, SetServicesRequest(parsed.map { SetServicesRequest.ServicePrice(it.first, it.second!!) })) }, strings.pricesSaved) },
                        modifier = Modifier.fillMaxWidth().heightIn(min = 48.dp)
                    ) { Text(strings.savePrices) }
                }
            }
        }

        // Event card
        if (p.isApproved) {
            val card = p.card
            Surface(shape = RoundedCornerShape(16.dp), color = Color.White, border = BorderStroke(1.dp, MaterialTheme.colorScheme.surfaceVariant)) {
                Column(Modifier.fillMaxWidth()) {
                    Surface(color = MaterialTheme.colorScheme.primary, contentColor = MaterialTheme.colorScheme.onPrimary, shape = RoundedCornerShape(topStart = 16.dp, topEnd = 16.dp)) {
                        Row(Modifier.fillMaxWidth().padding(14.dp), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically) {
                            Text(strings.cardTitle, fontWeight = FontWeight.Bold)
                            if (card != null) Text(card.status, fontWeight = FontWeight.Bold)
                        }
                    }
                    Column(Modifier.padding(14.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                        if (card == null) Text(strings.cardPendingNotice) else {
                            Text(strings.issuedByPoojaVahini, fontWeight = FontWeight.SemiBold)
                            Text(PurohitApiClient.currentPriestProfile?.full_name.orEmpty(), style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
                            Text(card.eventName)
                            Text("${strings.cardIdLabel}: ${card.cardCode}", style = MaterialTheme.typography.bodyMedium)
                            Text("${card.startsAt.take(10)} – ${card.endsAt.take(10)}", style = MaterialTheme.typography.bodyMedium)
                            Text("${PurohitApiClient.baseUrl}/api/v1/cards/${card.cardCode}", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                        }
                        Text(strings.notGovtNotice, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                }
            }
        }

        // Duty at ghat
        Card(colors = CardDefaults.cardColors(containerColor = Color.White)) {
            Column(Modifier.padding(16.dp).fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(10.dp)) {
                Text(strings.dutyTitle, style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
                if (!p.isApproved || !event.isLive) {
                    Text(strings.dutyNeedsApproval, color = MaterialTheme.colorScheme.onSurfaceVariant)
                } else {
                    val duty = p.duty
                    FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                        event.ghats.filter { it.isActive }.forEach { g ->
                            FilterChip(selected = g.id == selectedGhat, onClick = { selectedGhat = g.id }, label = { Text(g.displayName(language)) }, modifier = Modifier.heightIn(min = 48.dp))
                        }
                    }
                    if (duty == null) {
                        Button(enabled = !busy && selectedGhat != null, modifier = Modifier.fillMaxWidth().heightIn(min = 48.dp), onClick = {
                            val ghat = selectedGhat ?: return@Button
                            run({
                                PurohitApiClient.setEventDuty(event.id, DutyRequest(ghat, "AVAILABLE")).onSuccess {
                                    // Server marked us online; send a first point now, then the foreground service posts every 2 minutes.
                                    location.request { r -> r.onSuccess { c -> scope.launch { PurohitApiClient.updateLocation(c.latitude, c.longitude, c.capturedAt, c.accuracyM) } } }
                                    DutyServiceController.start()
                                }
                            })
                        }) { Text(strings.startDuty) }
                    } else {
                        Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                            listOf("AVAILABLE" to strings.dutyAvailable, "PERFORMING" to strings.dutyPerforming, "BREAK" to strings.dutyBreak).forEach { (code, label) ->
                                FilterChip(selected = duty.dutyStatus == code, onClick = {
                                    run({ PurohitApiClient.setEventDuty(event.id, DutyRequest(selectedGhat ?: duty.ghatId, code)) })
                                }, label = { Text(label) }, modifier = Modifier.heightIn(min = 48.dp))
                            }
                        }
                        if (selectedGhat != null && selectedGhat != duty.ghatId) {
                            OutlinedButton(enabled = !busy, onClick = { run({ PurohitApiClient.setEventDuty(event.id, DutyRequest(selectedGhat!!, duty.dutyStatus)) }) },
                                modifier = Modifier.fillMaxWidth().heightIn(min = 48.dp)) { Text(strings.startDuty) }
                        }
                        OutlinedButton(enabled = !busy, onClick = { if (hasActiveBooking) confirmEnd = true else run({ PurohitApiClient.endEventDuty(event.id).onSuccess { DutyServiceController.stop() } }) },
                            modifier = Modifier.fillMaxWidth().heightIn(min = 48.dp)) { Text(strings.endDuty) }
                    }
                }
            }
        }
        message?.let { Text(it, fontWeight = FontWeight.SemiBold, color = MaterialTheme.colorScheme.primary) }
    }

    if (confirmEnd) AlertDialog(
        onDismissRequest = { confirmEnd = false },
        text = { Text(strings.endDutyWarning) },
        confirmButton = { TextButton(onClick = { confirmEnd = false; run({ PurohitApiClient.endEventDuty(event.id).onSuccess { DutyServiceController.stop() } }) }) { Text(strings.endDuty) } },
        dismissButton = { TextButton(onClick = { confirmEnd = false }) { Text(com.purohit.priest.LocalPriestStrings.current.common.cancelAction) } }
    )
}
