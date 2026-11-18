package com.purohit.devotee.ui.resilience

import androidx.compose.animation.AnimatedVisibility
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
import androidx.compose.ui.draw.scale
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import com.purohit.contracts.i18n.StringContract
import com.purohit.devotee.motionAllowed
import com.purohit.devotee.SunLogo

@Composable
fun StickyOfflineBanner(
    isOffline: Boolean,
    strings: StringContract
) {
    if (isOffline) {
        Surface(
            modifier = Modifier.fillMaxWidth(),
            color = Color(0xFFF59E0B) // Amber
        ) {
            Row(
                modifier = Modifier.padding(horizontal = 16.dp, vertical = 6.dp),
                verticalAlignment = Alignment.CenterVertically,
                horizontalArrangement = Arrangement.Center
            ) {
                Text(
                    text = strings.error.offlineBannerNotice,
                    style = MaterialTheme.typography.labelSmall,
                    color = Color.Black,
                    fontWeight = FontWeight.Bold
                )
            }
        }
    }
}

@Composable
fun LogoutConfirmationModal(
    hasActiveRitual: Boolean,
    checkFailedMessage: String? = null,
    strings: StringContract,
    onConfirmLogout: () -> Unit,
    onDismiss: () -> Unit
) {
    AlertDialog(
        onDismissRequest = onDismiss,
        title = {
            Text(
                text = strings.common.logOutConfirmTitle,
                fontWeight = FontWeight.Bold
            )
        },
        text = {
            Column {
                Text(strings.common.logOutConfirmDescription)
                if (checkFailedMessage != null) {
                    Spacer(modifier = Modifier.height(10.dp))
                    Text(checkFailedMessage, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
                }
                if (hasActiveRitual) {
                    Spacer(modifier = Modifier.height(10.dp))
                    Surface(
                        color = MaterialTheme.colorScheme.surfaceVariant,
                        shape = RoundedCornerShape(8.dp)
                    ) {
                        Text(
                            text = strings.common.logOutActiveRitualWarning,
                            modifier = Modifier.padding(8.dp),
                            style = MaterialTheme.typography.bodySmall,
                            color = Color(0xFF92400E)
                        )
                    }
                }
            }
        },
        confirmButton = {
            Button(
                onClick = onConfirmLogout,
                colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.error)
            ) {
                Text(strings.common.logOutConfirmButton)
            }
        },
        dismissButton = {
            OutlinedButton(onClick = onDismiss) {
                Text(strings.common.cancelAction)
            }
        }
    )
}

@Composable
fun ShimmerCardSkeleton(
    modifier: Modifier = Modifier
) {
    Card(
        modifier = modifier.fillMaxWidth().height(120.dp),
        shape = RoundedCornerShape(12.dp),
        colors = CardDefaults.cardColors(containerColor = Color(0xFFE2E8F0))
    ) {
        Box(modifier = Modifier.fillMaxSize(), contentAlignment = Alignment.Center) {
            CircularProgressIndicator(strokeWidth = 2.dp, modifier = Modifier.size(24.dp))
        }
    }
}

/**
 * Full-Screen No Internet / Connection Lost Screen
 */
@Composable
fun NoInternetScreen(
    strings: StringContract,
    onRetry: () -> Unit,
    serverUrl: String = "",
    modifier: Modifier = Modifier
) {
    val pulseScale = if (motionAllowed()) {
        val infiniteTransition = rememberInfiniteTransition(label = "offline_pulse")
        val animated by infiniteTransition.animateFloat(
        initialValue = 0.92f,
        targetValue = 1.08f,
        animationSpec = infiniteRepeatable(
            animation = tween(1200, easing = EaseInOutQuad),
            repeatMode = RepeatMode.Reverse
        ),
        label = "pulse_scale"
    )
        animated
    } else 1f

    Column(
        modifier = modifier
            .fillMaxSize()
            .background(MaterialTheme.colorScheme.background)
            .padding(24.dp)
            .verticalScroll(rememberScrollState()),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center
    ) {
        // Pulsing Offline Emblem
        Surface(
            shape = CircleShape,
            color = Color(0xFFFEE2E2),
            modifier = Modifier
                .size(110.dp)
                .scale(pulseScale)
        ) {
            Box(contentAlignment = Alignment.Center) {
                Text(
                    text = "📡",
                    style = MaterialTheme.typography.displaySmall
                )
            }
        }

        Spacer(modifier = Modifier.height(24.dp))

        Text(
            text = strings.error.networkDisconnectedTitle.ifBlank { "No Internet Connection" },
            style = MaterialTheme.typography.headlineSmall,
            fontWeight = FontWeight.Bold,
            color = MaterialTheme.colorScheme.onBackground,
            textAlign = TextAlign.Center
        )

        Spacer(modifier = Modifier.height(8.dp))

        Text(
            text = strings.error.networkDisconnectedDescription.ifBlank {
                "Unable to connect to Puja Vaahini dispatch mesh. Please verify your mobile data or Wi-Fi network."
            },
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            textAlign = TextAlign.Center,
            modifier = Modifier.padding(horizontal = 12.dp)
        )

        if (serverUrl.isNotBlank()) {
            Spacer(modifier = Modifier.height(16.dp))
            Surface(
                shape = RoundedCornerShape(10.dp),
                color = MaterialTheme.colorScheme.surfaceVariant,
                modifier = Modifier.fillMaxWidth()
            ) {
                Column(modifier = Modifier.padding(12.dp)) {
                    Text(
                        text = "🌐 Active Gateway URL:",
                        style = MaterialTheme.typography.labelSmall,
                        color = MaterialTheme.colorScheme.primary,
                        fontWeight = FontWeight.Bold
                    )
                    Spacer(modifier = Modifier.height(2.dp))
                    Text(
                        text = serverUrl,
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }
            }
        }

        Spacer(modifier = Modifier.height(28.dp))

        Button(
            onClick = onRetry,
            modifier = Modifier
                .fillMaxWidth()
                .height(50.dp),
            shape = RoundedCornerShape(12.dp),
            colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.primary)
        ) {
            Text(
                text = "🔄 " + strings.error.tryAgainButton.ifBlank { "Retry Connection" },
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.titleSmall
            )
        }
    }
}

/**
 * Full-Screen Error / Failure Screen
 */
@Composable
fun ErrorScreen(
    title: String = "",
    errorMessage: String,
    strings: StringContract,
    onRetry: () -> Unit,
    onNavigateHome: (() -> Unit)? = null,
    modifier: Modifier = Modifier
) {
    var showTechDetails by remember { mutableStateOf(false) }

    Column(
        modifier = modifier
            .fillMaxSize()
            .background(MaterialTheme.colorScheme.background)
            .padding(24.dp)
            .verticalScroll(rememberScrollState()),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center
    ) {
        Surface(
            shape = CircleShape,
            color = MaterialTheme.colorScheme.surfaceVariant,
            modifier = Modifier.size(96.dp)
        ) {
            Box(contentAlignment = Alignment.Center) {
                Text(
                    text = "⚠️",
                    style = MaterialTheme.typography.headlineLarge
                )
            }
        }

        Spacer(modifier = Modifier.height(20.dp))

        Text(
            text = if (title.isNotBlank()) title else strings.error.genericError.ifBlank { "Something Went Wrong" },
            style = MaterialTheme.typography.headlineSmall,
            fontWeight = FontWeight.Bold,
            color = MaterialTheme.colorScheme.onBackground,
            textAlign = TextAlign.Center
        )

        Spacer(modifier = Modifier.height(8.dp))

        Text(
            text = errorMessage.ifBlank { "We encountered an unexpected error while communicating with the cloud services." },
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            textAlign = TextAlign.Center,
            modifier = Modifier.padding(horizontal = 8.dp)
        )

        Spacer(modifier = Modifier.height(16.dp))

        TextButton(onClick = { showTechDetails = !showTechDetails }) {
            Text(
                text = if (showTechDetails) "▲ Hide Diagnostics" else "▼ Show Technical Details",
                style = MaterialTheme.typography.labelSmall
            )
        }

        if (showTechDetails) {
            Surface(
                shape = RoundedCornerShape(8.dp),
                color = Color(0xFFF1F5F9),
                modifier = Modifier.fillMaxWidth()
            ) {
                Text(
                    text = errorMessage,
                    style = MaterialTheme.typography.bodySmall,
                    color = Color(0xFF334155),
                    modifier = Modifier.padding(12.dp)
                )
            }
        }

        Spacer(modifier = Modifier.height(24.dp))

        Button(
            onClick = onRetry,
            modifier = Modifier
                .fillMaxWidth()
                .height(50.dp),
            shape = RoundedCornerShape(12.dp),
            colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.primary)
        ) {
            Text(
                text = "🔄 " + strings.error.tryAgainButton.ifBlank { "Try Again" },
                fontWeight = FontWeight.Bold
            )
        }

        if (onNavigateHome != null) {
            Spacer(modifier = Modifier.height(10.dp))
            OutlinedButton(
                onClick = onNavigateHome,
                modifier = Modifier
                    .fillMaxWidth()
                    .height(48.dp),
                shape = RoundedCornerShape(12.dp)
            ) {
                Text("🏠 Return to Services")
            }
        }
    }
}

/**
 * Dedicated Vedic Loading Screen with Diya / Om animation
 */
@Composable
fun VedicLoadingScreen(
    title: String = "Connecting to Vedic Dispatch Mesh...",
    subtitle: String = "Loading available information...",
    modifier: Modifier = Modifier
) {
    val diyaScale = if (motionAllowed()) {
        val infiniteTransition = rememberInfiniteTransition(label = "vedic_diya")
        val animated by infiniteTransition.animateFloat(
        initialValue = 0.9f,
        targetValue = 1.1f,
        animationSpec = infiniteRepeatable(
            animation = tween(800, easing = FastOutSlowInEasing),
            repeatMode = RepeatMode.Reverse
        ),
        label = "diya_scale"
    )
        animated
    } else 1f

    Column(
        modifier = modifier
            .fillMaxSize()
            .background(MaterialTheme.colorScheme.background)
            .padding(24.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center
    ) {
        Surface(
            shape = CircleShape,
            color = MaterialTheme.colorScheme.surfaceVariant,
            modifier = Modifier
                .size(100.dp)
                .scale(diyaScale)
        ) {
            Box(contentAlignment = Alignment.Center) {
                SunLogo(Modifier.size(width = 64.dp, height = 44.dp))
            }
        }

        Spacer(modifier = Modifier.height(20.dp))

        Text(
            text = title,
            style = MaterialTheme.typography.titleLarge,
            fontWeight = FontWeight.Bold,
            color = MaterialTheme.colorScheme.primary,
            textAlign = TextAlign.Center
        )

        Spacer(modifier = Modifier.height(6.dp))

        Text(
            text = subtitle,
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant,
            textAlign = TextAlign.Center
        )

        Spacer(modifier = Modifier.height(24.dp))

        LinearProgressIndicator(
            color = MaterialTheme.colorScheme.primary,
            trackColor = MaterialTheme.colorScheme.surfaceVariant,
            modifier = Modifier
                .fillMaxWidth(0.6f)
                .height(4.dp)
        )
    }
}
