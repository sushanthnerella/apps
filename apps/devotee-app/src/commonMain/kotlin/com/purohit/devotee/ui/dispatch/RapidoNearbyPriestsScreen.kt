package com.purohit.devotee.ui.dispatch

import androidx.compose.animation.core.*
import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.purohit.devotee.LocalAppLanguage
import com.purohit.contracts.i18n.AppLanguage
import com.purohit.contracts.i18n.LocalizationManager
import com.purohit.contracts.i18n.StringContract
import com.purohit.contracts.idempotency.IdempotencyKeyGenerator
import com.purohit.contracts.resilience.JitteredBackoff
import com.purohit.devotee.location.rememberLocationAccess
import com.purohit.devotee.network.PriestSlotPayload
import com.purohit.devotee.network.NearbyPriestItem
import com.purohit.devotee.network.PurohitApiClient
import com.purohit.devotee.ui.resilience.ErrorScreen
import com.purohit.devotee.ui.resilience.NoInternetScreen
import com.purohit.devotee.ui.resilience.VedicLoadingScreen
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch
import kotlinx.datetime.Clock
import kotlinx.datetime.Instant
import kotlinx.datetime.LocalDate
import kotlinx.datetime.TimeZone
import kotlinx.datetime.atStartOfDayIn
import kotlinx.datetime.toLocalDateTime

@Composable
fun RapidoNearbyPriestsScreen(
    poojaTitle: String,
    discoveryNames: String,
    totalAmount: Double,
    poojaId: String = "5dccda01-fcb9-4bc2-9ef9-6015419533bc",
    strings: StringContract = LocalizationManager.getStrings(AppLanguage.ENGLISH),
    onPriestBooked: (priest: NearbyPriestItem, otp: String, bookingId: String, chatRoomId: String) -> Unit
) {
    val language = LocalAppLanguage.current
    fun tr(en: String, te: String, hi: String) = when (language) { AppLanguage.TELUGU -> te; AppLanguage.HINDI -> hi; else -> en }
    var locationPermissionDenied by remember { mutableStateOf(false) }
    var nearbyPriests by remember { mutableStateOf<List<NearbyPriestItem>>(emptyList()) }
    var selectedPriest by remember { mutableStateOf<NearbyPriestItem?>(null) }
    var priestToConfirm by remember { mutableStateOf<NearbyPriestItem?>(null) }
    var isLoading by remember { mutableStateOf(true) }
    var bookedPriest by remember { mutableStateOf<NearbyPriestItem?>(null) }
    var completionOtp by remember { mutableStateOf("") }
    var activeBookingId by remember { mutableStateOf("") }
    var confirmCancel by remember { mutableStateOf(false) }
    var isCancelling by remember { mutableStateOf(false) }
    var activeChatRoomId by remember { mutableStateOf("") }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    val coroutineScope = rememberCoroutineScope()
    val locationAccess = rememberLocationAccess()

    // 1. Date & Muhurtham Selection States
    var eventAddress by remember { mutableStateOf("") }
    var selectedDate by remember { mutableStateOf("") }
    var selectedSlot by remember { mutableStateOf("") }
    var availableSlots by remember { mutableStateOf<List<PriestSlotPayload>>(emptyList()) }
    var chosenSlot by remember { mutableStateOf<PriestSlotPayload?>(null) }
    var slotMessage by remember { mutableStateOf(tr("Choose a priest and select a date.", "పురోహితుడిని, తేదీని ఎంచుకోండి.", "पुरोहित और तारीख चुनें।")) }
    LaunchedEffect(selectedPriest?.id, selectedDate) {
        chosenSlot = null
        selectedSlot = ""
        availableSlots = emptyList()
        val priest = selectedPriest
        if (priest != null && Regex("\\d{4}-\\d{2}-\\d{2}").matches(selectedDate)) {
            slotMessage = tr("Loading availability…", "అందుబాటు పొందుతున్నాము…", "उपलब्धता लोड हो रही है…")
            PurohitApiClient.fetchPriestSlots(priest.id, selectedDate).onSuccess { slots ->
                availableSlots = slots.filter { !it.is_booked && !it.is_held }
                slotMessage = if (availableSlots.isEmpty()) tr("No available slots for this date.", "ఈ తేదీన అందుబాటు సమయాలు లేవు.", "इस तारीख पर कोई समय उपलब्ध नहीं है।") else tr("Select an available time.", "అందుబాటులో ఉన్న సమయాన్ని ఎంచుకోండి.", "उपलब्ध समय चुनें।")
            }.onFailure { slotMessage = tr("Could not load availability. Try again.", "అందుబాటు పొందలేకపోయాము. మళ్లీ ప్రయత్నించండి.", "उपलब्धता नहीं मिली। फिर प्रयास करें।") }
        }
    }
    var devoteeGotram by remember(PurohitApiClient.currentProfile) {
        mutableStateOf(PurohitApiClient.currentProfile?.gotram.orEmpty())
    }
    var devoteeNakshatram by remember(PurohitApiClient.currentProfile) {
        mutableStateOf(PurohitApiClient.currentProfile?.nakshatram.orEmpty())
    }
    var devoteeRashi by remember(PurohitApiClient.currentProfile) {
        mutableStateOf(PurohitApiClient.currentProfile?.rashi.orEmpty())
    }

    // 2. Priest Acceptance Waiting States
    var isAwaitingPriestAcceptance by remember { mutableStateOf(false) }
    var isRequestDispatched by remember { mutableStateOf(false) }
    var slotHoldKey by remember { mutableStateOf(IdempotencyKeyGenerator.generate()) }
    var bookingIntentKey by remember { mutableStateOf(IdempotencyKeyGenerator.generate()) }
    var cancelIntentKey by remember { mutableStateOf(IdempotencyKeyGenerator.generate()) }

    fun sendBookingRequest(targetPriest: NearbyPriestItem) {
        val slot = chosenSlot ?: return
        bookedPriest = targetPriest
        isAwaitingPriestAcceptance = true
        errorMessage = null
        coroutineScope.launch {
            val held = PurohitApiClient.holdPriestSlot(slot.id, idempotencyKey = slotHoldKey)
            if (held.isFailure) { errorMessage = tr("This slot is no longer available. Choose another.", "ఈ సమయం అందుబాటులో లేదు. మరొకటి ఎంచుకోండి.", "यह समय उपलब्ध नहीं है। दूसरा चुनें।"); isAwaitingPriestAcceptance = false; return@launch }
            val res = PurohitApiClient.createBooking(
                priestId = targetPriest.id,
                poojaId = poojaId,
                totalAmount = totalAmount,
                scheduledDate = selectedDate,
                slotTime = selectedSlot,
                slotId = slot.id,
                eventAddress = eventAddress.trim(),
                gotram = devoteeGotram,
                nakshatram = devoteeNakshatram,
                rashi = devoteeRashi,
                purpose = "",
                idempotencyKey = bookingIntentKey
            )
            val booking = res.getOrNull()
            if (booking != null) {
                slotHoldKey = IdempotencyKeyGenerator.generate()
                bookingIntentKey = IdempotencyKeyGenerator.generate()
                completionOtp = booking.completion_otp
                activeBookingId = booking.booking_id

                val roomRes = PurohitApiClient.getChatRoomByBooking(booking.booking_id)
                activeChatRoomId = roomRes.getOrNull()?.room_id ?: ""
            } else {
                errorMessage = tr("Could not send the booking request. Please try again.", "బుకింగ్ అభ్యర్థన పంపలేకపోయాము. మళ్లీ ప్రయత్నించండి.", "बुकिंग अनुरोध नहीं भेजा गया। फिर प्रयास करें।")
                isAwaitingPriestAcceptance = false
            }
        }
    }

    // Poll for Priest Acceptance
    LaunchedEffect(isAwaitingPriestAcceptance, activeBookingId) {
        if (isAwaitingPriestAcceptance && activeBookingId.isNotBlank()) {
            val backoff = JitteredBackoff(baseDelayMs = 5000L, maxDelayMs = 30000L)
            while (isAwaitingPriestAcceptance) {
                backoff.awaitNext()

                val detailRes = PurohitApiClient.fetchBookingDetails(activeBookingId)
                detailRes.onSuccess { details ->
                    backoff.recordSuccess()
                    if (details.status == "ACCEPTED" || details.status == "IN_TRANSIT" || details.status == "ARRIVED" || details.status == "POOJA_STARTED") {
                        isAwaitingPriestAcceptance = false
                        isRequestDispatched = true
                        bookedPriest?.let { p ->
                            onPriestBooked(p, completionOtp, activeBookingId, activeChatRoomId)
                        }
                    } else if (details.status == "CANCELLED" || details.status == "REJECTED") {
                        isAwaitingPriestAcceptance = false
                        errorMessage = if (details.cancellation_reason == "request_expired")
                            tr("The priest did not respond in time. Nothing is charged. Choose another priest or time.", "పురోహితుడు సమయానికి స్పందించలేదు. ఎలాంటి చార్జీ లేదు. మరొక పురోహితుడు లేదా సమయం ఎంచుకోండి.", "पुरोहित ने समय पर जवाब नहीं दिया। कोई शुल्क नहीं लगा। दूसरा पुरोहित या समय चुनें।")
                        else tr("The priest declined. Choose another priest or time.", "పురోహితుడు నిరాకరించారు. మరొక పురోహితుడు లేదా సమయం ఎంచుకోండి.", "पुरोहित ने मना किया। दूसरा पुरोहित या समय चुनें।")
                    } else {
                        errorMessage = null
                    }
                }.onFailure {
                    backoff.recordError()
                    errorMessage = tr("Could not check status. Retrying…", "స్థితి తనిఖీ కాలేదు. మళ్లీ ప్రయత్నిస్తున్నాము…", "स्थिति नहीं मिली। फिर प्रयास हो रहा है…")
                }
            }
        }
    }

    fun loadNearby() {
        isLoading = true
        errorMessage = null
        locationPermissionDenied = false
        nearbyPriests = emptyList()
        selectedPriest = null
        locationAccess.request { location ->
            location.onSuccess { coordinates ->
                coroutineScope.launch {
                    val res = PurohitApiClient.fetchNearbyPriests(coordinates.latitude, coordinates.longitude, discoveryNames)
                    isLoading = false
                    res.onSuccess { list ->
                        nearbyPriests = list
                        selectedPriest = list.firstOrNull()
                        errorMessage = null
                    }.onFailure { errorMessage = tr("Could not load nearby priests. Please try again.", "సమీప పురోహితులను పొందలేకపోయాము. మళ్లీ ప్రయత్నించండి.", "पास के पुरोहित नहीं मिले। फिर प्रयास करें।") }
                }
            }.onFailure {
                isLoading = false
                locationPermissionDenied = it.message.orEmpty().contains("permission", ignoreCase = true)
                errorMessage = if (locationPermissionDenied) tr(
                    "Location permission is off. Allow location for Puja Vaahini in your phone Settings (Apps → Puja Vaahini → Permissions), then tap Try again.",
                    "లొకేషన్ అనుమతి ఆఫ్‌లో ఉంది. ఫోన్ సెట్టింగ్స్‌లో (యాప్స్ → Puja Vaahini → అనుమతులు) లొకేషన్ అనుమతించి, మళ్లీ ప్రయత్నించండి నొక్కండి.",
                    "लोकेशन अनुमति बंद है। फोन की सेटिंग्स (ऐप्स → Puja Vaahini → अनुमतियाँ) में लोकेशन की अनुमति दें, फिर दोबारा प्रयास करें दबाएँ।"
                ) else tr("Could not get your location. Please turn on location and try again.", "మీ స్థానం పొందలేకపోయాము. లొకేషన్ ఆన్ చేసి మళ్లీ ప్రయత్నించండి.", "आपका स्थान नहीं मिला। लोकेशन चालू करके फिर प्रयास करें।")
            }
        }
    }

    LaunchedEffect(poojaTitle) {
        loadNearby()
    }

    if (confirmCancel) {
        AlertDialog(
            onDismissRequest = { confirmCancel = false },
            title = { Text(tr("Cancel this request?", "ఈ అభ్యర్థనను రద్దు చేయాలా?", "यह अनुरोध रद्द करें?")) },
            text = { Text(tr("The priest will no longer receive your request.", "పురోహితునికి మీ అభ్యర్థన ఇక చేరదు.", "पुरोहित को अब आपका अनुरोध नहीं मिलेगा।")) },
            confirmButton = { TextButton(onClick = {
                confirmCancel = false
                val bookingId = activeBookingId
                if (bookingId.isNotBlank() && !isCancelling) {
                    isCancelling = true
                    coroutineScope.launch {
                        try {
                            PurohitApiClient.cancelBooking(bookingId, "DEVOTEE_CANCELLED", idempotencyKey = cancelIntentKey).onSuccess {
                                isAwaitingPriestAcceptance = false
                                activeBookingId = ""
                                cancelIntentKey = IdempotencyKeyGenerator.generate()
                            }.onFailure { errorMessage = tr("Could not cancel the request. Please try again.", "అభ్యర్థన రద్దు కాలేదు. మళ్లీ ప్రయత్నించండి.", "अनुरोध रद्द नहीं हुआ। फिर प्रयास करें।") }
                        } finally { isCancelling = false }
                    }
                }
            }) { Text(tr("Cancel request", "అభ్యర్థన రద్దు చేయండి", "अनुरोध रद्द करें"), color = Color.Red) } },
            dismissButton = { TextButton(onClick = { confirmCancel = false }) { Text(tr("Keep waiting", "వేచి ఉండండి", "प्रतीक्षा करें")) } }
        )
    }

    Box(modifier = Modifier.fillMaxSize()) {
        if (isLoading && nearbyPriests.isEmpty()) {
            VedicLoadingScreen(
                title = tr("Finding priests", "పురోహితులను వెతుకుతున్నాము", "पुरोहित खोज रहे हैं"),
                subtitle = tr("Finding priests near your location…", "మీ దగ్గర పురోహితులను వెతుకుతున్నాము…", "आपके पास पुरोहित खोज रहे हैं…")
            )
        } else if (nearbyPriests.isEmpty() && errorMessage != null) {
            if (locationPermissionDenied) {
                Column(Modifier.fillMaxSize().padding(24.dp), verticalArrangement = Arrangement.Center, horizontalAlignment = Alignment.CenterHorizontally) {
                    Text(errorMessage!!, style = MaterialTheme.typography.bodyLarge)
                    Spacer(Modifier.height(16.dp))
                    Button(onClick = { loadNearby() }) { Text(tr("Try again", "మళ్లీ ప్రయత్నించండి", "दोबारा प्रयास करें")) }
                }
            } else if (errorMessage!!.contains("connect", ignoreCase = true) || errorMessage!!.contains("network", ignoreCase = true)) {
                NoInternetScreen(
                    strings = strings,
                    onRetry = { loadNearby() },
                    serverUrl = PurohitApiClient.baseUrl
                )
            } else {
                ErrorScreen(
                    errorMessage = errorMessage!!,
                    strings = strings,
                    onRetry = { loadNearby() }
                )
            }
        } else {
            LazyColumn(
            modifier = Modifier.fillMaxSize(),
            contentPadding = PaddingValues(start = 16.dp, end = 16.dp, top = 16.dp, bottom = 100.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            item { OutlinedButton(onClick = { loadNearby() }, enabled = !isLoading) { Text(tr("Refresh my location", "నా స్థానాన్ని నవీకరించండి", "मेरा स्थान अपडेट करें")) } }
            // Awaiting Priest Acceptance
            if (isAwaitingPriestAcceptance) {
                item {
                    Card(
                        modifier = Modifier.fillMaxWidth(),
                        shape = RoundedCornerShape(16.dp),
                        colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant),
                        elevation = CardDefaults.cardElevation(defaultElevation = 2.dp)
                    ) {
                        Column(
                            modifier = Modifier.padding(18.dp).fillMaxWidth(),
                            horizontalAlignment = Alignment.CenterHorizontally
                        ) {
                            Surface(
                                shape = CircleShape,
                                color = MaterialTheme.colorScheme.primary,
                                modifier = Modifier.size(56.dp)
                            ) {
                                Box(contentAlignment = Alignment.Center) {
                                    Text("⏳", style = MaterialTheme.typography.titleLarge)
                                }
                            }

                            Spacer(modifier = Modifier.height(10.dp))

                            Text(
                                text = tr("Request sent to priest", "పురోహితునికి అభ్యర్థన పంపబడింది", "पुरोहित को अनुरोध भेजा गया"),
                                fontWeight = FontWeight.Bold,
                                style = MaterialTheme.typography.titleMedium,
                                color = MaterialTheme.colorScheme.primary
                            )
                            Text(
                                text = tr("Waiting for ${bookedPriest?.full_name} to respond…", "${bookedPriest?.full_name} సమాధానం కోసం వేచి ఉన్నాము…", "${bookedPriest?.full_name} के उत्तर की प्रतीक्षा है…"),
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant
                            )

                            Spacer(modifier = Modifier.height(12.dp))

                            Surface(
                                color = Color.White,
                                shape = RoundedCornerShape(10.dp),
                                modifier = Modifier.fillMaxWidth()
                            ) {
                                Column(modifier = Modifier.padding(12.dp)) {
                                    Text("${tr("Date", "తేదీ", "तारीख")}: ${selectedDate.split("-").reversed().joinToString("/")} • ${tr("Time", "సమయం", "समय")}: $selectedSlot", fontWeight = FontWeight.Bold, style = MaterialTheme.typography.bodySmall)
                                    Spacer(modifier = Modifier.height(4.dp))
                                    Text("${tr("Sankalpam", "సంకల్పం", "संकल्प")}: $devoteeGotram • $devoteeNakshatram ($devoteeRashi)", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                                    Spacer(modifier = Modifier.height(4.dp))
                                    Text("${tr("Estimated charges", "అంచనా ఛార్జీలు", "अनुमानित शुल्क")}: ₹${totalAmount.toInt()} (${tr("confirm with priest", "పురోహితునితో నిర్ధారించండి", "पुरोहित से पुष्टि करें")})", style = MaterialTheme.typography.bodySmall, color = Color(0xFF15803D), fontWeight = FontWeight.Bold)
                                }
                            }

                            Spacer(modifier = Modifier.height(14.dp))

                            Text(tr("Waiting for the priest to respond.", "పురోహితుని సమాధానం కోసం వేచి ఉన్నాము.", "पुरोहित के उत्तर की प्रतीक्षा है।"), style = MaterialTheme.typography.bodySmall)
                            Spacer(modifier = Modifier.height(12.dp))
                            OutlinedButton(
                                onClick = { confirmCancel = true },
                                enabled = activeBookingId.isNotBlank() && !isCancelling,
                                colors = ButtonDefaults.outlinedButtonColors(contentColor = Color.Red),
                                shape = RoundedCornerShape(8.dp)
                            ) {
                                Text(tr("Cancel request", "అభ్యర్థన రద్దు చేయండి", "अनुरोध रद्द करें"))
                            }
                        }
                    }
                }
            } else if (bookedPriest != null && isRequestDispatched) {
                item {
                    Card(
                        modifier = Modifier.fillMaxWidth(),
                        shape = RoundedCornerShape(16.dp),
                        colors = CardDefaults.cardColors(containerColor = Color(0xFFDCFCE7))
                    ) {
                        Column(modifier = Modifier.padding(16.dp)) {
                            Text("✅ " + tr("Priest accepted your request", "పురోహితులు మీ అభ్యర్థనను అంగీకరించారు", "पुरोहित ने आपका अनुरोध स्वीकार किया"), fontWeight = FontWeight.Bold, color = Color(0xFF15803D), style = MaterialTheme.typography.titleMedium)
                            Text("${bookedPriest?.full_name} · ${selectedDate.split("-").reversed().joinToString("/")}")
                            Spacer(modifier = Modifier.height(8.dp))
                            Text("${tr("Time", "సమయం", "समय")}: $selectedSlot IST")
                            Text("${tr("Phone", "ఫోన్", "फ़ोन")}: ${bookedPriest?.phone}")
                            Spacer(modifier = Modifier.height(10.dp))
                            Surface(
                                color = Color.White,
                                shape = RoundedCornerShape(8.dp),
                                modifier = Modifier.fillMaxWidth()
                            ) {
                                Column(modifier = Modifier.padding(12.dp)) {
                                    Text("Completion OTP: $completionOtp", fontWeight = FontWeight.Bold, style = MaterialTheme.typography.titleMedium)
                                    Text("Share this code with the priest only after the puja is complete.", style = MaterialTheme.typography.bodySmall)
                                }
                            }
                            Spacer(modifier = Modifier.height(14.dp))
                            Button(
                                onClick = {
                                    bookedPriest?.let { onPriestBooked(it, completionOtp, activeBookingId, activeChatRoomId) }
                                },
                                modifier = Modifier.fillMaxWidth(),
                                shape = RoundedCornerShape(10.dp),
                                colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.primary)
                            ) {
                                Text(tr("View booking status & chat", "బుకింగ్ స్థితి & చాట్ చూడండి", "बुकिंग स्थिति और चैट देखें"))
                            }
                        }
                    }
                }
            } else {
                // Item 1: Location & Online Priests Card
                item {
                    OutlinedTextField(value = eventAddress, onValueChange = { eventAddress = it }, label = { Text(tr("Pooja address", "పూజ చిరునామా", "पूजा का पता")) }, modifier = Modifier.fillMaxWidth(), supportingText = { Text(tr("Enter the address the priest should visit.", "పురోహితుడు రావాల్సిన చిరునామా నమోదు చేయండి.", "वह पता दर्ज करें जहाँ पुरोहित को आना है।")) })
                }

                // Item 2: Date & Muhurtham Slot Selection Card
                item {
                    Card(
                        modifier = Modifier.fillMaxWidth(),
                        shape = RoundedCornerShape(14.dp),
                        colors = CardDefaults.cardColors(containerColor = Color.White),
                        elevation = CardDefaults.cardElevation(defaultElevation = 1.dp)
                    ) {
                        Column(modifier = Modifier.padding(14.dp)) {
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween,
                                verticalAlignment = Alignment.CenterVertically
                            ) {
                                Text(
                                    text = tr("Pooja date and time", "పూజ తేదీ మరియు సమయం", "पूजा की तारीख और समय"),
                                    fontWeight = FontWeight.Bold,
                                    style = MaterialTheme.typography.titleSmall,
                                    color = MaterialTheme.colorScheme.primary,
                                    modifier = Modifier.weight(1f)
                                )
                                Spacer(modifier = Modifier.width(8.dp))
                                Surface(
                                    color = MaterialTheme.colorScheme.surfaceVariant,
                                    shape = RoundedCornerShape(12.dp)
                                ) {
                                    Text(
                                        text = tr("Panchangam", "పంచాంగం", "पंचांग"),
                                        style = MaterialTheme.typography.labelSmall,
                                        color = MaterialTheme.colorScheme.primary,
                                        fontWeight = FontWeight.Bold,
                                        modifier = Modifier.padding(horizontal = 8.dp, vertical = 3.dp),
                                        maxLines = 1
                                    )
                                }
                            }

                            Spacer(modifier = Modifier.height(10.dp))

                            IndiaDatePicker(value = selectedDate, language = language, strings = strings, onDateSelected = { selectedDate = it })
                            Text(slotMessage, style = MaterialTheme.typography.bodySmall)
                            LazyRow(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
                                items(availableSlots) { slot ->
                                    val label = "${to12h(slot.start_time)} – ${to12h(slot.end_time)}"
                                    FilterChip(selected = chosenSlot?.id == slot.id, onClick = { chosenSlot = slot; selectedSlot = label }, label = { Text(label) })
                                }
                            }
                            Spacer(modifier = Modifier.height(10.dp))


                        }
                    }
                }

                // Item 3: Header
                item {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Column(modifier = Modifier.weight(1f)) {
                            Text(
                                text = if (selectedDate.isBlank()) tr("Available priests", "అందుబాటులో ఉన్న పురోహితులు", "उपलब्ध पुरोहित") else "${tr("Available priests", "అందుబాటులో ఉన్న పురోహితులు", "उपलब्ध पुरोहित")} (${selectedDate.split("-").reversed().joinToString("/")})",
                                fontWeight = FontWeight.Bold,
                                style = MaterialTheme.typography.titleMedium
                            )
                            Text(
                                text = poojaTitle,
                                style = MaterialTheme.typography.bodySmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant,
                                maxLines = 1
                            )
                        }
                        Surface(
                            color = MaterialTheme.colorScheme.primaryContainer,
                            shape = RoundedCornerShape(12.dp)
                        ) {
                            Text(
                                text = "₹${totalAmount.toInt()}",
                                fontWeight = FontWeight.Bold,
                                color = MaterialTheme.colorScheme.primary,
                                modifier = Modifier.padding(horizontal = 10.dp, vertical = 6.dp),
                                style = MaterialTheme.typography.titleSmall
                            )
                        }
                    }
                }

                if (errorMessage != null) {
                    item {
                        Surface(
                            color = Color(0xFFFEE2E2),
                            shape = RoundedCornerShape(8.dp),
                            modifier = Modifier.fillMaxWidth()
                        ) {
                            Text(
                                text = errorMessage!!,
                                color = Color(0xFFB91C1C),
                                style = MaterialTheme.typography.bodySmall,
                                modifier = Modifier.padding(8.dp)
                            )
                        }
                    }
                }

                if (nearbyPriests.isEmpty() && errorMessage == null) {
                    item { Text(tr("No online priests nearby. Refresh your location to try again.", "సమీపంలో ఆన్‌లైన్ పురోహితులు లేరు. మళ్లీ ప్రయత్నించడానికి స్థానాన్ని నవీకరించండి.", "आसपास कोई पुरोहित ऑनलाइन नहीं है। पुनः प्रयास के लिए स्थान अपडेट करें।"), style = MaterialTheme.typography.bodyMedium) }
                }

                // Item 4: Priests List
                items(nearbyPriests) { priest ->
                    val isSelected = (selectedPriest?.id == priest.id)
                    Card(
                        modifier = Modifier
                            .fillMaxWidth()
                            .clickable { selectedPriest = priest },
                        shape = RoundedCornerShape(16.dp),
                        colors = CardDefaults.cardColors(
                            containerColor = if (isSelected) MaterialTheme.colorScheme.background else MaterialTheme.colorScheme.surface
                        ),
                        border = if (isSelected) BorderStroke(2.dp, MaterialTheme.colorScheme.primary) else null,
                        elevation = CardDefaults.cardElevation(defaultElevation = if (isSelected) 4.dp else 2.dp)
                    ) {
                        Column(modifier = Modifier.padding(16.dp)) {
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween,
                                verticalAlignment = Alignment.Top
                            ) {
                                Column(modifier = Modifier.weight(1f)) {
                                    Text(
                                        text = priest.full_name,
                                        fontWeight = FontWeight.Bold,
                                        style = MaterialTheme.typography.titleSmall
                                    )
                                    Text(
                                        text = "${priest.sampradaya} • ${priest.veda_shakha}",
                                        style = MaterialTheme.typography.bodySmall,
                                        color = MaterialTheme.colorScheme.primary
                                    )
                                }
                                Surface(
                                    color = if (isSelected) MaterialTheme.colorScheme.primary else Color(0xFFDCFCE7),
                                    shape = RoundedCornerShape(12.dp)
                                ) {
                                    Text(
                                        text = if (isSelected) tr("Selected", "ఎంపికైంది", "चुना गया") else tr("Select to view times", "సమయాలు చూడడానికి ఎంచుకోండి", "समय देखने के लिए चुनें"),
                                        fontWeight = FontWeight.Bold,
                                        color = if (isSelected) Color.White else Color(0xFF15803D),
                                        modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
                                        style = MaterialTheme.typography.labelSmall
                                    )
                                }
                            }

                            Spacer(modifier = Modifier.height(8.dp))

                            Button(
                                onClick = {
                                    if (selectedPriest?.id == priest.id && chosenSlot != null && eventAddress.isNotBlank()) priestToConfirm = priest else { selectedPriest = priest; slotMessage = tr("Enter the address and select a date and time above.", "చిరునామా నమోదు చేసి పైన తేదీ, సమయం ఎంచుకోండి.", "पता दर्ज करें और ऊपर तारीख व समय चुनें।") }
                                },
                                modifier = Modifier.fillMaxWidth(),
                                shape = RoundedCornerShape(10.dp),
                                colors = ButtonDefaults.buttonColors(
                                    containerColor = if (isSelected) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.secondary
                                )
                            ) {
                                Text(if (isSelected && chosenSlot != null) tr("Review request", "అభ్యర్థన పరిశీలించండి", "अनुरोध जाँचें") else tr("Select priest", "పురోహితుడిని ఎంచుకోండి", "पुरोहित चुनें"))
                            }
                        }
                    }
                }
            }
        }
        }

        priestToConfirm?.let { priest ->
            AlertDialog(
                onDismissRequest = { priestToConfirm = null },
                title = { Text(tr("Review booking request", "బుకింగ్ అభ్యర్థన పరిశీలించండి", "बुकिंग अनुरोध जाँचें")) },
                text = { Text("${priest.full_name}\n$poojaTitle\n${tr("Date", "తేదీ", "तारीख")}: ${selectedDate.split("-").reversed().joinToString("/")}\n${tr("Time", "సమయం", "समय")}: $selectedSlot\n${tr("Address", "చిరునామా", "पता")}: $eventAddress\n${tr("Estimated charges", "అంచనా ఛార్జీలు", "अनुमानित शुल्क")}: ₹${totalAmount.toInt()}\n\n${tr("Confirm priest and material charges directly. No online payment is collected in this test.", "పురోహితులు, సామగ్రి ఛార్జీలను నేరుగా నిర్ధారించండి. ఈ పరీక్షలో ఆన్‌లైన్ చెల్లింపు వసూలు చేయరు.", "पुरोहित और सामग्री के शुल्क सीधे तय करें। इस परीक्षण में ऑनलाइन भुगतान नहीं लिया जाता।")}") },
                confirmButton = { TextButton(onClick = { priestToConfirm = null; sendBookingRequest(priest) }) { Text(tr("Send request", "అభ్యర్థన పంపండి", "अनुरोध भेजें")) } },
                dismissButton = { TextButton(onClick = { priestToConfirm = null }) { Text(strings.common.cancelAction) } }
            )
        }
    }
}


@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun IndiaDatePicker(value: String, language: AppLanguage, strings: StringContract, onDateSelected: (String) -> Unit) {
    var showPicker by remember { mutableStateOf(false) }
    val today = remember { Clock.System.now().toLocalDateTime(TimeZone.of("Asia/Kolkata")).date }
    val minimumUtc = remember(today) { today.atStartOfDayIn(TimeZone.UTC).toEpochMilliseconds() }
    val displayed = value.split("-").let { if (it.size == 3) "${it[2]}/${it[1]}/${it[0]}" else "Select date" }
    OutlinedButton(onClick = { showPicker = true }, modifier = Modifier.fillMaxWidth()) { Text(if (value.isEmpty()) when (language) { AppLanguage.TELUGU -> "తేదీ ఎంచుకోండి (DD/MM/YYYY)"; AppLanguage.HINDI -> "तारीख चुनें (DD/MM/YYYY)"; else -> "Select date (DD/MM/YYYY)" } else displayed) }
    if (showPicker) {
        val picker = rememberDatePickerState(initialSelectedDateMillis = value.takeIf { it.isNotBlank() }?.let { runCatching { LocalDate.parse(it).atStartOfDayIn(TimeZone.UTC).toEpochMilliseconds() }.getOrNull() } ?: minimumUtc,
            selectableDates = remember(minimumUtc) { object : SelectableDates { override fun isSelectableDate(utcTimeMillis: Long) = utcTimeMillis >= minimumUtc } })
        DatePickerDialog(onDismissRequest = { showPicker = false }, confirmButton = { TextButton(onClick = {
            picker.selectedDateMillis?.let { onDateSelected(Instant.fromEpochMilliseconds(it).toLocalDateTime(TimeZone.UTC).date.toString()) }
            showPicker = false
        }) { Text(when (language) { AppLanguage.TELUGU -> "సరే"; AppLanguage.HINDI -> "ठीक है"; else -> "OK" }) } }, dismissButton = { TextButton(onClick = { showPicker = false }) { Text(strings.common.cancelAction) } }) { DatePicker(state = picker, showModeToggle = false) }
    }
}

/** "14:30" -> "2:30 PM" */
private fun to12h(hhmm: String): String {
    val parts = hhmm.take(5).split(":")
    val h = parts.getOrNull(0)?.toIntOrNull() ?: return hhmm
    val m = parts.getOrNull(1) ?: "00"
    return "${if (h % 12 == 0) 12 else h % 12}:$m ${if (h < 12) "AM" else "PM"}"
}
