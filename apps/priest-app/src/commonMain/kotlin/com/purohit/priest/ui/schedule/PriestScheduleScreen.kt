package com.purohit.priest.ui.schedule

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.purohit.contracts.i18n.AppLanguage
import com.purohit.contracts.i18n.StringContract
import com.purohit.priest.LocalPriestLanguage
import com.purohit.priest.network.PriestSlotPayload
import com.purohit.priest.network.PurohitApiClient
import com.purohit.priest.network.SlotApiException
import kotlinx.coroutines.launch
import kotlinx.datetime.Clock
import kotlinx.datetime.Instant
import kotlinx.datetime.LocalDate
import kotlinx.datetime.LocalDateTime
import kotlinx.datetime.TimeZone
import kotlinx.datetime.atStartOfDayIn
import kotlinx.datetime.toLocalDateTime

private val IST = TimeZone.of("Asia/Kolkata")
private fun nowIst(): LocalDateTime = Clock.System.now().toLocalDateTime(IST)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PriestScheduleScreen(priestId: String = "", strings: StringContract) {
    val language = LocalPriestLanguage.current
    fun tr(en: String, te: String, hi: String) = when (language) { AppLanguage.TELUGU -> te; AppLanguage.HINDI -> hi; else -> en }
    var date by remember { mutableStateOf(nowIst().date) }
    var startMin by remember { mutableStateOf<Int?>(null) }
    var endMin by remember { mutableStateOf<Int?>(null) }
    var slots by remember { mutableStateOf<List<PriestSlotPayload>>(emptyList()) }
    var message by remember { mutableStateOf<String?>(null) }
    var isError by remember { mutableStateOf(false) }
    var loading by remember { mutableStateOf(false) }
    var pickingStart by remember { mutableStateOf(false) }
    var pickingEnd by remember { mutableStateOf(false) }
    var confirmRemove by remember { mutableStateOf<PriestSlotPayload?>(null) }
    val scope = rememberCoroutineScope()

    fun show(text: String, error: Boolean) { message = text; isError = error }
    fun problemText(p: SlotProblem) = when (p) {
        SlotProblem.PAST_DATE -> tr("This date has already passed. Choose today or a later date.", "ఈ తేదీ గడిచిపోయింది. ఈరోజు లేదా తర్వాతి తేదీని ఎంచుకోండి.", "यह तारीख बीत चुकी है। आज या आगे की तारीख चुनें।")
        SlotProblem.PAST_TIME -> tr("This start time has already passed today. Choose a later time.", "ఈ ప్రారంభ సమయం ఈరోజు గడిచిపోయింది. తర్వాతి సమయం ఎంచుకోండి.", "आज यह आरंभ समय बीत चुका है। बाद का समय चुनें।")
        SlotProblem.END_NOT_AFTER_START -> tr("End time must be after start time.", "ముగింపు సమయం ప్రారంభ సమయం తర్వాత ఉండాలి.", "समाप्ति समय आरंभ समय के बाद होना चाहिए।")
        SlotProblem.OVERLAP -> tr("This time overlaps a slot you already published. Choose a different time.", "ఈ సమయం మీరు ఇప్పటికే ప్రచురించిన సమయంతో కలుస్తోంది. వేరే సమయం ఎంచుకోండి.", "यह समय आपके पहले से प्रकाशित समय से टकराता है। अलग समय चुनें।")
    }
    fun serverText(e: Throwable, fallback: String) = when ((e as? SlotApiException)?.code) {
        "SLOT_OVERLAP" -> problemText(SlotProblem.OVERLAP)
        "SLOT_IN_PAST" -> problemText(SlotProblem.PAST_TIME)
        "SLOT_INVALID" -> problemText(SlotProblem.END_NOT_AFTER_START)
        "SLOT_NOT_REMOVABLE" -> tr("This slot is held or booked by a devotee and cannot be removed.", "ఈ సమయాన్ని భక్తులు నిలిపారు లేదా బుక్ చేశారు, తొలగించలేరు.", "यह समय किसी भक्त ने आरक्षित या बुक किया है, इसे हटाया नहीं जा सकता।")
        else -> fallback
    }
    suspend fun refreshNow(requestedDate: LocalDate) {
        PurohitApiClient.fetchPriestSlots(priestId, requestedDate.toString())
            .onSuccess { if (requestedDate == date) slots = it }
            .onFailure { if (requestedDate == date) { slots = emptyList(); show(tr("Could not load availability. Please retry.", "అందుబాటు పొందలేకపోయాము. మళ్లీ ప్రయత్నించండి.", "उपलब्धता नहीं मिली। फिर से प्रयास करें।"), true) } }
    }
    fun reload() { val d = date; loading = true; scope.launch { try { refreshNow(d) } finally { loading = false } } }
    LaunchedEffect(priestId) { reload() }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
        Text(tr("Availability", "అందుబాటు", "उपलब्धता"), style = MaterialTheme.typography.headlineMedium, color = MaterialTheme.colorScheme.primary)
        Text(tr("Publish the times when you're available so devotees can book you.", "భక్తులు మిమ్మల్ని బుక్ చేసుకునేలా మీరు అందుబాటులో ఉండే సమయాలను ప్రచురించండి.", "जिन समयों पर आप उपलब्ध हैं उन्हें प्रकाशित करें ताकि भक्त आपको बुक कर सकें।"), style = MaterialTheme.typography.bodyMedium)
        Text(tr("Each time block you publish is one bookable slot, for example 9:00 AM – 10:00 AM. All times are IST.", "మీరు ప్రచురించే ప్రతి సమయ భాగం ఒక బుకింగ్ సమయం, ఉదా. 9:00 AM – 10:00 AM. అన్ని సమయాలు IST.", "आपका प्रकाशित हर समय-खंड एक बुक करने योग्य स्लॉट है, जैसे 9:00 AM – 10:00 AM। सभी समय IST में हैं।"), style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)

        IndiaDatePicker(value = date, language = language, enabled = !loading, onDateSelected = {
            date = it; slots = emptyList(); message = null; reload()
        })
        Row(horizontalArrangement = Arrangement.spacedBy(8.dp), modifier = Modifier.fillMaxWidth()) {
            OutlinedButton(onClick = { pickingStart = true }, enabled = !loading, modifier = Modifier.weight(1f)) {
                Text(startMin?.let { "${tr("Start", "ప్రారంభం", "आरंभ")}: ${format12h(it)}" } ?: tr("Start time", "ప్రారంభ సమయం", "आरंभ समय"))
            }
            OutlinedButton(onClick = { pickingEnd = true }, enabled = !loading, modifier = Modifier.weight(1f)) {
                Text(endMin?.let { "${tr("End", "ముగింపు", "समाप्ति")}: ${format12h(it)}" } ?: tr("End time", "ముగింపు సమయం", "समाप्ति समय"))
            }
        }
        Button(onClick = {
            val s = startMin; val e = endMin
            if (s == null || e == null) { show(tr("Choose a start time and an end time.", "ప్రారంభ, ముగింపు సమయాలు ఎంచుకోండి.", "आरंभ और समाप्ति समय चुनें।"), true); return@Button }
            validateSlot(date, s, e, nowIst(), slots)?.let { show(problemText(it), true); return@Button }
            val d = date
            scope.publishThenRefresh(
                publish = { PurohitApiClient.createPriestSlot(priestId, d.toString(), hhmm(s), hhmm(e)).map { Unit } },
                refresh = {
                    refreshNow(d)
                    startMin = null; endMin = null
                    show(tr("Slot published: ${format12h(s)} – ${format12h(e)} IST", "సమయం ప్రచురించబడింది: ${format12h(s)} – ${format12h(e)} IST", "समय प्रकाशित हुआ: ${format12h(s)} – ${format12h(e)} IST"), false)
                },
                setLoading = { loading = it },
                onFailure = { err -> show(serverText(err, tr("Could not publish the slot. Please try again.", "సమయాన్ని ప్రచురించలేకపోయాము. మళ్లీ ప్రయత్నించండి.", "समय प्रकाशित नहीं हो सका। फिर से प्रयास करें।")), true) }
            )
        }, enabled = !loading, modifier = Modifier.fillMaxWidth()) { Text(tr("Publish slot", "సమయం ప్రచురించండి", "समय प्रकाशित करें")) }

        message?.let { Text(it, style = MaterialTheme.typography.bodyMedium, color = if (isError) MaterialTheme.colorScheme.error else MaterialTheme.colorScheme.primary) }
        if (loading) LinearProgressIndicator(Modifier.fillMaxWidth())

        Text(tr("Your slots on this date", "ఈ తేదీన మీ సమయాలు", "इस तारीख पर आपके समय"), style = MaterialTheme.typography.titleMedium)
        if (slots.isEmpty() && !loading) Text(tr("No slots published for this date yet.", "ఈ తేదీకి ఇంకా సమయాలు ప్రచురించలేదు.", "इस तारीख के लिए अभी कोई समय प्रकाशित नहीं है।"), style = MaterialTheme.typography.bodySmall)
        slots.forEach { slot ->
            val (status, color) = when {
                slot.is_booked -> tr("Booked", "బుక్ చేయబడింది", "बुक किया गया") to MaterialTheme.colorScheme.tertiary
                slot.is_held -> tr("Held", "తాత్కాలికంగా నిలిపారు", "अस्थायी रूप से आरक्षित") to MaterialTheme.colorScheme.secondary
                else -> tr("Available", "అందుబాటులో ఉంది", "उपलब्ध") to MaterialTheme.colorScheme.primary
            }
            Card(Modifier.fillMaxWidth(), colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant)) {
                Row(Modifier.padding(horizontal = 16.dp, vertical = 8.dp).fillMaxWidth(), verticalAlignment = Alignment.CenterVertically) {
                    Column(Modifier.weight(1f)) {
                        Text("${format12h(minutesOf(slot.start_time))} – ${format12h(minutesOf(slot.end_time))} IST", style = MaterialTheme.typography.bodyLarge)
                        Text(status, style = MaterialTheme.typography.labelMedium, color = color)
                    }
                    if (slot.isRemovable()) TextButton(onClick = { confirmRemove = slot }, enabled = !loading) {
                        Text(tr("Remove", "తొలగించు", "हटाएं"), color = MaterialTheme.colorScheme.error)
                    }
                }
            }
        }
    }

    if (pickingStart || pickingEnd) {
        val initial = (if (pickingStart) startMin else endMin) ?: (if (pickingStart) 9 * 60 else (startMin?.plus(60) ?: 10 * 60))
        val state = rememberTimePickerState(initialHour = (initial / 60) % 24, initialMinute = initial % 60, is24Hour = false)
        AlertDialog(
            onDismissRequest = { pickingStart = false; pickingEnd = false },
            title = { Text(if (pickingStart) tr("Start time (IST)", "ప్రారంభ సమయం (IST)", "आरंभ समय (IST)") else tr("End time (IST)", "ముగింపు సమయం (IST)", "समाप्ति समय (IST)")) },
            text = { TimePicker(state = state) },
            confirmButton = { TextButton(onClick = {
                val picked = state.hour * 60 + state.minute
                if (pickingStart) { startMin = picked; if (endMin == null || endMin!! <= picked) endMin = (picked + 60).coerceAtMost(23 * 60 + 59) } else endMin = picked
                message = null; pickingStart = false; pickingEnd = false
            }) { Text(tr("OK", "సరే", "ठीक है")) } },
            dismissButton = { TextButton(onClick = { pickingStart = false; pickingEnd = false }) { Text(tr("Cancel", "రద్దు", "रद्द करें")) } }
        )
    }

    confirmRemove?.let { slot ->
        val label = "${format12h(minutesOf(slot.start_time))} – ${format12h(minutesOf(slot.end_time))}"
        AlertDialog(
            onDismissRequest = { confirmRemove = null },
            title = { Text(tr("Remove this slot?", "ఈ సమయాన్ని తొలగించాలా?", "यह समय हटाएं?")) },
            text = { Text(tr("$label IST will no longer be bookable by devotees.", "$label IST ఇకపై భక్తులు బుక్ చేయలేరు.", "$label IST अब भक्त बुक नहीं कर पाएंगे।")) },
            confirmButton = { TextButton(onClick = {
                confirmRemove = null
                val d = date
                scope.publishThenRefresh(
                    publish = { PurohitApiClient.deletePriestSlot(slot.id) },
                    refresh = { refreshNow(d); show(tr("Slot removed.", "సమయం తొలగించబడింది.", "समय हटाया गया।"), false) },
                    setLoading = { loading = it },
                    onFailure = { err ->
                        show(serverText(err, tr("Could not remove the slot. Please try again.", "సమయాన్ని తొలగించలేకపోయాము. మళ్లీ ప్రయత్నించండి.", "समय हटाया नहीं जा सका। फिर से प्रयास करें।")), true)
                        scope.launch { refreshNow(d) }
                    }
                )
            }) { Text(tr("Remove", "తొలగించు", "हटाएं"), color = MaterialTheme.colorScheme.error) } },
            dismissButton = { TextButton(onClick = { confirmRemove = null }) { Text(tr("Keep", "ఉంచు", "रखें")) } }
        )
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun IndiaDatePicker(value: LocalDate, language: AppLanguage, enabled: Boolean, onDateSelected: (LocalDate) -> Unit) {
    var showPicker by remember { mutableStateOf(false) }
    val today = remember { nowIst().date }
    val minimumUtc = remember(today) { today.atStartOfDayIn(TimeZone.UTC).toEpochMilliseconds() }
    val label = when (language) { AppLanguage.TELUGU -> "తేదీ"; AppLanguage.HINDI -> "तारीख"; else -> "Date" }
    val dd = value.dayOfMonth.toString().padStart(2, '0'); val mm = value.monthNumber.toString().padStart(2, '0')
    OutlinedButton(onClick = { showPicker = true }, enabled = enabled, modifier = Modifier.fillMaxWidth()) { Text("$label: $dd/$mm/${value.year}") }
    if (showPicker) {
        val picker = rememberDatePickerState(initialSelectedDateMillis = value.atStartOfDayIn(TimeZone.UTC).toEpochMilliseconds(),
            selectableDates = remember(minimumUtc) { object : SelectableDates { override fun isSelectableDate(utcTimeMillis: Long) = utcTimeMillis >= minimumUtc } })
        DatePickerDialog(onDismissRequest = { showPicker = false }, confirmButton = { TextButton(onClick = {
            picker.selectedDateMillis?.let { onDateSelected(Instant.fromEpochMilliseconds(it).toLocalDateTime(TimeZone.UTC).date) }
            showPicker = false
        }) { Text(when (language) { AppLanguage.TELUGU -> "సరే"; AppLanguage.HINDI -> "ठीक है"; else -> "OK" }) } },
            dismissButton = { TextButton(onClick = { showPicker = false }) { Text(when (language) { AppLanguage.TELUGU -> "రద్దు"; AppLanguage.HINDI -> "रद्द करें"; else -> "Cancel" }) } }) { DatePicker(state = picker, showModeToggle = false) }
    }
}
