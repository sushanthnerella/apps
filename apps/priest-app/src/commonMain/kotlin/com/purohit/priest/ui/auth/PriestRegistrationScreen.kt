package com.purohit.priest.ui.auth

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
import com.purohit.priest.network.PurohitApiClient
import kotlinx.coroutines.launch

@Composable
fun PriestRegistrationScreen(
    onRegistrationSuccess: (priestName: String, sampradaya: String) -> Unit
) {
    var fullName by remember { mutableStateOf("") }
    var phone by remember { mutableStateOf("") }
    var sampradaya by remember { mutableStateOf("Smartha") }
    var vedaShakha by remember { mutableStateOf("Rigveda Shakha") }
    var experienceYears by remember { mutableStateOf("10") }
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
            text = "🙏 Puja Vaahini - Purohit Onboarding",
            style = MaterialTheme.typography.headlineMedium,
            fontWeight = FontWeight.Bold,
            color = MaterialTheme.colorScheme.primary
        )
        Text(
            text = "Register Vedic Qualifications, Sampradaya & Accept On-Demand Code of Conduct",
            style = MaterialTheme.typography.bodyMedium,
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )

        Spacer(modifier = Modifier.height(20.dp))

        OutlinedTextField(
            value = fullName,
            onValueChange = { fullName = it },
            label = { Text("Priest Full Name & Title (e.g. Brahmasri ...)") },
            placeholder = { Text("e.g. Brahmasri Sastri Garu") },
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
                value = sampradaya,
                onValueChange = { sampradaya = it },
                label = { Text("Sampradaya") },
                modifier = Modifier.weight(1f)
            )
            OutlinedTextField(
                value = vedaShakha,
                onValueChange = { vedaShakha = it },
                label = { Text("Veda Shakha") },
                modifier = Modifier.weight(1f)
            )
        }

        Spacer(modifier = Modifier.height(12.dp))

        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            OutlinedTextField(
                value = experienceYears,
                onValueChange = { experienceYears = it },
                label = { Text("Years of Experience") },
                modifier = Modifier.weight(1f)
            )
            OutlinedTextField(
                value = city,
                onValueChange = { city = it },
                label = { Text("Primary City") },
                modifier = Modifier.weight(1f)
            )
        }

        Spacer(modifier = Modifier.height(20.dp))

        // Ayagaryllu On-Demand Code of Conduct
        Card(
            modifier = Modifier.fillMaxWidth(),
            colors = CardDefaults.cardColors(
                containerColor = MaterialTheme.colorScheme.surfaceVariant
            )
        ) {
            Column(modifier = Modifier.padding(14.dp)) {
                Text(
                    text = "📜 Ayagaryllu On-Demand Honor Code & Vidhi",
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.titleSmall
                )
                Spacer(modifier = Modifier.height(6.dp))
                Text(
                    text = "1. I pledge to perform all Vedic rituals strictly as per traditional Shastras.\n" +
                           "2. I agree to prompt arrival within the estimated ETA for accepted instant pooja requests.\n" +
                           "3. Devotee Dakshina will be released automatically upon entering the ritual completion OTP.\n" +
                           "4. I will maintain highest sanctity, traditional attire (Dhoti & Angavastram), and sacred conduct.",
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
                        text = "I accept the Vedic Honor Code & Escrow Terms *",
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
                    errorMessage = "Please enter priest full name and phone number."
                    return@Button
                }
                if (!consentAgreed) {
                    errorMessage = "You must agree to the Puja Vaahini Honor Code to activate on-demand duty."
                    return@Button
                }
                errorMessage = null
                coroutineScope.launch {
                    isSubmitting = true
                    val exp = experienceYears.toIntOrNull() ?: 5
                    val res = PurohitApiClient.registerPriest(
                        fullName = fullName.trim(),
                        phone = phone.trim(),
                        sampradaya = sampradaya.trim(),
                        vedaShakha = vedaShakha.trim(),
                        experienceYears = exp,
                        city = city.trim(),
                        consentAgreed = consentAgreed
                    )
                    isSubmitting = false
                    res.onSuccess {
                        onRegistrationSuccess(fullName, sampradaya)
                    }.onFailure { err ->
                        errorMessage = "Registration error: ${err.message}"
                    }
                }
            },
            modifier = Modifier
                .fillMaxWidth()
                .height(50.dp),
            enabled = !isSubmitting,
            colors = ButtonDefaults.buttonColors(containerColor = Color(0xFFB45309))
        ) {
            Text(if (isSubmitting) "Registering in PostgreSQL..." else "Activate Priest Account & Go to Duty")
        }

        Spacer(modifier = Modifier.height(16.dp))
    }
}
