package com.purohit.priest.ui.duty

import androidx.compose.animation.core.*
import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.purohit.contracts.i18n.AppLanguage
import com.purohit.contracts.i18n.LocalizationManager
import com.purohit.contracts.i18n.StringContract
import com.purohit.priest.network.PurohitApiClient
import com.purohit.priest.location.rememberLocationAccess
import com.purohit.priest.location.DutyServiceController
import kotlinx.coroutines.launch

@Composable
fun DutyToggleScreen(
    onSimulateIncomingOrder: () -> Unit,
    hasActiveBooking: Boolean = false,
    activeBookingId: String = "",
    activeChatRoomId: String = "",
    devoteeName: String = "",
    devoteePhone: String = "",
    eventAddress: String = "",
    poojaTitle: String = "",
    scheduledSlot: String = "",
    dakshinaAmount: Double = 0.0,
    priestId: String = "6e7d277c-4ee2-469f-96ac-331517389762",
    initialBookingStatus: String = "IN_TRANSIT",
    currentLanguage: AppLanguage = AppLanguage.TELUGU,
    strings: StringContract = LocalizationManager.getStrings(AppLanguage.TELUGU),
    onOpenChat: () -> Unit = {},
    onOpenChecklist: () -> Unit = {},
    onBookingCompleted: () -> Unit = {},
    requiresCodeToStart: Boolean = false
) {
    var isOnline by remember { mutableStateOf(false) }
    var locationConfirmed by remember { mutableStateOf(false) }
    var dutyLoaded by remember { mutableStateOf(false) }
    var dutyBusy by remember { mutableStateOf(false) }
    var dutyError by remember { mutableStateOf<String?>(null) }
    var dutyNotice by remember { mutableStateOf<String?>(null) }
    var bookingStatus by remember(initialBookingStatus) { mutableStateOf(if (initialBookingStatus.isNotEmpty()) initialBookingStatus else "IN_TRANSIT") } // IN_TRANSIT, ARRIVED, POOJA_STARTED, COMPLETED
    var showOtpDialog by remember { mutableStateOf(false) }
    var enteredOtp by remember { mutableStateOf("") }
    // Ghat (event) bookings check the devotee's booking code before the ritual starts; it is reused at completion.
    var codeForStart by remember { mutableStateOf(false) }
    var verifiedStartCode by remember(activeBookingId) { mutableStateOf("") }
    var statusMessage by remember { mutableStateOf<String?>(null) }
    var statusIsError by remember { mutableStateOf(false) }
    var isUpdatingStatus by remember { mutableStateOf(false) }

    val coroutineScope = rememberCoroutineScope()
    val locationAccess = rememberLocationAccess()
    val isTelugu = currentLanguage == AppLanguage.TELUGU
    val isHindi = currentLanguage == AppLanguage.HINDI
    fun tr(en: String, te: String, hi: String) = when (currentLanguage) { AppLanguage.TELUGU -> te; AppLanguage.HINDI -> hi; else -> en }

    fun updateLocation(goOnline: Boolean) {
        dutyBusy = true
        dutyError = null
        dutyNotice = null
        locationAccess.request { result ->
            result.onSuccess { location ->
                coroutineScope.launch {
                    val saved = if (goOnline) {
                        goOnlineWithLocation(
                            setOnline = { PurohitApiClient.setDutyStatus(it) },
                            saveLocation = { PurohitApiClient.updateLocation(location.latitude, location.longitude, location.capturedAt, location.accuracyM) }
                        )
                    } else {
                        PurohitApiClient.updateLocation(location.latitude, location.longitude, location.capturedAt, location.accuracyM)
                    }
                    saved.onSuccess {
                        locationConfirmed = true
                        if (goOnline) isOnline = true
                        DutyServiceController.start()
                        dutyNotice = tr("Location updated. Devotees nearby can find you.", "స్థానం నవీకరించబడింది. సమీప భక్తులు మిమ్మల్ని కనుగొనగలరు.", "स्थान अपडेट हो गया। आसपास के भक्त आपको ढूँढ सकते हैं।")
                    }.onFailure {
                        locationConfirmed = false
                        PurohitApiClient.getDutyStatus().onSuccess { status -> isOnline = status.is_online; locationConfirmed = status.location_confirmed }
                        dutyError = tr("Could not update availability and location. Check your connection and try again.", "అందుబాటు, స్థానం నవీకరించలేకపోయాము. కనెక్షన్ తనిఖీ చేసి మళ్లీ ప్రయత్నించండి.", "उपलब्धता और स्थान अपडेट नहीं हुए। कनेक्शन जाँचकर फिर प्रयास करें।")
                    }
                    dutyBusy = false
                }
            }.onFailure {
                locationConfirmed = false
                dutyError = tr("Your location could not be refreshed. Devotees may not be able to find you; update your location.", "మీ స్థానం నవీకరించలేదు. భక్తులు మిమ్మల్ని కనుగొనలేకపోవచ్చు; మీ స్థానాన్ని నవీకరించండి.", "आपका स्थान अपडेट नहीं हुआ। भक्त आपको खोज नहीं पाएंगे; अपना स्थान अपडेट करें।")
                dutyBusy = false
            }
        }
    }

    LaunchedEffect(priestId) {
        dutyBusy = true
        // Retry with backoff (2s → 30s) until the status loads; a cold start often races the network.
        var retryMs = 2_000L
        while (!dutyLoaded) {
            PurohitApiClient.getDutyStatus().onSuccess {
                isOnline = it.is_online
                locationConfirmed = it.location_confirmed
                dutyLoaded = true
                dutyError = null
            }.onFailure {
                dutyError = tr("Could not load availability. Retrying…", "అందుబాటు పొందలేకపోయాము. మళ్లీ ప్రయత్నిస్తున్నాము…", "उपलब्धता नहीं मिली। फिर प्रयास हो रहा है…")
                dutyBusy = false
                kotlinx.coroutines.delay(retryMs)
                retryMs = (retryMs * 2).coerceAtMost(30_000L)
            }
        }
        dutyBusy = false
        // An online session may outlive the Android service. Refresh its fix and restart sharing.
        if (dutyLoaded && isOnline) updateLocation(false)
    }

    fun nextHiddenStep(current: String) = when (current) { "ACCEPTED" -> "IN_TRANSIT"; "IN_TRANSIT" -> "ARRIVED"; else -> "" }

    fun updateStatus(newStatus: String, otp: String? = null) {
        isUpdatingStatus = true
        coroutineScope.launch {
            // Ghat bookings have no travel: the server still walks ACCEPTED → IN_TRANSIT → ARRIVED,
            // so step through the skipped states before starting.
            if (requiresCodeToStart && newStatus == "POOJA_STARTED") {
                for (step in listOf("IN_TRANSIT", "ARRIVED").dropWhile { it != nextHiddenStep(bookingStatus) }) {
                    val hop = PurohitApiClient.updateBookingStatus(bookingId = activeBookingId, status = step, priestId = priestId, otp = null)
                    if (hop.isFailure) break
                    bookingStatus = step
                }
            }
            val res = PurohitApiClient.updateBookingStatus(
                bookingId = activeBookingId,
                status = newStatus,
                priestId = priestId,
                otp = otp
            )
            isUpdatingStatus = false
            res.onFailure { if (newStatus == "POOJA_STARTED") verifiedStartCode = "" }
            res.onSuccess {
                bookingStatus = newStatus
                statusIsError = false
                statusMessage = tr("Booking status updated.", "బుకింగ్ స్థితి నవీకరించబడింది.", "बुकिंग की स्थिति अपडेट हुई।")
                if (newStatus == "COMPLETED") {
                    onBookingCompleted()
                }
            }.onFailure { error ->
                statusIsError = true
                statusMessage = if (otp != null && error.message.orEmpty().let { it.contains("INVALID_OTP") || it.contains("invalid completion OTP", ignoreCase = true) }) {
                    tr("Incorrect code. Check the devotee's code and try again.", "కోడ్ తప్పు. భక్తుని కోడ్ తనిఖీ చేసి మళ్లీ ప్రయత్నించండి.", "कोड गलत है। भक्त का कोड जाँचकर फिर प्रयास करें।")
                } else {
                    tr("Could not update status. Please try again.", "స్థితి నవీకరించలేదు. మళ్లీ ప్రయత్నించండి.", "स्थिति अपडेट नहीं हुई। फिर प्रयास करें।")
                }
            }
        }
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(16.dp)
    ) {
        Card(modifier = Modifier.fillMaxWidth()) {
            Column(modifier = Modifier.padding(16.dp)) {
                Text(tr("Booking availability", "బుకింగ్ అందుబాటు", "बुकिंग उपलब्धता"), style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
                Row(verticalAlignment = Alignment.CenterVertically) {
                    Text(if (!dutyLoaded) tr("Status unavailable", "స్థితి అందుబాటులో లేదు", "स्थिति उपलब्ध नहीं है") else if (isOnline && !locationConfirmed) tr("Online — location needs updating", "ఆన్‌లైన్ — స్థానం నవీకరించాలి", "ऑनलाइन — स्थान अपडेट करें") else if (isOnline) tr("Online", "ఆన్‌లైన్", "ऑनलाइन") else tr("Offline", "ఆఫ్‌లైన్", "ऑफ़लाइन"), modifier = Modifier.weight(1f))
                    Switch(checked = isOnline, enabled = dutyLoaded && !dutyBusy, onCheckedChange = { desired ->
                        if (desired) updateLocation(true) else {
                            dutyBusy = true
                            coroutineScope.launch {
                                PurohitApiClient.setDutyStatus(false).onSuccess { isOnline = it; dutyError = null; DutyServiceController.stop() }
                                    .onFailure { dutyError = tr("Could not update availability. Please try again.", "అందుబాటు మార్చలేకపోయాము. మళ్లీ ప్రయత్నించండి.", "उपलब्धता नहीं बदली जा सकी। फिर प्रयास करें।") }
                                dutyBusy = false
                            }
                        }
                    })
                }
                Text(tr("Your location and online status help devotees find you. Update your location when you move.", "భక్తులు మిమ్మల్ని కనుగొనడానికి మీ స్థానం, ఆన్‌లైన్ స్థితి సహాయపడతాయి. స్థానం మారితే నవీకరించండి.", "आपका स्थान और ऑनलाइन स्थिति भक्तों को आपको खोजने में मदद करती है। स्थान बदलने पर अपडेट करें।"), style = MaterialTheme.typography.bodySmall)
                OutlinedButton(onClick = { updateLocation(false) }, enabled = dutyLoaded && !dutyBusy && isOnline) { Text(tr("Update my location", "నా స్థానం నవీకరించండి", "मेरा स्थान अपडेट करें")) }
                dutyError?.let { Text(it, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall) }
                if (dutyError == null) dutyNotice?.let { Text(it, color = MaterialTheme.colorScheme.primary, style = MaterialTheme.typography.bodySmall) }
            }
        }
        Spacer(modifier = Modifier.height(14.dp))

        // IF ACTIVE BOOKING ASSIGNED TO THIS PRIEST
        if (hasActiveBooking) {
            Card(
                modifier = Modifier.fillMaxWidth(),
                shape = RoundedCornerShape(16.dp),
                colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.background),
                elevation = CardDefaults.cardElevation(defaultElevation = 2.dp)
            ) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Surface(
                            color = when (bookingStatus) {
                                "IN_TRANSIT" -> Color(0xFFD97706)
                                "ARRIVED" -> Color(0xFF2563EB)
                                "POOJA_STARTED" -> Color(0xFF9333EA)
                                else -> Color(0xFF15803D)
                            },
                            shape = RoundedCornerShape(12.dp)
                        ) {
                            Text(
                                text = when (bookingStatus) {
                                    "IN_TRANSIT" -> strings.common.bookingInTransit
                                    "ARRIVED" -> strings.common.bookingArrived
                                    "POOJA_STARTED" -> strings.common.bookingInProgress
                                    "COMPLETED" -> strings.common.bookingCompleted
                                    else -> bookingStatus
                                },
                                color = Color.White,
                                fontWeight = FontWeight.Bold,
                                style = MaterialTheme.typography.labelSmall,
                                modifier = Modifier.padding(horizontal = 10.dp, vertical = 4.dp)
                            )
                        }

                        Text(
                            text = "₹${dakshinaAmount.toInt()}",
                            fontWeight = FontWeight.ExtraBold,
                            style = MaterialTheme.typography.titleLarge,
                            color = Color(0xFF15803D)
                        )
                    }

                    Spacer(modifier = Modifier.height(8.dp))

                    Text(
                        text = poojaTitle,
                        fontWeight = FontWeight.Bold,
                        style = MaterialTheme.typography.titleMedium
                    )
                    if (scheduledSlot.isNotBlank()) Text("📅 $scheduledSlot", style = MaterialTheme.typography.bodyMedium, fontWeight = FontWeight.SemiBold)

                    Spacer(modifier = Modifier.height(10.dp))

                    // Destination Address Box
                    Surface(
                        color = MaterialTheme.colorScheme.surfaceVariant,
                        shape = RoundedCornerShape(10.dp),
                        modifier = Modifier.fillMaxWidth()
                    ) {
                        Column(modifier = Modifier.padding(12.dp)) {
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween
                            ) {
                                Text("👤 ${tr("Devotee", "భక్తుడు", "भक्त")}: $devoteeName", fontWeight = FontWeight.Bold, style = MaterialTheme.typography.bodyMedium)
                                Text("📞 $devoteePhone", fontWeight = FontWeight.SemiBold, color = MaterialTheme.colorScheme.primary, style = MaterialTheme.typography.bodySmall)
                            }
                            Spacer(modifier = Modifier.height(6.dp))
                            Text(
                                text = "📍 $eventAddress",
                                style = MaterialTheme.typography.bodySmall,
                                color = Color(0xFF334155),
                                fontWeight = FontWeight.Medium
                            )
                            Spacer(modifier = Modifier.height(4.dp))
                            Text(
                                text = tr("Use the address provided in the booking.", "బుకింగ్‌లో ఇచ్చిన చిరునామాను ఉపయోగించండి.", "बुकिंग में दिया गया पता उपयोग करें।"),
                                style = MaterialTheme.typography.labelSmall,
                                color = Color(0xFF15803D)
                            )
                        }
                    }

                    Spacer(modifier = Modifier.height(12.dp))

                    // Communication & Samagri Actions Row
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(8.dp)
                    ) {
                        Button(
                            onClick = onOpenChecklist,
                            modifier = Modifier.weight(1f),
                            shape = RoundedCornerShape(10.dp),
                            colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.secondary)
                        ) {
                            Text(
                                "🧰 " + if (isTelugu) "సామాగ్రి జాబితా" else if (isHindi) "सामग्री सूची" else "Samagri Checklist",
                                style = MaterialTheme.typography.labelSmall
                            )
                        }

                        Button(
                            onClick = onOpenChat,
                            modifier = Modifier.weight(1f),
                            shape = RoundedCornerShape(10.dp),
                            colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.primary)
                        ) {
                            Text(
                                "💬 " + if (isTelugu) "భక్తునితో సంభాషణ" else if (isHindi) "भक्त से बातचीत" else "Chat with Devotee",
                                style = MaterialTheme.typography.labelSmall
                            )
                        }
                    }

                    Spacer(modifier = Modifier.height(12.dp))
                    Divider()
                    Spacer(modifier = Modifier.height(12.dp))

                    // Ritual Lifecycle Transit Stepper
                    Text(
                        if (isTelugu) "పూజా క్రతువు నిర్వహణ దశలు:" else if (isHindi) "पूजा अनुष्ठान के चरण:" else "Purohit Transit & Ritual Steps:",
                        fontWeight = FontWeight.Bold,
                        style = MaterialTheme.typography.labelMedium
                    )
                    Spacer(modifier = Modifier.height(8.dp))

                    Column(
                        modifier = Modifier.fillMaxWidth(),
                        verticalArrangement = Arrangement.spacedBy(8.dp)
                    ) {
                        // Only the next step is tappable; ghat bookings skip travel and arrival.
                        if (!requiresCodeToStart) {
                            Button(
                                onClick = { updateStatus("IN_TRANSIT") },
                                modifier = Modifier.fillMaxWidth().heightIn(min = 48.dp),
                                shape = RoundedCornerShape(8.dp),
                                enabled = !isUpdatingStatus && bookingStatus == "ACCEPTED"
                            ) {
                                Text(if (isTelugu) "1. ప్రయాణం" else if (isHindi) "1. प्रस्थान" else "1. In Transit", style = MaterialTheme.typography.labelSmall)
                            }

                            Button(
                                onClick = { updateStatus("ARRIVED") },
                                modifier = Modifier.fillMaxWidth().heightIn(min = 48.dp),
                                shape = RoundedCornerShape(8.dp),
                                enabled = !isUpdatingStatus && bookingStatus == "IN_TRANSIT"
                            ) {
                                Text(if (isTelugu) "2. చేరిక" else if (isHindi) "2. आगमन" else "2. Arrived", style = MaterialTheme.typography.labelSmall)
                            }
                        }

                        Button(
                            onClick = { if (requiresCodeToStart) { codeForStart = true; enteredOtp = ""; showOtpDialog = true } else updateStatus("POOJA_STARTED") },
                            modifier = Modifier.fillMaxWidth().heightIn(min = 48.dp),
                            shape = RoundedCornerShape(8.dp),
                            enabled = !isUpdatingStatus && (bookingStatus == "ARRIVED" || (requiresCodeToStart && bookingStatus in setOf("ACCEPTED", "IN_TRANSIT")))
                        ) {
                            val n = if (requiresCodeToStart) "1" else "3"
                            Text(if (isTelugu) "$n. పూజ మొదలు" else if (isHindi) "$n. पूजा प्रारंभ" else "$n. Start Puja", style = MaterialTheme.typography.labelSmall)
                        }

                        Button(
                            onClick = { if (requiresCodeToStart && verifiedStartCode.isNotBlank()) updateStatus("COMPLETED", verifiedStartCode) else { codeForStart = false; enteredOtp = ""; showOtpDialog = true } },
                            modifier = Modifier.fillMaxWidth().heightIn(min = 48.dp),
                            shape = RoundedCornerShape(8.dp),
                            colors = ButtonDefaults.buttonColors(containerColor = Color(0xFF15803D)),
                            enabled = !isUpdatingStatus && bookingStatus == "POOJA_STARTED"
                        ) {
                            val n = if (requiresCodeToStart) "2" else "4"
                            Text(if (isTelugu) "$n. సమాప్తం" else if (isHindi) "$n. संपन्न" else "$n. Complete", style = MaterialTheme.typography.labelSmall)
                        }
                    }
                }
            }

            Spacer(modifier = Modifier.height(14.dp))
        } else {
            // Awaiting Incoming Ritual Dispatch Card
            Card(
                modifier = Modifier.fillMaxWidth(),
                shape = RoundedCornerShape(16.dp),
                colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface),
                elevation = CardDefaults.cardElevation(defaultElevation = 1.dp)
            ) {
                Column(
                    modifier = Modifier.padding(20.dp),
                    horizontalAlignment = Alignment.CenterHorizontally
                ) {
                    Surface(
                        shape = CircleShape,
                        color = Color(0xFFDCFCE7),
                        modifier = Modifier.size(56.dp)
                    ) {
                        Box(contentAlignment = Alignment.Center) {
                            Text("📡", style = MaterialTheme.typography.headlineMedium)
                        }
                    }
                    Spacer(modifier = Modifier.height(10.dp))
                    Text(
                        text = if (!dutyLoaded) tr("Availability unavailable", "అందుబాటు తెలియదు", "उपलब्धता ज्ञात नहीं है") else if (isOnline) tr("Ready for booking requests", "బుకింగ్ అభ్యర్థనలకు సిద్ధం", "बुकिंग अनुरोधों के लिए तैयार") else tr("You are offline", "మీరు ఆఫ్‌లైన్‌లో ఉన్నారు", "आप ऑफ़लाइन हैं"),
                        fontWeight = FontWeight.Bold,
                        style = MaterialTheme.typography.titleMedium
                    )
                    Spacer(modifier = Modifier.height(4.dp))
                    Text(
                        text = if (isOnline) tr("Booking requests will appear here.", "బుకింగ్ అభ్యర్థనలు ఇక్కడ కనిపిస్తాయి.", "बुकिंग अनुरोध यहाँ दिखाई देंगे।") else tr("Go online when you are ready to accept bookings.", "బుకింగ్‌లు స్వీకరించడానికి సిద్ధమైనప్పుడు ఆన్‌లైన్‌కు వెళ్లండి.", "बुकिंग लेने के लिए तैयार होने पर ऑनलाइन हों।"),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant,
                        textAlign = androidx.compose.ui.text.style.TextAlign.Center
                    )
                }
            }

            Spacer(modifier = Modifier.height(14.dp))
        }

        // Daily Earnings & Metrics Card
        if (statusMessage != null) {
            Spacer(modifier = Modifier.height(8.dp))
            Text(
                text = statusMessage!!,
                style = MaterialTheme.typography.labelSmall,
                color = if (statusIsError) MaterialTheme.colorScheme.error else Color(0xFF15803D),
                fontWeight = FontWeight.Bold
            )
        }

        Spacer(modifier = Modifier.height(24.dp))
    }

    // OTP Completion Dialog
    if (showOtpDialog) {
        AlertDialog(
            onDismissRequest = { showOtpDialog = false },
            title = { Text(if (codeForStart) strings.event.enterBookingCode else tr("Enter completion code", "పూర్తి కోడ్ నమోదు చేయండి", "समापन कोड दर्ज करें"), fontWeight = FontWeight.Bold) },
            text = {
                Column {
                    Text(
                        text = if (codeForStart) tr("Ask the devotee for the 4-digit booking code shown in their app before you begin.", "ప్రారంభించే ముందు భక్తుని యాప్‌లో ఉన్న 4 అంకెల బుకింగ్ కోడ్ అడగండి.", "शुरू करने से पहले भक्त के ऐप में दिखा 4 अंकों का बुकिंग कोड लें।")
                            else tr("Ask the devotee for the 4-digit completion code after the pooja.", "పూజ తర్వాత భక్తుడిని 4 అంకెల పూర్తి కోడ్ అడగండి.", "पूजा के बाद भक्त से 4 अंकों का समापन कोड लें।"),
                        style = MaterialTheme.typography.bodySmall
                    )
                    Spacer(modifier = Modifier.height(10.dp))
                    OutlinedTextField(
                        value = enteredOtp,
                        onValueChange = { enteredOtp = it.filter { c -> c.isDigit() }.take(4) },
                        label = { Text(tr("4-digit code", "4 అంకెల కోడ్", "4 अंकों का कोड")) },
                        keyboardOptions = androidx.compose.foundation.text.KeyboardOptions(keyboardType = androidx.compose.ui.text.input.KeyboardType.NumberPassword),
                        modifier = Modifier.fillMaxWidth(),
                        singleLine = true
                    )
                }
            },
            confirmButton = {
                Button(
                    onClick = {
                        if (enteredOtp.length == 4) {
                            showOtpDialog = false
                            if (codeForStart) {
                                verifiedStartCode = enteredOtp.trim()
                                updateStatus("POOJA_STARTED", enteredOtp.trim())
                            } else updateStatus("COMPLETED", enteredOtp.trim())
                        }
                    },
                    enabled = enteredOtp.length == 4,
                    colors = ButtonDefaults.buttonColors(containerColor = Color(0xFF15803D))
                ) {
                    Text(if (codeForStart) tr("Verify and start", "నిర్ధారించి ప్రారంభించండి", "सत्यापित करें और शुरू करें") else tr("Verify completion", "పూర్తిని నిర్ధారించండి", "समापन सत्यापित करें"))
                }
            },
            dismissButton = {
                OutlinedButton(onClick = { showOtpDialog = false }) {
                    Text(strings.common.cancelAction)
                }
            }
        )
    }
}
