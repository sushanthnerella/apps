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

@Composable
fun PriestSamagriChecklistModal(
    bookingId: String,
    poojaTitle: String = "Sri Satyanarayana Swamy Vratam",
    onDismiss: () -> Unit,
    onRemindDevotee: (String) -> Unit = {}
) {
    var materials by remember { mutableStateOf<List<BookingMaterialPayload>>(emptyList()) }
    var isLoading by remember { mutableStateOf(true) }
    var isSaving by remember { mutableStateOf(false) }
    var saveSuccessMessage by remember { mutableStateOf<String?>(null) }
    var selectedTab by remember { mutableStateOf(0) } // 0: Sacred Samagri (Priest), 1: Devotee Household Items
    var showAddItemDialog by remember { mutableStateOf(false) }
    var newItemName by remember { mutableStateOf("") }
    var newItemQty by remember { mutableStateOf("1 set") }

    val coroutineScope = rememberCoroutineScope()

    LaunchedEffect(bookingId) {
        isLoading = true
        val res = PurohitApiClient.fetchBookingMaterials(bookingId)
        isLoading = false
        res.onSuccess { list ->
            materials = list
        }.onFailure {
            // Default ritual samagri if offline
            materials = listOf(
                BookingMaterialPayload(item_name = "Haldi (Pasupu) & Kumkuma", category = "Sacred Ritual Samagri", quantity = "100g each", is_included_by_priest = true),
                BookingMaterialPayload(item_name = "Akshatulu & Gandham Paste", category = "Sacred Ritual Samagri", quantity = "1 box", is_included_by_priest = true),
                BookingMaterialPayload(item_name = "Dhoop Sticks & Camphor (Karpooram)", category = "Sacred Ritual Samagri", quantity = "1 pack", is_included_by_priest = true),
                BookingMaterialPayload(item_name = "Navadhanya Pack (9 Grains)", category = "Sacred Ritual Samagri", quantity = "1 pack", is_included_by_priest = true),
                BookingMaterialPayload(item_name = "Sacred Yagnopaveetam & Raksha Kankanam", category = "Sacred Ritual Samagri", quantity = "2 sets", is_included_by_priest = true),
                BookingMaterialPayload(item_name = "Satyanarayana Vrata Katha Book & Navagraha Chart", category = "Sacred Ritual Samagri", quantity = "1 set", is_included_by_priest = true),
                BookingMaterialPayload(item_name = "Brass / Copper Puja Plates (Thalis)", category = "Devotee Household Items", quantity = "3 nos", is_included_by_priest = false),
                BookingMaterialPayload(item_name = "Brass Deepam / Diyas & Pure Cow Ghee", category = "Devotee Household Items", quantity = "2 nos, 500g", is_included_by_priest = false),
                BookingMaterialPayload(item_name = "Sitting Mats (Asanam)", category = "Devotee Household Items", quantity = "2 nos", is_included_by_priest = false),
                BookingMaterialPayload(item_name = "Raw Rice (Biyyam) for Mandapam", category = "Devotee Household Items", quantity = "2 kg", is_included_by_priest = false),
                BookingMaterialPayload(item_name = "Fresh Bananas, Variety Fruits & Flowers", category = "Devotee Household Items", quantity = "2 dozens", is_included_by_priest = false)
            )
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

    fun saveChecklist() {
        isSaving = true
        coroutineScope.launch {
            val res = PurohitApiClient.updateBookingMaterials(bookingId, materials)
            isSaving = false
            res.onSuccess {
                saveSuccessMessage = "✓ Samagri checklist updated & synced with Cloud DB!"
            }.onFailure { err ->
                saveSuccessMessage = "Updated locally (Sync error: ${err.message})"
            }
        }
    }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = {
            Column {
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text(
                        text = "🧰 Samagri Checklist",
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
                    color = Color.Gray
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
                        text = { Text("🪔 Sacred Kit (${sacredItems.size})", style = MaterialTheme.typography.labelSmall) }
                    )
                    Tab(
                        selected = selectedTab == 1,
                        onClick = { selectedTab = 1 },
                        text = { Text("🏡 Devotee Home (${householdItems.size})", style = MaterialTheme.typography.labelSmall) }
                    )
                }

                Spacer(modifier = Modifier.height(10.dp))

                if (saveSuccessMessage != null) {
                    Surface(
                        color = Color(0xFFDCFCE7),
                        shape = RoundedCornerShape(8.dp),
                        modifier = Modifier.fillMaxWidth()
                    ) {
                        Text(
                            text = saveSuccessMessage!!,
                            color = Color(0xFF15803D),
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
                        text = "Check items as you pack them into your ritual travel bag before departure:",
                        style = MaterialTheme.typography.bodySmall,
                        color = Color.DarkGray
                    )
                    Spacer(modifier = Modifier.height(8.dp))

                    LazyColumn(
                        modifier = Modifier.weight(1f, fill = false).heightIn(max = 280.dp),
                        verticalArrangement = Arrangement.spacedBy(6.dp)
                    ) {
                        itemsIndexed(sacredItems) { _, item ->
                            Surface(
                                shape = RoundedCornerShape(8.dp),
                                color = if (item.is_included_by_priest) Color(0xFFFEF3C7) else Color(0xFFF1F5F9),
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
                                            text = "Qty: ${item.quantity}",
                                            style = MaterialTheme.typography.labelSmall,
                                            color = Color.Gray
                                        )
                                    }
                                    if (item.is_included_by_priest) {
                                        Text("✓ Packed", color = Color(0xFFB45309), style = MaterialTheme.typography.labelSmall, fontWeight = FontWeight.Bold)
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
                            Text("➕ Add Item", style = MaterialTheme.typography.labelSmall)
                        }

                        Button(
                            onClick = { saveChecklist() },
                            modifier = Modifier.weight(1f),
                            shape = RoundedCornerShape(8.dp),
                            enabled = !isSaving
                        ) {
                            Text(if (isSaving) "Saving..." else "💾 Save Sync", style = MaterialTheme.typography.labelSmall)
                        }
                    }
                } else {
                    // TAB 1: Devotee Household Items
                    Text(
                        text = "Standard essentials devotee must arrange at home before your arrival:",
                        style = MaterialTheme.typography.bodySmall,
                        color = Color.DarkGray
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
                                            text = "Required: ${item.quantity} (Devotee's responsibility)",
                                            style = MaterialTheme.typography.labelSmall,
                                            color = Color.Gray
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
                        Text("💬 Send Reminder via Devotee Chat")
                    }
                }
            }
        },
        confirmButton = {
            OutlinedButton(onClick = onDismiss) {
                Text("Close")
            }
        }
    )

    // Add Custom Item Dialog
    if (showAddItemDialog) {
        AlertDialog(
            onDismissRequest = { showAddItemDialog = false },
            title = { Text("Add Custom Samagri Item", fontWeight = FontWeight.Bold) },
            text = {
                Column {
                    OutlinedTextField(
                        value = newItemName,
                        onValueChange = { newItemName = it },
                        label = { Text("Samagri Name (e.g. Navagraha Samidhalu)") },
                        modifier = Modifier.fillMaxWidth()
                    )
                    Spacer(modifier = Modifier.height(8.dp))
                    OutlinedTextField(
                        value = newItemQty,
                        onValueChange = { newItemQty = it },
                        label = { Text("Quantity (e.g. 1 bundle / 250g)") },
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
                            materials = materials + newItem
                            newItemName = ""
                            newItemQty = "1 set"
                            showAddItemDialog = false
                            saveChecklist()
                        }
                    }
                ) {
                    Text("Add & Save")
                }
            },
            dismissButton = {
                OutlinedButton(onClick = { showAddItemDialog = false }) {
                    Text("Cancel")
                }
            }
        )
    }
}
