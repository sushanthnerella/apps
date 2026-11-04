package com.purohit.devotee.ui.tracking

import androidx.compose.animation.core.*
import androidx.compose.foundation.background
import androidx.compose.foundation.border
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
import com.purohit.contracts.booking.BookingState
import com.purohit.devotee.network.PurohitApiClient
import com.purohit.devotee.ui.catalog.RequiredItemsModal
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

@Composable
fun LiveTrackingScreen(
    priestName: String = "Brahmasri Sushanth Sastry",
    priestPhone: String = "+91 98765 43210",
    priestSampradaya: String = "Smartha • Rigveda Shakha",
    poojaTitle: String = "Vedic Puja Ceremony",
    totalAmount: Double = 3716.0,
    completionOtp: String = "4921",
    initialDistanceKm: Double = 1.8,
    bookingId: String = "f312db8f-1d71-4cab-b5ba-77fbead70c71",
    chatRoomId: String = "d66283f3-1eb0-40b3-afc0-c37d355b90f8",
    poojaId: String = "5dccda01-fcb9-4bc2-9ef9-6015419533bc",
    eventAddress: String = "Plot 42, Sri Rama Nagar, Madhav Nagar, Hyderabad (Landmark: Behind Sri Rama Temple)",
    isSamagriIncluded: Boolean = true,
    onOpenChat: () -> Unit = {},
    onOpenLiveStream: () -> Unit = {},
    onCancelBooking: () -> Unit = {},
    onBackToHome: () -> Unit = {}
) {
    var currentState by remember { mutableStateOf(BookingState.IN_TRANSIT) }
    var dakshinaAdded by remember { mutableStateOf(0) }
    var currentDistanceKm by remember { mutableStateOf(initialDistanceKm) }
    var currentSpeedKmh by remember { mutableStateOf(28) }
    var etaMinutes by remember { mutableStateOf((initialDistanceKm * 3.5).toInt().coerceAtLeast(3)) }
    var gnssAccuracyMeters by remember { mutableStateOf(3.8) }
    var satelliteCount by remember { mutableStateOf(21) }
    var isEscrowReleased by remember { mutableStateOf(false) }
    var showSamagriModal by remember { mutableStateOf(false) }

    // Pulsing animation for live GPS signal
    val infiniteTransition = rememberInfiniteTransition(label = "gps_pulse")
    val pulseAlpha by infiniteTransition.animateFloat(
        initialValue = 0.3f,
        targetValue = 1.0f,
        animationSpec = infiniteRepeatable(
            animation = tween(1000, easing = LinearEasing),
            repeatMode = RepeatMode.Reverse
        ),
        label = "pulse_alpha"
    )

    // Dynamic GPS Traveling Progression Simulation
    LaunchedEffect(currentState) {
        if (currentState == BookingState.IN_TRANSIT) {
            while (currentDistanceKm > 0.05) {
                delay(3000L) // update every 3 seconds
                currentDistanceKm = (currentDistanceKm - 0.3).coerceAtLeast(0.0)
                currentDistanceKm = (Math.round(currentDistanceKm * 10.0) / 10.0)
                etaMinutes = (currentDistanceKm * 3.5).toInt().coerceAtLeast(1)
                currentSpeedKmh = if (currentDistanceKm > 0.2) (24..34).random() else 12
                gnssAccuracyMeters = (32..42).random() / 10.0
                satelliteCount = (19..24).random()

                if (currentDistanceKm <= 0.05) {
                    currentState = BookingState.ARRIVED
                    currentSpeedKmh = 0
                    etaMinutes = 0
                    break
                }
            }
        }
    }

    val coroutineScope = rememberCoroutineScope()

    // 2-Way Realtime Cloud Sync: Reflects Priest's Actions (Arrival, Puja Started, Completed)
    LaunchedEffect(bookingId) {
        if (bookingId.isNotBlank()) {
            while (true) {
                val res = PurohitApiClient.fetchBookingDetails(bookingId)
                res.onSuccess { details ->
                    when (details.status) {
                        "IN_TRANSIT" -> currentState = BookingState.IN_TRANSIT
                        "ARRIVED" -> {
                            currentState = BookingState.ARRIVED
                            currentDistanceKm = 0.0
                            etaMinutes = 0
                            currentSpeedKmh = 0
                        }
                        "IN_PROGRESS", "POOJA_STARTED" -> {
                            currentState = BookingState.POOJA_STARTED
                            currentDistanceKm = 0.0
                            etaMinutes = 0
                            currentSpeedKmh = 0
                        }
                        "COMPLETED" -> {
                            currentState = BookingState.COMPLETED
                            isEscrowReleased = (details.escrow_status == "RELEASED")
                        }
                    }
                }
                delay(2500L)
            }
        }
    }

    fun syncStatusToCloud(newStatus: String, otp: String? = null, dakshina: Double = 0.0) {
        coroutineScope.launch {
            if (bookingId.isNotBlank()) {
                PurohitApiClient.updateBookingStatus(
                    bookingId = bookingId,
                    status = newStatus,
                    priestId = "6e7d277c-4ee2-469f-96ac-331517389762",
                    otp = otp,
                    dakshina = dakshina
                )
            }
        }
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(rememberScrollState())
            .padding(16.dp)
    ) {
        // Top Header
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            Column {
                Text(
                    text = "🛵 Purohit Live Tracking",
                    style = MaterialTheme.typography.headlineSmall,
                    fontWeight = FontWeight.Bold,
                    color = MaterialTheme.colorScheme.primary
                )
                Text(
                    text = "Live telemetry & GPS navigation radar",
                    style = MaterialTheme.typography.bodySmall,
                    color = Color.Gray
                )
            }
            Surface(
                shape = RoundedCornerShape(12.dp),
                color = when (currentState) {
                    BookingState.IN_TRANSIT -> Color(0xFFFEF3C7)
                    BookingState.ARRIVED -> Color(0xFFDCFCE7)
                    BookingState.POOJA_STARTED -> Color(0xFFDBEAFE)
                    BookingState.COMPLETED -> Color(0xFFE0E7FF)
                    else -> MaterialTheme.colorScheme.surfaceVariant
                }
            ) {
                Text(
                    text = when (currentState) {
                        BookingState.IN_TRANSIT -> "🛵 EN ROUTE"
                        BookingState.ARRIVED -> "📍 ARRIVED"
                        BookingState.POOJA_STARTED -> "🪔 PUJA ACTIVE"
                        BookingState.COMPLETED -> "✅ COMPLETED"
                        else -> currentState.name
                    },
                    modifier = Modifier.padding(horizontal = 10.dp, vertical = 6.dp),
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.labelSmall,
                    color = when (currentState) {
                        BookingState.IN_TRANSIT -> Color(0xFFB45309)
                        BookingState.ARRIVED -> Color(0xFF15803D)
                        BookingState.POOJA_STARTED -> Color(0xFF1D4ED8)
                        BookingState.COMPLETED -> Color(0xFF4338CA)
                        else -> Color.Black
                    }
                )
            }
        }

        Spacer(modifier = Modifier.height(12.dp))

        // Devotee Residence (Pickup / Ritual Venue) Card
        Card(
            modifier = Modifier.fillMaxWidth(),
            shape = RoundedCornerShape(14.dp),
            colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant)
        ) {
            Column(modifier = Modifier.padding(14.dp)) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text(
                        text = "🏡 Devotee Residence (Puja Venue)",
                        style = MaterialTheme.typography.labelSmall,
                        color = Color.DarkGray,
                        fontWeight = FontWeight.Bold
                    )
                    Surface(
                        shape = RoundedCornerShape(8.dp),
                        color = Color(0xFFDCFCE7)
                    ) {
                        Text(
                            text = "🛰️ Dual L1+L5 Locked",
                            style = MaterialTheme.typography.labelSmall,
                            color = Color(0xFF15803D),
                            fontWeight = FontWeight.Bold,
                            modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp)
                        )
                    }
                }
                Spacer(modifier = Modifier.height(4.dp))
                Text(
                    text = eventAddress,
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.titleSmall
                )
                Spacer(modifier = Modifier.height(2.dp))
                Text(
                    text = "GPS: 17.3850° N, 78.4867° E • Flat 402, Behind Sri Rama Temple",
                    style = MaterialTheme.typography.bodySmall,
                    color = Color.DarkGray
                )
            }
        }

        Spacer(modifier = Modifier.height(12.dp))

        // 1. GNSS Location Accuracy & Dual-Band Sensor Pill
        Card(
            modifier = Modifier.fillMaxWidth(),
            colors = CardDefaults.cardColors(containerColor = Color(0xFF0F172A)),
            shape = RoundedCornerShape(14.dp)
        ) {
            Column(modifier = Modifier.padding(12.dp)) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Row(verticalAlignment = Alignment.CenterVertically) {
                        Box(
                            modifier = Modifier
                                .size(10.dp)
                                .clip(CircleShape)
                                .background(Color(0xFF22C55E).copy(alpha = pulseAlpha))
                        )
                        Spacer(modifier = Modifier.width(8.dp))
                        Text(
                            text = "GNSS HIGH-PRECISION LOCK",
                            color = Color(0xFF4ADE80),
                            fontWeight = FontWeight.Bold,
                            style = MaterialTheme.typography.labelSmall
                        )
                    }
                    Text(
                        text = "±${gnssAccuracyMeters}m Accuracy",
                        color = Color.White,
                        fontWeight = FontWeight.Bold,
                        style = MaterialTheme.typography.labelSmall
                    )
                }

                Spacer(modifier = Modifier.height(6.dp))

                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween
                ) {
                    Text(
                        text = "🛰️ $satelliteCount Satellites (NavIC L5 + GPS L1)",
                        color = Color(0xFF94A3B8),
                        style = MaterialTheme.typography.labelSmall
                    )
                    Text(
                        text = "Carrier Lock: RTK FIXED",
                        color = Color(0xFF38BDF8),
                        fontWeight = FontWeight.SemiBold,
                        style = MaterialTheme.typography.labelSmall
                    )
                }
            }
        }

        Spacer(modifier = Modifier.height(12.dp))

        // 2. Simulated Live Route Map & Distance Progress Card
        Card(
            modifier = Modifier.fillMaxWidth(),
            shape = RoundedCornerShape(16.dp),
            colors = CardDefaults.cardColors(containerColor = Color(0xFFF1F5F9))
        ) {
            Column(modifier = Modifier.padding(16.dp)) {
                // Live Traveling Metrics
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Column {
                        Text(
                            text = if (currentDistanceKm > 0.05) "$currentDistanceKm km away" else "Purohit at your Doorstep!",
                            style = MaterialTheme.typography.titleLarge,
                            fontWeight = FontWeight.Bold,
                            color = if (currentDistanceKm > 0.05) Color(0xFF0F172A) else Color(0xFF15803D)
                        )
                        Text(
                            text = if (currentDistanceKm > 0.05) "Estimated arrival in ~$etaMinutes mins • Speed: ${currentSpeedKmh} km/h" else "Doorstep transit complete",
                            style = MaterialTheme.typography.bodySmall,
                            color = Color(0xFF475569)
                        )
                    }
                    Text(
                        text = if (currentDistanceKm > 0.05) "🛵" else "🏡",
                        style = MaterialTheme.typography.headlineMedium
                    )
                }

                Spacer(modifier = Modifier.height(14.dp))

                // Transit Progress Bar
                val progressFraction = if (initialDistanceKm > 0) {
                    ((initialDistanceKm - currentDistanceKm) / initialDistanceKm).coerceIn(0.0, 1.0).toFloat()
                } else 1.0f

                LinearProgressIndicator(
                    progress = { progressFraction },
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(8.dp)
                        .clip(RoundedCornerShape(4.dp)),
                    color = MaterialTheme.colorScheme.primary,
                    trackColor = Color(0xFFCBD5E1)
                )

                Spacer(modifier = Modifier.height(10.dp))

                // Progress Points: Priest Origin -> In Transit -> Devotee Residence
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween
                ) {
                    Text("🛵 Ayagaryllu", style = MaterialTheme.typography.labelSmall, color = Color.Gray)
                    Text("📍 Real-time Transit", style = MaterialTheme.typography.labelSmall, fontWeight = FontWeight.Bold, color = MaterialTheme.colorScheme.primary)
                    Text("🏡 Devotee Home", style = MaterialTheme.typography.labelSmall, color = Color.Gray)
                }
            }
        }

        Spacer(modifier = Modifier.height(12.dp))

        // 3. Assigned Purohit Details & Contact Card
        Card(
            modifier = Modifier.fillMaxWidth(),
            shape = RoundedCornerShape(16.dp),
            colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface),
            elevation = CardDefaults.cardElevation(defaultElevation = 2.dp)
        ) {
            Column(modifier = Modifier.padding(16.dp)) {
                Text("Assigned Vedic Purohit:", style = MaterialTheme.typography.labelSmall, color = Color.Gray)
                Text(
                    text = priestName,
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.titleMedium
                )
                Text(
                    text = priestSampradaya,
                    color = MaterialTheme.colorScheme.primary,
                    style = MaterialTheme.typography.bodySmall,
                    fontWeight = FontWeight.SemiBold
                )

                Spacer(modifier = Modifier.height(10.dp))

                // Sacred Samagri Kit & Household Checklist Row
                Surface(
                    color = Color(0xFFFEF3C7),
                    shape = RoundedCornerShape(10.dp),
                    modifier = Modifier.fillMaxWidth()
                ) {
                    Row(
                        modifier = Modifier.padding(10.dp).fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Column(modifier = Modifier.weight(1f)) {
                            Text(
                                text = if (isSamagriIncluded) "🪔 Sacred Samagri Kit: Included" else "Sacred Samagri: Excluded",
                                fontWeight = FontWeight.Bold,
                                style = MaterialTheme.typography.labelSmall,
                                color = Color(0xFF78350F)
                            )
                            Text(
                                text = "Purohit brings homa sticks, pasupu, kumkum & sacred grains.",
                                style = MaterialTheme.typography.labelSmall,
                                color = Color(0xFF92400E)
                            )
                        }
                        OutlinedButton(
                            onClick = { showSamagriModal = true },
                            shape = RoundedCornerShape(8.dp),
                            colors = ButtonDefaults.outlinedButtonColors(contentColor = Color(0xFF78350F))
                        ) {
                            Text("👁️ View Items", style = MaterialTheme.typography.labelSmall)
                        }
                    }
                }

                Spacer(modifier = Modifier.height(10.dp))

                // Devotee Communication Buttons: Chat & Call
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    Button(
                        onClick = onOpenChat,
                        modifier = Modifier.weight(1.2f),
                        shape = RoundedCornerShape(10.dp),
                        colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.primary)
                    ) {
                        Text("💬 Chat with Purohit", style = MaterialTheme.typography.labelSmall)
                    }

                    Button(
                        onClick = { /* Native phone call intent */ },
                        modifier = Modifier.weight(1f),
                        shape = RoundedCornerShape(10.dp),
                        colors = ButtonDefaults.buttonColors(containerColor = Color(0xFF15803D))
                    ) {
                        Text("📞 Call", style = MaterialTheme.typography.labelSmall)
                    }

                    OutlinedButton(
                        onClick = { /* WhatsApp chat */ },
                        modifier = Modifier.weight(0.8f),
                        shape = RoundedCornerShape(10.dp)
                    ) {
                        Text("WhatsApp", style = MaterialTheme.typography.labelSmall)
                    }
                }
            }
        }

        Spacer(modifier = Modifier.height(12.dp))

        // 4. Escrow Protection & Completion OTP Card
        Card(
            modifier = Modifier.fillMaxWidth(),
            shape = RoundedCornerShape(16.dp),
            colors = CardDefaults.cardColors(containerColor = Color(0xFFFFFBEB)),
            elevation = CardDefaults.cardElevation(defaultElevation = 1.dp)
        ) {
            Column(modifier = Modifier.padding(16.dp)) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Column {
                        Text("Ritual Escrow Protection", fontWeight = FontWeight.Bold, style = MaterialTheme.typography.titleSmall)
                        Text(poojaTitle, style = MaterialTheme.typography.bodySmall, color = Color.Gray)
                    }
                    Text(
                        text = "₹${totalAmount.toInt()}",
                        fontWeight = FontWeight.Bold,
                        style = MaterialTheme.typography.titleMedium,
                        color = MaterialTheme.colorScheme.primary
                    )
                }

                Spacer(modifier = Modifier.height(10.dp))

                // OTP Box
                Surface(
                    color = Color.White,
                    shape = RoundedCornerShape(10.dp),
                    modifier = Modifier.fillMaxWidth()
                ) {
                    Row(
                        modifier = Modifier.padding(12.dp).fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Column {
                            Text("Completion OTP Code", style = MaterialTheme.typography.labelSmall, color = Color.Gray)
                            Text(
                                text = completionOtp,
                                fontWeight = FontWeight.ExtraBold,
                                style = MaterialTheme.typography.headlineMedium,
                                color = MaterialTheme.colorScheme.primary
                            )
                        }
                        Text(
                            text = "🔐 Give to priest ONLY after pooja is finished to release payout",
                            style = MaterialTheme.typography.labelSmall,
                            color = Color.DarkGray,
                            modifier = Modifier.width(160.dp)
                        )
                    }
                }
            }
        }

        Spacer(modifier = Modifier.height(12.dp))

        // 5. Ritual State Machine Stepper Buttons
        Text("Booking State Controls:", fontWeight = FontWeight.Bold, style = MaterialTheme.typography.labelMedium)
        Spacer(modifier = Modifier.height(6.dp))

        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            Button(
                onClick = {
                    currentState = BookingState.ARRIVED
                    currentDistanceKm = 0.0
                    etaMinutes = 0
                    syncStatusToCloud("ARRIVED")
                },
                modifier = Modifier.weight(1f),
                shape = RoundedCornerShape(10.dp),
                enabled = currentState == BookingState.IN_TRANSIT
            ) {
                Text("Priest Arrived", style = MaterialTheme.typography.labelSmall)
            }

            Button(
                onClick = {
                    currentState = BookingState.POOJA_STARTED
                    syncStatusToCloud("POOJA_STARTED")
                },
                modifier = Modifier.weight(1f),
                shape = RoundedCornerShape(10.dp),
                enabled = currentState == BookingState.ARRIVED
            ) {
                Text("Start Puja", style = MaterialTheme.typography.labelSmall)
            }

            Button(
                onClick = {
                    currentState = BookingState.COMPLETED
                    isEscrowReleased = true
                    syncStatusToCloud("COMPLETED", otp = completionOtp, dakshina = dakshinaAdded.toDouble())
                },
                modifier = Modifier.weight(1f),
                shape = RoundedCornerShape(10.dp),
                colors = ButtonDefaults.buttonColors(containerColor = Color(0xFF15803D)),
                enabled = currentState == BookingState.POOJA_STARTED
            ) {
                Text("Complete", style = MaterialTheme.typography.labelSmall)
            }
        }

        Spacer(modifier = Modifier.height(10.dp))

        // 6. Live E-Pooja Telecast / WebRTC Sanctum Studio
        Button(
            onClick = onOpenLiveStream,
            modifier = Modifier.fillMaxWidth(),
            shape = RoundedCornerShape(10.dp),
            colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.secondary)
        ) {
            Text("🎥 Open Sanctum Studio (Live E-Pooja Stream)")
        }

        // 7. Completion & Digital Dakshina (if pooja started or completed)
        if (currentState == BookingState.POOJA_STARTED || currentState == BookingState.COMPLETED) {
            Spacer(modifier = Modifier.height(12.dp))
            Card(
                modifier = Modifier.fillMaxWidth(),
                shape = RoundedCornerShape(14.dp),
                colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.4f))
            ) {
                Column(modifier = Modifier.padding(14.dp)) {
                    Text("Offer Digital Tambulam / Dakshina:", fontWeight = FontWeight.Bold, style = MaterialTheme.typography.titleSmall)
                    Text("Express gratitude for the sacred blessings with a direct guru dakshina.", style = MaterialTheme.typography.bodySmall)
                    Spacer(modifier = Modifier.height(8.dp))
                    Row(horizontalArrangement = Arrangement.spacedBy(8.dp)) {
                        listOf(251, 501, 1116).forEach { amount ->
                            FilterChip(
                                selected = dakshinaAdded == amount,
                                onClick = { dakshinaAdded = amount },
                                label = { Text("₹$amount") }
                            )
                        }
                    }
                    if (dakshinaAdded > 0) {
                        Spacer(modifier = Modifier.height(6.dp))
                        Text(
                            text = "✓ ₹$dakshinaAdded will be transferred directly to ${priestName} along with escrow.",
                            style = MaterialTheme.typography.labelSmall,
                            color = Color(0xFF15803D),
                            fontWeight = FontWeight.Bold
                        )
                    }
                }
            }
        }

        Spacer(modifier = Modifier.height(12.dp))

        // 8. Cancellation or Return to Services
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            if (currentState != BookingState.COMPLETED) {
                OutlinedButton(
                    onClick = onCancelBooking,
                    modifier = Modifier.weight(1f),
                    shape = RoundedCornerShape(10.dp),
                    colors = ButtonDefaults.outlinedButtonColors(contentColor = MaterialTheme.colorScheme.error)
                ) {
                    Text("❌ Cancel Booking")
                }
            }

            Button(
                onClick = onBackToHome,
                modifier = Modifier.weight(1f),
                shape = RoundedCornerShape(10.dp)
            ) {
                Text("🏠 Home / Services")
            }
        }

        Spacer(modifier = Modifier.height(24.dp))
    }

    if (showSamagriModal) {
        RequiredItemsModal(
            poojaId = poojaId,
            poojaTitle = poojaTitle,
            onDismiss = { showSamagriModal = false }
        )
    }
}
