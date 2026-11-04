package com.purohit.devotee.ui.auth

import androidx.compose.foundation.layout.*
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
import kotlinx.coroutines.launch

@Composable
fun DevoteeRegistrationScreen(
    onRegistrationSuccess: (userName: String, phone: String) -> Unit,
    onNavigateToLogin: () -> Unit = {}
) {
    var fullName by remember { mutableStateOf("") }
    var phone by remember { mutableStateOf("") }
    var gotram by remember { mutableStateOf("") }
    var nakshatram by remember { mutableStateOf("") }
    var rashi by remember { mutableStateOf("") }
    var city by remember { mutableStateOf("Hyderabad") }
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
            text = "🕉️ Devotee Registration",
            style = MaterialTheme.typography.headlineMedium,
            fontWeight = FontWeight.Bold,
            color = MaterialTheme.colorScheme.primary
        )
        Text(
            text = "Create your Vedic profile for Sankalpam & Pooja Bookings",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )

        Spacer(modifier = Modifier.height(20.dp))

        OutlinedTextField(
            value = fullName,
            onValueChange = { fullName = it },
            label = { Text("Full Name (e.g. Sushanth)") },
            placeholder = { Text("Enter your full name") },
            modifier = Modifier.fillMaxWidth()
        )

        Spacer(modifier = Modifier.height(12.dp))

        OutlinedTextField(
            value = phone,
            onValueChange = { phone = it },
            label = { Text("Mobile Phone Number") },
            placeholder = { Text("e.g. +91 98765 43210") },
            modifier = Modifier.fillMaxWidth()
        )

        Spacer(modifier = Modifier.height(12.dp))

        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            OutlinedTextField(
                value = gotram,
                onValueChange = { gotram = it },
                label = { Text("Gotram") },
                placeholder = { Text("e.g. Bharadwaja") },
                modifier = Modifier.weight(1f)
            )
            OutlinedTextField(
                value = nakshatram,
                onValueChange = { nakshatram = it },
                label = { Text("Nakshatram") },
                placeholder = { Text("e.g. Rohini") },
                modifier = Modifier.weight(1f)
            )
        }

        Spacer(modifier = Modifier.height(12.dp))

        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            OutlinedTextField(
                value = rashi,
                onValueChange = { rashi = it },
                label = { Text("Rashi") },
                placeholder = { Text("e.g. Vrishabha") },
                modifier = Modifier.weight(1f)
            )
            OutlinedTextField(
                value = city,
                onValueChange = { city = it },
                label = { Text("City / Region") },
                placeholder = { Text("Hyderabad") },
                modifier = Modifier.weight(1f)
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
                    text = "📜 Terms of Service & Vedic Ritual Consent",
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.titleSmall
                )
                Spacer(modifier = Modifier.height(6.dp))
                Text(
                    text = "1. I authorize verified Purohits to perform rituals as per traditional Vedic vidhanam.\n" +
                           "2. Payment is held in secure escrow and released only upon ritual completion with OTP.\n" +
                           "3. Devotee details (Gotram & Nakshatram) will be utilized strictly for Sankalpam chanting.",
                    style = MaterialTheme.typography.bodySmall
                )
                Spacer(modifier = Modifier.height(10.dp))
                Row(
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Checkbox(
                        checked = consentAgreed,
                        onCheckedChange = { consentAgreed = it }
                    )
                    Text(
                        text = "I have read and agree to the Consent & Escrow Terms *",
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

        Spacer(modifier = Modifier.height(24.dp))

        Button(
            onClick = {
                if (fullName.isBlank() || phone.isBlank()) {
                    errorMessage = "Please enter your full name and phone number."
                    return@Button
                }
                if (!consentAgreed) {
                    errorMessage = "You must agree to the Vedic Consent & Escrow terms to proceed."
                    return@Button
                }
                errorMessage = null
                coroutineScope.launch {
                    isSubmitting = true
                    val res = PurohitApiClient.registerDevotee(
                        fullName = fullName.trim(),
                        phone = phone.trim(),
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
                        errorMessage = "Registration error: ${err.message}"
                    }
                }
            },
            modifier = Modifier
                .fillMaxWidth()
                .height(50.dp),
            enabled = !isSubmitting
        ) {
            Text(if (isSubmitting) "Saving to Database..." else "Complete Registration & Continue")
        }

        Spacer(modifier = Modifier.height(12.dp))

        TextButton(onClick = onNavigateToLogin) {
            Text("Already registered? Login with Phone & OTP")
        }

        Spacer(modifier = Modifier.height(16.dp))
    }
}
