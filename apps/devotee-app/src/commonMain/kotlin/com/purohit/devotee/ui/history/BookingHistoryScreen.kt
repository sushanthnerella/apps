package com.purohit.devotee.ui.history

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.purohit.contracts.i18n.AppLanguage
import com.purohit.devotee.LocalAppLanguage
import com.purohit.devotee.network.BookingDetailPayload
import com.purohit.devotee.network.PurohitApiClient

internal fun isActiveBooking(status: String): Boolean = status.uppercase() in setOf(
    "REQUESTED", "ACCEPTED", "IN_TRANSIT", "ARRIVED", "POOJA_STARTED", "IN_PROGRESS"
)

@Composable
internal fun BookingHistoryScreen(onBrowse: () -> Unit, onOpenActive: (BookingDetailPayload) -> Unit) {
    val language = LocalAppLanguage.current
    fun tr(en: String, te: String, hi: String) = when (language) {
        AppLanguage.TELUGU -> te
        AppLanguage.HINDI -> hi
        else -> en
    }
    var bookings by remember { mutableStateOf<List<BookingDetailPayload>?>(null) }
    var failed by remember { mutableStateOf(false) }
    var attempt by remember { mutableStateOf(0) }
    LaunchedEffect(attempt) {
        bookings = null
        failed = false
        PurohitApiClient.fetchDevoteeBookings(PurohitApiClient.currentProfile?.id.orEmpty())
            .onSuccess { bookings = it }
            .onFailure { failed = true }
    }
    Column(Modifier.fillMaxSize().verticalScroll(rememberScrollState()).padding(20.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
        Text(tr("My bookings", "నా బుకింగ్‌లు", "मेरी बुकिंग"), style = MaterialTheme.typography.headlineMedium, fontWeight = FontWeight.Bold)
        when {
            failed -> {
                Text(tr("Could not load bookings. Check your connection and try again.", "బుకింగ్‌లను పొందలేకపోయాము. కనెక్షన్ తనిఖీ చేసి మళ్లీ ప్రయత్నించండి.", "बुकिंग लोड नहीं हुईं। कनेक्शन जाँचकर फिर से प्रयास करें."), color = MaterialTheme.colorScheme.error)
                Button(onClick = { attempt++ }) { Text(tr("Retry", "మళ్లీ ప్రయత్నించండి", "फिर प्रयास करें")) }
            }
            bookings == null -> CircularProgressIndicator()
            bookings!!.isEmpty() -> {
                Text(tr("You have no bookings yet.", "మీకు ఇంకా బుకింగ్‌లు లేవు.", "अभी आपकी कोई बुकिंग नहीं है."))
                Button(onClick = onBrowse) { Text(tr("Browse pujas", "పూజలు చూడండి", "पूजाएँ देखें")) }
            }
            else -> bookings!!.forEach { booking ->
                Card(Modifier.fillMaxWidth()) {
                    Column(Modifier.padding(16.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                        Text(booking.pooja_title.ifBlank { tr("Puja booking", "పూజ బుకింగ్", "पूजा बुकिंग") }, style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.SemiBold)
                        val status = when (booking.status.uppercase()) {
                            "REQUESTED" -> tr("Requested", "అభ్యర్థించబడింది", "अनुरोध भेजा गया")
                            "ACCEPTED" -> tr("Accepted", "అంగీకరించబడింది", "स्वीकार किया गया")
                            "IN_TRANSIT" -> tr("Priest is on the way", "పురోహితుడు దారిలో ఉన్నారు", "पुरोहित रास्ते में हैं")
                            "ARRIVED" -> tr("Priest arrived", "పురోహితుడు చేరుకున్నారు", "पुरोहित पहुँच गए")
                            "POOJA_STARTED", "IN_PROGRESS" -> tr("Puja in progress", "పూజ జరుగుతోంది", "पूजा चल रही है")
                            "COMPLETED" -> tr("Completed", "పూర్తయింది", "पूर्ण हुई")
                            "CANCELLED", "CANCELED" -> tr("Cancelled", "రద్దయింది", "रद्द हुई")
                            "REJECTED" -> tr("Declined", "తిరస్కరించబడింది", "अस्वीकार हुई")
                            else -> booking.status.replace('_', ' ')
                        }
                        Text(status, color = MaterialTheme.colorScheme.primary)
                        if (booking.scheduled_date.isNotBlank()) Text(listOf(booking.scheduled_date, booking.slot_time).filter { it.isNotBlank() }.joinToString(" • "))
                        if (booking.priest_name.isNotBlank()) Text(booking.priest_name)
                        if (booking.event_address.isNotBlank()) Text(booking.event_address, style = MaterialTheme.typography.bodyMedium)
                        if (isActiveBooking(booking.status)) {
                            OutlinedButton(onClick = { onOpenActive(booking) }) { Text(tr("View booking", "బుకింగ్ చూడండి", "बुकिंग देखें")) }
                        }
                    }
                }
            }
        }
    }
}
