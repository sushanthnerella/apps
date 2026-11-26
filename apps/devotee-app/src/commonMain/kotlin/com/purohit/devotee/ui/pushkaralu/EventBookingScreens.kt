package com.purohit.devotee.ui.pushkaralu

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.selection.selectableGroup
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.semantics.Role
import androidx.compose.ui.semantics.contentDescription
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.purohit.contracts.event.*
import com.purohit.contracts.i18n.AppLanguage
import com.purohit.contracts.i18n.models.EventStrings
import com.purohit.contracts.idempotency.IdempotencyKeyGenerator
import com.purohit.contracts.money.formatMinorToRupees
import com.purohit.devotee.network.EventApiException
import com.purohit.devotee.network.PurohitApiClient
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.datetime.*

private val FreshBg = Color(0xFFDFF1E3)
private val FreshFg = Color(0xFF1B5E30)
private val DelayBg = Color(0xFFFFEBC2)
private val DelayFg = Color(0xFF6B4200)
internal val IST = TimeZone.of("Asia/Kolkata")

@Composable
private fun BackHeader(title: String, subtitle: String?, onBack: () -> Unit) {
    Row(Modifier.fillMaxWidth().padding(horizontal = 4.dp, vertical = 4.dp), verticalAlignment = Alignment.CenterVertically) {
        TextButton(onClick = onBack, modifier = Modifier.heightIn(min = 48.dp)) { Text("←", fontSize = 22.sp) }
        Column(Modifier.weight(1f)) {
            Text(title, style = MaterialTheme.typography.titleLarge, color = MaterialTheme.colorScheme.primary, modifier = Modifier.semantics { heading() })
            if (subtitle != null) Text(subtitle, style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
    }
}

@Composable
private fun Chip(text: String, bg: Color, fg: Color) {
    Surface(color = bg, contentColor = fg, shape = RoundedCornerShape(50)) {
        Text(text, style = MaterialTheme.typography.labelMedium, fontWeight = FontWeight.SemiBold, modifier = Modifier.padding(horizontal = 10.dp, vertical = 4.dp))
    }
}

// ---------------- D3: priests at a ghat ----------------

@Composable
fun GhatPriestsScreen(event: EventConfig, ghat: Ghat, language: AppLanguage, strings: EventStrings, onBack: () -> Unit, onRequest: (GhatPriest) -> Unit) {
    var priests by remember { mutableStateOf<List<GhatPriest>?>(null) }
    var failed by remember { mutableStateOf(false) }
    var reload by remember { mutableStateOf(0) }

    // Refresh every 20 s; back off to 60 s after errors. Stops when the screen leaves composition.
    LaunchedEffect(event.id, ghat.id, reload) {
        var wait = 20_000L
        while (true) {
            PurohitApiClient.fetchGhatPriests(event.id, ghat.id)
                .onSuccess { priests = it; failed = false; wait = 20_000L }
                .onFailure { failed = true; wait = (wait * 2).coerceAtMost(60_000L) }
            delay(wait)
        }
    }

    Column(Modifier.fillMaxSize()) {
        val list = priests
        BackHeader(ghat.displayName(language), list?.let { strings.availableNowTemplate.replace("{count}", it.size.toString()) }, onBack)
        when {
            list == null && failed -> Column(Modifier.fillMaxSize().padding(24.dp), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.Center) {
                Text(strings.noPriestsAvailable, textAlign = TextAlign.Center)
                Button(onClick = { reload++ }, modifier = Modifier.padding(top = 12.dp).heightIn(min = 48.dp)) { Text(strings.retryAction) }
            }
            list == null -> Box(Modifier.fillMaxSize(), contentAlignment = Alignment.Center) { CircularProgressIndicator() }
            list.isEmpty() -> Box(Modifier.fillMaxSize().padding(24.dp), contentAlignment = Alignment.Center) {
                Text(strings.noPriestsAvailable, textAlign = TextAlign.Center, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
            else -> LazyColumn(contentPadding = PaddingValues(16.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
                items(list, key = { it.priestId }) { p ->
                    Card(colors = CardDefaults.cardColors(containerColor = Color.White), shape = RoundedCornerShape(14.dp)) {
                        Column(Modifier.padding(12.dp), verticalArrangement = Arrangement.spacedBy(10.dp)) {
                            Row(horizontalArrangement = Arrangement.spacedBy(12.dp), verticalAlignment = Alignment.CenterVertically) {
                                Surface(shape = CircleShape, color = MaterialTheme.colorScheme.primaryContainer, contentColor = MaterialTheme.colorScheme.onPrimaryContainer, modifier = Modifier.size(48.dp)) {
                                    Box(contentAlignment = Alignment.Center) { Text(p.fullName.split(" ").filter { it.isNotBlank() }.take(2).joinToString("") { it.take(1) }, fontWeight = FontWeight.Bold) }
                                }
                                Column(Modifier.weight(1f)) {
                                    Text(p.fullName, style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
                                    Text("★ ${(p.ratingAvg * 10).toInt() / 10.0}", style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
                                    Text(strings.verifiedBadge, style = MaterialTheme.typography.labelMedium, color = MaterialTheme.colorScheme.secondary, fontWeight = FontWeight.SemiBold)
                                }
                            }
                            FlowRowCompat {
                                if (p.freshness == Freshness.FRESH) Chip(strings.updatedAgoTemplate.replace("{min}", p.locationAgeMinutes.toString()), FreshBg, FreshFg)
                                else Chip(strings.locationDelayedTemplate.replace("{min}", p.locationAgeMinutes.toString()), DelayBg, DelayFg)
                                p.lowestPriceMinor?.let { Text(strings.fromPriceTemplate.replace("{price}", formatMinorToRupees(it)), fontWeight = FontWeight.SemiBold) }
                            }
                            if (event.acceptsBookings) Button(onClick = { onRequest(p) }, modifier = Modifier.fillMaxWidth().heightIn(min = 48.dp)) { Text(strings.requestAction) }
                        }
                    }
                }
                if (!event.acceptsBookings) item { Text(strings.bookingsPaused, fontWeight = FontWeight.Bold) }
            }
        }
    }
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
private fun FlowRowCompat(content: @Composable () -> Unit) {
    FlowRow(horizontalArrangement = Arrangement.spacedBy(10.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) { content() }
}

// ---------------- D4: booking request ----------------

/** Hourly start times on a given IST date, at least 30 minutes ahead and inside the event window. */
internal fun scheduleSlots(event: EventConfig, date: LocalDate, now: Instant): List<Instant> {
    val start = runCatching { Instant.parse(event.startsAt) }.getOrNull() ?: return emptyList()
    val end = runCatching { Instant.parse(event.endsAt) }.getOrNull() ?: return emptyList()
    return (5..20).map { LocalDateTime(date, LocalTime(it, 0)).toInstant(IST) }
        .filter { it >= now.plus(30, DateTimeUnit.MINUTE) && it >= start && it <= end }
}

internal fun scheduleDays(event: EventConfig, now: Instant): List<LocalDate> {
    val end = runCatching { Instant.parse(event.endsAt).toLocalDateTime(IST).date }.getOrNull() ?: return emptyList()
    val start = runCatching { Instant.parse(event.startsAt).toLocalDateTime(IST).date }.getOrNull() ?: return emptyList()
    var d = maxOf(start, now.toLocalDateTime(IST).date)
    val out = mutableListOf<LocalDate>()
    while (d <= end && out.size < 5) {
        if (scheduleSlots(event, d, now).isNotEmpty()) out += d
        d = d.plus(1, DateTimeUnit.DAY)
    }
    return out
}

@OptIn(ExperimentalLayoutApi::class)
@Composable
fun EventRequestScreen(
    event: EventConfig, ghat: Ghat, priest: GhatPriest, language: AppLanguage, strings: EventStrings,
    onBack: () -> Unit, onBooked: (EventBookingResponse) -> Unit
) {
    var serviceId by remember { mutableStateOf(priest.services.firstOrNull()?.id) }
    var scheduled by remember { mutableStateOf(false) }
    var day by remember { mutableStateOf<LocalDate?>(null) }
    var slot by remember { mutableStateOf<Instant?>(null) }
    var family by remember { mutableStateOf(1) }
    var busy by remember { mutableStateOf(false) }
    var error by remember { mutableStateOf<String?>(null) }
    // One key per intended request so a retried tap never creates two bookings.
    var idempotencyKey by remember { mutableStateOf(IdempotencyKeyGenerator.generate()) }
    val scope = rememberCoroutineScope()
    val service = priest.services.firstOrNull { it.id == serviceId }
    val now = remember { Clock.System.now() }
    val days = remember(event) { scheduleDays(event, now) }

    Column(Modifier.fillMaxSize()) {
        BackHeader(priest.fullName, ghat.displayName(language), onBack)
        Column(Modifier.weight(1f).verticalScroll(rememberScrollState()).padding(16.dp), verticalArrangement = Arrangement.spacedBy(14.dp)) {
            Text(strings.ritualLabel, style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
            Column(Modifier.selectableGroup(), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                priest.services.forEach { s ->
                    Surface(
                        shape = RoundedCornerShape(12.dp), color = Color.White,
                        border = androidx.compose.foundation.BorderStroke(2.dp, if (s.id == serviceId) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.surfaceVariant),
                        modifier = Modifier.fillMaxWidth().selectable(selected = s.id == serviceId, role = Role.RadioButton) { serviceId = s.id; idempotencyKey = IdempotencyKeyGenerator.generate() }
                    ) {
                        Row(Modifier.padding(12.dp).heightIn(min = 48.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                            RadioButton(selected = s.id == serviceId, onClick = null)
                            Column(Modifier.weight(1f)) {
                                Text(s.displayName(language), fontWeight = FontWeight.SemiBold)
                                Text("${s.durationMinutes} min", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                            }
                            Text(formatMinorToRupees(s.priceMinor), fontWeight = FontWeight.Bold)
                        }
                    }
                }
            }

            Text(strings.whenLabel, style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
            Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                FilterChip(selected = !scheduled, onClick = { scheduled = false; idempotencyKey = IdempotencyKeyGenerator.generate() }, label = { Text(strings.whenNow) }, modifier = Modifier.heightIn(min = 48.dp))
                if (days.isNotEmpty()) FilterChip(selected = scheduled, onClick = { scheduled = true; day = day ?: days.first(); idempotencyKey = IdempotencyKeyGenerator.generate() }, label = { Text(strings.whenSchedule) }, modifier = Modifier.heightIn(min = 48.dp))
            }
            if (scheduled) {
                FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                    days.forEach { d ->
                        FilterChip(selected = d == day, onClick = { day = d; slot = null }, label = { Text("${d.dayOfMonth} ${d.month.name.take(3).lowercase().replaceFirstChar { it.uppercase() }}") }, modifier = Modifier.heightIn(min = 48.dp))
                    }
                }
                day?.let { d ->
                    FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalArrangement = Arrangement.spacedBy(6.dp)) {
                        scheduleSlots(event, d, now).forEach { t ->
                            val local = t.toLocalDateTime(IST)
                            val h = local.hour
                            val label = "${if (h % 12 == 0) 12 else h % 12}:00 ${if (h < 12) "AM" else "PM"}"
                            FilterChip(selected = t == slot, onClick = { slot = t; idempotencyKey = IdempotencyKeyGenerator.generate() }, label = { Text(label) }, modifier = Modifier.heightIn(min = 48.dp))
                        }
                    }
                }
            }

            Text(strings.familySizeLabel, style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
            Row(verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp)) {
                OutlinedButton(onClick = { if (family > 1) family-- }, modifier = Modifier.size(48.dp).semantics { contentDescription = "-" }, contentPadding = PaddingValues(0.dp)) { Text("−") }
                Text(family.toString(), style = MaterialTheme.typography.titleLarge)
                OutlinedButton(onClick = { if (family < 50) family++ }, modifier = Modifier.size(48.dp).semantics { contentDescription = "+" }, contentPadding = PaddingValues(0.dp)) { Text("+") }
            }

            if (service != null) {
                Surface(color = MaterialTheme.colorScheme.surfaceVariant, shape = RoundedCornerShape(12.dp)) {
                    Row(Modifier.fillMaxWidth().padding(14.dp), horizontalArrangement = Arrangement.SpaceBetween) {
                        Text(strings.dakshinaLabel, fontWeight = FontWeight.Bold)
                        Text(formatMinorToRupees(service.priceMinor), fontWeight = FontWeight.Bold)
                    }
                }
            }
            Text(strings.payAtGhatNotice, style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
            Text(strings.requestExpiryNotice, style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
            error?.let { Text(it, color = MaterialTheme.colorScheme.error, fontWeight = FontWeight.SemiBold) }
        }
        Surface(tonalElevation = 2.dp) {
            Button(
                enabled = !busy && service != null && (!scheduled || slot != null),
                onClick = {
                    val s = service ?: return@Button
                    busy = true; error = null
                    scope.launch {
                        val req = CreateEventBookingRequest(
                            priestId = priest.priestId, ghatId = ghat.id, serviceId = s.id,
                            `when` = if (scheduled) "SCHEDULED" else "NOW", scheduledAt = if (scheduled) slot?.toString() else null, familySize = family
                        )
                        PurohitApiClient.createEventBooking(event.id, req, idempotencyKey)
                            .onSuccess { onBooked(it) }
                            .onFailure { e ->
                                error = when ((e as? EventApiException)?.code) {
                                    EventErrorCodes.BOOKING_DISABLED -> strings.bookingsPaused
                                    EventErrorCodes.PRIEST_UNAVAILABLE, EventErrorCodes.SERVICE_UNAVAILABLE -> strings.errorPriestUnavailable
                                    EventErrorCodes.PRIEST_BUSY -> strings.errorPriestBusyAtTime
                                    EventErrorCodes.INVALID_SCHEDULE -> strings.errorInvalidSchedule
                                    else -> strings.noPriestsAvailable
                                }
                            }
                        busy = false
                    }
                },
                modifier = Modifier.fillMaxWidth().padding(16.dp).heightIn(min = 52.dp)
            ) { if (busy) CircularProgressIndicator(Modifier.size(20.dp), strokeWidth = 2.dp) else Text(strings.sendRequest) }
        }
    }
}

// ---------------- D5: waiting, confirmed, booking code ----------------

internal enum class EventBookingPhase { WAITING, CONFIRMED, STARTED, COMPLETED, EXPIRED, CANCELLED }

internal fun eventBookingPhase(status: String, expiresAt: Instant?, now: Instant): EventBookingPhase = when (status) {
    "REQUESTED", "DISPATCHED" -> if (expiresAt != null && now > expiresAt) EventBookingPhase.EXPIRED else EventBookingPhase.WAITING
    "ACCEPTED", "IN_TRANSIT", "ARRIVED" -> EventBookingPhase.CONFIRMED
    "POOJA_STARTED" -> EventBookingPhase.STARTED
    "COMPLETED" -> EventBookingPhase.COMPLETED
    // The server expires unanswered requests as CANCELLED; past the deadline that means expired.
    "CANCELLED" -> if (expiresAt != null && now > expiresAt) EventBookingPhase.EXPIRED else EventBookingPhase.CANCELLED
    else -> EventBookingPhase.CANCELLED
}

@Composable
fun EventBookingScreen(
    booking: EventBookingResponse, ghatName: String, priestName: String, strings: EventStrings,
    onChat: (roomId: String) -> Unit, onDone: () -> Unit
) {
    val expiresAt = remember(booking.requestExpiresAt) { runCatching { Instant.parse(booking.requestExpiresAt) }.getOrNull() }
    var status by remember { mutableStateOf(booking.status) }
    var title by remember { mutableStateOf("") }
    var scheduledText by remember { mutableStateOf("") }
    var now by remember { mutableStateOf(Clock.System.now()) }
    var confirmCancel by remember { mutableStateOf(false) }
    var isCancelling by remember { mutableStateOf(false) }
    var cancelError by remember { mutableStateOf<String?>(null) }
    val language = com.purohit.devotee.LocalAppLanguage.current
    fun tr(en: String, te: String, hi: String) = when (language) { AppLanguage.TELUGU -> te; AppLanguage.HINDI -> hi; else -> en }
    val scope = rememberCoroutineScope()
    val phase = eventBookingPhase(status, expiresAt, now)

    LaunchedEffect(booking.bookingId) {
        var wait = 4_000L
        while (true) {
            PurohitApiClient.fetchBookingDetails(booking.bookingId)
                .onSuccess { status = it.status; title = it.pooja_title; scheduledText = listOf(it.scheduled_date, it.slot_time).filter { v -> v.isNotBlank() }.joinToString(" • "); wait = 4_000L }
                .onFailure { wait = (wait * 2).coerceAtMost(10_000L) }
            if (eventBookingPhase(status, expiresAt, Clock.System.now()) in setOf(EventBookingPhase.COMPLETED, EventBookingPhase.CANCELLED, EventBookingPhase.EXPIRED)) break
            delay(wait)
        }
    }
    LaunchedEffect(Unit) { while (true) { now = Clock.System.now(); delay(1_000) } }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(16.dp), verticalArrangement = Arrangement.spacedBy(14.dp)) {
        val headline = when (phase) {
            EventBookingPhase.WAITING -> strings.waitingForPriest
            EventBookingPhase.CONFIRMED -> strings.bookingConfirmed
            EventBookingPhase.STARTED -> com.purohit.devotee.LocalAppStrings.current.common.bookingInProgress
            EventBookingPhase.COMPLETED -> strings.ritualCompleted
            EventBookingPhase.EXPIRED -> strings.requestExpired
            EventBookingPhase.CANCELLED -> strings.priestDeclined
        }
        Text(headline, style = MaterialTheme.typography.headlineSmall, color = MaterialTheme.colorScheme.primary, modifier = Modifier.semantics { heading() })

        val stepIndex = when (phase) { EventBookingPhase.WAITING -> 0; EventBookingPhase.CONFIRMED -> 1; EventBookingPhase.STARTED -> 2; EventBookingPhase.COMPLETED -> 3; else -> -1 }
        if (stepIndex >= 0) Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
            listOf(strings.stepRequested, strings.stepAccepted, strings.stepStarted, strings.stepCompleted).forEachIndexed { i, label ->
                Column(Modifier.weight(1f), horizontalAlignment = Alignment.CenterHorizontally) {
                    val done = i <= stepIndex
                    Surface(shape = CircleShape, color = if (done) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.surfaceVariant,
                        contentColor = if (done) MaterialTheme.colorScheme.onPrimary else MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.size(28.dp)) {
                        Box(contentAlignment = Alignment.Center) { Text(if (done) "✓" else "${i + 1}", fontWeight = FontWeight.Bold) }
                    }
                    Text(label, style = MaterialTheme.typography.labelMedium, textAlign = TextAlign.Center)
                }
            }
        }

        if (phase == EventBookingPhase.WAITING && expiresAt != null) {
            val left = (expiresAt - now).inWholeSeconds.coerceAtLeast(0)
            val time = if (left >= 3600) "${left / 3600}h ${(left % 3600) / 60}m" else "${left / 60}:${(left % 60).toString().padStart(2, '0')}"
            Text(strings.expiresInTemplate.replace("{time}", time), style = MaterialTheme.typography.titleMedium)
        }

        if (phase in setOf(EventBookingPhase.WAITING, EventBookingPhase.CONFIRMED, EventBookingPhase.STARTED)) {
            Surface(shape = RoundedCornerShape(18.dp), color = Color.White, border = androidx.compose.foundation.BorderStroke(2.dp, MaterialTheme.colorScheme.primary)) {
                Column(Modifier.fillMaxWidth().padding(20.dp), horizontalAlignment = Alignment.CenterHorizontally, verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text(strings.bookingCodeTitle, color = MaterialTheme.colorScheme.onSurfaceVariant, fontWeight = FontWeight.SemiBold)
                    Text(booking.bookingCode, style = MaterialTheme.typography.displayMedium, fontWeight = FontWeight.Bold, letterSpacing = 12.sp, color = MaterialTheme.colorScheme.primary,
                        modifier = Modifier.semantics { contentDescription = booking.bookingCode.toList().joinToString(" ") })
                    Text(strings.bookingCodeHint, textAlign = TextAlign.Center)
                }
            }
        }

        Surface(color = MaterialTheme.colorScheme.surfaceVariant, shape = RoundedCornerShape(14.dp)) {
            Column(Modifier.fillMaxWidth().padding(14.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                if (title.isNotBlank()) Text(title, fontWeight = FontWeight.Bold)
                if (scheduledText.isNotBlank()) Text("📅 $scheduledText", fontWeight = FontWeight.SemiBold)
                Text(priestName)
                Text("${strings.meetAt}: $ghatName", color = MaterialTheme.colorScheme.onSurfaceVariant)
                Text("${strings.dakshinaLabel}: ${formatMinorToRupees(booking.totalAmountMinor)}", color = MaterialTheme.colorScheme.onSurfaceVariant)
                Text(strings.payAtGhatNotice, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
        }

        if (phase in setOf(EventBookingPhase.CONFIRMED, EventBookingPhase.STARTED) && booking.chatRoomId.isNotBlank()) {
            Button(onClick = { onChat(booking.chatRoomId) }, modifier = Modifier.fillMaxWidth().heightIn(min = 48.dp)) { Text(strings.chatAction) }
        }
        if (phase == EventBookingPhase.WAITING || phase == EventBookingPhase.CONFIRMED) {
            TextButton(onClick = { confirmCancel = true }, enabled = !isCancelling, modifier = Modifier.fillMaxWidth().heightIn(min = 48.dp)) { Text(strings.cancelRequest) }
        }
        cancelError?.let { Text(it, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodyMedium) }
        if (phase in setOf(EventBookingPhase.COMPLETED, EventBookingPhase.CANCELLED, EventBookingPhase.EXPIRED)) {
            Button(onClick = onDone, modifier = Modifier.fillMaxWidth().heightIn(min = 48.dp)) { Text(strings.chooseGhatTitle) }
        }
    }

    if (confirmCancel) AlertDialog(
        onDismissRequest = { confirmCancel = false },
        title = { Text(strings.cancelConfirm) },
        confirmButton = {
            TextButton(onClick = {
                confirmCancel = false
                isCancelling = true
                cancelError = null
                scope.launch {
                    try {
                        PurohitApiClient.updateBookingStatus(booking.bookingId, "CANCELLED")
                            .onSuccess { status = "CANCELLED" }
                            .onFailure { cancelError = tr("Could not cancel the booking. Please try again.", "బుకింగ్ రద్దు కాలేదు. మళ్లీ ప్రయత్నించండి.", "बुकिंग रद्द नहीं हुई। फिर प्रयास करें।") }
                    } finally { isCancelling = false }
                }
            }) { Text(strings.cancelRequest) }
        },
        dismissButton = { TextButton(onClick = { confirmCancel = false }) { Text(com.purohit.devotee.LocalAppStrings.current.common.closeAction) } }
    )
}
