package com.purohit.devotee.ui.auth

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.semantics.liveRegion
import androidx.compose.ui.semantics.LiveRegionMode
import androidx.compose.ui.semantics.semantics
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.unit.dp
import com.purohit.devotee.SunLogo
import com.purohit.devotee.LocalAppLanguage
import com.purohit.contracts.i18n.AppLanguage
import com.purohit.contracts.i18n.LocalizationManager
import com.purohit.contracts.i18n.StringContract
import com.purohit.devotee.network.PurohitApiClient
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

private fun parseErrorMessage(raw: String, fallback: String): String {
    val messageRegex = """"message"\s*:\s*"([^"]+)"""".toRegex()
    val match = messageRegex.find(raw)
    if (match != null) {
        return match.groupValues[1]
    }
    val errorRegex = """"error"\s*:\s*"([^"]+)"""".toRegex()
    val errMatch = errorRegex.find(raw)
    if (errMatch != null) {
        return errMatch.groupValues[1].replace('_', ' ')
    }
    val clean = raw.trim().removePrefix("Exception: ").removePrefix("Server returned ")
    return if (clean.isNotBlank()) clean else fallback
}

@Composable
fun DevoteeLoginScreen(
    onLoginSuccess: () -> Unit,
    onNavigateToRegister: () -> Unit,
    strings: StringContract = LocalizationManager.getStrings(AppLanguage.ENGLISH)
) {
    val language = LocalAppLanguage.current
    fun tr(en: String, te: String, hi: String) = when (language) { AppLanguage.TELUGU -> te; AppLanguage.HINDI -> hi; else -> en }
    var phone by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var passwordVisible by remember { mutableStateOf(false) }

    var isLoading by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var successMessage by remember { mutableStateOf<String?>(null) }

    // Password Recovery Dialog State
    var showForgotPasswordModal by remember { mutableStateOf(false) }
    var recoveryEmail by remember { mutableStateOf("") }
    var recoveryError by remember { mutableStateOf<String?>(null) }
    var isRequestingLink by remember { mutableStateOf(false) }
    var emailSentSuccess by remember { mutableStateOf(false) }
    var cooldownSeconds by remember { mutableStateOf(0) }

    val coroutineScope = rememberCoroutineScope()

    // Resend cooldown countdown timer
    LaunchedEffect(cooldownSeconds) {
        if (cooldownSeconds > 0) {
            delay(1000L)
            cooldownSeconds -= 1
        }
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(24.dp)
            .verticalScroll(rememberScrollState()),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center
    ) {
        Spacer(modifier = Modifier.height(16.dp))
        SunLogo(Modifier.size(width = 104.dp, height = 72.dp))
        Spacer(Modifier.height(8.dp))
        Text(
            text = "Puja Vaahini",
            style = MaterialTheme.typography.headlineLarge,
            fontWeight = FontWeight.Bold,
            color = MaterialTheme.colorScheme.primary
        )
        Text(
            text = tr("Poojas and priests", "పూజలు మరియు పురోహితులు", "पूजा और पुरोहित"),
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )

        Spacer(modifier = Modifier.height(24.dp))

        Card(
            modifier = Modifier.fillMaxWidth(),
            shape = RoundedCornerShape(16.dp),
            colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface)
        ) {
            Column(modifier = Modifier.padding(20.dp)) {
                Text(
                    text = strings.auth.loginTitle,
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.titleMedium
                )
                Text(
                    text = tr("Sign in with your phone number and password", "ఫోన్ నంబరు, పాస్‌వర్డ్‌తో సైన్ ఇన్ చేయండి", "फ़ोन नंबर और पासवर्ड से साइन इन करें"),
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )

                Spacer(modifier = Modifier.height(14.dp))

                OutlinedTextField(
                    value = phone,
                    onValueChange = { phone = it; errorMessage = null },
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Phone),
                    label = { Text(strings.auth.enterPhone) },
                    placeholder = { Text(strings.auth.phonePlaceholder) },
                    modifier = Modifier.fillMaxWidth(),
                    singleLine = true
                )

                Spacer(modifier = Modifier.height(10.dp))

                OutlinedTextField(
                    value = password,
                    onValueChange = { password = it; errorMessage = null },
                    keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password),
                    label = { Text(tr("Password", "పాస్‌వర్డ్", "पासवर्ड")) },
                    visualTransformation = if (passwordVisible) VisualTransformation.None else PasswordVisualTransformation(),
                    trailingIcon = {
                        TextButton(onClick = { passwordVisible = !passwordVisible }) {
                            Text(if (passwordVisible) tr("Hide", "దాచు", "छिपाएँ") else tr("Show", "చూపు", "दिखाएँ"), style = MaterialTheme.typography.labelSmall)
                        }
                    },
                    modifier = Modifier.fillMaxWidth(),
                    singleLine = true
                )

                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.End
                ) {
                    TextButton(
                        onClick = {
                            showForgotPasswordModal = true
                        }
                    ) {
                        Text(
                            strings.auth.forgotPasswordButton,
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.primary
                        )
                    }
                }

                errorMessage?.let {
                    Spacer(modifier = Modifier.height(6.dp))
                    Text(it, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall, modifier = Modifier.semantics { liveRegion = LiveRegionMode.Polite })
                }

                successMessage?.let {
                    Spacer(modifier = Modifier.height(6.dp))
                    Text(it, color = Color(0xFF15803D), style = MaterialTheme.typography.bodySmall, fontWeight = FontWeight.Bold)
                }

                Spacer(modifier = Modifier.height(14.dp))

                Button(
                    onClick = {
                        if (phone.isBlank()) {
                            errorMessage = strings.auth.invalidPhone
                            return@Button
                        }
                        if (password.isBlank()) {
                            errorMessage = tr("Enter your password.", "మీ పాస్‌వర్డ్ నమోదు చేయండి.", "अपना पासवर्ड दर्ज करें।")
                            return@Button
                        }
                        isLoading = true
                        errorMessage = null
                        coroutineScope.launch {
                            val res = PurohitApiClient.login(phone = phone.trim(), password = password.trim())
                            isLoading = false
                            res.onSuccess {
                                onLoginSuccess()
                            }.onFailure { err ->
                                val raw = (err.message ?: "").lowercase()
                                errorMessage = when {
                                    raw.contains("invalid") || raw.contains("unauthorized") || raw.contains("credentials") || raw.contains("password") || raw.contains("401") -> {
                                        tr(
                                            "Invalid phone number or password. Please try again.",
                                            "తప్పు ఫోన్ నంబరు లేదా పాస్‌వర్డ్. దయచేసి మళ్లీ ప్రయత్నించండి.",
                                            "अमान्य फ़ोन नंबर या पासवर्ड। कृपया पुनः प्रयास करें।"
                                        )
                                    }
                                    raw.contains("timeout") || raw.contains("timed out") || raw.contains("connectexception") || raw.contains("sockettimeoutexception") || raw.contains("failed to connect") -> {
                                        tr(
                                            "Connection timed out. The server may be waking up, please wait a moment and try again.",
                                            "సర్వర్ కనెక్షన్ సమయం ముగిసింది. దయచేసి కాసేపు ఆగి మళ్లీ ప్రయత్నించండి.",
                                            "सर्वर कनेक्शन का समय समाप्त हो गया। कृपया थोड़ी देर बाद फिर प्रयास करें।"
                                        )
                                    }
                                    raw.contains("too_many_requests") || raw.contains("lockout") || raw.contains("throttled") || raw.contains("429") -> {
                                        tr(
                                            "Too many failed attempts. Please try again after 15 minutes.",
                                            "చాలా విఫల ప్రయత్నాలు జరిగాయి. దయచేసి 15 నిమిషాల తర్వాత ప్రయత్నించండి.",
                                            "कई बार गलत प्रयास किए गए। कृपया 15 मिनट बाद पुनः प्रयास करें।"
                                        )
                                    }
                                    else -> {
                                        tr(
                                            "Could not sign in. Check your details and connection, then try again.",
                                            "సైన్ ఇన్ కాలేదు. మీ వివరాలు మరియు కనెక్షన్‌ను తనిఖీ చేసి మళ్లీ ప్రయత్నించండి.",
                                            "साइन इन नहीं हो सका। अपनी जानकारी और कनेक्शन जाँचकर फिर से प्रयास करें।"
                                        )
                                    }
                                }
                            }
                        }
                    },
                    modifier = Modifier
                        .fillMaxWidth()
                        .heightIn(min = 50.dp),
                    enabled = !isLoading,
                    colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.primary)
                ) {
                    if (isLoading) {
                        CircularProgressIndicator(color = MaterialTheme.colorScheme.onPrimary, modifier = Modifier.size(24.dp))
                    } else {
                        Text(tr("Log in", "లాగిన్", "लॉग इन"))
                    }
                }

                Spacer(modifier = Modifier.height(8.dp))

                TextButton(
                    onClick = onNavigateToRegister,
                    modifier = Modifier.fillMaxWidth()
                ) {
                    Text(strings.auth.newToPujaVaahini)
                }

                Spacer(modifier = Modifier.height(4.dp))


            }
        }

        Spacer(modifier = Modifier.height(24.dp))
    }

    // Email Password Recovery Dialog
    if (showForgotPasswordModal) {
        AlertDialog(
            onDismissRequest = {
                if (!isRequestingLink) {
                    showForgotPasswordModal = false
                }
            },
            title = {
                Text(
                    text = strings.auth.resetPasswordTitle,
                    fontWeight = FontWeight.Bold
                )
            },
            text = {
                Column {
                    if (!emailSentSuccess) {
                        Text(
                            text = strings.auth.forgotPasswordEmailPrompt,
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                        Spacer(modifier = Modifier.height(14.dp))
                        OutlinedTextField(
                            value = recoveryEmail,
                            onValueChange = {
                                recoveryEmail = it
                                recoveryError = null
                            },
                            label = { Text(strings.auth.emailLabel) },
                            placeholder = { Text(strings.auth.emailPlaceholder) },
                            modifier = Modifier.fillMaxWidth(),
                            singleLine = true,
                            enabled = !isRequestingLink
                        )
                    } else {
                        Surface(
                            shape = RoundedCornerShape(8.dp),
                            color = Color(0xFFF0FDF4),
                            modifier = Modifier.fillMaxWidth()
                        ) {
                            Column(modifier = Modifier.padding(12.dp)) {
                                Text(
                                    text = "✉️ ${strings.auth.resetLinkSentSuccess}",
                                    style = MaterialTheme.typography.bodyMedium,
                                    color = Color(0xFF15803D),
                                    fontWeight = FontWeight.Medium
                                )
                                Spacer(modifier = Modifier.height(6.dp))
                                Text(
                                    text = "Destination: $recoveryEmail\nThe reset link will expire in 15 minutes.",
                                    style = MaterialTheme.typography.bodySmall,
                                    color = Color(0xFF166534)
                                )
                            }
                        }

                        Spacer(modifier = Modifier.height(12.dp))

                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.End,
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            TextButton(
                                onClick = {
                                    if (cooldownSeconds <= 0 && !isRequestingLink) {
                                        isRequestingLink = true
                                        recoveryError = null
                                        coroutineScope.launch {
                                            val res = PurohitApiClient.requestPasswordReset(email = recoveryEmail.trim(), role = "devotee")
                                            isRequestingLink = false
                                            res.onSuccess { startRes ->
                                                cooldownSeconds = if (startRes.cooldown_seconds > 0) startRes.cooldown_seconds else 60
                                            }.onFailure { err ->
                                                val msg = parseErrorMessage(err.message ?: "", "Failed to resend reset link")
                                                recoveryError = if (msg.contains("email_service_unavailable") || msg.contains("unavailable") || msg.contains("503")) {
                                                    strings.auth.emailUnavailableNotice
                                                } else {
                                                    strings.auth.emailDeliveryFailed
                                                }
                                            }
                                        }
                                    }
                                },
                                enabled = cooldownSeconds <= 0 && !isRequestingLink
                            ) {
                                Text(
                                    if (cooldownSeconds > 0) {
                                        strings.auth.resendCooldown.replace("%d", cooldownSeconds.toString())
                                    } else {
                                        strings.auth.resendResetLink
                                    },
                                    style = MaterialTheme.typography.bodySmall
                                )
                            }
                        }
                    }

                    recoveryError?.let {
                        Spacer(modifier = Modifier.height(10.dp))
                        Text(it, color = Color.Red, style = MaterialTheme.typography.bodySmall)
                    }
                }
            },
            confirmButton = {
                if (!emailSentSuccess) {
                    Button(
                        onClick = {
                            val trimmedEmail = recoveryEmail.trim()
                            if (trimmedEmail.isBlank() || !trimmedEmail.contains("@") || !trimmedEmail.contains(".")) {
                                recoveryError = strings.auth.invalidEmail
                                return@Button
                            }
                            isRequestingLink = true
                            recoveryError = null
                            coroutineScope.launch {
                                val res = PurohitApiClient.requestPasswordReset(email = trimmedEmail, role = "devotee")
                                isRequestingLink = false
                                res.onSuccess { startRes ->
                                    emailSentSuccess = true
                                    cooldownSeconds = if (startRes.cooldown_seconds > 0) startRes.cooldown_seconds else 60
                                }.onFailure { err ->
                                    val msg = parseErrorMessage(err.message ?: "", "Failed to send reset link")
                                    recoveryError = if (msg.contains("email_service_unavailable") || msg.contains("unavailable") || msg.contains("503")) {
                                        strings.auth.emailUnavailableNotice
                                    } else {
                                        strings.auth.emailDeliveryFailed
                                    }
                                }
                            }
                        },
                        enabled = !isRequestingLink
                    ) {
                        if (isRequestingLink) {
                            CircularProgressIndicator(color = Color.White, modifier = Modifier.size(16.dp), strokeWidth = 2.dp)
                            Spacer(modifier = Modifier.width(8.dp))
                        }
                        Text(strings.auth.sendResetLink)
                    }
                } else {
                    Button(
                        onClick = { showForgotPasswordModal = false }
                    ) {
                        Text(strings.auth.backToLogin)
                    }
                }
            },
            dismissButton = {
                if (!emailSentSuccess) {
                    OutlinedButton(
                        onClick = { showForgotPasswordModal = false },
                        enabled = !isRequestingLink
                    ) {
                        Text(strings.common.cancelAction)
                    }
                }
            }
        )
    }
}
