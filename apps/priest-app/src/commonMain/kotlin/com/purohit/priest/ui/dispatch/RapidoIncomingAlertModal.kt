package com.purohit.priest.ui.dispatch

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.platform.LocalSoftwareKeyboardController
import androidx.compose.ui.semantics.LiveRegionMode
import androidx.compose.ui.semantics.liveRegion
import androidx.compose.ui.semantics.semantics
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import kotlinx.coroutines.delay

import com.purohit.contracts.i18n.AppLanguage
import com.purohit.contracts.i18n.LocalizationManager
import com.purohit.contracts.i18n.StringContract

@Composable
fun RapidoIncomingAlertModal(
    bookingId: String = "f312db8f-1d71-4cab-b5ba-77fbead70c71",
    poojaTitle: String = "Sri Satyanarayana Swamy Vratam",
    devoteeName: String = "Sushanth2",
    devoteePhone: String = "9177961089",
    eventAddress: String = "Plot 42, Sri Rama Nagar, Madhav Nagar, Hyderabad (Landmark: Behind Sri Rama Temple)",
    dakshinaAmount: Double = 3716.0,
    isSamagriIncluded: Boolean = true,
    scheduledDate: String = "Tomorrow (Nov 4)",
    slotTime: String = "Abhijit Muhurtham (11:45 AM - 01:45 PM)",
    devoteeGotram: String = "Kashyapa",
    devoteeNakshatram: String = "Rohini",
    devoteeRashi: String = "Vrishabha",
    sankalpamPurpose: String = "Family Prosperity & Health",
    currentLanguage: AppLanguage = AppLanguage.TELUGU,
    strings: StringContract = LocalizationManager.getStrings(AppLanguage.TELUGU),
    errorMessage: String? = null,
    isSubmitting: Boolean = false,
    onAccept: () -> Unit,
    onReject: () -> Unit,
    // Server deadline (ISO-8601) for event requests. When set, the countdown follows it and,
    // at zero, the alert just closes: the server expires the request, nothing is declined.
    expiresAt: String? = null,
    onExpired: () -> Unit = {}
) {
    val deadline = remember(expiresAt) { expiresAt?.let { runCatching { kotlinx.datetime.Instant.parse(it) }.getOrNull() } }
    fun secondsToDeadline() = deadline?.let { (it - kotlinx.datetime.Clock.System.now()).inWholeSeconds.toInt().coerceAtLeast(0) }
    var timeLeftSeconds by remember { mutableStateOf(secondsToDeadline() ?: 45) }
    val isTelugu = currentLanguage == AppLanguage.TELUGU
    val isHindi = currentLanguage == AppLanguage.HINDI
    // Decline needs a second tap so a slip next to Accept does not drop the request.
    var declineArmed by remember { mutableStateOf(false) }
    LaunchedEffect(declineArmed) { if (declineArmed) { kotlinx.coroutines.delay(4_000); declineArmed = false } }
    val focusManager = LocalFocusManager.current
    val keyboard = LocalSoftwareKeyboardController.current

    LaunchedEffect(Unit) {
        focusManager.clearFocus(force = true)
        keyboard?.hide()
    }
    LaunchedEffect(Unit) {
        while (timeLeftSeconds > 0) {
            delay(1000L)
            timeLeftSeconds = secondsToDeadline() ?: (timeLeftSeconds - 1)
        }
        // Never auto-decline: the server expires unanswered requests itself.
        onExpired()
    }

    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(Color.Black.copy(alpha = 0.85f))
            .padding(20.dp),
        contentAlignment = Alignment.Center
    ) {
        Card(
            modifier = Modifier.fillMaxWidth(),
            shape = RoundedCornerShape(20.dp),
            colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface)
        ) {
            Column(
                modifier = Modifier.verticalScroll(rememberScrollState()).padding(20.dp),
                horizontalAlignment = Alignment.CenterHorizontally
            ) {
                // Countdown Timer Ring
                Surface(
                    shape = CircleShape,
                    color = if (timeLeftSeconds > 15) Color(0xFFD97706) else Color.Red,
                    modifier = Modifier.size(60.dp)
                ) {
                    Box(contentAlignment = Alignment.Center) {
                        Text(
                            text = if (timeLeftSeconds >= 60) "${timeLeftSeconds / 60}:${(timeLeftSeconds % 60).toString().padStart(2, '0')}" else "${timeLeftSeconds}s",
                            color = Color.White,
                            fontWeight = FontWeight.Bold,
                            style = MaterialTheme.typography.titleLarge
                        )
                    }
                }

                Spacer(modifier = Modifier.height(14.dp))

                Text(
                    text = strings.dispatch.incomingBookingAlert,
                    modifier = Modifier.semantics { liveRegion = LiveRegionMode.Assertive },
                    style = MaterialTheme.typography.labelMedium,
                    fontWeight = FontWeight.Bold,
                    color = MaterialTheme.colorScheme.primary
                )

                Text(
                    text = poojaTitle,
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.headlineSmall
                )

                Spacer(modifier = Modifier.height(6.dp))

                // Date & Muhurtham Slot Chip
                Surface(
                    shape = RoundedCornerShape(8.dp),
                    color = Color(0xFFEFF6FF),
                    modifier = Modifier.fillMaxWidth()
                ) {
                    Row(
                        modifier = Modifier.padding(horizontal = 12.dp, vertical = 8.dp).fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Text(
                            text = "📅 $scheduledDate",
                            fontWeight = FontWeight.Bold,
                            style = MaterialTheme.typography.labelMedium,
                            color = Color(0xFF1D4ED8)
                        )
                        Text(
                            text = "⏰ $slotTime",
                            fontWeight = FontWeight.Bold,
                            style = MaterialTheme.typography.labelSmall,
                            color = Color(0xFF1E40AF)
                        )
                    }
                }

                Spacer(modifier = Modifier.height(8.dp))

                // Vedic Sankalpam Details Card
                Surface(
                    shape = RoundedCornerShape(10.dp),
                    color = Color(0xFFFDF4FF),
                    modifier = Modifier.fillMaxWidth()
                ) {
                    Column(modifier = Modifier.padding(10.dp)) {
                        Text(
                            text = if (isTelugu) "✨ వైదిక సంకల్ప వివరాలు" else if (isHindi) "✨ वैदिक संकल्प विवरण" else "✨ VEDIC SANKALPAM ALIGNMENT",
                            style = MaterialTheme.typography.labelSmall,
                            fontWeight = FontWeight.Bold,
                            color = Color(0xFF86198F)
                        )
                        Spacer(modifier = Modifier.height(2.dp))
                        Text(
                            text = "${if (isTelugu) "గోత్రం" else if (isHindi) "गोत्र" else "Gotram"}: $devoteeGotram • ${if (isTelugu) "నక్షత్రం" else if (isHindi) "नक्षत्र" else "Nakshatram"}: $devoteeNakshatram ($devoteeRashi)",
                            style = MaterialTheme.typography.bodySmall,
                            fontWeight = FontWeight.Bold,
                            color = Color(0xFF701A75)
                        )
                        if (sankalpamPurpose.isNotBlank()) Text(
                            text = "${if (isTelugu) "సంకల్ప ఉద్దేశం" else if (isHindi) "संकल्प उद्देश्य" else "Sankalpam"}: $sankalpamPurpose",
                            style = MaterialTheme.typography.labelSmall,
                            color = Color(0xFF86198F)
                        )
                    }
                }

                Spacer(modifier = Modifier.height(8.dp))

                // Devotee and Address Card
                Surface(
                    shape = RoundedCornerShape(12.dp),
                    color = MaterialTheme.colorScheme.surfaceVariant,
                    modifier = Modifier.fillMaxWidth()
                ) {
                    Column(modifier = Modifier.padding(12.dp)) {
                        Column(modifier = Modifier.fillMaxWidth(), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                            Text(
                                "${if (isTelugu) "👤 యజమాని / భక్తుడు" else if (isHindi) "👤 यजमान / भक्त" else "👤 Devotee"}: $devoteeName",
                                fontWeight = FontWeight.Bold,
                                style = MaterialTheme.typography.titleSmall
                            )
                            Text("📞 $devoteePhone", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                        }
                        Spacer(modifier = Modifier.height(4.dp))
                        Text(
                            text = "📍 $eventAddress",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                        Spacer(modifier = Modifier.height(2.dp))
                        Text(
                            text = if (isTelugu) "స్థాన భాగస్వామ్యం అందుబాటులో లేదు" else if (isHindi) "स्थान साझा करना उपलब्ध नहीं है" else "Location sharing is unavailable",
                            style = MaterialTheme.typography.labelSmall,
                            color = Color(0xFF15803D),
                            fontWeight = FontWeight.Bold
                        )
                    }
                }

                Spacer(modifier = Modifier.height(10.dp))

                // Dakshina & Samagri breakdown
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceAround
                ) {
                    Column(horizontalAlignment = Alignment.CenterHorizontally) {
                        Text(strings.catalog.dakshinaLabel, style = MaterialTheme.typography.labelSmall)
                        Text("₹${dakshinaAmount.toInt()}", fontWeight = FontWeight.Bold, style = MaterialTheme.typography.titleLarge, color = Color(0xFF15803D))
                        Text(if (isTelugu) "ఛార్జీలను భక్తునితో నిర్ధారించండి" else if (isHindi) "शुल्क की पुष्टि भक्त से करें" else "Confirm charges with devotee", style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    }
                    Column(horizontalAlignment = Alignment.CenterHorizontally) {
                        Text(if (isTelugu) "సామగ్రి" else if (isHindi) "सामग्री" else "Materials", style = MaterialTheme.typography.labelSmall)
                        Text(if (isTelugu) "భక్తునితో నిర్ధారించండి" else if (isHindi) "भक्त से पुष्टि करें" else "Confirm with devotee", style = MaterialTheme.typography.bodySmall)
                    }
                }

                errorMessage?.let {
                    Spacer(modifier = Modifier.height(12.dp))
                    Text(it, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodyMedium,
                        modifier = Modifier.fillMaxWidth().semantics { liveRegion = LiveRegionMode.Assertive })
                }
                Spacer(modifier = Modifier.height(20.dp))

                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(14.dp)
                ) {
                    OutlinedButton(
                        onClick = { if (declineArmed) onReject() else declineArmed = true },
                        enabled = !isSubmitting,
                        modifier = Modifier.weight(1f),
                        colors = ButtonDefaults.outlinedButtonColors(contentColor = Color.Red)
                    ) {
                        Text(if (declineArmed) (if (isTelugu) "నిర్ధారించడానికి మళ్లీ నొక్కండి" else if (isHindi) "पुष्टि के लिए फिर दबाएं" else "Tap again to decline") else strings.dispatch.declineDutyCTA)
                    }
                    Button(
                        onClick = onAccept,
                        enabled = !isSubmitting,
                        modifier = Modifier.weight(1.5f),
                        colors = ButtonDefaults.buttonColors(containerColor = Color(0xFF15803D))
                    ) {
                        Text(strings.dispatch.acceptDutyCTA)
                    }
                }
            }
        }
    }
}
