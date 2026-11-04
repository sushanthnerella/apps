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
import kotlinx.coroutines.launch

@Composable
fun DutyToggleScreen(
    onSimulateIncomingOrder: () -> Unit,
    hasActiveBooking: Boolean = false,
    activeBookingId: String = "f312db8f-1d71-4cab-b5ba-77fbead70c71",
    activeChatRoomId: String = "d66283f3-1eb0-40b3-afc0-c37d355b90f8",
    devoteeName: String = "Sushanth2",
    devoteePhone: String = "9177961089",
    eventAddress: String = "Plot 42, Sri Rama Nagar, Madhav Nagar, Hyderabad (Landmark: Behind Sri Rama Temple)",
    poojaTitle: String = "Sri Satyanarayana Swamy Vratam",
    dakshinaAmount: Double = 3716.0,
    currentLanguage: AppLanguage = AppLanguage.TELUGU,
    strings: StringContract = LocalizationManager.getStrings(AppLanguage.TELUGU),
    onOpenChat: () -> Unit = {},
    onOpenChecklist: () -> Unit = {},
    onBookingCompleted: () -> Unit = {}
) {
    var isOnline by remember { mutableStateOf(true) }
    var bookingStatus by remember { mutableStateOf("IN_TRANSIT") } // IN_TRANSIT, ARRIVED, POOJA_STARTED, COMPLETED
    var showOtpDialog by remember { mutableStateOf(false) }
    var enteredOtp by remember { mutableStateOf("") }
    var statusMessage by remember { mutableStateOf<String?>(null) }
    var isUpdatingStatus by remember { mutableStateOf(false) }

    val coroutineScope = rememberCoroutineScope()
    val isTelugu = currentLanguage == AppLanguage.TELUGU
    val isHindi = currentLanguage == AppLanguage.HINDI

    // Pulsing animation for active GPS telemetry beacon
    val infiniteTransition = rememberInfiniteTransition(label = "priest_gps_beacon")
    val pulseAlpha by infiniteTransition.animateFloat(
        initialValue = 0.4f,
        targetValue = 1.0f,
        animationSpec = infiniteRepeatable(
            animation = tween(900, easing = LinearEasing),
            repeatMode = RepeatMode.Reverse
        ),
        label = "pulse_alpha"
    )

    fun updateStatus(newStatus: String, otp: String? = null) {
        isUpdatingStatus = true
        coroutineScope.launch {
            val res = PurohitApiClient.updateBookingStatus(
                bookingId = activeBookingId,
                status = newStatus,
                priestId = "6e7d277c-4ee2-469f-96ac-331517389762",
                otp = otp
            )
            isUpdatingStatus = false
            bookingStatus = newStatus
            res.onSuccess {
                statusMessage = "Status updated to $newStatus on Cloud DB"
                if (newStatus == "COMPLETED") {
                    onBookingCompleted()
                }
            }.onFailure {
                statusMessage = "Status set locally to $newStatus (Sync: ${it.message})"
            }
        }
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(16.dp)
    ) {
        // Duty Status Header
        Card(
            modifier = Modifier.fillMaxWidth(),
            colors = CardDefaults.cardColors(
                containerColor = if (isOnline) Color(0xFFDCFCE7) else Color(0xFFFEE2E2)
            ),
            shape = RoundedCornerShape(14.dp)
        ) {
            Row(
                modifier = Modifier.padding(16.dp).fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Column {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Surface(
                            shape = CircleShape,
                            color = if (isOnline) Color(0xFF15803D).copy(alpha = pulseAlpha) else Color.Red,
                            modifier = Modifier.size(10.dp)
                        ) {}
                        Spacer(modifier = Modifier.width(8.dp))
                        Text(
                            text = if (isOnline) {
                                if (isTelugu) "🟢 మీరు విధుల్లో ఉన్నారు (ఆన్‌లైన్)" else if (isHindi) "🟢 आप ड्यूटी पर हैं (ऑनलाइन)" else "🟢 YOU ARE ONLINE & ON DUTY"
                            } else {
                                if (isTelugu) "🔴 విధుల నుండి విరామం (ఆఫ్‌లైన్)" else if (isHindi) "🔴 आप ऑफलाइन हैं" else "🔴 YOU ARE OFFLINE"
                            },
                            fontWeight = FontWeight.Bold,
                            color = if (isOnline) Color(0xFF15803D) else Color(0xFFB91C1C)
                        )
                    }
                    Spacer(modifier = Modifier.height(2.dp))
                    Text(
                        text = if (isOnline) {
                            if (isTelugu) "హైదరాబాద్ పురోహిత డిస్పాచ్ నెట్‌వర్క్‌లో అందుబాటులో ఉన్నారు" else if (isHindi) "हैदराबाद पुरोहित नेटवर्क पर सक्रिय" else "Active on Hyderabad Purohit Dispatch Mesh"
                        } else {
                            if (isTelugu) "పూజా పిలుపులు అందుకోవడానికి విధులను ప్రారంభించండి" else if (isHindi) "बुकिंग प्राप्त करने के लिए ड्यूटी चालू करें" else "Switch ON duty to receive bookings"
                        },
                        style = MaterialTheme.typography.bodySmall
                    )
                }
                Switch(
                    checked = isOnline,
                    onCheckedChange = { isOnline = it }
                )
            }
        }

        Spacer(modifier = Modifier.height(12.dp))

        // High Precision GNSS Telemetry Beacon Card
        Card(
            modifier = Modifier.fillMaxWidth(),
            colors = CardDefaults.cardColors(containerColor = Color(0xFF0F172A)),
            shape = RoundedCornerShape(14.dp)
        ) {
            Column(modifier = Modifier.padding(14.dp)) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text(
                        text = "🛰️ HIGH-PRECISION GNSS TELEMETRY",
                        color = Color(0xFF4ADE80),
                        fontWeight = FontWeight.Bold,
                        style = MaterialTheme.typography.labelSmall
                    )
                    Text(
                        text = "±3.4m (L1+L5 RTK FIXED)",
                        color = Color.White,
                        fontWeight = FontWeight.Bold,
                        style = MaterialTheme.typography.labelSmall
                    )
                }
                Spacer(modifier = Modifier.height(6.dp))
                Text(
                    text = "Coordinates: 17.3865° N, 78.4880° E • Hyderabad Hub",
                    color = Color(0xFFCBD5E1),
                    style = MaterialTheme.typography.bodySmall
                )
                Spacer(modifier = Modifier.height(4.dp))
                Text(
                    text = "Broadcasting live transit beacon to devotee booking radar via WebSocket",
                    color = Color(0xFF94A3B8),
                    style = MaterialTheme.typography.labelSmall
                )
            }
        }

        Spacer(modifier = Modifier.height(14.dp))

        // IF ACTIVE BOOKING ASSIGNED TO THIS PRIEST
        if (hasActiveBooking) {
            Card(
                modifier = Modifier.fillMaxWidth(),
                shape = RoundedCornerShape(16.dp),
                colors = CardDefaults.cardColors(containerColor = Color(0xFFFFFBEB)),
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
                                text = "🛵 $bookingStatus",
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

                    Spacer(modifier = Modifier.height(10.dp))

                    // Destination Address Box
                    Surface(
                        color = Color.White,
                        shape = RoundedCornerShape(10.dp),
                        modifier = Modifier.fillMaxWidth()
                    ) {
                        Column(modifier = Modifier.padding(12.dp)) {
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween
                            ) {
                                Text("👤 Devotee: $devoteeName", fontWeight = FontWeight.Bold, style = MaterialTheme.typography.bodyMedium)
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
                                text = "🛰️ Devotee GPS: 17.3850° N, 78.4867° E (Landmark: Behind Sri Rama Temple)",
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

                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(6.dp)
                    ) {
                        Button(
                            onClick = { updateStatus("IN_TRANSIT") },
                            modifier = Modifier.weight(1f),
                            shape = RoundedCornerShape(8.dp),
                            enabled = bookingStatus != "IN_TRANSIT" && bookingStatus != "COMPLETED"
                        ) {
                            Text(if (isTelugu) "1. ప్రయాణం" else if (isHindi) "1. प्रस्थान" else "1. In Transit", style = MaterialTheme.typography.labelSmall)
                        }

                        Button(
                            onClick = { updateStatus("ARRIVED") },
                            modifier = Modifier.weight(1f),
                            shape = RoundedCornerShape(8.dp),
                            enabled = bookingStatus == "IN_TRANSIT"
                        ) {
                            Text(if (isTelugu) "2. చేరిక" else if (isHindi) "2. आगमन" else "2. Arrived", style = MaterialTheme.typography.labelSmall)
                        }

                        Button(
                            onClick = { updateStatus("POOJA_STARTED") },
                            modifier = Modifier.weight(1f),
                            shape = RoundedCornerShape(8.dp),
                            enabled = bookingStatus == "ARRIVED"
                        ) {
                            Text(if (isTelugu) "3. పూజ మొదలు" else if (isHindi) "3. पूजा प्रारंभ" else "3. Start Puja", style = MaterialTheme.typography.labelSmall)
                        }

                        Button(
                            onClick = { showOtpDialog = true },
                            modifier = Modifier.weight(1f),
                            shape = RoundedCornerShape(8.dp),
                            colors = ButtonDefaults.buttonColors(containerColor = Color(0xFF15803D)),
                            enabled = bookingStatus == "POOJA_STARTED"
                        ) {
                            Text(if (isTelugu) "4. సమాప్తం" else if (isHindi) "4. संपन्न" else "4. Complete", style = MaterialTheme.typography.labelSmall)
                        }
                    }
                }
            }

            Spacer(modifier = Modifier.height(14.dp))
        }

        // Daily Earnings & Metrics Card
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            Card(modifier = Modifier.weight(1f)) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Text(if (isTelugu) "నేటి దక్షిణ" else if (isHindi) "आज की दक्षिणा" else "Today's Dakshina", style = MaterialTheme.typography.labelSmall)
                    Text("₹7,632", fontWeight = FontWeight.Bold, style = MaterialTheme.typography.headlineSmall)
                    Text(if (isTelugu) "2 పూజలు పూర్తయ్యాయి" else if (isHindi) "2 पूजा संपन्न" else "2 Poojas completed", style = MaterialTheme.typography.bodySmall, color = Color.Gray)
                }
            }
            Card(modifier = Modifier.weight(1f)) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Text(if (isTelugu) "రేటింగ్ & అనుభవం" else if (isHindi) "रेटिंग एवं प्रतिष्ठा" else "Ranking & Score", style = MaterialTheme.typography.labelSmall)
                    Text("★ 4.95", fontWeight = FontWeight.Bold, style = MaterialTheme.typography.headlineSmall, color = MaterialTheme.colorScheme.primary)
                    Text(if (isTelugu) "42 పునరావృత భక్తులు" else if (isHindi) "42 नियमित भक्त" else "42 Repeat Devotees", style = MaterialTheme.typography.bodySmall, color = Color.Gray)
                }
            }
        }

        Spacer(modifier = Modifier.height(18.dp))

        // Test Incoming Alert Simulator
        Text(if (isTelugu) "డిస్పాచ్ ఇంజిన్ పిలుపు (టెస్ట్):" else if (isHindi) "डिस्पैच इंजन कॉल:" else "Dispatch Engine Trigger:", fontWeight = FontWeight.Bold)
        Spacer(modifier = Modifier.height(6.dp))
        Button(
            onClick = onSimulateIncomingOrder,
            modifier = Modifier.fillMaxWidth(),
            enabled = isOnline
        ) {
            Text(if (isTelugu) "⚡ నూతన పూజా అలర్ట్ (పరీక్ష)" else if (isHindi) "⚡ नई पूजा अलर्ट (परीक्षण)" else "⚡ Trigger Incoming Rapido Alert (Test)")
        }

        if (statusMessage != null) {
            Spacer(modifier = Modifier.height(8.dp))
            Text(
                text = statusMessage!!,
                style = MaterialTheme.typography.labelSmall,
                color = Color(0xFF15803D),
                fontWeight = FontWeight.Bold
            )
        }

        Spacer(modifier = Modifier.height(24.dp))
    }

    // OTP Completion Dialog
    if (showOtpDialog) {
        AlertDialog(
            onDismissRequest = { showOtpDialog = false },
            title = { Text("Enter Devotee Completion OTP", fontWeight = FontWeight.Bold) },
            text = {
                Column {
                    Text(
                        text = "Ask devotee for the 4-digit OTP shown on their Live Tracking screen (e.g. 4480) to release escrow payout:",
                        style = MaterialTheme.typography.bodySmall
                    )
                    Spacer(modifier = Modifier.height(10.dp))
                    OutlinedTextField(
                        value = enteredOtp,
                        onValueChange = { if (it.length <= 6) enteredOtp = it },
                        label = { Text("4-Digit OTP Code") },
                        modifier = Modifier.fillMaxWidth(),
                        singleLine = true
                    )
                }
            },
            confirmButton = {
                Button(
                    onClick = {
                        if (enteredOtp.isNotBlank()) {
                            showOtpDialog = false
                            updateStatus("COMPLETED", enteredOtp.trim())
                        }
                    },
                    colors = ButtonDefaults.buttonColors(containerColor = Color(0xFF15803D))
                ) {
                    Text("Verify & Release ₹${dakshinaAmount.toInt()}")
                }
            },
            dismissButton = {
                OutlinedButton(onClick = { showOtpDialog = false }) {
                    Text("Cancel")
                }
            }
        )
    }
}
