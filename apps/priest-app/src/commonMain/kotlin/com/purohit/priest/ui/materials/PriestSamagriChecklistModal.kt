package com.purohit.priest.ui.materials

import androidx.compose.foundation.background
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.itemsIndexed
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.purohit.priest.network.BookingMaterialPayload
import com.purohit.priest.network.PurohitApiClient
import kotlinx.coroutines.launch
import com.purohit.contracts.i18n.AppLanguage
import com.purohit.priest.LocalPriestLanguage

@Composable
fun PriestSamagriChecklistModal(
    bookingId: String,
    poojaTitle: String = "Sri Satyanarayana Swamy Vratam",
    onDismiss: () -> Unit,
    onRemindDevotee: (String) -> Unit = {}
) {
    var materials by remember { mutableStateOf<List<BookingMaterialPayload>>(emptyList()) }
    var isLoading by remember { mutableStateOf(true) }
    var loadError by remember { mutableStateOf<String?>(null) }
    var isSaving by remember { mutableStateOf(false) }
    var saveSuccessMessage by remember { mutableStateOf<String?>(null) }
    var selectedTab by remember { mutableStateOf(0) } // 0: Sacred Samagri (Priest), 1: Devotee Household Items
    var showAddItemDialog by remember { mutableStateOf(false) }
    var newItemName by remember { mutableStateOf("") }
    var newItemQty by remember { mutableStateOf("1 set") }
    var pendingRemove by remember { mutableStateOf<BookingMaterialPayload?>(null) }

    var saveFailed by remember { mutableStateOf(false) }
    val language = LocalPriestLanguage.current
    fun tr(en: String, te: String, hi: String) = when (language) { AppLanguage.TELUGU -> te; AppLanguage.HINDI -> hi; else -> en }
    val coroutineScope = rememberCoroutineScope()

    LaunchedEffect(bookingId) {
        isLoading = true
        val res = PurohitApiClient.fetchBookingMaterials(bookingId)
        isLoading = false
        res.onSuccess { list ->
            materials = list
        }.onFailure {
            materials = emptyList()
            loadError = tr("Could not load this checklist. Please reopen it to retry.", "ఈ జాబితాను పొందలేకపోయాము. మళ్లీ తెరిచి ప్రయత్నించండి.", "यह सूची नहीं मिली। दोबारा खोलकर प्रयास करें।")
        }
    }

    val sacredItems = materials.filter { it.category != "Devotee Household Items" }
    val householdItems = materials.filter { it.category == "Devotee Household Items" }

    fun toggleItemPacked(itemName: String) {
        materials = materials.map { item ->
            if (item.item_name == itemName) {
                item.copy(is_included_by_priest = !item.is_included_by_priest)
            } else {
                item
            }
        }
    }

    // Saves [updated]; on failure the list goes back to [previous] so nothing looks saved that isn't.
    fun saveChecklist(updated: List<BookingMaterialPayload> = materials, previous: List<BookingMaterialPayload> = materials) {
        if (loadError != null || isSaving) return
        materials = updated
        isSaving = true
        coroutineScope.launch {
            val res = PurohitApiClient.updateBookingMaterials(bookingId, updated)
            isSaving = false
            res.onSuccess {
                saveFailed = false
                saveSuccessMessage = tr("✓ Checklist saved.", "✓ జాబితా సేవ్ అయింది.", "✓ सूची सहेजी गई।")
            }.onFailure {
                materials = previous
                saveFailed = true
                saveSuccessMessage = tr("Could not save the checklist. Please try again.", "జాబితాను సేవ్ చేయలేకపోయాము. మళ్లీ ప్రయత్నించండి.", "सूची सहेजी नहीं जा सकी। फिर प्रयास करें।")
            }
        }
    }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = {
            Column {
                loadError?.let { Text(it, color = MaterialTheme.colorScheme.error) }
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text(
                        text = "🧰 " + tr("Samagri Checklist", "సామాగ్రి జాబితా", "सामग्री सूची"),
                        fontWeight = FontWeight.Bold,
                        style = MaterialTheme.typography.titleMedium,
                        color = MaterialTheme.colorScheme.primary
                    )
                    IconButton(onClick = onDismiss) {
                        Text("✕", fontWeight = FontWeight.Bold)
                    }
                }
                Text(
                    text = poojaTitle,
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }
        },
        text = {
            Column(modifier = Modifier.fillMaxWidth()) {
                // Tab Row: Sacred Samagri vs Household
                TabRow(selectedTabIndex = selectedTab) {
                    Tab(
                        selected = selectedTab == 0,
                        onClick = { selectedTab = 0 },
                        text = { Text("🪔 ${tr("Sacred Kit", "పూజా కిట్", "पूजा किट")} (${sacredItems.size})", style = MaterialTheme.typography.labelSmall) }
                    )
                    Tab(
                        selected = selectedTab == 1,
                        onClick = { selectedTab = 1 },
                        text = { Text("🏡 ${tr("Devotee Home", "భక్తుని ఇల్లు", "भक्त का घर")} (${householdItems.size})", style = MaterialTheme.typography.labelSmall) }
                    )
                }

                Spacer(modifier = Modifier.height(10.dp))

                if (saveSuccessMessage != null) {
                    Surface(
                        color = if (saveFailed) MaterialTheme.colorScheme.errorContainer else Color(0xFFDCFCE7),
                        shape = RoundedCornerShape(8.dp),
                        modifier = Modifier.fillMaxWidth()
                    ) {
                        Text(
                            text = saveSuccessMessage!!,
                            color = if (saveFailed) MaterialTheme.colorScheme.error else Color(0xFF15803D),
                            style = MaterialTheme.typography.labelSmall,
                            modifier = Modifier.padding(8.dp),
                            fontWeight = FontWeight.Bold
                        )
                    }
                    Spacer(modifier = Modifier.height(8.dp))
                }

                if (isLoading) {
                    Box(modifier = Modifier.fillMaxWidth().height(180.dp), contentAlignment = Alignment.Center) {
                        CircularProgressIndicator()
                    }
                } else if (selectedTab == 0) {
                    // TAB 0: Sacred Kit (Priest Bag Checklist)
                    Text(
                        text = tr("Tick each item as you pack it before leaving:", "బయలుదేరే ముందు ప్యాక్ చేసిన ప్రతి వస్తువును టిక్ చేయండి:", "निकलने से पहले पैक की गई हर वस्तु पर टिक करें:"),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                    Spacer(modifier = Modifier.height(8.dp))

                    LazyColumn(
                        modifier = Modifier.weight(1f, fill = false).heightIn(max = 280.dp),
                        verticalArrangement = Arrangement.spacedBy(6.dp)
                    ) {
                        itemsIndexed(sacredItems) { _, item ->
                            Surface(
                                shape = RoundedCornerShape(8.dp),
                                color = if (item.is_included_by_priest) MaterialTheme.colorScheme.surfaceVariant else Color(0xFFF1F5F9),
                                modifier = Modifier.fillMaxWidth()
                            ) {
                                Row(
                                    modifier = Modifier.padding(horizontal = 10.dp, vertical = 6.dp).fillMaxWidth(),
                                    verticalAlignment = Alignment.CenterVertically
                                ) {
                                    Checkbox(
                                        checked = item.is_included_by_priest,
                                        onCheckedChange = { toggleItemPacked(item.item_name) }
                                    )
                                    Spacer(modifier = Modifier.width(6.dp))
                                    Column(modifier = Modifier.weight(1f)) {
                                        Text(
                                            text = item.item_name,
                                            fontWeight = if (item.is_included_by_priest) FontWeight.Bold else FontWeight.Normal,
                                            style = MaterialTheme.typography.bodySmall
                                        )
                                        Text(
                                            text = "${tr("Qty", "పరిమాణం", "मात्रा")}: ${item.quantity}",
                                            style = MaterialTheme.typography.labelSmall,
                                            color = MaterialTheme.colorScheme.onSurfaceVariant
                                        )
                                    }
                                    if (item.is_included_by_priest) {
                                        Text("✓ " + tr("Packed", "ప్యాక్ అయింది", "पैक हुआ"), color = MaterialTheme.colorScheme.primary, style = MaterialTheme.typography.labelSmall, fontWeight = FontWeight.Bold)
                                    }
                                    IconButton(onClick = { pendingRemove = item }, enabled = !isSaving) {
                                        Text("✕", color = MaterialTheme.colorScheme.error)
                                    }
                                }
                            }
                        }
                    }

                    Spacer(modifier = Modifier.height(10.dp))

                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(8.dp)
                    ) {
                        OutlinedButton(
                            onClick = { showAddItemDialog = true },
                            modifier = Modifier.weight(1f),
                            shape = RoundedCornerShape(8.dp)
                        ) {
                            Text("➕ " + tr("Add Item", "వస్తువు జోడించండి", "वस्तु जोड़ें"), style = MaterialTheme.typography.labelSmall)
                        }

                        Button(
                            onClick = { saveChecklist() },
                            modifier = Modifier.weight(1f),
                            shape = RoundedCornerShape(8.dp),
                            enabled = !isSaving
                        ) {
                            Text(if (isSaving) tr("Saving...", "సేవ్ అవుతోంది...", "सहेजा जा रहा है...") else "💾 " + tr("Save", "సేవ్ చేయండి", "सहेजें"), style = MaterialTheme.typography.labelSmall)
                        }
                    }
                } else {
                    // TAB 1: Devotee Household Items
                    Text(
                        text = tr("Items the devotee must keep ready at home before you arrive:", "మీరు రాకముందే భక్తుడు ఇంట్లో సిద్ధం చేయాల్సినవి:", "आपके आने से पहले भक्त को घर पर तैयार रखनी वस्तुएं:"),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                    Spacer(modifier = Modifier.height(8.dp))

                    LazyColumn(
                        modifier = Modifier.weight(1f, fill = false).heightIn(max = 280.dp),
                        verticalArrangement = Arrangement.spacedBy(6.dp)
                    ) {
                        itemsIndexed(householdItems) { _, item ->
                            Surface(
                                shape = RoundedCornerShape(8.dp),
                                color = Color(0xFFF8FAFC),
                                modifier = Modifier.fillMaxWidth()
                            ) {
                                Row(
                                    modifier = Modifier.padding(horizontal = 10.dp, vertical = 8.dp).fillMaxWidth(),
                                    verticalAlignment = Alignment.CenterVertically
                                ) {
                                    Text("🏡", style = MaterialTheme.typography.bodyMedium)
                                    Spacer(modifier = Modifier.width(8.dp))
                                    Column(modifier = Modifier.weight(1f)) {
                                        Text(
                                            text = item.item_name,
                                            fontWeight = FontWeight.SemiBold,
                                            style = MaterialTheme.typography.bodySmall
                                        )
                                        Text(
                                            text = "${tr("Needed", "కావాలి", "आवश्यक")}: ${item.quantity}",
                                            style = MaterialTheme.typography.labelSmall,
                                            color = MaterialTheme.colorScheme.onSurfaceVariant
                                        )
                                    }
                                }
                            }
                        }
                    }

                    Spacer(modifier = Modifier.height(10.dp))

                    Button(
                        onClick = {
                            val itemsSummary = householdItems.take(4).joinToString(", ") { it.item_name }
                            onRemindDevotee("Namaskaram! Please keep devotee household essentials ready: $itemsSummary.")
                            onDismiss()
                        },
                        modifier = Modifier.fillMaxWidth(),
                        shape = RoundedCornerShape(8.dp),
                        colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.secondary)
                    ) {
                        Text("💬 " + tr("Remind devotee in chat", "చాట్‌లో భక్తునికి గుర్తు చేయండి", "चैट में भक्त को याद दिलाएं"))
                    }
                }
            }
        },
        confirmButton = {
            OutlinedButton(onClick = onDismiss) {
                Text(tr("Close", "మూసివేయండి", "बंद करें"))
            }
        }
    )

    // Add Custom Item Dialog
    if (showAddItemDialog) {
        AlertDialog(
            onDismissRequest = { showAddItemDialog = false },
            title = { Text(tr("Add samagri item", "సామాగ్రి వస్తువు జోడించండి", "सामग्री वस्तु जोड़ें"), fontWeight = FontWeight.Bold) },
            text = {
                Column {
                    OutlinedTextField(
                        value = newItemName,
                        onValueChange = { newItemName = it },
                        label = { Text(tr("Item name (e.g. Navagraha Samidhalu)", "వస్తువు పేరు (ఉదా: నవగ్రహ సమిధలు)", "वस्तु का नाम (उदा: नवग्रह समिधा)")) },
                        modifier = Modifier.fillMaxWidth()
                    )
                    Spacer(modifier = Modifier.height(8.dp))
                    OutlinedTextField(
                        value = newItemQty,
                        onValueChange = { newItemQty = it },
                        label = { Text(tr("Quantity (e.g. 1 bundle / 250g)", "పరిమాణం (ఉదా: 1 కట్ట / 250గ్రా)", "मात्रा (उदा: 1 गट्ठर / 250 ग्राम)")) },
                        modifier = Modifier.fillMaxWidth()
                    )
                }
            },
            confirmButton = {
                Button(
                    onClick = {
                        if (newItemName.isNotBlank()) {
                            val newItem = BookingMaterialPayload(
                                booking_id = bookingId,
                                item_name = newItemName.trim(),
                                category = "Sacred Ritual Samagri",
                                quantity = newItemQty.trim(),
                                is_included_by_priest = true
                            )
                            newItemName = ""
                            newItemQty = "1 set"
                            showAddItemDialog = false
                            saveChecklist(updated = materials + newItem)
                        }
                    },
                    enabled = !isSaving && newItemName.isNotBlank()
                ) {
                    Text(tr("Add & Save", "జోడించి సేవ్ చేయండి", "जोड़ें और सहेजें"))
                }
            },
            dismissButton = {
                OutlinedButton(onClick = { showAddItemDialog = false }) {
                    Text(tr("Cancel", "రద్దు", "रद्द करें"))
                }
            }
        )
    }

    pendingRemove?.let { item ->
        AlertDialog(
            onDismissRequest = { pendingRemove = null },
            title = { Text(tr("Remove item?", "వస్తువును తొలగించాలా?", "वस्तु हटाएं?")) },
            text = { Text(item.item_name) },
            confirmButton = { TextButton(onClick = {
                pendingRemove = null
                saveChecklist(updated = materials.filter { it !== item })
            }) { Text(tr("Remove", "తొలగించు", "हटाएं"), color = MaterialTheme.colorScheme.error) } },
            dismissButton = { TextButton(onClick = { pendingRemove = null }) { Text(tr("Keep", "ఉంచు", "रखें")) } }
        )
    }
}
