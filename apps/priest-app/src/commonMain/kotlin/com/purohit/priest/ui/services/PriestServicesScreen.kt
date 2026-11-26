package com.purohit.priest.ui.services

import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.purohit.contracts.i18n.AppLanguage
import com.purohit.contracts.i18n.StringContract
import com.purohit.contracts.services.CustomServiceItem
import com.purohit.priest.network.CreateCustomServicePayload
import com.purohit.priest.network.PurohitApiClient
import kotlinx.coroutines.launch

@Composable
fun PriestServicesScreen(
    priestId: String = "6e7d277c-4ee2-469f-96ac-331517389762",
    currentLanguage: AppLanguage = AppLanguage.TELUGU,
    strings: StringContract
) {
    val coroutineScope = rememberCoroutineScope()
    val customServices = remember { mutableStateListOf<CustomServiceItem>() }
    var isLoading by remember { mutableStateOf(true) }
    var loadError by remember { mutableStateOf<String?>(null) }
    var createError by remember { mutableStateOf<String?>(null) }

    fun loadServices() {
        // Always scope to this priest: an empty priest_id lists every priest's offerings.
        val ownerId = priestId.ifBlank { PurohitApiClient.currentPriestProfile?.id.orEmpty() }
        if (ownerId.isBlank()) { isLoading = false; return }
        isLoading = true
        loadError = null
        coroutineScope.launch {
            val res = PurohitApiClient.fetchCustomServices(priestId = ownerId, lang = currentLanguage.isoCode)
            res.onSuccess { remoteList ->
                customServices.clear()
                run {
                    remoteList.forEach { r ->
                        customServices.add(
                            CustomServiceItem(
                                id = r.id,
                                priestId = r.priest_id,
                                title = r.title,
                                category = r.category,
                                dakshinaFee = r.dakshina_fee,
                                samagriKitPrice = r.samagri_kit_price,
                                durationMinutes = r.duration_minutes,
                                isOnlineEligible = r.is_online_eligible,
                                description = r.description
                            )
                        )
                    }
                }
            }.onFailure { loadError = "Could not load services. Please try again." }
            isLoading = false
        }
    }

    LaunchedEffect(currentLanguage, priestId) {
        loadServices()
    }

    var showCreateModal by remember { mutableStateOf(false) }
    var newTitle by remember { mutableStateOf("") }
    var newDakshina by remember { mutableStateOf("") }
    var newSamagriFee by remember { mutableStateOf("") }
    var newDuration by remember { mutableStateOf("") }
    var newDescription by remember { mutableStateOf("") }
    var isSubmitting by remember { mutableStateOf(false) }
    var pendingRemove by remember { mutableStateOf<CustomServiceItem?>(null) }

    val isTelugu = currentLanguage == AppLanguage.TELUGU
    val isHindi = currentLanguage == AppLanguage.HINDI

    Column(modifier = Modifier.fillMaxSize().padding(16.dp)) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = "📋 " + strings.common.tabCustomServices,
                    style = MaterialTheme.typography.headlineMedium,
                    fontWeight = FontWeight.Bold,
                    color = MaterialTheme.colorScheme.primary
                )
                Text(
                    text = if (isTelugu) "మీ వ్యక్తిగత దక్షిణ మరియు ప్రత్యేక వైదిక సేవల జాబితా" else if (isHindi) "आपकी व्यक्तिगत दक्षिणा एवं विशिष्ट पूजा सेवाओं की सूची" else "Customized ritual offerings with personalized Dakshina fees",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
            Button(onClick = { showCreateModal = true }) {
                Text(if (isTelugu) "+ నూతన సేవ" else if (isHindi) "+ नई सेवा" else "+ New Offering")
            }
        }

        Spacer(modifier = Modifier.height(16.dp))

        if (isLoading) {
            LinearProgressIndicator(modifier = Modifier.fillMaxWidth())
        } else if (loadError != null) {
            Text(loadError!!, color = MaterialTheme.colorScheme.error)
            TextButton(onClick = { loadServices() }) { Text((if (isTelugu) "మళ్లీ ప్రయత్నించండి" else if (isHindi) "फिर प्रयास करें" else "Retry")) }
        } else if (customServices.isEmpty()) {
            Text(if (isTelugu) "సేవలు ఇంకా లేవు" else if (isHindi) "अभी कोई सेवा नहीं है" else "No services yet", color = MaterialTheme.colorScheme.onSurfaceVariant)
        }
        LazyColumn(
            verticalArrangement = Arrangement.spacedBy(12.dp),
            modifier = Modifier.weight(1f)
        ) {
            items(customServices) { cs ->
                Card(
                    modifier = Modifier.fillMaxWidth(),
                    shape = RoundedCornerShape(14.dp),
                    colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant)
                ) {
                    Column(modifier = Modifier.padding(16.dp)) {
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.SpaceBetween,
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            Text(
                                text = cs.title,
                                style = MaterialTheme.typography.titleMedium,
                                fontWeight = FontWeight.Bold,
                                modifier = Modifier.weight(1f)
                            )
                            Surface(
                                color = MaterialTheme.colorScheme.primary.copy(alpha = 0.15f),
                                shape = RoundedCornerShape(8.dp)
                            ) {
                                Text(
                                    text = "${cs.durationMinutes} ${strings.catalog.durationMinutes}",
                                    modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp),
                                    style = MaterialTheme.typography.labelSmall,
                                    fontWeight = FontWeight.Bold,
                                    color = MaterialTheme.colorScheme.primary
                                )
                            }
                        }

                        Spacer(modifier = Modifier.height(6.dp))

                        Text(cs.description, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)

                        Spacer(modifier = Modifier.height(12.dp))

                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.SpaceBetween,
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            Column {
                                Text(
                                    "${strings.catalog.dakshinaLabel}: ₹${cs.dakshinaFee.toInt()}",
                                    fontWeight = FontWeight.Bold,
                                    color = MaterialTheme.colorScheme.primary
                                )
                                Text(
                                    "${if (isTelugu) "సామాగ్రి కిట్" else if (isHindi) "सामग्री किट" else "Samagri Kit"}: +₹${cs.samagriKitPrice.toInt()}",
                                    style = MaterialTheme.typography.labelSmall,
                                    color = MaterialTheme.colorScheme.onSurfaceVariant
                                )
                            }
                            TextButton(onClick = { pendingRemove = cs }) { Text(if (isTelugu) "తొలగించు" else if (isHindi) "हटाएँ" else "Remove", color = MaterialTheme.colorScheme.error) }
                        }
                    }
                }
            }
        }
    }

    if (showCreateModal) {
        AlertDialog(
            // Back / outside tap only closes an untouched form, so typed details are not lost.
            onDismissRequest = { if (listOf(newTitle, newDakshina, newSamagriFee, newDuration, newDescription).all { it.isBlank() }) showCreateModal = false },
            title = {
                Text(
                    if (isTelugu) "నూతన సేవను చేర్చండి" else if (isHindi) "नई पूजा सेवा जोड़ें" else "Create Custom Offering",
                    fontWeight = FontWeight.Bold
                )
            },
            text = {
                Column(modifier = Modifier.verticalScroll(rememberScrollState()), verticalArrangement = Arrangement.spacedBy(10.dp)) {
                    createError?.let { Text(it, color = MaterialTheme.colorScheme.error) }
                    OutlinedTextField(
                        value = newTitle,
                        onValueChange = { newTitle = it },
                        label = { Text(if (isTelugu) "పూజా పేరు (ఉదా: శ్రీ రుద్రాభిషేకం)" else if (isHindi) "पूजा का नाम (उदा: रुद्राभिषेक)" else "Offering Title") },
                        modifier = Modifier.fillMaxWidth()
                    )
                    OutlinedTextField(
                        value = newDakshina,
                        onValueChange = { newDakshina = it.filter { c -> c.isDigit() || c == '.' }.take(9) },
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                        singleLine = true,
                        isError = newDakshina.isNotBlank() && newDakshina.toDoubleOrNull() == null,
                        prefix = { Text("₹") },
                        label = { Text("${strings.catalog.dakshinaLabel} (₹)") },
                        modifier = Modifier.fillMaxWidth()
                    )
                    OutlinedTextField(
                        value = newSamagriFee,
                        onValueChange = { newSamagriFee = it.filter { c -> c.isDigit() || c == '.' }.take(9) },
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Decimal),
                        singleLine = true,
                        isError = newSamagriFee.isNotBlank() && newSamagriFee.toDoubleOrNull() == null,
                        prefix = { Text("₹") },
                        label = { Text(if (isTelugu) "సామాగ్రి కిట్ ఖర్చు (₹)" else if (isHindi) "सामग्री किट मूल्य (₹)" else "Samagri Kit Included Cost (₹)") },
                        modifier = Modifier.fillMaxWidth()
                    )
                    OutlinedTextField(
                        value = newDuration,
                        onValueChange = { newDuration = it.filter { c -> c.isDigit() }.take(4) },
                        keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Number),
                        singleLine = true,
                        isError = newDuration.isNotBlank() && (newDuration.toIntOrNull() ?: 0) <= 0,
                        label = { Text("${strings.catalog.durationLabel} (${strings.catalog.durationMinutes})") },
                        modifier = Modifier.fillMaxWidth()
                    )
                    OutlinedTextField(
                        value = newDescription,
                        onValueChange = { newDescription = it },
                        label = { Text(if (isTelugu) "పూజా వివరాలు & విశిష్టత" else if (isHindi) "पूजा का विवरण एवं महत्व" else "Description") },
                        modifier = Modifier.fillMaxWidth()
                    )
                }
            },
            confirmButton = {
                Button(
                    onClick = {
                        if (newTitle.isNotBlank() && (newDakshina.toDoubleOrNull() ?: -1.0) >= 0 && (newSamagriFee.ifBlank { "0" }.toDoubleOrNull() ?: -1.0) >= 0 && (newDuration.toIntOrNull() ?: 0) > 0) {
                            createError = null
                            isSubmitting = true
                            coroutineScope.launch {
                                val payload = CreateCustomServicePayload(
                                    priest_id = priestId,
                                    title = newTitle,
                                    dakshina_fee = newDakshina.toDouble(),
                                    samagri_kit_price = newSamagriFee.ifBlank { "0" }.toDouble(),
                                    duration_minutes = newDuration.toInt(),
                                    description = newDescription
                                )
                                PurohitApiClient.createCustomService(payload).onSuccess {
                                    showCreateModal = false
                                    loadServices()
                                }.onFailure {
                                    createError = (if (isTelugu) "సేవను సేవ్ చేయలేకపోయాము. మళ్లీ ప్రయత్నించండి." else if (isHindi) "सेवा सहेजी नहीं जा सकी। फिर प्रयास करें।" else "Could not save the service. Please try again.")
                                }
                                isSubmitting = false
                            }
                        } else {
                            createError = (if (isTelugu) "పేరు, సరైన ధరలు (₹), నిమిషాల్లో సమయం నమోదు చేయండి." else if (isHindi) "नाम, सही मूल्य (₹) और मिनटों में अवधि दर्ज करें।" else "Enter a title, valid prices (₹) and a duration in minutes.")
                        }
                    },
                    enabled = !isSubmitting
                ) {
                    Text(if (isTelugu) "సేవను ప్రచురించండి" else if (isHindi) "सेवा प्रकाशित करें" else "Publish Offering")
                }
            },
            dismissButton = {
                TextButton(onClick = { showCreateModal = false }) {
                    Text(strings.common.cancelAction)
                }
            }
        )
    }

    pendingRemove?.let { item ->
        AlertDialog(
            onDismissRequest = { pendingRemove = null },
            title = { Text(if (isTelugu) "ఈ సేవను తొలగించాలా?" else if (isHindi) "यह सेवा हटाएँ?" else "Remove this offering?") },
            confirmButton = { TextButton(onClick = {
                pendingRemove = null
                                coroutineScope.launch {
                                    PurohitApiClient.deactivateCustomService(item.id)
                                        .onSuccess { customServices.remove(item); loadError = null }
                                        .onFailure { loadError = if (isTelugu) "తొలగించలేకపోయాము. మళ్లీ ప్రయత్నించండి." else if (isHindi) "हटा नहीं सके। फिर कोशिश करें।" else "Could not remove this offering. Try again." }
                                }
            }) { Text(if (isTelugu) "తొలగించు" else if (isHindi) "हटाएँ" else "Remove", color = MaterialTheme.colorScheme.error) } },
            dismissButton = { TextButton(onClick = { pendingRemove = null }) { Text(if (isTelugu) "ఉంచు" else if (isHindi) "रखें" else "Keep") } }
        )
    }
}
