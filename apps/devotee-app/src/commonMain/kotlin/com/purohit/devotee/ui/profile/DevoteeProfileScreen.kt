package com.purohit.devotee.ui.profile

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.purohit.devotee.network.PurohitApiClient
import com.purohit.devotee.LocalAppStrings
import kotlinx.coroutines.launch

@Composable
fun DevoteeProfileScreen(
    onLogout: () -> Unit
) {
    val strings = LocalAppStrings.current
    val language = com.purohit.devotee.LocalAppLanguage.current
    fun tr(en: String, te: String, hi: String) = when (language) { com.purohit.contracts.i18n.AppLanguage.TELUGU -> te; com.purohit.contracts.i18n.AppLanguage.HINDI -> hi; else -> en }
    var profile by remember { mutableStateOf(PurohitApiClient.currentProfile) }
    var isLoading by remember { mutableStateOf(false) }
    var showEmailDialog by remember { mutableStateOf(false) }
    var emailInput by remember { mutableStateOf("") }
    var emailLoading by remember { mutableStateOf(false) }
    var emailMessage by remember { mutableStateOf<String?>(null) }
    var emailError by remember { mutableStateOf<String?>(null) }
    val coroutineScope = rememberCoroutineScope()

    fun reloadProfile() {
        if (PurohitApiClient.authToken.isNotBlank()) {
            coroutineScope.launch {
                isLoading = true
                val res = PurohitApiClient.fetchProfile()
                isLoading = false
                res.onSuccess {
                    profile = it
                }
            }
        }
    }

    LaunchedEffect(Unit) {
        reloadProfile()
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(16.dp)
            .verticalScroll(rememberScrollState()),
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        Spacer(modifier = Modifier.height(8.dp))

        // Avatar
        Surface(
            shape = CircleShape,
            color = MaterialTheme.colorScheme.primaryContainer,
            modifier = Modifier.size(80.dp)
        ) {
            Box(contentAlignment = Alignment.Center) {
                Text(
                    text = "🕉️",
                    style = MaterialTheme.typography.headlineMedium
                )
            }
        }

        Spacer(modifier = Modifier.height(12.dp))

        Text(
            text = profile?.full_name?.ifBlank { null } ?: tr("Devotee Profile", "భక్తుని ప్రొఫైల్", "भक्त प्रोफ़ाइल"),
            style = MaterialTheme.typography.headlineSmall,
            fontWeight = FontWeight.Bold
        )

        Text(
            text = profile?.phone?.ifBlank { null } ?: tr("Phone not set", "ఫోన్ లేదు", "फ़ोन नहीं जोड़ा"),
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )

        Spacer(modifier = Modifier.height(20.dp))

        // Vedic Astrological Profile Card
        Card(
            modifier = Modifier.fillMaxWidth(),
            shape = RoundedCornerShape(16.dp),
            colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant)
        ) {
            Column(modifier = Modifier.padding(16.dp)) {
                Text(
                    text = "🪔 " + tr("Sankalpam details", "సంకల్ప వివరాలు", "संकल्प विवरण"),
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.titleMedium,
                    color = MaterialTheme.colorScheme.primary
                )
                Spacer(modifier = Modifier.height(10.dp))
                Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                    Text(strings.auth.gotram, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    Text(profile?.gotram?.ifBlank { null } ?: tr("Not specified", "పేర్కొనలేదు", "नहीं बताया"), fontWeight = FontWeight.Bold)
                }
                Divider(modifier = Modifier.padding(vertical = 8.dp))
                Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                    Text(strings.auth.nakshatram, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    Text(profile?.nakshatram?.ifBlank { null } ?: tr("Not specified", "పేర్కొనలేదు", "नहीं बताया"), fontWeight = FontWeight.Bold)
                }
                Divider(modifier = Modifier.padding(vertical = 8.dp))
                Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                    Text(strings.auth.rashi, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    Text(profile?.rashi?.ifBlank { null } ?: tr("Not specified", "పేర్కొనలేదు", "नहीं बताया"), fontWeight = FontWeight.Bold)
                }
                Divider(modifier = Modifier.padding(vertical = 8.dp))
                Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                    Text(strings.auth.city, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    Text(profile?.city?.ifBlank { null } ?: tr("Not specified", "పేర్కొనలేదు", "नहीं बताया"), fontWeight = FontWeight.Bold)
                }
            }
        }

        Spacer(modifier = Modifier.height(16.dp))

        // Email Password Recovery Card
        Card(
            modifier = Modifier.fillMaxWidth(),
            shape = RoundedCornerShape(16.dp),
            colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface)
        ) {
            Column(modifier = Modifier.padding(16.dp)) {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text(
                        text = "📧 " + tr("Password recovery email", "పాస్‌వర్డ్ రికవరీ ఇమెయిల్", "पासवर्ड रिकवरी ईमेल"),
                        fontWeight = FontWeight.Bold,
                        style = MaterialTheme.typography.titleSmall
                    )
                    IconButton(onClick = { reloadProfile() }, modifier = Modifier.size(28.dp)) {
                        Text("🔄", style = MaterialTheme.typography.bodySmall)
                    }
                }
                Spacer(modifier = Modifier.height(6.dp))
                val currentEmail = profile?.email.orEmpty()
                val isVerified = profile?.email_verified == true

                if (currentEmail.isNotBlank()) {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Text(currentEmail, style = MaterialTheme.typography.bodyMedium, fontWeight = FontWeight.SemiBold)
                        if (isVerified) {
                            Surface(
                                shape = RoundedCornerShape(8.dp),
                                color = Color(0xFFDCFCE7)
                            ) {
                                Text(
                                    text = "🟢 " + tr("Verified", "ధృవీకరించబడింది", "सत्यापित"),
                                    color = Color(0xFF15803D),
                                    style = MaterialTheme.typography.labelSmall,
                                    fontWeight = FontWeight.Bold,
                                    modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp)
                                )
                            }
                        } else {
                            Surface(
                                shape = RoundedCornerShape(8.dp),
                                color = MaterialTheme.colorScheme.surfaceVariant
                            ) {
                                Text(
                                    text = "⚠️ " + tr("Not verified", "ధృవీకరించలేదు", "सत्यापित नहीं"),
                                    color = MaterialTheme.colorScheme.primary,
                                    style = MaterialTheme.typography.labelSmall,
                                    fontWeight = FontWeight.Bold,
                                    modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp)
                                )
                            }
                        }
                    }
                    Spacer(modifier = Modifier.height(6.dp))
                    Text(
                        text = if (isVerified) {
                            tr("You can reset your password using a link sent to this email.", "ఈ ఇమెయిల్‌కు పంపే లింక్‌తో పాస్‌వర్డ్ రీసెట్ చేయవచ్చు.", "इस ईमेल पर भेजे गए लिंक से आप पासवर्ड रीसेट कर सकते हैं।")
                        } else {
                            tr("Open the link we emailed you to verify this address. Password reset works only after that.", "ఈ చిరునామాను ధృవీకరించడానికి మేము పంపిన లింక్ తెరవండి. ఆ తర్వాతే పాస్‌వర్డ్ రీసెట్ పనిచేస్తుంది.", "इस पते को सत्यापित करने के लिए भेजा गया लिंक खोलें। उसके बाद ही पासवर्ड रीसेट काम करेगा।")
                        },
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                } else {
                    Text(
                        text = tr("Add an email so you can reset your password if you forget it.", "పాస్‌వర్డ్ మర్చిపోతే రీసెట్ చేయడానికి ఇమెయిల్ జోడించండి.", "पासवर्ड भूलने पर रीसेट करने के लिए ईमेल जोड़ें।"),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                }

                Spacer(modifier = Modifier.height(12.dp))
                Button(
                    onClick = {
                        emailInput = currentEmail
                        emailMessage = null
                        emailError = null
                        showEmailDialog = true
                    },
                    modifier = Modifier.fillMaxWidth(),
                    shape = RoundedCornerShape(10.dp)
                ) {
                    Text(if (currentEmail.isNotBlank()) tr("Change email", "ఇమెయిల్ మార్చండి", "ईमेल बदलें") else tr("Add email", "ఇమెయిల్ జోడించండి", "ईमेल जोड़ें"))
                }
            }
        }

        if (showEmailDialog) {
            val emailValid = Regex("^[^@\\s]+@[^@\\s]+\\.[^@\\s]+$").matches(emailInput)
            AlertDialog(
                onDismissRequest = { if (!emailLoading) showEmailDialog = false },
                title = { Text(tr("Recovery email", "రికవరీ ఇమెయిల్", "रिकवरी ईमेल")) },
                text = {
                    Column {
                        Text(
                            tr("Enter your email. We'll send a link to verify it. Your current email stays active until the new one is verified.", "మీ ఇమెయిల్ నమోదు చేయండి. ధృవీకరణ లింక్ పంపుతాము. కొత్తది ధృవీకరించే వరకు ప్రస్తుత ఇమెయిల్ కొనసాగుతుంది.", "अपना ईमेल दर्ज करें। हम सत्यापन लिंक भेजेंगे। नया सत्यापित होने तक मौजूदा ईमेल चालू रहेगा।"),
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                        Spacer(modifier = Modifier.height(12.dp))
                        OutlinedTextField(
                            value = emailInput,
                            onValueChange = { emailInput = it.trim() },
                            label = { Text(tr("Email address", "ఇమెయిల్ చిరునామా", "ईमेल पता")) },
                            keyboardOptions = androidx.compose.foundation.text.KeyboardOptions(keyboardType = androidx.compose.ui.text.input.KeyboardType.Email),
                            isError = emailInput.isNotBlank() && !emailValid,
                            singleLine = true,
                            modifier = Modifier.fillMaxWidth(),
                            enabled = !emailLoading
                        )

                        emailMessage?.let { msg ->
                            Spacer(modifier = Modifier.height(8.dp))
                            Text(msg, color = Color(0xFF15803D), style = MaterialTheme.typography.bodySmall, fontWeight = FontWeight.Bold)
                        }

                        emailError?.let { err ->
                            Spacer(modifier = Modifier.height(8.dp))
                            Text(err, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall)
                        }
                    }
                },
                confirmButton = {
                    Button(
                        onClick = {
                            if (emailValid) {
                                emailLoading = true
                                emailMessage = null
                                emailError = null
                                coroutineScope.launch {
                                    val res = PurohitApiClient.requestEmailVerification(emailInput.trim())
                                    emailLoading = false
                                    res.fold(
                                        onSuccess = {
                                            emailMessage = tr("Verification link sent. Check your inbox and open the link.", "ధృవీకరణ లింక్ పంపాము. మీ ఇన్‌బాక్స్ చూసి లింక్ తెరవండి.", "सत्यापन लिंक भेजा गया। अपना इनबॉक्स देखें और लिंक खोलें।")
                                            reloadProfile()
                                        },
                                        onFailure = {
                                            emailError = tr("Could not send the verification email. Please try again.", "ధృవీకరణ ఇమెయిల్ పంపలేకపోయాము. మళ్లీ ప్రయత్నించండి.", "सत्यापन ईमेल नहीं भेजा जा सका। फिर प्रयास करें।")
                                        }
                                    )
                                }
                            }
                        },
                        enabled = !emailLoading && emailValid
                    ) {
                        if (emailLoading) {
                            CircularProgressIndicator(modifier = Modifier.size(16.dp), strokeWidth = 2.dp, color = Color.White)
                        } else {
                            Text(tr("Send link", "లింక్ పంపండి", "लिंक भेजें"))
                        }
                    }
                },
                dismissButton = {
                    TextButton(
                        onClick = { showEmailDialog = false },
                        enabled = !emailLoading
                    ) {
                        Text(tr("Close", "మూసివేయండి", "बंद करें"))
                    }
                }
            )
        }

        Spacer(modifier = Modifier.height(16.dp))

        // Logout Button
        OutlinedButton(
            onClick = onLogout,
            modifier = Modifier.fillMaxWidth(),
            colors = ButtonDefaults.outlinedButtonColors(contentColor = Color.Red)
        ) {
            Text(strings.common.logOut)
        }
    }
}
