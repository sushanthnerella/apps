package com.purohit.devotee.ui.tracking

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.fadeIn
import androidx.compose.animation.fadeOut
import androidx.compose.animation.core.tween
import androidx.compose.ui.semantics.liveRegion
import androidx.compose.ui.semantics.LiveRegionMode
import androidx.compose.ui.semantics.semantics
import com.purohit.contracts.i18n.AppLanguage
import com.purohit.devotee.LocalAppLanguage
import com.purohit.devotee.motionAllowed
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.purohit.devotee.network.PurohitApiClient
import kotlinx.coroutines.delay

@Composable
fun LiveTrackingScreen(
    priestName: String = "", priestPhone: String = "", priestSampradaya: String = "",
    poojaTitle: String = "", totalAmount: Double = 0.0, completionOtp: String = "",
    initialDistanceKm: Double = 0.0, bookingId: String = "", chatRoomId: String = "",
    poojaId: String = "", eventAddress: String = "", isSamagriIncluded: Boolean = false,
    onOpenChat: () -> Unit = {}, onOpenLiveStream: () -> Unit = {},
    onCancelBooking: () -> Unit = {}, onBackToHome: () -> Unit = {}
) {
    val strings = com.purohit.devotee.LocalAppStrings.current
    val language = LocalAppLanguage.current
    fun tr(en: String, te: String, hi: String) = when (language) { AppLanguage.TELUGU -> te; AppLanguage.HINDI -> hi; else -> en }
    var status by remember(bookingId) { mutableStateOf<String?>(null) }
    var error by remember(bookingId) { mutableStateOf<String?>(null) }
    var otp by remember(bookingId) { mutableStateOf(completionOtp) }
    var otpError by remember(bookingId) { mutableStateOf<String?>(null) }
    LaunchedEffect(bookingId) {
        if (bookingId.isBlank()) { error = strings.common.bookingMissing; return@LaunchedEffect }
        var refreshDelay = 5000L
        while (true) {
            PurohitApiClient.fetchBookingDetails(bookingId).onSuccess { status = it.status; error = null; refreshDelay = 5000L }
                .onFailure { error = strings.common.bookingRefreshError; refreshDelay = (refreshDelay * 2).coerceAtMost(60000L) }
            if (otp.isBlank()) PurohitApiClient.fetchBookingOtp(bookingId).onSuccess { otp = it.completion_otp; otpError = null }
                .onFailure { otpError = tr("Could not load the completion code. Check your connection.", "పూర్తి కోడ్ పొందలేకపోయాము. కనెక్షన్ తనిఖీ చేయండి.", "समापन कोड नहीं मिला। कनेक्शन जाँचें।") }
            if (status in listOf("COMPLETED", "CANCELLED", "REJECTED")) break
            delay(refreshDelay + kotlin.random.Random.nextLong(1000))
        }
    }
    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
        Text(strings.common.bookingStatusTitle, style = MaterialTheme.typography.headlineMedium, fontWeight = FontWeight.Bold)
        error?.let { Text(it, color = MaterialTheme.colorScheme.error) }
        Card(Modifier.fillMaxWidth(), shape = RoundedCornerShape(20.dp), colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant)) {
            Column(Modifier.padding(20.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                Text(when (status) {
                    "REQUESTED" -> strings.common.bookingRequested
                    "ACCEPTED" -> strings.common.bookingAccepted
                    "IN_TRANSIT" -> strings.common.bookingInTransit
                    "ARRIVED" -> strings.common.bookingArrived
                    "POOJA_STARTED" -> strings.common.bookingInProgress
                    "COMPLETED" -> strings.common.bookingCompleted
                    "CANCELLED" -> strings.common.bookingCancelled
                    "REJECTED" -> strings.common.bookingRejected
                    null -> strings.common.bookingLoading
                    else -> "${tr("Status", "స్థితి", "स्थिति")}: $status"
                }, modifier = Modifier.semantics { liveRegion = LiveRegionMode.Polite }, style = MaterialTheme.typography.titleLarge, color = MaterialTheme.colorScheme.primary, fontWeight = FontWeight.Bold)
                if (poojaTitle.isNotBlank()) Text(poojaTitle)
                if (priestName.isNotBlank()) Text("${tr("Priest", "పురోహితుడు", "पुरोहित")}: $priestName")
                if (priestPhone.isNotBlank()) Text("${tr("Phone", "ఫోన్", "फ़ोन")}: $priestPhone")
                if (eventAddress.isNotBlank()) Text("${tr("Address", "చిరునామా", "पता")}: $eventAddress")
                Text("${tr("Estimated charges", "అంచనా ఛార్జీలు", "अनुमानित शुल्क")}: ₹${totalAmount.toInt()}. ${tr("Confirm charges directly with the priest.", "ఛార్జీలను పురోహితునితో నేరుగా నిర్ధారించండి.", "शुल्क की पुष्टि पुरोहित से सीधे करें।")}")
                Text(strings.common.trackingUnavailable, style = MaterialTheme.typography.bodySmall)
            }
        }
        if (motionAllowed()) {
            AnimatedVisibility(visible = status == "COMPLETED", enter = fadeIn(tween(180)), exit = fadeOut(tween(120))) {
                Text("✓  ${strings.common.bookingCompleted}", color = MaterialTheme.colorScheme.primary, style = MaterialTheme.typography.titleMedium, modifier = Modifier.semantics { liveRegion = LiveRegionMode.Polite })
            }
        } else if (status == "COMPLETED") {
            Text("✓  ${strings.common.bookingCompleted}", color = MaterialTheme.colorScheme.primary, style = MaterialTheme.typography.titleMedium)
        }
        if (status != "COMPLETED" && status != "CANCELLED" && status != "REJECTED") {
            Card(Modifier.fillMaxWidth()) {
                Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    Text(tr("Completion code", "పూర్తి కోడ్", "समापन कोड"), fontWeight = FontWeight.Bold)
                    if (otp.isNotBlank()) Text(otp, style = MaterialTheme.typography.headlineMedium)
                    else Text(otpError ?: tr("Loading completion code…", "పూర్తి కోడ్ పొందుతున్నాము…", "समापन कोड लोड हो रहा है…"))
                    Text(tr("Share this code with the priest only after the pooja is complete.", "పూజ పూర్తయిన తర్వాత మాత్రమే ఈ కోడ్‌ను పురోహితునితో పంచుకోండి.", "पूजा पूरी होने के बाद ही यह कोड पुरोहित को दें।"), style = MaterialTheme.typography.bodySmall)
                }
            }
            if (chatRoomId.isNotBlank()) OutlinedButton(onClick = onOpenChat, modifier = Modifier.fillMaxWidth()) { Text(tr("Chat with priest", "పురోహితునితో సంభాషణ", "पुरोहित से बातचीत")) }
            // Server lets devotees cancel only until the priest is on the way (lifecycle/fsm.go).
            if (status in setOf(null, "REQUESTED", "ACCEPTED", "IN_TRANSIT")) OutlinedButton(onClick = onCancelBooking, modifier = Modifier.fillMaxWidth()) { Text(tr("Cancel booking", "బుకింగ్ రద్దు చేయండి", "बुकिंग रद्द करें")) }
        }
        TextButton(onClick = onBackToHome) { Text(tr("Back to services", "సేవలకు తిరిగి వెళ్లండి", "सेवाओं पर वापस जाएँ")) }
    }
}
