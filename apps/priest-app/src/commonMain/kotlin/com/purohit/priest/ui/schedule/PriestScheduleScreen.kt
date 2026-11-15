package com.purohit.priest.ui.schedule

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import com.purohit.contracts.i18n.AppLanguage
import com.purohit.priest.LocalPriestLanguage
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.purohit.contracts.i18n.StringContract
import com.purohit.priest.network.PriestSlotPayload
import com.purohit.priest.network.PurohitApiClient
import kotlinx.coroutines.launch
import kotlinx.datetime.Clock
import kotlinx.datetime.Instant
import kotlinx.datetime.LocalDate
import kotlinx.datetime.TimeZone
import kotlinx.datetime.atStartOfDayIn
import kotlinx.datetime.toLocalDateTime

@Composable
fun PriestScheduleScreen(priestId: String = "", strings: StringContract) {
    val language = LocalPriestLanguage.current
    fun tr(en: String, te: String, hi: String) = when (language) { AppLanguage.TELUGU -> te; AppLanguage.HINDI -> hi; else -> en }
    var date by remember { mutableStateOf("") }
    var start by remember { mutableStateOf("") }
    var end by remember { mutableStateOf("") }
    var slots by remember { mutableStateOf<List<PriestSlotPayload>>(emptyList()) }
    var message by remember(language) { mutableStateOf(tr("Enter a date to view or publish availability.", "అందుబాటు చూడడానికి లేదా ప్రచురించడానికి తేదీని ఎంచుకోండి.", "उपलब्धता देखने या प्रकाशित करने के लिए तारीख चुनें।")) }
    var loading by remember { mutableStateOf(false) }
    val scope = rememberCoroutineScope()
    val validDate = Regex("\\d{4}-\\d{2}-\\d{2}")
    val validTime = Regex("(?:[01]\\d|2[0-3]):[0-5]\\d")
    suspend fun refreshNow(requestedDate: String) {
        PurohitApiClient.fetchPriestSlots(priestId, requestedDate).onSuccess {
            slots = it
            message = if (it.isEmpty()) tr("No slots published", "సమయాలు ప్రచురించలేదు", "कोई समय प्रकाशित नहीं है") else tr("Published availability", "ప్రచురించిన అందుబాటు", "प्रकाशित उपलब्धता")
        }.onFailure { slots = emptyList(); message = tr("Could not load availability. Please retry.", "అందుబాటు పొందలేకపోయాము. మళ్లీ ప్రయత్నించండి.", "उपलब्धता नहीं मिली। फिर से प्रयास करें।") }
    }
    fun refresh() {
        if (!validDate.matches(date)) { message = tr("Enter a valid date.", "సరైన తేదీని ఎంచుకోండి.", "सही तारीख चुनें।"); return }
        val requestedDate = date
        loading = true
        scope.launch {
            try { refreshNow(requestedDate) } finally { loading = false }
        }
    }

    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
        Text(tr("Availability", "అందుబాటు", "उपलब्धता"), style = MaterialTheme.typography.headlineMedium, color = MaterialTheme.colorScheme.primary)
        IndiaDatePicker(value = date, language = language, enabled = !loading, onDateSelected = { date = it; slots = emptyList() })
        Button(onClick = { refresh() }, enabled = !loading) { Text(tr("View slots", "సమయాలు చూడండి", "समय देखें")) }
        Text(message, style = MaterialTheme.typography.bodySmall)
        OutlinedTextField(value = start, onValueChange = { start = it }, label = { Text(tr("Start time (24-hour IST)", "ప్రారంభ సమయం (24 గంటలు IST)", "आरंभ समय (24 घंटे IST)")) }, singleLine = true, modifier = Modifier.fillMaxWidth())
        OutlinedTextField(value = end, onValueChange = { end = it }, label = { Text(tr("End time (24-hour IST)", "ముగింపు సమయం (24 గంటలు IST)", "समाप्ति समय (24 घंटे IST)")) }, singleLine = true, modifier = Modifier.fillMaxWidth())
        Button(onClick = {
            if (!validDate.matches(date) || !validTime.matches(start) || !validTime.matches(end) || start >= end) {
                message = tr("Enter a valid date and times; end must be after start.", "సరైన తేదీ, సమయాలు నమోదు చేయండి; ముగింపు ప్రారంభం తర్వాత ఉండాలి.", "सही तारीख और समय दर्ज करें; समाप्ति समय बाद में होना चाहिए।")
            } else {
                val requestedDate = date
                val requestedStart = start
                val requestedEnd = end
                scope.publishThenRefresh(
                    publish = { PurohitApiClient.createPriestSlot(priestId, requestedDate, requestedStart, requestedEnd).map { Unit } },
                    refresh = { refreshNow(requestedDate) },
                    setLoading = { loading = it },
                    onFailure = { message = tr("Could not publish the slot. Please try again.", "సమయాన్ని ప్రచురించలేకపోయాము. మళ్లీ ప్రయత్నించండి.", "समय प्रकाशित नहीं हो सका। फिर से प्रयास करें।") }
                )
            }
        }, enabled = !loading) { Text(tr("Publish slot", "సమయం ప్రచురించండి", "समय प्रकाशित करें")) }
        slots.forEach { slot ->
            Card(Modifier.fillMaxWidth(), colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant)) {
                val status = if (slot.is_booked) tr("Booked", "బుక్ చేయబడింది", "बुक किया गया") else if (slot.is_held) tr("Held", "తాత్కాలికంగా నిలిపారు", "अस्थायी रूप से आरक्षित") else tr("Available", "అందుబాటులో ఉంది", "उपलब्ध")
                Text("${slot.start_time.take(5)} – ${slot.end_time.take(5)}  •  $status", modifier = Modifier.padding(16.dp))
            }
        }
    }
}


@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun IndiaDatePicker(value: String, language: AppLanguage, enabled: Boolean, onDateSelected: (String) -> Unit) {
    var showPicker by remember { mutableStateOf(false) }
    val today = remember { Clock.System.now().toLocalDateTime(TimeZone.of("Asia/Kolkata")).date }
    val minimumUtc = remember(today) { today.atStartOfDayIn(TimeZone.UTC).toEpochMilliseconds() }
    val displayed = value.split("-").let { if (it.size == 3) "${it[2]}/${it[1]}/${it[0]}" else "Select date" }
    OutlinedButton(onClick = { showPicker = true }, enabled = enabled, modifier = Modifier.fillMaxWidth()) { Text(if (value.isEmpty()) when (language) { AppLanguage.TELUGU -> "తేదీ ఎంచుకోండి (DD/MM/YYYY)"; AppLanguage.HINDI -> "तारीख चुनें (DD/MM/YYYY)"; else -> "Select date (DD/MM/YYYY)" } else displayed) }
    if (showPicker) {
        val picker = rememberDatePickerState(initialSelectedDateMillis = value.takeIf { it.isNotBlank() }?.let { runCatching { LocalDate.parse(it).atStartOfDayIn(TimeZone.UTC).toEpochMilliseconds() }.getOrNull() } ?: minimumUtc,
            selectableDates = remember(minimumUtc) { object : SelectableDates { override fun isSelectableDate(utcTimeMillis: Long) = utcTimeMillis >= minimumUtc } })
        DatePickerDialog(onDismissRequest = { showPicker = false }, confirmButton = { TextButton(onClick = {
            picker.selectedDateMillis?.let { onDateSelected(Instant.fromEpochMilliseconds(it).toLocalDateTime(TimeZone.UTC).date.toString()) }
            showPicker = false
        }) { Text("OK") } }, dismissButton = { TextButton(onClick = { showPicker = false }) { Text("Cancel") } }) { DatePicker(state = picker, showModeToggle = false) }
    }
}
