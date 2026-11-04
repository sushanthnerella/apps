package com.purohit.devotee

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.purohit.contracts.cancellation.CancellationQuote
import com.purohit.contracts.i18n.AppLanguage
import com.purohit.contracts.i18n.LocalizationManager
import com.purohit.contracts.i18n.StringContract
import com.purohit.devotee.network.NearbyPriestItem
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
import kotlinx.coroutines.launch

val LocalAppStrings = compositionLocalOf<StringContract> {
    LocalizationManager.getStrings(AppLanguage.ENGLISH)
}

enum class DevoteeScreen {
    LOGIN,
    REGISTRATION,
    SERVICES,       // Tab 1: 🪔 Services & Catalog
    TEMPLES,        // Tab 2: 🏛️ Sacred Temples & VIP Ticketing
    PUJA_STORE,     // Tab 3: 🛒 Puja Store & Local Vendors
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
    var currentLanguage by remember { mutableStateOf(AppLanguage.ENGLISH) }
    val strings = remember(currentLanguage) { LocalizationManager.getStrings(currentLanguage) }

    var selectedPoojaTitle by remember { mutableStateOf("Sri Satyanarayana Swamy Vratam") }
    var selectedPoojaAmount by remember { mutableStateOf(3716.0) }
    var activePoojaId by remember { mutableStateOf("5dccda01-fcb9-4bc2-9ef9-6015419533bc") }
    var activeBookingId by remember { mutableStateOf("f312db8f-1d71-4cab-b5ba-77fbead70c71") }
    var activeChatRoomId by remember { mutableStateOf("d66283f3-1eb0-40b3-afc0-c37d355b90f8") }
    var activeEventAddress by remember { mutableStateOf("Plot 42, Sri Rama Nagar, Madhav Nagar, Hyderabad (Landmark: Behind Sri Rama Temple)") }
    var activePriest by remember { mutableStateOf<NearbyPriestItem?>(null) }
    var activeOtp by remember { mutableStateOf("4921") }
    var showLogoutModal by remember { mutableStateOf(false) }
    var showLanguageModal by remember { mutableStateOf(false) }
    var showCancellationModal by remember { mutableStateOf(false) }
    var activeCancellationQuote by remember { mutableStateOf<CancellationQuote?>(null) }
    var isOffline by remember { mutableStateOf(false) }
    val coroutineScope = rememberCoroutineScope()

    CompositionLocalProvider(LocalAppStrings provides strings) {
        MaterialTheme(
            colorScheme = lightColorScheme(
                primary = Color(0xFFD97706), // Vedic saffron/amber
                secondary = Color(0xFFB45309),
                background = Color(0xFFFFFBEB)
            )
        ) {
            Scaffold(
                topBar = {
                    if (isAuthenticated) {
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
                                    text = "🕉️ Puja Vaahini",
                                    color = Color.White,
                                    fontWeight = FontWeight.Bold,
                                    style = MaterialTheme.typography.titleMedium
                                )
                                // Language Switcher Chip
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
                    if (isAuthenticated && currentScreen != DevoteeScreen.CHAT) {
                        NavigationBar {
                            NavigationBarItem(
                                selected = currentScreen == DevoteeScreen.SERVICES,
                                onClick = { currentScreen = DevoteeScreen.SERVICES },
                                label = { Text(strings.common.tabServices) },
                                icon = { Text("🪔") }
                            )
                            NavigationBarItem(
                                selected = currentScreen == DevoteeScreen.TEMPLES,
                                onClick = { currentScreen = DevoteeScreen.TEMPLES },
                                label = { Text(strings.common.tabTemples) },
                                icon = { Text("🏛️") }
                            )
                            NavigationBarItem(
                                selected = currentScreen == DevoteeScreen.PUJA_STORE,
                                onClick = { currentScreen = DevoteeScreen.PUJA_STORE },
                                label = { Text(strings.common.tabPujaStore) },
                                icon = { Text("🛒") }
                            )
                            NavigationBarItem(
                                selected = currentScreen == DevoteeScreen.BOOKINGS_RADAR || currentScreen == DevoteeScreen.LIVE_TRACKING,
                                onClick = {
                                    currentScreen = if (activePriest != null) DevoteeScreen.LIVE_TRACKING else DevoteeScreen.BOOKINGS_RADAR
                                },
                                label = { Text(if (currentScreen == DevoteeScreen.LIVE_TRACKING) "Tracking" else strings.common.tabBookings) },
                                icon = { Text(if (currentScreen == DevoteeScreen.LIVE_TRACKING) "🛵" else "📍") }
                            )
                            NavigationBarItem(
                                selected = currentScreen == DevoteeScreen.PROFILE,
                                onClick = { currentScreen = DevoteeScreen.PROFILE },
                                label = { Text(strings.common.tabProfile) },
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
                            DevoteeScreen.SERVICES -> PoojaCatalogScreen(
                                currentLanguage = currentLanguage,
                                strings = strings,
                                onSelectPoojaForDispatch = { title, id, amount ->
                                    selectedPoojaTitle = title
                                    activePoojaId = id
                                    selectedPoojaAmount = amount
                                    currentScreen = DevoteeScreen.BOOKINGS_RADAR
                                }
                            )
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
                            DevoteeScreen.BOOKINGS_RADAR -> RapidoNearbyPriestsScreen(
                                poojaTitle = selectedPoojaTitle,
                                totalAmount = selectedPoojaAmount,
                                poojaId = activePoojaId,
                                onPriestBooked = { priest, otp, bId, roomId ->
                                    activePriest = priest
                                    activeOtp = otp
                                    if (bId.isNotBlank()) activeBookingId = bId
                                    if (roomId.isNotBlank()) activeChatRoomId = roomId
                                    currentScreen = DevoteeScreen.LIVE_TRACKING
                                }
                            )
                            DevoteeScreen.LIVE_TRACKING -> LiveTrackingScreen(
                                priestName = activePriest?.full_name ?: "Brahmasri Sushanth Sastry",
                                priestPhone = activePriest?.phone ?: "9876543210",
                                priestSampradaya = "${activePriest?.sampradaya ?: "Smartha"} • ${activePriest?.veda_shakha ?: "Rigveda Shakha"}",
                                poojaTitle = selectedPoojaTitle,
                                totalAmount = selectedPoojaAmount,
                                completionOtp = activeOtp,
                                initialDistanceKm = activePriest?.distance_km ?: 1.8,
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
                                            activeCancellationQuote = CancellationQuote(
                                                bookingId = activeBookingId,
                                                currentStatus = "IN_TRANSIT",
                                                advancePaid = selectedPoojaAmount,
                                                cancellationFee = 100.0,
                                                travelFee = 50.0,
                                                priestCompensation = 50.0,
                                                refundAmount = (selectedPoojaAmount - 100.0).coerceAtLeast(0.0),
                                                isFreeCancellation = false,
                                                policyNotice = "Purohit is en route. Cancellation fee applies for transit compensation."
                                            )
                                            showCancellationModal = true
                                        }
                                    }
                                },
                                onBackToHome = {
                                    currentScreen = DevoteeScreen.SERVICES
                                }
                            )
                            DevoteeScreen.CHAT -> ChatScreen(
                                roomId = activeChatRoomId,
                                priestName = activePriest?.full_name ?: "Brahmasri Sushanth Sastry",
                                priestPhone = activePriest?.phone ?: "9876543210",
                                eventAddress = activeEventAddress,
                                onBack = { currentScreen = DevoteeScreen.LIVE_TRACKING }
                            )
                            DevoteeScreen.PROFILE -> DevoteeProfileScreen(
                                onLogout = { showLogoutModal = true }
                            )
                        }
                    }
                }

                // 1. Language Switcher Radio Modal
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
                        hasActiveRitual = true,
                        strings = strings,
                        onConfirmLogout = {
                            showLogoutModal = false
                            isAuthenticated = false
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
                        onConfirmCancel = { reason ->
                            coroutineScope.launch {
                                PurohitApiClient.cancelBooking(
                                    bookingId = activeBookingId,
                                    reasonCode = reason,
                                    reasonText = reason
                                )
                            }
                            showCancellationModal = false
                            currentScreen = DevoteeScreen.SERVICES
                        },
                        onRescheduleInstead = {
                            showCancellationModal = false
                            currentScreen = DevoteeScreen.SERVICES
                        },
                        onDismiss = { showCancellationModal = false }
                    )
                }
            }
        }
    }
}
