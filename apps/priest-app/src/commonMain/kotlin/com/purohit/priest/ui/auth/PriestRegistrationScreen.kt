package com.purohit.priest.ui.auth

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.selection.toggleable
import androidx.compose.ui.semantics.Role
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.unit.dp
import com.purohit.priest.SunLogo
import com.purohit.priest.LocalPriestLanguage
import com.purohit.contracts.i18n.AppLanguage
import com.purohit.contracts.i18n.LocalizationManager
import com.purohit.contracts.i18n.StringContract
import com.purohit.priest.network.PurohitApiClient
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
fun PriestRegistrationScreen(
    onRegistrationSuccess: (priestName: String, sampradaya: String, priestId: String) -> Unit,
    strings: StringContract = LocalizationManager.getStrings(AppLanguage.ENGLISH)
) {
    val language = LocalPriestLanguage.current
    fun tr(en: String, te: String, hi: String) = when (language) { AppLanguage.TELUGU -> te; AppLanguage.HINDI -> hi; else -> en }
    var isLoginMode by remember { mutableStateOf(true) }
    var phone by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var passwordVisible by remember { mutableStateOf(false) }

    // Registration fields
    var fullName by remember { mutableStateOf("") }
    var sampradaya by remember { mutableStateOf("") }
    var vedaShakha by remember { mutableStateOf("") }
    var experienceYears by remember { mutableStateOf("") }
    var city by remember { mutableStateOf("") }
    var consentAgreed by remember { mutableStateOf(false) }

    var errorMessage by remember { mutableStateOf<String?>(null) }
    var successMessage by remember { mutableStateOf<String?>(null) }
    var isSubmitting by remember { mutableStateOf(false) }

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
            .padding(20.dp)
            .verticalScroll(rememberScrollState()),
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        Spacer(modifier = Modifier.height(16.dp))
        SunLogo(Modifier.size(width = 104.dp, height = 72.dp))
        Spacer(Modifier.height(8.dp))
        Text(
            text = "అయ్యగారు (Purohit Portal)",
            style = MaterialTheme.typography.headlineMedium,
            fontWeight = FontWeight.Bold,
            color = MaterialTheme.colorScheme.primary
        )
        Text(
            text = tr("Manage bookings and availability", "బుకింగ్‌లు మరియు అందుబాటును నిర్వహించండి", "बुकिंग और उपलब्धता प्रबंधित करें"),
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )

        Spacer(modifier = Modifier.height(20.dp))

        // Tab Selector: Login vs Register
        Surface(
            shape = RoundedCornerShape(12.dp),
            color = MaterialTheme.colorScheme.surfaceVariant,
            modifier = Modifier.fillMaxWidth()
        ) {
            Row(modifier = Modifier.padding(4.dp)) {
                Surface(
                    shape = RoundedCornerShape(8.dp),
                    color = if (isLoginMode) MaterialTheme.colorScheme.primary else Color.Transparent,
                    modifier = Modifier
                        .weight(1f)
                        .clickable {
                            isLoginMode = true
                            errorMessage = null
                        }
                ) {
                    Text(
                        text = tr("Log in", "లాగిన్", "लॉग इन"),
                        color = if (isLoginMode) Color.White else MaterialTheme.colorScheme.onSurfaceVariant,
                        fontWeight = FontWeight.Bold,
                        style = MaterialTheme.typography.bodyMedium,
                        modifier = Modifier.padding(vertical = 10.dp),
                        textAlign = androidx.compose.ui.text.style.TextAlign.Center
                    )
                }

                Surface(
                    shape = RoundedCornerShape(8.dp),
                    color = if (!isLoginMode) MaterialTheme.colorScheme.primary else Color.Transparent,
                    modifier = Modifier
                        .weight(1f)
                        .clickable {
                            isLoginMode = false
                            errorMessage = null
                        }
                ) {
                    Text(
                        text = tr("Register", "నమోదు", "पंजीकरण"),
                        color = if (!isLoginMode) Color.White else MaterialTheme.colorScheme.onSurfaceVariant,
                        fontWeight = FontWeight.Bold,
                        style = MaterialTheme.typography.bodyMedium,
                        modifier = Modifier.padding(vertical = 10.dp),
                        textAlign = androidx.compose.ui.text.style.TextAlign.Center
                    )
                }
            }
        }

        Spacer(modifier = Modifier.height(16.dp))

        if (isLoginMode) {
            // LOGIN MODE
            Card(
                modifier = Modifier.fillMaxWidth(),
                shape = RoundedCornerShape(14.dp),
                colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surface)
            ) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Text(
                        text = tr("Priest sign in", "పురోహితుల లాగిన్", "पुरोहित लॉग इन"),
                        fontWeight = FontWeight.Bold,
                        style = MaterialTheme.typography.titleMedium
                    )
                    Text(
                        text = tr("Sign in to see requests and your schedule", "అభ్యర్థనలు, షెడ్యూల్ చూడడానికి లాగిన్ చేయండి", "अनुरोध और समय-सारणी देखने के लिए लॉग इन करें"),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )

                    Spacer(modifier = Modifier.height(14.dp))

                    OutlinedTextField(
                        value = phone,
                        onValueChange = { phone = it },
                        label = { Text(strings.auth.enterPhone) },
                        placeholder = { Text(strings.auth.phonePlaceholder) },
                        modifier = Modifier.fillMaxWidth(),
                        singleLine = true
                    )

                    Spacer(modifier = Modifier.height(10.dp))

                    OutlinedTextField(
                        value = password,
                        onValueChange = { password = it },
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
                            Text(strings.auth.forgotPasswordButton, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.primary)
                        }
                    }

                    errorMessage?.let {
                        Spacer(modifier = Modifier.height(8.dp))
                        Text(it, color = Color.Red, style = MaterialTheme.typography.bodySmall)
                    }

                    successMessage?.let {
                        Spacer(modifier = Modifier.height(8.dp))
                        Text(it, color = Color(0xFF15803D), style = MaterialTheme.typography.bodySmall, fontWeight = FontWeight.Bold)
                    }

                    Spacer(modifier = Modifier.height(14.dp))

                    Button(
                        onClick = {
                            if (phone.isBlank()) {
                                errorMessage = strings.auth.invalidPhone
                                return@Button
                            }
                            isSubmitting = true
                            errorMessage = null
                            coroutineScope.launch {
                                val res = PurohitApiClient.loginPriest(phone = phone.trim(), password = password.trim())
                                isSubmitting = false
                                res.onSuccess { auth ->
                                    val priest = auth.user
                                    if (priest == null || priest.id.isBlank()) {
                                        errorMessage = tr("Could not load your profile. Please try again.", "మీ వివరాలు పొందలేకపోయాము. మళ్లీ ప్రయత్నించండి.", "प्रोफ़ाइल नहीं मिली। फिर प्रयास करें।")
                                    } else {
                                        onRegistrationSuccess(priest.full_name, priest.sampradaya, priest.id)
                                    }
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
                            .height(50.dp),
                        enabled = !isSubmitting,
                        colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.primary)
                    ) {
                        Text(if (isSubmitting) strings.common.loadingState else tr("Log in", "లాగిన్", "लॉग इन"))
                    }

                    Spacer(modifier = Modifier.height(10.dp))


                }
            }
        } else {
            // REGISTRATION MODE
            OutlinedTextField(
                value = fullName,
                onValueChange = { fullName = it },
                label = { Text(strings.auth.fullName) },
                placeholder = { Text(strings.auth.fullName) },
                modifier = Modifier.fillMaxWidth()
            )

            Spacer(modifier = Modifier.height(12.dp))

            OutlinedTextField(
                value = phone,
                onValueChange = { phone = it },
                label = { Text(strings.auth.enterPhone) },
                placeholder = { Text(strings.auth.phonePlaceholder) },
                modifier = Modifier.fillMaxWidth()
            )

            Spacer(modifier = Modifier.height(12.dp))

            OutlinedTextField(
                value = password,
                onValueChange = { password = it },
                label = { Text(tr("Password (at least 8 characters) *", "పాస్‌వర్డ్ (కనీసం 8 అక్షరాలు) *", "पासवर्ड (कम से कम 8 अक्षर) *")) },
                placeholder = { Text(tr("Enter a password", "పాస్‌వర్డ్ నమోదు చేయండి", "पासवर्ड दर्ज करें")) },
                visualTransformation = androidx.compose.ui.text.input.PasswordVisualTransformation(),
                modifier = Modifier.fillMaxWidth()
            )

            Spacer(modifier = Modifier.height(12.dp))

            Column(
                modifier = Modifier.fillMaxWidth(),
                verticalArrangement = Arrangement.spacedBy(12.dp)
            ) {
                OutlinedTextField(
                    value = sampradaya,
                    onValueChange = { sampradaya = it },
                    label = { Text(strings.auth.sampradaya) },
                    modifier = Modifier.fillMaxWidth()
                )
                OutlinedTextField(
                    value = vedaShakha,
                    onValueChange = { vedaShakha = it },
                    label = { Text(strings.auth.vedaShakha) },
                    modifier = Modifier.fillMaxWidth()
                )
            }

            Spacer(modifier = Modifier.height(12.dp))

            Column(
                modifier = Modifier.fillMaxWidth(),
                verticalArrangement = Arrangement.spacedBy(12.dp)
            ) {
                OutlinedTextField(
                    value = experienceYears,
                    onValueChange = { experienceYears = it },
                    label = { Text(strings.auth.experienceYears) },
                    modifier = Modifier.fillMaxWidth()
                )
                OutlinedTextField(
                    value = city,
                    onValueChange = { city = it },
                    label = { Text(strings.auth.city) },
                    modifier = Modifier.fillMaxWidth()
                )
            }

            Spacer(modifier = Modifier.height(16.dp))

            // Ayagaryllu On-Demand Code of Conduct
            Card(
                modifier = Modifier.fillMaxWidth(),
                colors = CardDefaults.cardColors(
                    containerColor = MaterialTheme.colorScheme.surfaceVariant
                )
            ) {
                Column(modifier = Modifier.padding(14.dp)) {
                    Text(
                        text = tr("Code of conduct", "ప్రవర్తనా నియమావళి", "आचार संहिता"),
                        fontWeight = FontWeight.Bold,
                        style = MaterialTheme.typography.titleSmall
                    )
                    Spacer(modifier = Modifier.height(6.dp))
                    Text(
                        text = tr(
                        "1. I will perform rituals according to traditional Shastras.\n2. I agree to arrive promptly within the estimated time for accepted requests.\n3. Discuss priest and materials charges with the devotee; no online payment is collected during this test.\n4. I will maintain sanctity, traditional attire and sacred conduct.",
                        "1. సంప్రదాయ శాస్త్రాల ప్రకారం పూజలు నిర్వహిస్తాను.\n2. అంగీకరించిన అభ్యర్థనలకు అంచనా సమయంలో చేరుకుంటాను.\n3. పురోహితులు, సామగ్రి ఛార్జీలను భక్తునితో చర్చించాలి; ఈ పరీక్షలో ఆన్‌లైన్ చెల్లింపు వసూలు చేయరు.\n4. పవిత్రత, సంప్రదాయ దుస్తులు, సత్ప్రవర్తన పాటిస్తాను.",
                        "1. मैं पारंपरिक शास्त्रों के अनुसार पूजा करूँगा/करूँगी।\n2. स्वीकृत अनुरोधों के लिए अनुमानित समय पर पहुँचूँगा/पहुँचूँगी।\n3. पुरोहित और सामग्री के शुल्क भक्त से तय करें; इस परीक्षण में ऑनलाइन भुगतान नहीं लिया जाता।\n4. मैं पवित्रता, पारंपरिक वेश और सदाचार बनाए रखूँगा/रखूँगी।"),
                        style = MaterialTheme.typography.bodySmall
                    )
                    Spacer(modifier = Modifier.height(10.dp))
                    Row(
                        modifier = Modifier.fillMaxWidth().heightIn(min = 48.dp).toggleable(value = consentAgreed, role = Role.Checkbox, onValueChange = { consentAgreed = it }),
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Checkbox(
                            checked = consentAgreed,
                            onCheckedChange = null
                        )
                        Text(
                            text = tr("I accept the code of conduct *", "నేను ప్రవర్తనా నియమావళిని అంగీకరిస్తున్నాను *", "मैं आचार संहिता स्वीकार करता/करती हूँ *"),
                            fontWeight = FontWeight.SemiBold,
                            style = MaterialTheme.typography.bodyMedium,
                            color = if (consentAgreed) Color(0xFF15803D) else MaterialTheme.colorScheme.onSurface
                        )
                    }
                }
            }

            errorMessage?.let {
                Spacer(modifier = Modifier.height(8.dp))
                Text(it, color = Color.Red, style = MaterialTheme.typography.bodySmall)
            }

            Spacer(modifier = Modifier.height(20.dp))

            Button(
                onClick = {
                    if (fullName.isBlank() || phone.isBlank() || password.isBlank()) {
                        errorMessage = "Please enter priest full name, phone number, and password."
                        return@Button
                    }
                    if (password.trim().length < 8) {
                        errorMessage = "Password must be at least 8 characters long."
                        return@Button
                    }
                    if (experienceYears.toIntOrNull()?.let { it >= 0 } != true) {
                        errorMessage = tr("Enter valid years of experience.", "చెల్లుబాటు అయ్యే అనుభవ సంవత్సరాలు నమోదు చేయండి.", "अनुभव के सही वर्ष दर्ज करें।")
                        return@Button
                    }
                    if (!consentAgreed) {
                        errorMessage = "You must agree to the Puja Vaahini Honor Code to activate on-demand duty."
                        return@Button
                    }
                    errorMessage = null
                    coroutineScope.launch {
                        isSubmitting = true
                        val exp = experienceYears.toInt()
                        val res = PurohitApiClient.registerPriest(
                            fullName = fullName.trim(),
                            phone = phone.trim(),
                            password = password.trim(),
                            sampradaya = sampradaya.trim(),
                            vedaShakha = vedaShakha.trim(),
                            experienceYears = exp,
                            city = city.trim(),
                            consentAgreed = consentAgreed
                        )
                        isSubmitting = false
                        res.onSuccess {
                            isLoginMode = true
                            successMessage = tr("Registration submitted. Please log in.", "నమోదు పూర్తయింది. దయచేసి లాగిన్ చేయండి.", "पंजीकरण पूरा हुआ। कृपया लॉग इन करें।")
                        }.onFailure { err ->
                            val raw = (err.message ?: "").lowercase()
                            errorMessage = when {
                                raw.contains("already exists") || raw.contains("already registered") || raw.contains("account_already_exists") || raw.contains("409") -> {
                                    tr(
                                        "A profile with this phone number already exists. Please log in.",
                                        "ఈ ఫోన్ నంబరుతో ఇప్పటికే పురోహిత ప్రొఫైల్ ఉంది. దయచేసి లాగిన్ అవ్వండి.",
                                        "इस फ़ोन नंबर से पहले से पुरोहित प्रोफ़ाइल मौजूद है। कृपया लॉग इन करें।"
                                    )
                                }
                                raw.contains("timeout") || raw.contains("timed out") || raw.contains("connectexception") || raw.contains("sockettimeoutexception") || raw.contains("failed to connect") -> {
                                    tr(
                                        "Connection timed out. The server may be waking up, please wait a moment and try again.",
                                        "సర్వర్ కనెక్షన్ సమయం ముగిసింది. దయచేసి కాసేపు ఆగి మళ్లీ ప్రయత్నించండి.",
                                        "सर्वर कनेक्शन का समय समाप्त हो गया। कृपया थोड़ी देर बाद फिर प्रयास करें।"
                                    )
                                }
                                raw.contains("password") && (raw.contains("8") || raw.contains("length")) -> {
                                    tr(
                                        "Password must have at least 8 characters.",
                                        "పాస్‌వర్డ్‌లో కనీసం 8 అక్షరాలు ఉండాలి.",
                                        "पासवर्ड में कम से कम 8 अक्षर होने चाहिए।"
                                    )
                                }
                                raw.contains("consent") -> {
                                    strings.auth.consentRequiredError
                                }
                                else -> {
                                    tr(
                                        "Could not complete registration. Check your details and connection, then try again.",
                                        "నమోదు పూర్తి కాలేదు. మీ వివరాలు మరియు కనెక్షన్‌ను తనిఖీ చేసి మళ్లీ ప్రయత్నించండి.",
                                        "पंजीकरण पूरा नहीं हो सका। अपना विवरण और कनेक्शन जाँचकर फिर प्रयास करें।"
                                    )
                                }
                            }
                        }
                    }
                },
                modifier = Modifier
                    .fillMaxWidth()
                    .height(50.dp),
                enabled = !isSubmitting,
                colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.primary)
            ) {
                Text(if (isSubmitting) strings.common.loadingState else strings.auth.registerButton)
            }
        }

        Spacer(modifier = Modifier.height(16.dp))

        Spacer(modifier = Modifier.height(20.dp))
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
                                            val res = PurohitApiClient.requestPasswordReset(email = recoveryEmail.trim(), role = "priest")
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
                                val res = PurohitApiClient.requestPasswordReset(email = trimmedEmail, role = "priest")
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
