package com.purohit.priest.ui.dispatch

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
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
    onAccept: () -> Unit,
    onReject: () -> Unit
) {
    var timeLeftSeconds by remember { mutableStateOf(45) }
    val isTelugu = currentLanguage == AppLanguage.TELUGU
    val isHindi = currentLanguage == AppLanguage.HINDI

    LaunchedEffect(Unit) {
        while (timeLeftSeconds > 0) {
            delay(1000L)
            timeLeftSeconds--
        }
        onReject()
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
                modifier = Modifier.padding(20.dp),
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
                            text = "${timeLeftSeconds}s",
                            color = Color.White,
                            fontWeight = FontWeight.Bold,
                            style = MaterialTheme.typography.titleLarge
                        )
                    }
                }

                Spacer(modifier = Modifier.height(14.dp))

                Text(
                    text = strings.dispatch.incomingBookingAlert,
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
                        Text(
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
                    color = Color(0xFFFEF3C7),
                    modifier = Modifier.fillMaxWidth()
                ) {
                    Column(modifier = Modifier.padding(12.dp)) {
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.SpaceBetween
                        ) {
                            Text(
                                "${if (isTelugu) "👤 యజమాని / భక్తుడు" else if (isHindi) "👤 यजमान / भक्त" else "👤 Devotee"}: $devoteeName",
                                fontWeight = FontWeight.Bold,
                                style = MaterialTheme.typography.titleSmall
                            )
                            Text("📞 $devoteePhone", style = MaterialTheme.typography.bodySmall, color = Color(0xFF78350F))
                        }
                        Spacer(modifier = Modifier.height(4.dp))
                        Text(
                            text = "📍 $eventAddress",
                            style = MaterialTheme.typography.bodySmall,
                            color = Color(0xFF78350F)
                        )
                        Spacer(modifier = Modifier.height(2.dp))
                        Text(
                            text = "🛰️ GNSS: 17.3850° N, 78.4867° E (Dual L1+L5 RTK Locked)",
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
                        Text("Escrow Protected", style = MaterialTheme.typography.labelSmall, color = Color.Gray)
                    }
                    Column(horizontalAlignment = Alignment.CenterHorizontally) {
                        Text(if (isTelugu) "సామాగ్రి స్థితి" else if (isHindi) "सामग्री स्थिति" else "Samagri Status", style = MaterialTheme.typography.labelSmall)
                        Text(
                            if (isSamagriIncluded) (if (isTelugu) "🪔 కిట్ చేర్చబడింది" else if (isHindi) "🪔 किट शामिल" else "🪔 Kit Included")
                            else (if (isTelugu) "గృహ ద్రవ్యాలు" else if (isHindi) "घर की सामग्री" else "Household Only"),
                            fontWeight = FontWeight.Bold,
                            style = MaterialTheme.typography.titleMedium,
                            color = Color(0xFFB45309)
                        )
                        Text("+₹1,200 Purohit Samagri", style = MaterialTheme.typography.labelSmall, color = Color.Gray)
                    }
                }

                Spacer(modifier = Modifier.height(20.dp))

                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(14.dp)
                ) {
                    OutlinedButton(
                        onClick = onReject,
                        modifier = Modifier.weight(1f),
                        colors = ButtonDefaults.outlinedButtonColors(contentColor = Color.Red)
                    ) {
                        Text(strings.dispatch.declineDutyCTA)
                    }
                    Button(
                        onClick = onAccept,
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
