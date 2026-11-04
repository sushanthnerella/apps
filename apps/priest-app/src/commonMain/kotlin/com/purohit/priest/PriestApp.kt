package com.purohit.priest

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.purohit.contracts.i18n.AppLanguage
import com.purohit.contracts.i18n.LocalizationManager
import com.purohit.contracts.i18n.StringContract
import com.purohit.priest.network.PurohitApiClient
import com.purohit.priest.ui.auth.PriestRegistrationScreen
import com.purohit.priest.ui.chat.PriestChatScreen
import com.purohit.priest.ui.dispatch.RapidoIncomingAlertModal
import com.purohit.priest.ui.duty.DutyToggleScreen
import com.purohit.priest.ui.materials.PriestSamagriChecklistModal
import com.purohit.priest.ui.referral.ColleagueReferralModal
import com.purohit.priest.ui.schedule.PriestScheduleScreen
import com.purohit.priest.ui.services.PriestServicesScreen
import com.purohit.priest.ui.studio.BroadcasterStudioScreen
import kotlinx.coroutines.launch

val LocalPriestStrings = compositionLocalOf<StringContract> {
    LocalizationManager.getStrings(AppLanguage.ENGLISH)
}

enum class PriestScreen {
    REGISTRATION,
    DUTY_RADAR,         // Tab 1: ⚡ Radar & Duty Dashboard
    SCHEDULE_WORKBENCH, // Tab 2: 📅 Schedule Workbench
    CUSTOM_SERVICES,    // Tab 3: 📋 My Services & Dynamic Dakshina
    LIVE_STUDIO,        // Tab 4: 📹 Live Studio & Active Rituals
    CHAT,               // 💬 Bidirectional Chat with Devotee
    PROFILE             // Tab 5: 👤 Vedic Credentials & Language
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun PriestAppRoot() {
    var isRegistered by remember { mutableStateOf(true) } // Verified priest
    var currentScreen by remember { mutableStateOf(PriestScreen.DUTY_RADAR) }
    var priestName by remember { mutableStateOf("Brahmasri Sushanth Sastry") }
    var priestId by remember { mutableStateOf("6e7d277c-4ee2-469f-96ac-331517389762") }

    // Active booking details connected to PostgreSQL DB
    var hasActiveBooking by remember { mutableStateOf(true) }
    var activeBookingId by remember { mutableStateOf("f312db8f-1d71-4cab-b5ba-77fbead70c71") }
    var activeChatRoomId by remember { mutableStateOf("d66283f3-1eb0-40b3-afc0-c37d355b90f8") }
    var activeDevoteeName by remember { mutableStateOf("Sushanth2") }
    var activeDevoteePhone by remember { mutableStateOf("9177961089") }
    var activeAddress by remember { mutableStateOf("Plot 42, Sri Rama Nagar, Madhav Nagar, Hyderabad (Landmark: Behind Sri Rama Temple)") }
    var activePoojaTitle by remember { mutableStateOf("Sri Satyanarayana Swamy Vratam") }
    var activeDakshina by remember { mutableStateOf(3716.0) }
    var activeScheduledDate by remember { mutableStateOf("Tomorrow (Nov 4)") }
    var activeSlotTime by remember { mutableStateOf("Abhijit Muhurtham (11:45 AM - 01:45 PM)") }
    var activeDevoteeGotram by remember { mutableStateOf("Kashyapa") }
    var activeDevoteeNakshatram by remember { mutableStateOf("Rohini") }
    var activeDevoteeRashi by remember { mutableStateOf("Vrishabha") }
    var activeSankalpamPurpose by remember { mutableStateOf("Family Prosperity & Health") }

    val coroutineScope = rememberCoroutineScope()

    LaunchedEffect(activeBookingId) {
        val res = PurohitApiClient.fetchBookingDetails(activeBookingId)
        res.onSuccess { b ->
            if (b.devotee_name.isNotEmpty()) activeDevoteeName = b.devotee_name
            if (b.devotee_phone.isNotEmpty()) activeDevoteePhone = b.devotee_phone
            if (b.event_address.isNotEmpty()) activeAddress = b.event_address
            if (b.pooja_title.isNotEmpty()) activePoojaTitle = b.pooja_title
            if (b.total_amount > 0) activeDakshina = b.total_amount
            if (b.chat_room_id.isNotEmpty()) activeChatRoomId = b.chat_room_id
            if (b.scheduled_date.isNotEmpty()) activeScheduledDate = b.scheduled_date
            if (b.slot_time.isNotEmpty()) activeSlotTime = b.slot_time
            if (b.devotee_gotram.isNotEmpty()) activeDevoteeGotram = b.devotee_gotram
            if (b.devotee_nakshatram.isNotEmpty()) activeDevoteeNakshatram = b.devotee_nakshatram
            if (b.devotee_rashi.isNotEmpty()) activeDevoteeRashi = b.devotee_rashi
            if (b.sankalpam_purpose.isNotEmpty()) activeSankalpamPurpose = b.sankalpam_purpose
        }
    }

    var hasSimulatedIncomingAlert by remember { mutableStateOf(false) }
    var showChecklistModal by remember { mutableStateOf(false) }
    var prefilledChatMessage by remember { mutableStateOf("") }
    var showReferralModal by remember { mutableStateOf(false) }
    var showLanguageModal by remember { mutableStateOf(false) }
    var showLogoutModal by remember { mutableStateOf(false) }

    var currentLanguage by remember { mutableStateOf(AppLanguage.TELUGU) } // Default Telugu for priests
    val strings = remember(currentLanguage) { LocalizationManager.getStrings(currentLanguage) }

    CompositionLocalProvider(LocalPriestStrings provides strings) {
        MaterialTheme(
            colorScheme = lightColorScheme(
                primary = Color(0xFFB45309), // Deep Ochre / Gerua
                secondary = Color(0xFFD97706),
                background = Color(0xFFFFFBEB)
            )
        ) {
            Box(modifier = Modifier.fillMaxSize()) {
                Scaffold(
                    topBar = {
                        if (isRegistered && currentScreen != PriestScreen.CHAT) {
                            Surface(
                                color = MaterialTheme.colorScheme.primary,
                                modifier = Modifier.fillMaxWidth()
                            ) {
                                Row(
                                    modifier = Modifier
                                        .fillMaxWidth()
                                        .padding(horizontal = 16.dp, vertical = 10.dp),
                                    horizontalArrangement = Arrangement.SpaceBetween,
                                    verticalAlignment = Alignment.CenterVertically
                                ) {
                                    Text(
                                        text = "🕉️ అయ్యగారు (Purohit Portal)",
                                        color = Color.White,
                                        fontWeight = FontWeight.Bold,
                                        style = MaterialTheme.typography.titleMedium
                                    )
                                    Surface(
                                        shape = MaterialTheme.shapes.small,
                                        color = Color.White.copy(alpha = 0.2f),
                                        modifier = Modifier.clickable { showLanguageModal = true }
                                    ) {
                                        Text(
                                            text = "🌐 ${currentLanguage.nativeName}",
                                            color = Color.White,
                                            fontWeight = FontWeight.Bold,
                                            style = MaterialTheme.typography.labelSmall,
                                            modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp)
                                        )
                                    }
                                }
                            }
                        }
                    },
                    bottomBar = {
                        if (isRegistered && currentScreen != PriestScreen.CHAT) {
                            NavigationBar {
                                NavigationBarItem(
                                    selected = currentScreen == PriestScreen.DUTY_RADAR,
                                    onClick = { currentScreen = PriestScreen.DUTY_RADAR },
                                    label = { Text(strings.common.tabDuty) },
                                    icon = { Text("⚡") }
                                )
                                NavigationBarItem(
                                    selected = currentScreen == PriestScreen.SCHEDULE_WORKBENCH,
                                    onClick = { currentScreen = PriestScreen.SCHEDULE_WORKBENCH },
                                    label = { Text(strings.common.tabSchedule) },
                                    icon = { Text("📅") }
                                )
                                NavigationBarItem(
                                    selected = currentScreen == PriestScreen.CUSTOM_SERVICES,
                                    onClick = { currentScreen = PriestScreen.CUSTOM_SERVICES },
                                    label = { Text(strings.common.tabCustomServices) },
                                    icon = { Text("📋") }
                                )
                                NavigationBarItem(
                                    selected = currentScreen == PriestScreen.LIVE_STUDIO,
                                    onClick = { currentScreen = PriestScreen.LIVE_STUDIO },
                                    label = { Text("Live Studio") },
                                    icon = { Text("📹") }
                                )
                                NavigationBarItem(
                                    selected = currentScreen == PriestScreen.PROFILE,
                                    onClick = { currentScreen = PriestScreen.PROFILE },
                                    label = { Text(strings.common.tabProfile) },
                                    icon = { Text("👤") }
                                )
                            }
                        }
                    }
                ) { paddingValues ->
                    Box(modifier = Modifier.padding(paddingValues).fillMaxSize()) {
                        when (currentScreen) {
                            PriestScreen.REGISTRATION -> PriestRegistrationScreen(
                                onRegistrationSuccess = { name, _ ->
                                    priestName = name
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
                                currentLanguage = currentLanguage,
                                strings = strings,
                                onOpenChat = { currentScreen = PriestScreen.CHAT },
                                onOpenChecklist = { showChecklistModal = true },
                                onBookingCompleted = { hasActiveBooking = false }
                            )
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
                                    modifier = Modifier.fillMaxSize().padding(20.dp),
                                    horizontalAlignment = Alignment.CenterHorizontally
                                ) {
                                    Text(
                                        text = priestName,
                                        style = MaterialTheme.typography.headlineSmall,
                                        fontWeight = FontWeight.Bold
                                    )
                                    Text(
                                        text = "Rigveda Shakha • Smartha Sampradaya",
                                        style = MaterialTheme.typography.bodyMedium,
                                        color = Color.Gray
                                    )
                                    Spacer(modifier = Modifier.height(20.dp))
                                    Button(
                                        onClick = { showReferralModal = true },
                                        modifier = Modifier.fillMaxWidth()
                                    ) {
                                        Text("🤝 Delegate Booking (Colleague Referral)")
                                    }
                                    Spacer(modifier = Modifier.height(10.dp))
                                    OutlinedButton(
                                        onClick = { showLanguageModal = true },
                                        modifier = Modifier.fillMaxWidth()
                                    ) {
                                        Text("🌐 Change Language (${currentLanguage.nativeName})")
                                    }
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
                        currentLanguage = currentLanguage,
                        strings = strings,
                        onAccept = {
                            coroutineScope.launch {
                                PurohitApiClient.updateBookingStatus(
                                    bookingId = activeBookingId,
                                    status = "ACCEPTED",
                                    priestId = priestId
                                )
                            }
                            hasSimulatedIncomingAlert = false
                            hasActiveBooking = true
                        },
                        onReject = {
                            coroutineScope.launch {
                                PurohitApiClient.updateBookingStatus(
                                    bookingId = activeBookingId,
                                    status = "CANCELLED",
                                    priestId = priestId
                                )
                            }
                            hasSimulatedIncomingAlert = false
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
                        text = { Text("Are you sure you want to log out of duty?") },
                        confirmButton = {
                            Button(
                                onClick = {
                                    showLogoutModal = false
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
