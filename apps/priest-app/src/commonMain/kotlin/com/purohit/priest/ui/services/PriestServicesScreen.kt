package com.purohit.priest.ui.services

import androidx.compose.foundation.layout.*
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
    val customServices = remember {
        mutableStateListOf(
            CustomServiceItem(
                id = "cs1",
                priestId = priestId,
                title = if (currentLanguage == AppLanguage.TELUGU) "నవగ్రహ + ధన్వంతరి మహా హోమం (ప్రత్యేక పౌరోహిత్యం)" else "Navagraha + Dhanvantari Maha Homa Special",
                category = "Remedial Homa",
                dakshinaFee = 7500.0,
                samagriKitPrice = 2200.0,
                durationMinutes = 120,
                description = if (currentLanguage == AppLanguage.TELUGU) "గ్రహ శాంతి, వ్యాధి నివారణ మరియు సంపూర్ణ కుటుంబ సౌభాగ్యము కొరకు ఆచార్యుల ప్రత్యేక హోమము." else "Specialized Vedic ritual invoking longevity, planetary harmony, and family peace."
            ),
            CustomServiceItem(
                id = "cs2",
                priestId = priestId,
                title = if (currentLanguage == AppLanguage.TELUGU) "శ్రీ లలితా సహస్రనామ కుంకుమార్చన + చండీ పారాయణం" else "Sri Lalitha Sahasranama Kumkumarchana + Chandi Parayanam",
                category = "Devi Archana",
                dakshinaFee = 3516.0,
                samagriKitPrice = 1100.0,
                durationMinutes = 90,
                description = if (currentLanguage == AppLanguage.TELUGU) "వేద మంత్రోచ్ఛారణతో సాగే పవిత్ర దేవీ కుంకుమార్చన." else "Sacred Devi worship with authentic Vedic chanting and pure Kumkumarchana."
            )
        )
    }

    fun loadServices() {
        coroutineScope.launch {
            val res = PurohitApiClient.fetchCustomServices(lang = currentLanguage.isoCode)
            res.onSuccess { remoteList ->
                if (remoteList.isNotEmpty()) {
                    customServices.clear()
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
            }
        }
    }

    LaunchedEffect(currentLanguage, priestId) {
        loadServices()
    }

    var showCreateModal by remember { mutableStateOf(false) }
    var newTitle by remember { mutableStateOf("") }
    var newDakshina by remember { mutableStateOf("5116") }
    var newSamagriFee by remember { mutableStateOf("1500") }
    var newDuration by remember { mutableStateOf("90") }
    var newDescription by remember { mutableStateOf("") }
    var isSubmitting by remember { mutableStateOf(false) }

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
                    color = Color.Gray
                )
            }
            Button(onClick = { showCreateModal = true }) {
                Text(if (isTelugu) "+ నూతన సేవ" else if (isHindi) "+ नई सेवा" else "+ New Offering")
            }
        }

        Spacer(modifier = Modifier.height(16.dp))

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

                        Text(cs.description, style = MaterialTheme.typography.bodySmall, color = Color.DarkGray)

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
                                    color = Color.Gray
                                )
                            }
                            Surface(
                                shape = RoundedCornerShape(8.dp),
                                color = Color(0xFFDCFCE7)
                            ) {
                                Text(
                                    text = if (isTelugu) "3 భాషల అనువాదం సిద్ధం" else if (isHindi) "3 भाषाओं में अनुवादित" else "3 Languages Active",
                                    color = Color(0xFF15803D),
                                    fontWeight = FontWeight.Bold,
                                    style = MaterialTheme.typography.labelSmall,
                                    modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp)
                                )
                            }
                        }
                    }
                }
            }
        }
    }

    if (showCreateModal) {
        AlertDialog(
            onDismissRequest = { showCreateModal = false },
            title = {
                Text(
                    if (isTelugu) "నూతన సేవను చేర్చండి" else if (isHindi) "नई पूजा सेवा जोड़ें" else "Create Custom Offering",
                    fontWeight = FontWeight.Bold
                )
            },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                    OutlinedTextField(
                        value = newTitle,
                        onValueChange = { newTitle = it },
                        label = { Text(if (isTelugu) "పూజా పేరు (ఉదా: శ్రీ రుద్రాభిషేకం)" else if (isHindi) "पूजा का नाम (उदा: रुद्राभिषेक)" else "Offering Title") },
                        modifier = Modifier.fillMaxWidth()
                    )
                    OutlinedTextField(
                        value = newDakshina,
                        onValueChange = { newDakshina = it },
                        label = { Text("${strings.catalog.dakshinaLabel} (₹)") },
                        modifier = Modifier.fillMaxWidth()
                    )
                    OutlinedTextField(
                        value = newSamagriFee,
                        onValueChange = { newSamagriFee = it },
                        label = { Text(if (isTelugu) "సామాగ్రి కిట్ ఖర్చు (₹)" else if (isHindi) "सामग्री किट मूल्य (₹)" else "Samagri Kit Included Cost (₹)") },
                        modifier = Modifier.fillMaxWidth()
                    )
                    OutlinedTextField(
                        value = newDuration,
                        onValueChange = { newDuration = it },
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
                        if (newTitle.isNotBlank()) {
                            isSubmitting = true
                            coroutineScope.launch {
                                val payload = CreateCustomServicePayload(
                                    priest_id = priestId,
                                    title = newTitle,
                                    dakshina_fee = newDakshina.toDoubleOrNull() ?: 5116.0,
                                    samagri_kit_price = newSamagriFee.toDoubleOrNull() ?: 1500.0,
                                    duration_minutes = newDuration.toIntOrNull() ?: 90,
                                    description = if (newDescription.isNotBlank()) newDescription else "పురోహితులు నిర్వహించే పవిత్ర వేదోక్త పూజా విధానము."
                                )
                                PurohitApiClient.createCustomService(payload)
                                isSubmitting = false
                                showCreateModal = false
                                loadServices()
                            }
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
}
