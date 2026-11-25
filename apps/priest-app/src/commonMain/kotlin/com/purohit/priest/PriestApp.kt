package com.purohit.priest

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.purohit.contracts.i18n.AppLanguage
import com.purohit.contracts.i18n.LocalizationManager
import com.purohit.contracts.i18n.StringContract
import com.purohit.contracts.resilience.JitteredBackoff
import com.purohit.priest.location.DutyServiceController
import com.purohit.priest.network.PurohitApiClient
import com.purohit.priest.ui.auth.PriestRegistrationScreen
import com.purohit.priest.ui.chat.PriestChatScreen
import com.purohit.priest.ui.dispatch.RapidoIncomingAlertModal
import com.purohit.priest.ui.duty.DutyToggleScreen
import com.purohit.priest.ui.materials.PriestSamagriChecklistModal
import com.purohit.priest.ui.referral.ColleagueReferralModal
import com.purohit.priest.ui.resilience.PriestErrorScreen
import com.purohit.priest.ui.resilience.PriestNoInternetScreen
import com.purohit.priest.ui.resilience.PriestVedicLoadingScreen
import com.purohit.priest.ui.schedule.PriestScheduleScreen
import com.purohit.priest.ui.services.PriestServicesScreen
import com.purohit.priest.ui.studio.BroadcasterStudioScreen
import kotlinx.coroutines.launch

val LocalPriestStrings = compositionLocalOf<StringContract> {
    LocalizationManager.getStrings(AppLanguage.ENGLISH)
}

val LocalPriestLanguage = staticCompositionLocalOf { AppLanguage.TELUGU }

enum class PriestScreen {
    REGISTRATION,
    DUTY_RADAR,         // Tab 1: ⚡ Radar & Duty Dashboard
    SCHEDULE_WORKBENCH, // Tab 2: 📅 Schedule Workbench
    CUSTOM_SERVICES,    // Tab 3: 📋 My Services & Dynamic Dakshina
    LIVE_STUDIO,        // Tab 4: 📹 Live Studio & Active Rituals
    PUSHKARALU,         // Tab 4 while an event is live (replaces Live Studio; PRD v2 defers live E-Pooja)
    CHAT,               // 💬 Bidirectional Chat with Devotee
    PROFILE             // Tab 5: 👤 Vedic Credentials & Language
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PriestAppRoot() {
    var isRegistered by remember { mutableStateOf(false) } // Require priest login/registration for pilot
    var currentScreen by remember { mutableStateOf(PriestScreen.REGISTRATION) }
    var priestName by remember { mutableStateOf("") }
    var priestId by remember { mutableStateOf("") }

    // Active booking details connected to PostgreSQL DB
    var hasActiveBooking by remember { mutableStateOf(false) }
    var activeBookingId by remember { mutableStateOf("") }
    var activeChatRoomId by remember { mutableStateOf("") }
    var activeDevoteeName by remember { mutableStateOf("") }
    var activeDevoteePhone by remember { mutableStateOf("") }
    var activeAddress by remember { mutableStateOf("") }
    var activePoojaTitle by remember { mutableStateOf("") }
    var activeDakshina by remember { mutableStateOf(0.0) }
    var activeScheduledDate by remember { mutableStateOf("") }
    var activeSlotTime by remember { mutableStateOf("") }
    var activeDevoteeGotram by remember { mutableStateOf("") }
    var activeDevoteeNakshatram by remember { mutableStateOf("") }
    var activeDevoteeRashi by remember { mutableStateOf("") }
    var activeSankalpamPurpose by remember { mutableStateOf("") }
    var activeBookingStatus by remember { mutableStateOf("IN_TRANSIT") }
    var activeBookingType by remember { mutableStateOf("") }
    var activeRequestExpiresAt by remember { mutableStateOf<String?>(null) }
    var eventConfig by remember { mutableStateOf<com.purohit.contracts.event.EventConfig?>(null) }

    var hasSimulatedIncomingAlert by remember { mutableStateOf(false) }
    var incomingError by remember { mutableStateOf<String?>(null) }
    var incomingBusy by remember { mutableStateOf(false) }
    var showChecklistModal by remember { mutableStateOf(false) }
    var prefilledChatMessage by remember { mutableStateOf("") }
    var showReferralModal by remember { mutableStateOf(false) }
    var showLanguageModal by remember { mutableStateOf(false) }
    var showAppearance by remember { mutableStateOf(false) }
    var appearance by remember { mutableStateOf(appearanceFromStoredValue(readAppearance())) }
    var showLogoutModal by remember { mutableStateOf(false) }
    var isNetworkError by remember { mutableStateOf(false) }
    var networkErrorMessage by remember { mutableStateOf("") }

    // Follow the phone language when the app supports it; Telugu stays the default for other languages.
    var currentLanguage by rememberSaveable { mutableStateOf(when (androidx.compose.ui.text.intl.Locale.current.language) { "en" -> AppLanguage.ENGLISH; "hi" -> AppLanguage.HINDI; else -> AppLanguage.TELUGU }) }
    fun ptr(en: String, te: String, hi: String) = when (currentLanguage) { AppLanguage.TELUGU -> te; AppLanguage.HINDI -> hi; else -> en }
    val strings = remember(currentLanguage) { LocalizationManager.getStrings(currentLanguage) }

    val coroutineScope = rememberCoroutineScope()

    var priestProfile by remember { mutableStateOf(PurohitApiClient.currentPriestProfile) }
    var showEmailDialog by remember { mutableStateOf(false) }
    var emailInput by remember { mutableStateOf("") }
    var emailLoading by remember { mutableStateOf(false) }
    var emailMessage by remember { mutableStateOf<String?>(null) }
    var emailError by remember { mutableStateOf<String?>(null) }

    fun reloadPriestProfile() {
        if (PurohitApiClient.authToken.isNotBlank()) {
            coroutineScope.launch {
                val res = PurohitApiClient.fetchPriestProfile()
                res.onSuccess {
                    priestProfile = it
                }
            }
        }
    }

    LaunchedEffect(currentScreen) {
        if (currentScreen == PriestScreen.PROFILE) {
            reloadPriestProfile()
        }
    }

    LaunchedEffect(Unit) {
        if (PurohitApiClient.authToken.isNotBlank()) {
            PurohitApiClient.fetchPriestProfile().onSuccess {
                priestProfile = it
                priestId = it.id
                priestName = it.full_name
                isRegistered = true
                currentScreen = PriestScreen.DUTY_RADAR
            }.onFailure { networkErrorMessage = it.message.orEmpty(); isNetworkError = true }
        }
    }

    // Event config: on login and every 10 minutes (no config push on iOS yet) (the server enforces every flag regardless).
    LaunchedEffect(isRegistered) {
        if (!isRegistered) return@LaunchedEffect
        while (true) {
            PurohitApiClient.fetchEventConfig().onSuccess { eventConfig = it }
            kotlinx.coroutines.delay(10 * 60 * 1000L)
        }
    }

    // Realtime Cloud Dispatch Sync: Polls for incoming Devotee Requests & Active Rituals
    LaunchedEffect(priestId, hasActiveBooking) {
        if (priestId.isBlank()) return@LaunchedEffect
        val backoff = JitteredBackoff(baseDelayMs = 5000L, maxDelayMs = 30000L)
        while (true) {
            try {
                if (!hasActiveBooking) {
                    // 1. Check for incoming REQUESTED bookings for this priest
                    val reqRes = PurohitApiClient.fetchPriestBookings(priestId, "REQUESTED")
                    reqRes.onSuccess { bookings ->
                        isNetworkError = false
                        val incoming = bookings.firstOrNull { it.status == "REQUESTED" }
                        // Only reconcile a server-backed alert after a successful poll. A
                        // failed poll must not dismiss a request, and accepting/declining
                        // it may be in flight while the poll response arrives.
                        if (!incomingBusy && !hasActiveBooking && incoming == null &&
                            hasSimulatedIncomingAlert && activeBookingId.isNotBlank()) {
                            hasSimulatedIncomingAlert = false
                            activeBookingId = ""
                            activeRequestExpiresAt = null
                            incomingError = null
                        }
                        if (!incomingBusy && !hasActiveBooking && incoming != null && incoming.id != activeBookingId) {
                            incomingError = null
                            activeBookingId = incoming.id
                            activePoojaTitle = incoming.pooja_title
                            activeDevoteeName = incoming.devotee_name
                            activeDevoteePhone = incoming.devotee_phone
                            activeAddress = incoming.event_address
                            activeDakshina = incoming.total_amount
                            activeScheduledDate = incoming.scheduled_date
                            activeSlotTime = incoming.slot_time
                            activeDevoteeGotram = incoming.devotee_gotram
                            activeDevoteeNakshatram = incoming.devotee_nakshatram
                            activeDevoteeRashi = incoming.devotee_rashi
                            activeSankalpamPurpose = incoming.sankalpam_purpose
                            activeChatRoomId = incoming.chat_room_id
                            activeBookingType = incoming.booking_type
                            activeRequestExpiresAt = incoming.request_expires_at
                            hasSimulatedIncomingAlert = true
                        }
                    }

                    // 2. Check if an ongoing accepted/in-transit ritual is active to resume
                    val actRes = PurohitApiClient.fetchPriestBookings(priestId, "ACTIVE")
                    actRes.onSuccess { bookings ->
                        val ongoing = bookings.firstOrNull {
                            it.status in listOf("ACCEPTED", "IN_TRANSIT", "ARRIVED", "POOJA_STARTED", "IN_PROGRESS")
                        }
                        if (ongoing != null && !incomingBusy) {
                            hasSimulatedIncomingAlert = false
                            activeBookingId = ongoing.id
                            activePoojaTitle = ongoing.pooja_title
                            activeDevoteeName = ongoing.devotee_name
                            activeDevoteePhone = ongoing.devotee_phone
                            activeAddress = ongoing.event_address
                            activeDakshina = ongoing.total_amount
                            activeScheduledDate = ongoing.scheduled_date
                            activeSlotTime = ongoing.slot_time
                            activeChatRoomId = ongoing.chat_room_id
                            activeBookingStatus = ongoing.status
                            activeBookingType = ongoing.booking_type
                            hasActiveBooking = true
                        }
                    }
                } else if (activeBookingId.isNotBlank()) {
                    // Check active booking status updates
                    val bRes = PurohitApiClient.fetchBookingDetails(activeBookingId)
                    bRes.onSuccess { b ->
                        activeBookingStatus = b.status
                        if (b.status == "COMPLETED" || b.status == "CANCELLED") {
                            hasActiveBooking = false
                            activeBookingId = ""
                        }
                    }
                }
            } catch (t: Throwable) {
                backoff.recordError()
            }
            backoff.awaitNext()
        }
    }

    CompositionLocalProvider(LocalPriestStrings provides strings, LocalPriestLanguage provides currentLanguage) {
        MaterialTheme(
            colorScheme = appearanceScheme(appearance),
            typography = appTypography(),
            shapes = Shapes(small = androidx.compose.foundation.shape.RoundedCornerShape(8.dp), medium = androidx.compose.foundation.shape.RoundedCornerShape(16.dp), large = androidx.compose.foundation.shape.RoundedCornerShape(20.dp))
        ) {
            Box(modifier = Modifier.fillMaxSize()) {
                Scaffold(
                    topBar = {
                        if (currentScreen != PriestScreen.CHAT) {
                            AppHeader(currentLanguage, onAppearance = { showAppearance = true }, onLanguage = { showLanguageModal = true })
                        }
                    },
                    bottomBar = {
                        if (isRegistered && currentScreen != PriestScreen.CHAT) {
                            NavigationBar(containerColor = MaterialTheme.colorScheme.primaryContainer, contentColor = MaterialTheme.colorScheme.onPrimaryContainer) {
                                NavigationBarItem(
                                    selected = currentScreen == PriestScreen.DUTY_RADAR,
                                    onClick = { currentScreen = PriestScreen.DUTY_RADAR },
                                    label = { Text(strings.common.tabDuty, style = MaterialTheme.typography.labelSmall, maxLines = 2, textAlign = androidx.compose.ui.text.style.TextAlign.Center) },
                                    icon = { Text("⚡") }
                                )
                                NavigationBarItem(
                                    selected = currentScreen == PriestScreen.SCHEDULE_WORKBENCH,
                                    onClick = { currentScreen = PriestScreen.SCHEDULE_WORKBENCH },
                                    label = { Text(strings.common.tabSchedule, style = MaterialTheme.typography.labelSmall, maxLines = 2, textAlign = androidx.compose.ui.text.style.TextAlign.Center) },
                                    icon = { Text("📅") }
                                )
                                NavigationBarItem(
                                    selected = currentScreen == PriestScreen.CUSTOM_SERVICES,
                                    onClick = { currentScreen = PriestScreen.CUSTOM_SERVICES },
                                    label = { Text(strings.common.tabCustomServices, style = MaterialTheme.typography.labelSmall, maxLines = 2, textAlign = androidx.compose.ui.text.style.TextAlign.Center) },
                                    icon = { Text("📋") }
                                )
                                if (eventConfig?.isOpenForPriests == true || currentScreen == PriestScreen.PUSHKARALU) NavigationBarItem(
                                    selected = currentScreen == PriestScreen.PUSHKARALU,
                                    onClick = { currentScreen = PriestScreen.PUSHKARALU },
                                    label = { Text(strings.event.tabEvent, style = MaterialTheme.typography.labelSmall, maxLines = 2, textAlign = androidx.compose.ui.text.style.TextAlign.Center) },
                                    icon = { Text("🌊") }
                                ) else NavigationBarItem(
                                    selected = currentScreen == PriestScreen.LIVE_STUDIO,
                                    onClick = { currentScreen = PriestScreen.LIVE_STUDIO },
                                    label = { Text("Live Studio", style = MaterialTheme.typography.labelSmall, maxLines = 2, textAlign = androidx.compose.ui.text.style.TextAlign.Center) },
                                    icon = { Text("📹") }
                                )
                                NavigationBarItem(
                                    selected = currentScreen == PriestScreen.PROFILE,
                                    onClick = { currentScreen = PriestScreen.PROFILE },
                                    label = { Text(strings.common.tabProfile, style = MaterialTheme.typography.labelSmall, maxLines = 2, textAlign = androidx.compose.ui.text.style.TextAlign.Center) },
                                    icon = { Text("👤") }
                                )
                            }
                        }
                    }
                ) { paddingValues ->
                    Box(modifier = Modifier.padding(paddingValues).fillMaxSize()) {
                        if (isNetworkError) {
                            PriestNoInternetScreen(
                                strings = strings,
                                onRetry = { isNetworkError = false },
                                serverUrl = PurohitApiClient.baseUrl
                            )
                        } else {
                            when (currentScreen) {
                                PriestScreen.REGISTRATION -> PriestRegistrationScreen(
                                    strings = strings,
                                    onRegistrationSuccess = { name, _, id ->
                                        priestName = name
                                        priestId = id
                                        isRegistered = true
                                        currentScreen = PriestScreen.DUTY_RADAR
                                    }
                                )
                            PriestScreen.DUTY_RADAR -> DutyToggleScreen(
                                onSimulateIncomingOrder = { hasSimulatedIncomingAlert = true },
                                hasActiveBooking = hasActiveBooking,
                                activeBookingId = activeBookingId,
                                activeChatRoomId = activeChatRoomId,
                                devoteeName = activeDevoteeName,
                                devoteePhone = activeDevoteePhone,
                                eventAddress = activeAddress,
                                poojaTitle = activePoojaTitle,
                                dakshinaAmount = activeDakshina,
                                priestId = priestId,
                                initialBookingStatus = activeBookingStatus,
                                currentLanguage = currentLanguage,
                                strings = strings,
                                onOpenChat = { currentScreen = PriestScreen.CHAT },
                                onOpenChecklist = { showChecklistModal = true },
                                onBookingCompleted = { hasActiveBooking = false; activeBookingId = "" },
                                requiresCodeToStart = activeBookingType.startsWith("EVENT")
                            )
                            PriestScreen.PUSHKARALU -> com.purohit.priest.ui.pushkaralu.PriestEventScreen(eventConfig, currentLanguage, strings.event, hasActiveBooking)
                            PriestScreen.SCHEDULE_WORKBENCH -> PriestScheduleScreen(
                                priestId = priestId,
                                strings = strings
                            )
                            PriestScreen.CUSTOM_SERVICES -> PriestServicesScreen(
                                priestId = priestId,
                                currentLanguage = currentLanguage,
                                strings = strings
                            )
                            PriestScreen.LIVE_STUDIO -> BroadcasterStudioScreen()
                            PriestScreen.CHAT -> PriestChatScreen(
                                roomId = activeChatRoomId,
                                priestId = priestId,
                                devoteeName = activeDevoteeName,
                                devoteePhone = activeDevoteePhone,
                                eventAddress = activeAddress,
                                initialPrefillText = prefilledChatMessage,
                                onBack = { currentScreen = PriestScreen.DUTY_RADAR }
                            )
                            PriestScreen.PROFILE -> {
                                Column(
                                    modifier = Modifier.fillMaxSize().padding(20.dp).verticalScroll(rememberScrollState()),
                                    horizontalAlignment = Alignment.CenterHorizontally
                                ) {
                                    Text(
                                        text = priestName,
                                        style = MaterialTheme.typography.headlineSmall,
                                        fontWeight = FontWeight.Bold
                                    )
                                    Text(
                                        text = listOfNotNull(priestProfile?.veda_shakha?.takeIf { it.isNotBlank() }, priestProfile?.sampradaya?.takeIf { it.isNotBlank() }).joinToString(" • "),
                                        style = MaterialTheme.typography.bodyMedium,
                                        color = Color.Gray
                                    )
                                    Spacer(modifier = Modifier.height(16.dp))

                                    // Email Password Recovery Card
                                    Card(
                                        modifier = Modifier.fillMaxWidth(),
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
                                                    text = ptr("📧 Password Recovery Email", "📧 పాస్‌వర్డ్ రికవరీ ఈమెయిల్", "📧 पासवर्ड रिकवरी ईमेल"),
                                                    fontWeight = FontWeight.Bold,
                                                    style = MaterialTheme.typography.titleSmall
                                                )
                                                IconButton(onClick = { reloadPriestProfile() }, modifier = Modifier.size(28.dp)) {
                                                    Text("🔄", style = MaterialTheme.typography.bodySmall)
                                                }
                                            }
                                            Spacer(modifier = Modifier.height(6.dp))
                                            val currentEmail = priestProfile?.email.orEmpty()
                                            val isVerified = priestProfile?.email_verified == true

                                            if (currentEmail.isNotBlank()) {
                                                Row(
                                                    modifier = Modifier.fillMaxWidth(),
                                                    horizontalArrangement = Arrangement.SpaceBetween,
                                                    verticalAlignment = Alignment.CenterVertically
                                                ) {
                                                    Text(text = currentEmail, style = MaterialTheme.typography.bodyMedium, fontWeight = FontWeight.SemiBold)
                                                    if (isVerified) {
                                                        Surface(
                                                            shape = RoundedCornerShape(8.dp),
                                                            color = Color(0xFFDCFCE7)
                                                        ) {
                                                            Text(
                                                                text = ptr("🟢 Verified", "🟢 ధృవీకరించబడింది", "🟢 सत्यापित"),
                                                                color = Color(0xFF15803D),
                                                                style = MaterialTheme.typography.labelSmall,
                                                                fontWeight = FontWeight.Bold,
                                                                modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp)
                                                            )
                                                        }
                                                    } else {
                                                        Surface(
                                                            shape = RoundedCornerShape(8.dp),
                                                            color = Color(0xFFFEF3C7)
                                                        ) {
                                                            Text(
                                                                text = ptr("⚠️ Unverified", "⚠️ ధృవీకరించలేదు", "⚠️ असत्यापित"),
                                                                color = Color(0xFFB45309),
                                                                style = MaterialTheme.typography.labelSmall,
                                                                fontWeight = FontWeight.Bold,
                                                                modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp)
                                                            )
                                                        }
                                                    }
                                                }
                                                Spacer(modifier = Modifier.height(6.dp))
                                                Text(
                                                    text = if (isVerified) {
                                                        ptr("Self-service password reset is enabled via single-use reset links sent to this email.", "ఈ ఈమెయిల్‌కు పంపే లింక్‌తో పాస్‌వర్డ్ రీసెట్ చేయవచ్చు.", "इस ईमेल पर भेजे गए लिंक से पासवर्ड रीसेट कर सकते हैं।")
                                                    } else {
                                                        ptr("Verification link pending. Complete verification to enable password reset.", "ధృవీకరణ పెండింగ్‌లో ఉంది. పాస్‌వర్డ్ రీసెట్ కోసం ధృవీకరించండి.", "सत्यापन बाकी है। पासवर्ड रीसेट के लिए सत्यापन पूरा करें।")
                                                    },
                                                    style = MaterialTheme.typography.bodySmall,
                                                    color = Color.Gray
                                                )
                                            } else {
                                                Text(
                                                    text = ptr("No verified email configured. Password recovery via email is disabled.", "ధృవీకరించిన ఈమెయిల్ లేదు. ఈమెయిల్ ద్వారా పాస్‌వర్డ్ రికవరీ సాధ్యం కాదు.", "कोई सत्यापित ईमेल नहीं है। ईमेल से पासवर्ड रिकवरी बंद है।"),
                                                    style = MaterialTheme.typography.bodySmall,
                                                    color = Color.Gray
                                                )
                                            }

                                            Spacer(modifier = Modifier.height(12.dp))
                                            Button(
                                                onClick = {
                                                    emailInput = currentEmail
                                                    emailMessage = null
                                                    emailError = null
                                                    showEmailDialog = true
                                                },
                                                modifier = Modifier.fillMaxWidth(),
                                                shape = RoundedCornerShape(10.dp)
                                            ) {
                                                Text(if (currentEmail.isNotBlank()) ptr("Update Recovery Email", "రికవరీ ఈమెయిల్ మార్చండి", "रिकवरी ईमेल बदलें") else ptr("Add Recovery Email", "రికవరీ ఈమెయిల్ జోడించండి", "रिकवरी ईमेल जोड़ें"))
                                            }
                                        }
                                    }

                                    if (showEmailDialog) {
                                        AlertDialog(
                                            onDismissRequest = { if (!emailLoading) showEmailDialog = false },
                                            title = { Text(ptr("Set Priest Recovery Email", "రికవరీ ఈమెయిల్ సెట్ చేయండి", "रिकवरी ईमेल सेट करें")) },
                                            text = {
                                                Column {
                                                    Text(
                                                        ptr("Enter your email address to receive a secure single-use verification link. Your existing recovery email is preserved until the replacement is verified.", "ధృవీకరణ లింక్ కోసం మీ ఈమెయిల్ నమోదు చేయండి. కొత్తది ధృవీకరించే వరకు ప్రస్తుత ఈమెయిల్ కొనసాగుతుంది.", "सत्यापन लिंक के लिए अपना ईमेल दर्ज करें। नया ईमेल सत्यापित होने तक मौजूदा ईमेल बना रहेगा।"),
                                                        style = MaterialTheme.typography.bodySmall,
                                                        color = Color.Gray
                                                    )
                                                    Spacer(modifier = Modifier.height(12.dp))
                                                    OutlinedTextField(
                                                        value = emailInput,
                                                        onValueChange = { emailInput = it },
                                                        label = { Text(ptr("Email Address", "ఈమెయిల్ చిరునామా", "ईमेल पता")) },
                                                        singleLine = true,
                                                        modifier = Modifier.fillMaxWidth(),
                                                        enabled = !emailLoading
                                                    )

                                                    emailMessage?.let { msg ->
                                                        Spacer(modifier = Modifier.height(8.dp))
                                                        Text(msg, color = Color(0xFF15803D), style = MaterialTheme.typography.bodySmall, fontWeight = FontWeight.Bold)
                                                    }

                                                    emailError?.let { err ->
                                                        Spacer(modifier = Modifier.height(8.dp))
                                                        Text(err, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
                                                    }
                                                }
                                            },
                                            confirmButton = {
                                                Button(
                                                    onClick = {
                                                        if (emailInput.isNotBlank()) {
                                                            emailLoading = true
                                                            emailMessage = null
                                                            emailError = null
                                                            coroutineScope.launch {
                                                                val res = PurohitApiClient.requestEmailVerification(emailInput.trim())
                                                                emailLoading = false
                                                                res.fold(
                                                                    onSuccess = {
                                                                        emailMessage = ptr("Verification link sent! Please check your inbox and click the single-use link to confirm.", "ధృవీకరణ లింక్ పంపాము. మీ ఇన్‌బాక్స్ చూసి లింక్ నొక్కండి.", "सत्यापन लिंक भेजा गया। इनबॉक्स देखकर लिंक पर क्लिक करें।")
                                                                        reloadPriestProfile()
                                                                    },
                                                                    onFailure = { ex ->
                                                                        emailError = ex.message ?: ptr("Failed to dispatch verification email.", "ధృవీకరణ ఈమెయిల్ పంపలేకపోయాము.", "सत्यापन ईमेल नहीं भेजा जा सका।")
                                                                    }
                                                                )
                                                            }
                                                        }
                                                    },
                                                    enabled = !emailLoading && emailInput.isNotBlank()
                                                ) {
                                                    if (emailLoading) {
                                                        CircularProgressIndicator(modifier = Modifier.size(16.dp), strokeWidth = 2.dp, color = Color.White)
                                                    } else {
                                                        Text(ptr("Send Link", "లింక్ పంపండి", "लिंक भेजें"))
                                                    }
                                                }
                                            },
                                            dismissButton = {
                                                TextButton(
                                                    onClick = { showEmailDialog = false },
                                                    enabled = !emailLoading
                                                ) {
                                                    Text(ptr("Close", "మూసివేయి", "बंद करें"))
                                                }
                                            }
                                        )
                                    }

                                    Spacer(modifier = Modifier.height(16.dp))
                                    Button(
                                        onClick = { showReferralModal = true },
                                        modifier = Modifier.fillMaxWidth()
                                    ) {
                                        Text(ptr("🤝 Delegate Booking (Colleague Referral)", "🤝 బుకింగ్ సహచరుడికి అప్పగించండి", "🤝 बुकिंग सहकर्मी को सौंपें"))
                                    }
                                    Spacer(modifier = Modifier.height(10.dp))
                                    OutlinedButton(
                                        onClick = { showLanguageModal = true },
                                        modifier = Modifier.fillMaxWidth()
                                    ) {
                                        Text("🌐 Change Language (${currentLanguage.nativeName})")
                                    }
                                    Spacer(modifier = Modifier.height(10.dp))
                                    Spacer(modifier = Modifier.height(10.dp))
                                    Button(
                                        onClick = { showLogoutModal = true },
                                        colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.error),
                                        modifier = Modifier.fillMaxWidth()
                                    ) {
                                        Text(strings.common.logOut)
                                    }
                                }
                            }
                        }
                        }
                    }
                }

                // 1. Incoming Rapido Alert Modal
                if (hasSimulatedIncomingAlert) {
                    RapidoIncomingAlertModal(
                        bookingId = activeBookingId,
                        poojaTitle = activePoojaTitle,
                        devoteeName = activeDevoteeName,
                        devoteePhone = activeDevoteePhone,
                        eventAddress = activeAddress,
                        dakshinaAmount = activeDakshina,
                        scheduledDate = activeScheduledDate,
                        slotTime = activeSlotTime,
                        devoteeGotram = activeDevoteeGotram,
                        devoteeNakshatram = activeDevoteeNakshatram,
                        devoteeRashi = activeDevoteeRashi,
                        sankalpamPurpose = activeSankalpamPurpose,
                        expiresAt = activeRequestExpiresAt,
                        onExpired = { hasSimulatedIncomingAlert = false; activeBookingId = "" },
                        currentLanguage = currentLanguage,
                        strings = strings,
                        errorMessage = incomingError,
                        isSubmitting = incomingBusy,
                        onAccept = {
                            incomingBusy = true
                            incomingError = null
                            coroutineScope.launch {
                                PurohitApiClient.updateBookingStatus(
                                    bookingId = activeBookingId,
                                    status = "ACCEPTED",
                                    priestId = priestId
                                ).onSuccess {
                                    hasSimulatedIncomingAlert = false
                                    hasActiveBooking = true
                                    activeBookingStatus = "ACCEPTED"
                                    // The active booking card lives on Duty Radar; take the priest there.
                                    currentScreen = PriestScreen.DUTY_RADAR
                                }.onFailure { err ->
                                    incomingError = if (err.message.orEmpty().contains("priest_busy")) strings.event.priestBusyError
                                    else if (err.message.orEmpty().contains("request_expired")) strings.event.priestDeclined
                                    else when (currentLanguage) {
                                        AppLanguage.TELUGU -> "బుకింగ్ స్వీకరించలేకపోయాము. సమయం అందుబాటులో ఉందో తనిఖీ చేసి మళ్లీ ప్రయత్నించండి."
                                        AppLanguage.HINDI -> "बुकिंग स्वीकार नहीं हुई। समय की उपलब्धता जाँचकर फिर प्रयास करें।"
                                        else -> "Could not accept this booking. Check the slot and try again."
                                    }
                                }
                                incomingBusy = false
                            }
                        },
                        onReject = {
                            // Capture the ID first: activeBookingId is cleared below before the request runs.
                            val declinedId = activeBookingId
                            coroutineScope.launch {
                                PurohitApiClient.updateBookingStatus(
                                    bookingId = declinedId,
                                    status = "CANCELLED",
                                    priestId = priestId
                                )
                            }
                            hasSimulatedIncomingAlert = false
                            activeBookingId = ""
                        }
                    )
                }

                // 2. Samagri Checklist Modal (Editable & Synced)
                if (showChecklistModal) {
                    PriestSamagriChecklistModal(
                        bookingId = activeBookingId,
                        poojaTitle = activePoojaTitle,
                        onDismiss = { showChecklistModal = false },
                        onRemindDevotee = { msg ->
                            prefilledChatMessage = msg
                            showChecklistModal = false
                            currentScreen = PriestScreen.CHAT
                        }
                    )
                }

                // 3. Colleague Referral Modal
                if (showReferralModal) {
                    ColleagueReferralModal(
                        bookingId = activeBookingId,
                        referringPriestId = priestId,
                        bookingTitle = activePoojaTitle,
                        scheduledSlot = "$activeScheduledDate • $activeSlotTime",
                        strings = strings,
                        onReferralConfirmed = { _, colleagueName, _ ->
                            showReferralModal = false
                            hasActiveBooking = false
                        },
                        onDismiss = { showReferralModal = false }
                    )
                }

                // 4. Language Selector Modal
                if (showAppearance) {
                    AppearanceDialog(appearance, currentLanguage, onDismiss = { showAppearance = false }, onSelect = { appearance = it; writeAppearance(it.name) })
                }

                if (showLanguageModal) {
                    AlertDialog(
                        onDismissRequest = { showLanguageModal = false },
                        title = { Text(strings.common.selectLanguage, fontWeight = FontWeight.Bold) },
                        text = {
                            Column {
                                AppLanguage.entries.forEach { lang ->
                                    Row(
                                        modifier = Modifier
                                            .fillMaxWidth()
                                            .clickable {
                                                currentLanguage = lang
                                                showLanguageModal = false
                                            }
                                            .padding(vertical = 8.dp),
                                        verticalAlignment = Alignment.CenterVertically
                                    ) {
                                        RadioButton(
                                            selected = currentLanguage == lang,
                                            onClick = {
                                                currentLanguage = lang
                                                showLanguageModal = false
                                            }
                                        )
                                        Spacer(modifier = Modifier.width(8.dp))
                                        Text(
                                            text = "${lang.nativeName} (${lang.englishName})",
                                            fontWeight = if (currentLanguage == lang) FontWeight.Bold else FontWeight.Normal
                                        )
                                    }
                                }
                            }
                        },
                        confirmButton = {
                            OutlinedButton(onClick = { showLanguageModal = false }) {
                                Text(strings.common.closeAction)
                            }
                        }
                    )
                }

                // 5. Logout Confirmation Modal
                if (showLogoutModal) {
                    AlertDialog(
                        onDismissRequest = { showLogoutModal = false },
                        title = { Text(strings.common.logOutConfirmTitle, fontWeight = FontWeight.Bold) },
                        text = { Text(strings.common.logOutConfirmDescription) },
                        confirmButton = {
                            Button(
                                onClick = {
                                    showLogoutModal = false
                                    DutyServiceController.stop()
                                    PurohitApiClient.authToken = ""
                                    PurohitApiClient.currentPriestProfile = null
                                    isRegistered = false
                                    currentScreen = PriestScreen.REGISTRATION
                                },
                                colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.error)
                            ) {
                                Text(strings.common.logOutConfirmButton)
                            }
                        },
                        dismissButton = {
                            OutlinedButton(onClick = { showLogoutModal = false }) {
                                Text(strings.common.cancelAction)
                            }
                        }
                    )
                }
            }
        }
    }
}
