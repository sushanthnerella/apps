package com.purohit.devotee.ui.auth

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.selection.toggleable
import androidx.compose.ui.semantics.Role
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.purohit.devotee.network.PurohitApiClient
import com.purohit.devotee.LocalAppLanguage
import com.purohit.devotee.LocalAppStrings
import com.purohit.contracts.i18n.AppLanguage
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.input.PasswordVisualTransformation
import androidx.compose.ui.text.input.VisualTransformation
import androidx.compose.ui.semantics.liveRegion
import androidx.compose.ui.semantics.LiveRegionMode
import androidx.compose.ui.semantics.semantics
import kotlinx.coroutines.launch

@Composable
fun DevoteeRegistrationScreen(
    onRegistrationSuccess: (userName: String, phone: String) -> Unit,
    onNavigateToLogin: () -> Unit = {}
) {
    val language = LocalAppLanguage.current
    val strings = LocalAppStrings.current
    fun tr(en: String, te: String, hi: String) = when (language) { AppLanguage.TELUGU -> te; AppLanguage.HINDI -> hi; else -> en }
    var fullName by remember { mutableStateOf("") }
    var phone by remember { mutableStateOf("") }
    var password by remember { mutableStateOf("") }
    var passwordVisible by remember { mutableStateOf(false) }
    var gotram by remember { mutableStateOf("") }
    var nakshatram by remember { mutableStateOf("") }
    var rashi by remember { mutableStateOf("") }
    var city by remember { mutableStateOf("") }
    var consentAgreed by remember { mutableStateOf(false) }
    var errorMessage by remember { mutableStateOf<String?>(null) }
    var isSubmitting by remember { mutableStateOf(false) }
    val coroutineScope = rememberCoroutineScope()

    Column(
        modifier = Modifier
            .fillMaxSize()
            .padding(20.dp)
            .verticalScroll(rememberScrollState()),
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        Spacer(modifier = Modifier.height(12.dp))
        Text(
            text = strings.auth.devoteeRegistrationTitle,
            style = MaterialTheme.typography.headlineMedium,
            fontWeight = FontWeight.Bold,
            color = MaterialTheme.colorScheme.primary
        )
        Text(
            text = tr("Create your profile for Sankalpam and pooja bookings", "సంకల్పం మరియు పూజ బుకింగ్‌ల కోసం మీ వివరాలను నమోదు చేయండి", "संकल्प और पूजा बुकिंग के लिए अपनी जानकारी दर्ज करें"),
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )

        Spacer(modifier = Modifier.height(20.dp))

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
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Phone),
            label = { Text(strings.auth.enterPhone) },
            placeholder = { Text(strings.auth.phonePlaceholder) },
            modifier = Modifier.fillMaxWidth()
        )

        Spacer(modifier = Modifier.height(12.dp))

        OutlinedTextField(
            value = password,
            onValueChange = { password = it },
            keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Password),
            label = { Text(tr("Password (at least 8 characters) *", "పాస్‌వర్డ్ (కనీసం 8 అక్షరాలు) *", "पासवर्ड (कम से कम 8 अक्षर) *")) },
            placeholder = { Text(tr("Enter a password", "పాస్‌వర్డ్ నమోదు చేయండి", "पासवर्ड दर्ज करें")) },
            visualTransformation = if (passwordVisible) VisualTransformation.None else PasswordVisualTransformation(),
            trailingIcon = {
                TextButton(onClick = { passwordVisible = !passwordVisible }) {
                    Text(if (passwordVisible) tr("Hide", "దాచు", "छिपाएँ") else tr("Show", "చూపు", "दिखाएँ"))
                }
            },
            singleLine = true,
            modifier = Modifier.fillMaxWidth()
        )

        Spacer(modifier = Modifier.height(12.dp))

        Column(
            modifier = Modifier.fillMaxWidth(),
            verticalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            OutlinedTextField(
                value = gotram,
                onValueChange = { gotram = it },
                label = { Text(strings.auth.gotram) },
                placeholder = { Text(strings.auth.gotram) },
                modifier = Modifier.fillMaxWidth()
            )
            OutlinedTextField(
                value = nakshatram,
                onValueChange = { nakshatram = it },
                label = { Text(strings.auth.nakshatram) },
                placeholder = { Text(strings.auth.nakshatram) },
                modifier = Modifier.fillMaxWidth()
            )
        }

        Spacer(modifier = Modifier.height(12.dp))

        Column(
            modifier = Modifier.fillMaxWidth(),
            verticalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            OutlinedTextField(
                value = rashi,
                onValueChange = { rashi = it },
                label = { Text(strings.auth.rashi) },
                placeholder = { Text(strings.auth.rashi) },
                modifier = Modifier.fillMaxWidth()
            )
            OutlinedTextField(
                value = city,
                onValueChange = { city = it },
                label = { Text(strings.auth.city) },
                placeholder = { Text(strings.auth.city) },
                modifier = Modifier.fillMaxWidth()
            )
        }

        Spacer(modifier = Modifier.height(20.dp))

        // Mandatory Consent Card
        Card(
            modifier = Modifier.fillMaxWidth(),
            colors = CardDefaults.cardColors(
                containerColor = MaterialTheme.colorScheme.surfaceVariant
            )
        ) {
            Column(modifier = Modifier.padding(14.dp)) {
                Text(
                    text = tr("Terms and ritual consent", "నిబంధనలు మరియు పూజ సమ్మతి", "नियम और पूजा सहमति"),
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.titleSmall
                )
                Spacer(modifier = Modifier.height(6.dp))
                Text(
                    text = tr(
                        "1. I authorize assigned Purohits to perform rituals according to traditional Vedic practice.\n2. No online payment is collected during this test; confirm priest and materials charges directly.\n3. Gotram and Nakshatram details will be used for Sankalpam chanting.",
                        "1. సంప్రదాయ వేద విధానం ప్రకారం కేటాయించిన పురోహితులు పూజ నిర్వహించడానికి నేను అనుమతిస్తున్నాను.\n2. ఈ పరీక్షలో ఆన్‌లైన్ చెల్లింపు వసూలు చేయరు; పురోహితులు మరియు సామగ్రి ఛార్జీలను నేరుగా నిర్ధారించుకోండి.\n3. గోత్రం మరియు నక్షత్ర వివరాలను సంకల్పం కోసం ఉపయోగిస్తారు.",
                        "1. मैं नियुक्त पुरोहित को पारंपरिक वैदिक विधि से पूजा करने की अनुमति देता/देती हूँ।\n2. इस परीक्षण में ऑनलाइन भुगतान नहीं लिया जाता; पुरोहित और सामग्री के शुल्क सीधे सुनिश्चित करें।\n3. गोत्र और नक्षत्र की जानकारी संकल्प के लिए उपयोग की जाएगी।"),
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
                        text = tr("I have read and agree to the consent terms *", "నేను సమ్మతి నిబంధనలను చదివి అంగీకరిస్తున్నాను *", "मैंने सहमति की शर्तें पढ़ ली हैं और सहमत हूँ *"),
                        fontWeight = FontWeight.SemiBold,
                        style = MaterialTheme.typography.bodyMedium,
                        color = MaterialTheme.colorScheme.onSurface
                    )
                }
            }
        }

        errorMessage?.let {
            Spacer(modifier = Modifier.height(8.dp))
            Text(it, color = MaterialTheme.colorScheme.error, style = MaterialTheme.typography.bodySmall, modifier = Modifier.semantics { liveRegion = LiveRegionMode.Polite })
        }

        Spacer(modifier = Modifier.height(24.dp))

        Button(
            onClick = {
                if (fullName.isBlank() || phone.isBlank() || password.isBlank()) {
                    errorMessage = tr("Enter your name, phone number and password.", "పేరు, ఫోన్ నంబరు మరియు పాస్‌వర్డ్ నమోదు చేయండి.", "अपना नाम, फ़ोन नंबर और पासवर्ड दर्ज करें।")
                    return@Button
                }
                if (password.trim().length < 8) {
                    errorMessage = tr("Password must have at least 8 characters.", "పాస్‌వర్డ్‌లో కనీసం 8 అక్షరాలు ఉండాలి.", "पासवर्ड में कम से कम 8 अक्षर होने चाहिए।")
                    return@Button
                }
                if (!consentAgreed) {
                    errorMessage = strings.auth.consentRequiredError
                    return@Button
                }
                errorMessage = null
                coroutineScope.launch {
                    isSubmitting = true
                    val res = PurohitApiClient.registerDevotee(
                        fullName = fullName.trim(),
                        phone = phone.trim(),
                        password = password.trim(),
                        gotram = gotram.trim(),
                        nakshatram = nakshatram.trim(),
                        rashi = rashi.trim(),
                        city = city.trim(),
                        consentAgreed = consentAgreed
                    )
                    isSubmitting = false
                    res.onSuccess {
                        onRegistrationSuccess(fullName, phone)
                    }.onFailure { err ->
                        val raw = (err.message ?: "").lowercase()
                        errorMessage = when {
                            raw.contains("already exists") || raw.contains("already registered") || raw.contains("account_already_exists") || raw.contains("409") -> {
                                tr(
                                    "An account with this phone number already exists. Please log in.",
                                    "ఈ ఫోన్ నంబరుతో ఇప్పటికే ఖాతా ఉంది. దయచేసి లాగిన్ అవ్వండి.",
                                    "इस फ़ोन नंबर से पहले से खाता मौजूद है। कृपया लॉग इन करें।"
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
                                    "Could not complete registration. Check your connection and details, then try again.",
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
                .heightIn(min = 50.dp),
            enabled = !isSubmitting
        ) {
            Text(if (isSubmitting) strings.common.loadingState else strings.auth.registerButton)
        }

        Spacer(modifier = Modifier.height(12.dp))

        TextButton(onClick = onNavigateToLogin) {
            Text(strings.auth.alreadyHaveAccount)
        }

        Spacer(modifier = Modifier.height(16.dp))
    }
}
