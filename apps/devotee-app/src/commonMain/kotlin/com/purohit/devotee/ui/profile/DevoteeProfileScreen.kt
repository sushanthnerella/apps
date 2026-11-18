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
            text = profile?.full_name?.ifBlank { "Devotee Profile" } ?: "Devotee Profile",
            style = MaterialTheme.typography.headlineSmall,
            fontWeight = FontWeight.Bold
        )

        Text(
            text = profile?.phone?.ifBlank { "Phone Not Set" } ?: "Phone Not Set",
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
                    text = "🪔 Vedic Sankalpam Profile",
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.titleMedium,
                    color = MaterialTheme.colorScheme.primary
                )
                Spacer(modifier = Modifier.height(10.dp))
                Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                    Text(strings.auth.gotram, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    Text(profile?.gotram?.ifBlank { "Not Specified" } ?: "Not Specified", fontWeight = FontWeight.Bold)
                }
                Divider(modifier = Modifier.padding(vertical = 8.dp))
                Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                    Text(strings.auth.nakshatram, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    Text(profile?.nakshatram?.ifBlank { "Not Specified" } ?: "Not Specified", fontWeight = FontWeight.Bold)
                }
                Divider(modifier = Modifier.padding(vertical = 8.dp))
                Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                    Text(strings.auth.rashi, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    Text(profile?.rashi?.ifBlank { "Not Specified" } ?: "Not Specified", fontWeight = FontWeight.Bold)
                }
                Divider(modifier = Modifier.padding(vertical = 8.dp))
                Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                    Text(strings.auth.city, color = MaterialTheme.colorScheme.onSurfaceVariant)
                    Text(profile?.city?.ifBlank { "Hyderabad" } ?: "Hyderabad", fontWeight = FontWeight.Bold)
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
                        text = "📧 Password Recovery Email",
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
                                    text = "🟢 Verified",
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
                                    text = "⚠️ Unverified",
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
                            "Self-service password reset is enabled via single-use reset links sent to this email."
                        } else {
                            "Verification link pending or unverified. Complete verification to enable password reset."
                        },
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                } else {
                    Text(
                        text = "No verified email configured. Password reset via email is disabled.",
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
                    Text(if (currentEmail.isNotBlank()) "Update Recovery Email" else "Add Recovery Email")
                }
            }
        }

        if (showEmailDialog) {
            AlertDialog(
                onDismissRequest = { if (!emailLoading) showEmailDialog = false },
                title = { Text("Set Recovery Email") },
                text = {
                    Column {
                        Text(
                            "Enter your email address. We will send a secure single-use verification link. Your existing recovery email is preserved until the new email is verified.",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onSurfaceVariant
                        )
                        Spacer(modifier = Modifier.height(12.dp))
                        OutlinedTextField(
                            value = emailInput,
                            onValueChange = { emailInput = it },
                            label = { Text("Email Address") },
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
                            if (emailInput.isNotBlank()) {
                                emailLoading = true
                                emailMessage = null
                                emailError = null
                                coroutineScope.launch {
                                    val res = PurohitApiClient.requestEmailVerification(emailInput.trim())
                                    emailLoading = false
                                    res.fold(
                                        onSuccess = {
                                            emailMessage = "Verification link sent! Please check your inbox and click the single-use link to confirm."
                                            reloadProfile()
                                        },
                                        onFailure = { ex ->
                                            emailError = ex.message ?: "Failed to dispatch verification email."
                                        }
                                    )
                                }
                            }
                        },
                        enabled = !emailLoading && emailInput.isNotBlank()
                    ) {
                        if (emailLoading) {
                            CircularProgressIndicator(modifier = Modifier.size(16.dp), strokeWidth = 2.dp, color = Color.White)
                        } else {
                            Text("Send Link")
                        }
                    }
                },
                dismissButton = {
                    TextButton(
                        onClick = { showEmailDialog = false },
                        enabled = !emailLoading
                    ) {
                        Text("Close")
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
