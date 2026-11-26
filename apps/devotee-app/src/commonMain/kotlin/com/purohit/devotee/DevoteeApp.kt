package com.purohit.devotee

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.runtime.saveable.rememberSaveable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.purohit.contracts.cancellation.CancellationQuote
import com.purohit.contracts.i18n.AppLanguage
import com.purohit.contracts.i18n.LocalizationManager
import com.purohit.contracts.idempotency.IdempotencyKeyGenerator
import com.purohit.contracts.i18n.StringContract
import com.purohit.devotee.network.NearbyPriestItem
import com.purohit.devotee.ui.history.BookingHistoryScreen
import com.purohit.devotee.ui.history.isActiveBooking
import com.purohit.devotee.network.PurohitApiClient
import com.purohit.devotee.ui.auth.DevoteeLoginScreen
import com.purohit.devotee.ui.auth.DevoteeRegistrationScreen
import com.purohit.devotee.ui.cancellation.SwiggyCancellationModal
import com.purohit.devotee.ui.catalog.PoojaCatalogScreen
import com.purohit.devotee.ui.chat.ChatScreen
import com.purohit.devotee.ui.dispatch.RapidoNearbyPriestsScreen
import com.purohit.devotee.ui.profile.DevoteeProfileScreen
import com.purohit.devotee.ui.resilience.LogoutConfirmationModal
import com.purohit.devotee.ui.resilience.StickyOfflineBanner
import com.purohit.devotee.ui.store.PujaStoreScreen
import com.purohit.devotee.ui.temple.TempleDirectoryScreen
import com.purohit.devotee.ui.tracking.LiveTrackingScreen
import com.purohit.devotee.event.EventConfigStore
import com.purohit.devotee.ui.pushkaralu.EventStrip
import com.purohit.devotee.ui.pushkaralu.PushkaraluScreen
import com.purohit.devotee.ui.pushkaralu.GhatPriestsScreen
import com.purohit.devotee.ui.pushkaralu.EventRequestScreen
import com.purohit.devotee.ui.pushkaralu.EventBookingScreen
import com.purohit.contracts.event.Ghat
import com.purohit.contracts.event.GhatPriest
import com.purohit.contracts.event.EventBookingResponse
import kotlinx.coroutines.launch

val LocalAppStrings = compositionLocalOf<StringContract> {
    LocalizationManager.getStrings(AppLanguage.ENGLISH)
}

val LocalAppLanguage = staticCompositionLocalOf { AppLanguage.ENGLISH }

enum class DevoteeScreen {
    LOGIN,
    REGISTRATION,
    SERVICES,       // Tab 1: 🪔 Services & Catalog
    TEMPLES,        // Tab 2: 🏛️ Sacred Temples & VIP Ticketing
    PUJA_STORE,     // Tab 3: 🛒 Puja Store & Local Vendors
    PUSHKARALU,     // Tab 3 while an event is live (replaces Store; PRD v2 defers the store)
    PUSHKARALU_PRIESTS,
    PUSHKARALU_REQUEST,
    PUSHKARALU_BOOKING,
    BOOKING_HISTORY,
    BOOKINGS_RADAR, // Tab 4: 📍 My Bookings & Live Radar
    LIVE_TRACKING,  // 🛵 Live Traveling & GNSS Carrier Accuracy
    CHAT,           // 💬 Live Chat with Assigned Purohit
    PROFILE         // Tab 5: 👤 Vedic Sankalpam Profile & Language
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun DevoteeAppRoot() {
    var isAuthenticated by remember { mutableStateOf(false) }
    var currentScreen by remember { mutableStateOf(DevoteeScreen.LOGIN) }
    var currentLanguage by rememberSaveable { mutableStateOf(AppLanguage.ENGLISH) }
    val strings = remember(currentLanguage) { LocalizationManager.getStrings(currentLanguage) }

    var selectedPoojaTitle by remember { mutableStateOf("") }
    var selectedDiscoveryNames by remember { mutableStateOf("") }
    var selectedPoojaAmount by remember { mutableStateOf(0.0) }
    var activePoojaId by remember { mutableStateOf("") }
    var activeBookingId by remember { mutableStateOf("") }
    var activeChatRoomId by remember { mutableStateOf("") }
    var activeEventAddress by remember { mutableStateOf("") }
    var activePriest by remember { mutableStateOf<NearbyPriestItem?>(null) }
    var activeOtp by remember { mutableStateOf("") }
    var showLogoutModal by remember { mutableStateOf(false) }
    var logoutHasActiveRitual by remember { mutableStateOf(false) }
    var logoutCheckFailed by remember { mutableStateOf(false) }
    var logoutCheckInProgress by remember { mutableStateOf(false) }
    var showLanguageModal by remember { mutableStateOf(false) }
    var showAppearance by remember { mutableStateOf(false) }
    var appearance by remember { mutableStateOf(appearanceFromStoredValue(readAppearance())) }
    var showCancellationModal by remember { mutableStateOf(false) }
    var cancellationBusy by remember { mutableStateOf(false) }
    var cancellationIntentKey by remember { mutableStateOf(IdempotencyKeyGenerator.generate()) }
    var activeCancellationQuote by remember { mutableStateOf<CancellationQuote?>(null) }
    var isOffline by remember { mutableStateOf(false) }
    var restorationError by remember { mutableStateOf<String?>(null) }
    val coroutineScope = rememberCoroutineScope()
    val eventConfig by EventConfigStore.config.collectAsState()
    val eventLive = eventConfig?.isLive == true
    var selectedGhat by remember { mutableStateOf<Ghat?>(null) }
    var selectedGhatPriest by remember { mutableStateOf<GhatPriest?>(null) }
    var eventBooking by remember { mutableStateOf<EventBookingResponse?>(null) }
    var chatReturnScreen by remember { mutableStateOf(DevoteeScreen.LIVE_TRACKING) }
    val inEventFlow = currentScreen in setOf(DevoteeScreen.PUSHKARALU, DevoteeScreen.PUSHKARALU_PRIESTS, DevoteeScreen.PUSHKARALU_REQUEST, DevoteeScreen.PUSHKARALU_BOOKING)

    // While the app is open, recheck every 10 minutes (covers iOS, which has no config push yet).
    LaunchedEffect(Unit) {
        while (true) {
            kotlinx.coroutines.delay(10 * 60 * 1000L)
            EventConfigStore.refreshIfOlderThan(kotlin.time.Duration.parse("10m"))
        }
    }

    LaunchedEffect(Unit) {
        EventConfigStore.refreshIfStale()
        // A blip (slow network, server busy) must not log the devotee out: retry while the
        // token is still held. fetchProfile clears the token only on a real 401.
        var attempt = 0
        while (PurohitApiClient.authToken.isNotBlank() && !isAuthenticated) {
            val result = PurohitApiClient.fetchProfile()
            if (result.isSuccess) {
                isAuthenticated = true
                currentScreen = DevoteeScreen.SERVICES
            } else if (PurohitApiClient.authToken.isNotBlank() && ++attempt < 4) {
                kotlinx.coroutines.delay(2000L * attempt)
            } else {
                // Token cleared means the session really ended: just show login, no error popup.
                // Token kept means the server could not be reached.
                if (PurohitApiClient.authToken.isNotBlank()) restorationError = when (currentLanguage) {
                    AppLanguage.TELUGU -> "Puja Vaahini సర్వర్‌ను చేరలేకపోయాము. ఇంటర్నెట్ తనిఖీ చేసి యాప్‌ను మళ్లీ తెరవండి."
                    AppLanguage.HINDI -> "Puja Vaahini सर्वर तक नहीं पहुँच सके। इंटरनेट जाँचकर ऐप फिर खोलें।"
                    else -> "Could not reach Puja Vaahini. Check your internet and reopen the app."
                }
                break
            }
        }
    }

    // Resume existing or ongoing booking from PostgreSQL upon authentication
    LaunchedEffect(isAuthenticated) {
        if (isAuthenticated) {
            val userId = PurohitApiClient.currentProfile?.id ?: ""
            if (userId.isNotBlank()) {
                val res = PurohitApiClient.fetchDevoteeBookings(userId)
                res.onSuccess { bookings ->
                    val ongoing = bookings.firstOrNull {
                        it.status in listOf("REQUESTED", "ACCEPTED", "IN_TRANSIT", "ARRIVED", "POOJA_STARTED", "IN_PROGRESS")
                    }
                    if (ongoing != null) {
                        activeBookingId = ongoing.id
                        selectedPoojaTitle = ongoing.pooja_title
                        selectedPoojaAmount = ongoing.total_amount
                        activeOtp = ongoing.completion_otp
                        activeChatRoomId = ongoing.chat_room_id
                        activeEventAddress = ongoing.event_address
                        activePriest = NearbyPriestItem(
                            id = ongoing.priest_id,
                            full_name = ongoing.priest_name,
                            phone = ongoing.priest_phone,
                            sampradaya = "",
                            veda_shakha = "",
                            rating_avg = 0.0,
                            total_completed_poojas = 0,
                            repeat_booking_count = 0,
                            distance_meters = 0.0,
                            distance_km = 0.0,
                            eta_minutes = 0,
                            vehicle_type = ""
                        )
                    }
                }.onFailure { restorationError = "Could not load your bookings. Please check your connection and sign in again." }
            }
        }
    }

    CompositionLocalProvider(LocalAppStrings provides strings, LocalAppLanguage provides currentLanguage) {
        MaterialTheme(
            colorScheme = appearanceScheme(appearance),
            typography = appTypography(),
            shapes = Shapes(small = androidx.compose.foundation.shape.RoundedCornerShape(8.dp), medium = androidx.compose.foundation.shape.RoundedCornerShape(16.dp), large = androidx.compose.foundation.shape.RoundedCornerShape(20.dp))
        ) {
            Scaffold(
                topBar = {
                    if (currentScreen != DevoteeScreen.CHAT) {
                        AppHeader(currentLanguage, onAppearance = { showAppearance = true }, onLanguage = { showLanguageModal = true })
                    }
                },
                bottomBar = {
                    if (isAuthenticated && currentScreen != DevoteeScreen.CHAT) {
                        NavigationBar(containerColor = MaterialTheme.colorScheme.primaryContainer, contentColor = MaterialTheme.colorScheme.onPrimaryContainer) {
                            NavigationBarItem(
                                selected = currentScreen == DevoteeScreen.SERVICES,
                                onClick = { currentScreen = DevoteeScreen.SERVICES },
                                label = { Text(strings.common.tabServices, style = MaterialTheme.typography.labelSmall, maxLines = 2, textAlign = androidx.compose.ui.text.style.TextAlign.Center) },
                                icon = { Text("🪔") }
                            )
                            NavigationBarItem(
                                selected = currentScreen == DevoteeScreen.TEMPLES,
                                onClick = { currentScreen = DevoteeScreen.TEMPLES },
                                label = { Text(strings.common.tabTemples, style = MaterialTheme.typography.labelSmall, maxLines = 2, textAlign = androidx.compose.ui.text.style.TextAlign.Center) },
                                icon = { Text("🏛️") }
                            )
                            if (eventLive || inEventFlow) NavigationBarItem(
                                selected = inEventFlow,
                                onClick = { currentScreen = DevoteeScreen.PUSHKARALU },
                                // ponytail: one long word ("Pushkaralu") breaks mid-word at large font scales; keep it at 1.0x on one line.
                                label = { val fs = androidx.compose.ui.platform.LocalDensity.current.fontScale; Text(strings.event.tabEvent, style = MaterialTheme.typography.labelSmall.let { it.copy(fontSize = it.fontSize * (minOf(fs, 1f) / fs)) }, maxLines = 1, softWrap = false, textAlign = androidx.compose.ui.text.style.TextAlign.Center) },
                                icon = { Text("🌊") }
                            ) else NavigationBarItem(
                                selected = currentScreen == DevoteeScreen.PUJA_STORE,
                                onClick = { currentScreen = DevoteeScreen.PUJA_STORE },
                                label = { Text(strings.common.tabPujaStore, style = MaterialTheme.typography.labelSmall, maxLines = 2, textAlign = androidx.compose.ui.text.style.TextAlign.Center) },
                                icon = { Text("🛒") }
                            )
                            NavigationBarItem(
                                selected = currentScreen == DevoteeScreen.BOOKING_HISTORY || currentScreen == DevoteeScreen.BOOKINGS_RADAR || currentScreen == DevoteeScreen.LIVE_TRACKING,
                                onClick = { currentScreen = DevoteeScreen.BOOKING_HISTORY },
                                label = { Text(strings.common.tabBookings, style = MaterialTheme.typography.labelSmall, maxLines = 2, textAlign = androidx.compose.ui.text.style.TextAlign.Center) },
                                icon = { Text(if (currentScreen == DevoteeScreen.LIVE_TRACKING) "🛵" else "📍") }
                            )
                            NavigationBarItem(
                                selected = currentScreen == DevoteeScreen.PROFILE,
                                onClick = { currentScreen = DevoteeScreen.PROFILE },
                                label = { Text(strings.common.tabProfile, style = MaterialTheme.typography.labelSmall, maxLines = 2, textAlign = androidx.compose.ui.text.style.TextAlign.Center) },
                                icon = { Text("👤") }
                            )
                        }
                    }
                }
            ) { paddingValues ->
                Column(modifier = Modifier.padding(paddingValues).fillMaxSize()) {
                    // Sticky Offline Banner
                    StickyOfflineBanner(isOffline = isOffline, strings = strings)

                    Box(modifier = Modifier.weight(1f).fillMaxWidth()) {
                        when (currentScreen) {
                            DevoteeScreen.LOGIN -> DevoteeLoginScreen(
                                strings = strings,
                                onLoginSuccess = {
                                    isAuthenticated = true
                                    currentScreen = DevoteeScreen.SERVICES
                                },
                                onNavigateToRegister = {
                                    currentScreen = DevoteeScreen.REGISTRATION
                                }
                            )
                            DevoteeScreen.REGISTRATION -> DevoteeRegistrationScreen(
                                onRegistrationSuccess = { _, _ ->
                                    isAuthenticated = true
                                    currentScreen = DevoteeScreen.SERVICES
                                },
                                onNavigateToLogin = {
                                    currentScreen = DevoteeScreen.LOGIN
                                }
                            )
                            DevoteeScreen.SERVICES -> Column(Modifier.fillMaxSize()) {
                                eventConfig?.takeIf { it.isLive }?.let { event ->
                                    EventStrip(event, currentLanguage, strings.event, onOpen = { currentScreen = DevoteeScreen.PUSHKARALU })
                                }
                                Box(Modifier.weight(1f).fillMaxWidth()) {
                                    PoojaCatalogScreen(
                                        currentLanguage = currentLanguage,
                                        strings = strings,
                                        onSelectPoojaForDispatch = { title, id, amount, discoveryNames ->
                                            selectedPoojaTitle = title
                                            selectedDiscoveryNames = discoveryNames
                                            activePoojaId = id
                                            selectedPoojaAmount = amount
                                            currentScreen = DevoteeScreen.BOOKINGS_RADAR
                                        }
                                    )
                                }
                            }
                            DevoteeScreen.PUSHKARALU -> PushkaraluScreen(eventConfig, currentLanguage, strings.event, onOpenGhat = {
                                selectedGhat = it
                                currentScreen = DevoteeScreen.PUSHKARALU_PRIESTS
                            })
                            DevoteeScreen.PUSHKARALU_PRIESTS -> {
                                val ev = eventConfig; val ghat = selectedGhat
                                if (ev == null || ghat == null) PushkaraluScreen(ev, currentLanguage, strings.event, onOpenGhat = { selectedGhat = it })
                                else GhatPriestsScreen(ev, ghat, currentLanguage, strings.event,
                                    onBack = { currentScreen = DevoteeScreen.PUSHKARALU },
                                    onRequest = { selectedGhatPriest = it; currentScreen = DevoteeScreen.PUSHKARALU_REQUEST })
                            }
                            DevoteeScreen.PUSHKARALU_REQUEST -> {
                                val ev = eventConfig; val ghat = selectedGhat; val priest = selectedGhatPriest
                                if (ev == null || ghat == null || priest == null) PushkaraluScreen(ev, currentLanguage, strings.event, onOpenGhat = { selectedGhat = it; currentScreen = DevoteeScreen.PUSHKARALU_PRIESTS })
                                else EventRequestScreen(ev, ghat, priest, currentLanguage, strings.event,
                                    onBack = { currentScreen = DevoteeScreen.PUSHKARALU_PRIESTS },
                                    onBooked = { eventBooking = it; currentScreen = DevoteeScreen.PUSHKARALU_BOOKING })
                            }
                            DevoteeScreen.PUSHKARALU_BOOKING -> eventBooking?.let { b ->
                                EventBookingScreen(b, selectedGhat?.displayName(currentLanguage).orEmpty(), selectedGhatPriest?.fullName.orEmpty(), strings.event,
                                    onChat = { roomId ->
                                        activeChatRoomId = roomId
                                        activePriest = NearbyPriestItem(id = "", full_name = selectedGhatPriest?.fullName.orEmpty(), phone = "", sampradaya = "", veda_shakha = "",
                                            rating_avg = 0.0, total_completed_poojas = 0, repeat_booking_count = 0, distance_meters = 0.0, distance_km = 0.0, eta_minutes = 0, vehicle_type = "")
                                        activeEventAddress = selectedGhat?.name.orEmpty()
                                        chatReturnScreen = DevoteeScreen.PUSHKARALU_BOOKING
                                        currentScreen = DevoteeScreen.CHAT
                                    },
                                    onDone = { eventBooking = null; currentScreen = DevoteeScreen.PUSHKARALU })
                            } ?: PushkaraluScreen(eventConfig, currentLanguage, strings.event, onOpenGhat = { selectedGhat = it; currentScreen = DevoteeScreen.PUSHKARALU_PRIESTS })
                            DevoteeScreen.TEMPLES -> TempleDirectoryScreen(
                                strings = strings,
                                onBookVipPuja = { _, _ ->
                                    currentScreen = DevoteeScreen.BOOKINGS_RADAR
                                }
                            )
                            DevoteeScreen.PUJA_STORE -> PujaStoreScreen(
                                strings = strings,
                                onOrderPlaced = { _, _ ->
                                    currentScreen = DevoteeScreen.BOOKINGS_RADAR
                                }
                            )
                            DevoteeScreen.BOOKING_HISTORY -> BookingHistoryScreen(
                                onBrowse = { currentScreen = DevoteeScreen.SERVICES },
                                onOpenActive = { booking ->
                                    activeBookingId = booking.id
                                    selectedPoojaTitle = booking.pooja_title
                                    selectedPoojaAmount = booking.total_amount
                                    activeOtp = booking.completion_otp
                                    activeChatRoomId = booking.chat_room_id
                                    activeEventAddress = booking.event_address
                                    activePriest = NearbyPriestItem(
                                        id = booking.priest_id, full_name = booking.priest_name,
                                        phone = booking.priest_phone, sampradaya = "", veda_shakha = "",
                                        rating_avg = 0.0, total_completed_poojas = 0, repeat_booking_count = 0,
                                        distance_meters = 0.0, distance_km = 0.0, eta_minutes = 0, vehicle_type = ""
                                    )
                                    currentScreen = DevoteeScreen.LIVE_TRACKING
                                }
                            )
                            DevoteeScreen.BOOKINGS_RADAR -> if (activePoojaId.isBlank()) {
                                Column(modifier = Modifier.fillMaxSize().padding(24.dp), verticalArrangement = Arrangement.Center, horizontalAlignment = Alignment.CenterHorizontally) {
                                    Text("Choose a puja before requesting a priest.")
                                    Button(onClick = { currentScreen = DevoteeScreen.SERVICES }) { Text("Browse pujas") }
                                }
                            } else RapidoNearbyPriestsScreen(
                                poojaTitle = selectedPoojaTitle,
                                discoveryNames = selectedDiscoveryNames,
                                totalAmount = selectedPoojaAmount,
                                poojaId = activePoojaId,
                                strings = strings,
                                onPriestBooked = { priest, otp, bId, roomId ->
                                    activePriest = priest
                                    activeOtp = otp
                                    if (bId.isNotBlank()) activeBookingId = bId
                                    if (roomId.isNotBlank()) activeChatRoomId = roomId
                                    if (bId.isNotBlank()) coroutineScope.launch {
                                        PurohitApiClient.fetchBookingDetails(bId).onSuccess { activeEventAddress = it.event_address }
                                    }
                                    currentScreen = DevoteeScreen.LIVE_TRACKING
                                }
                            )
                            DevoteeScreen.LIVE_TRACKING -> LiveTrackingScreen(
                                priestName = activePriest?.full_name ?: "Priest",
                                priestPhone = activePriest?.phone ?: "",
                                priestSampradaya = "${activePriest?.sampradaya.orEmpty()} ${activePriest?.veda_shakha.orEmpty()}",
                                poojaTitle = selectedPoojaTitle,
                                totalAmount = selectedPoojaAmount,
                                completionOtp = activeOtp,
                                initialDistanceKm = activePriest?.distance_km ?: 0.0,
                                bookingId = activeBookingId,
                                chatRoomId = activeChatRoomId,
                                poojaId = activePoojaId,
                                eventAddress = activeEventAddress,
                                onOpenChat = { currentScreen = DevoteeScreen.CHAT },
                                onOpenLiveStream = {},
                                onCancelBooking = {
                                    coroutineScope.launch {
                                        val quoteRes = PurohitApiClient.fetchCancellationQuote(activeBookingId)
                                        quoteRes.onSuccess { q ->
                                            activeCancellationQuote = CancellationQuote(
                                                bookingId = q.booking_id,
                                                currentStatus = q.current_status,
                                                advancePaid = q.advance_paid,
                                                cancellationFee = q.cancellation_fee,
                                                travelFee = q.travel_fee,
                                                priestCompensation = q.priest_compensation,
                                                refundAmount = q.refund_amount,
                                                isFreeCancellation = q.is_free_cancellation,
                                                policyNotice = q.policy_notice
                                            )
                                            showCancellationModal = true
                                        }.onFailure {
                                            restorationError = "Could not load cancellation details. Please try again."

                                        }
                                    }
                                },
                                onBackToHome = {
                                    currentScreen = DevoteeScreen.SERVICES
                                }
                            )
                            DevoteeScreen.CHAT -> ChatScreen(
                                roomId = activeChatRoomId,
                                priestName = activePriest?.full_name ?: "Priest",
                                priestPhone = activePriest?.phone ?: "",
                                eventAddress = activeEventAddress,
                                onBack = { currentScreen = chatReturnScreen; chatReturnScreen = DevoteeScreen.LIVE_TRACKING }
                            )
                            DevoteeScreen.PROFILE -> DevoteeProfileScreen(
                                onLogout = {
                                    if (!logoutCheckInProgress) {
                                        logoutCheckInProgress = true
                                        coroutineScope.launch {
                                            val result = PurohitApiClient.fetchDevoteeBookings(PurohitApiClient.currentProfile?.id.orEmpty())
                                            logoutHasActiveRitual = result.getOrNull()?.any { isActiveBooking(it.status) } == true
                                            logoutCheckFailed = result.isFailure
                                            logoutCheckInProgress = false
                                            showLogoutModal = true
                                        }
                                    }
                                }
                            )
                        }
                    }
                }

                restorationError?.let { error -> AlertDialog(onDismissRequest = { restorationError = null }, title = { Text(when (currentLanguage) { AppLanguage.TELUGU -> "సమస్య ఎదురైంది"; AppLanguage.HINDI -> "समस्या हुई"; else -> "Something went wrong" }) }, text = { Text(error) }, confirmButton = { TextButton(onClick = { restorationError = null }) { Text("OK") } }) }

                // 1. Language Switcher Radio Modal
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

                // 2. Logout Confirmation Modal
                if (showLogoutModal) {
                    LogoutConfirmationModal(
                        hasActiveRitual = logoutHasActiveRitual,
                        checkFailedMessage = if (!logoutCheckFailed) null else when (currentLanguage) {
                            AppLanguage.TELUGU -> "బుకింగ్ స్థితిని తనిఖీ చేయలేకపోయాము. లాగ్ అవుట్ చేసినా బుకింగ్‌లు రద్దు కావు."
                            AppLanguage.HINDI -> "बुकिंग की स्थिति जाँची नहीं जा सकी। लॉग आउट करने से बुकिंग रद्द नहीं होंगी।"
                            else -> "Could not check booking status. Logging out does not cancel bookings."
                        },
                        strings = strings,
                        onConfirmLogout = {
                            showLogoutModal = false
                            PurohitApiClient.authToken = ""
                            PurohitApiClient.currentProfile = null
                            isAuthenticated = false
                            activeBookingId = ""
                            activePoojaId = ""
                            activeChatRoomId = ""
                            activeEventAddress = ""
                            activeOtp = ""
                            activePriest = null
                            selectedPoojaTitle = ""
                            selectedPoojaAmount = 0.0
                            currentScreen = DevoteeScreen.LOGIN
                        },
                        onDismiss = { showLogoutModal = false }
                    )
                }

                // 3. Swiggy Cancellation Bottom Sheet
                if (showCancellationModal && activeCancellationQuote != null) {
                    SwiggyCancellationModal(
                        quote = activeCancellationQuote!!,
                        strings = strings,
                        isSubmitting = cancellationBusy,
                        onConfirmCancel = { reason ->
                            if (!cancellationBusy) {
                                cancellationBusy = true
                                coroutineScope.launch {
                                    PurohitApiClient.cancelBooking(
                                        bookingId = activeBookingId,
                                        reasonCode = reason,
                                        reasonText = reason,
                                        idempotencyKey = cancellationIntentKey
                                    ).onSuccess {
                                        cancellationIntentKey = IdempotencyKeyGenerator.generate()
                                        showCancellationModal = false
                                        activeBookingId = ""
                                        activePriest = null
                                        currentScreen = DevoteeScreen.BOOKING_HISTORY
                                    }.onFailure {
                                        restorationError = when (currentLanguage) {
                                            AppLanguage.TELUGU -> "బుకింగ్ రద్దు కాలేదు. కనెక్షన్ తనిఖీ చేసి మళ్లీ ప్రయత్నించండి."
                                            AppLanguage.HINDI -> "बुकिंग रद्द नहीं हुई। कनेक्शन जाँचकर फिर प्रयास करें।"
                                            else -> "Could not confirm cancellation. Check your connection and try again."
                                        }
                                    }
                                    cancellationBusy = false
                                }
                            }
                        },
                        onDismiss = { showCancellationModal = false }
                    )
                }
            }
        }
    }
}
