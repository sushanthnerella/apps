package com.purohit.vendor.ui.inventory

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
import com.purohit.contracts.i18n.StringContract
import com.purohit.contracts.store.VendorInventoryItem

@Composable
fun VendorInventoryScreen(
    strings: StringContract
) {
    val items = remember {
        mutableStateListOf(
            VendorInventoryItem("p1", "v1", "Pure Gir Cow Ghee (500ml)", "Ghee & Oils", 480.0, 45, "jar", "", true),
            VendorInventoryItem("p2", "v1", "Fresh Jasmine & Marigold Garlands", "Fresh Offerings", 150.0, 30, "garland", "", true),
            VendorInventoryItem("p3", "v1", "Organic Betel Leaves (25 Nos)", "Fresh Offerings", 60.0, 100, "bunch", "", true),
            VendorInventoryItem("p4", "v1", "Vedic Havan Wood (Sami, Palasha)", "Havan Samagri", 250.0, 15, "bundle", "", true),
            VendorInventoryItem("p5", "v1", "Pure Bhimseni Camphor (Karpuram)", "Dry Essentials", 180.0, 40, "box", "", true),
            VendorInventoryItem("p6", "v1", "Navadhanya Sacred 9-Grain Pack", "Havan Samagri", 160.0, 25, "pack", "", true)
        )
    }

    var showAddProductModal by remember { mutableStateOf(false) }
    var newName by remember { mutableStateOf("") }
    var newPrice by remember { mutableStateOf("120") }
    var newCategory by remember { mutableStateOf("Fresh Offerings") }
    var newStock by remember { mutableStateOf("25") }

    Column(modifier = Modifier.fillMaxSize().padding(16.dp)) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            Column {
                Text(
                    text = "🏪 " + strings.common.tabInventory,
                    style = MaterialTheme.typography.headlineMedium,
                    fontWeight = FontWeight.Bold,
                    color = MaterialTheme.colorScheme.primary
                )
                Text(
                    text = "Manage product prices, stock units, and instant availability",
                    style = MaterialTheme.typography.bodySmall,
                    color = Color.Gray
                )
            }
            Button(onClick = { showAddProductModal = true }) {
                Text("+ Item")
            }
        }

        Spacer(modifier = Modifier.height(14.dp))

        LazyColumn(
            verticalArrangement = Arrangement.spacedBy(10.dp),
            modifier = Modifier.weight(1f)
        ) {
            items(items) { itm ->
                Card(
                    modifier = Modifier.fillMaxWidth(),
                    shape = RoundedCornerShape(12.dp),
                    colors = CardDefaults.cardColors(
                        containerColor = if (itm.isAvailable) MaterialTheme.colorScheme.surfaceVariant else Color(0xFFF1F5F9)
                    )
                ) {
                    Row(
                        modifier = Modifier.padding(14.dp).fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Column(modifier = Modifier.weight(1f)) {
                            Text(itm.itemName, fontWeight = FontWeight.Bold, style = MaterialTheme.typography.bodyMedium)
                            Text("${itm.category} • ${itm.stockQuantity} ${itm.unit} in stock", style = MaterialTheme.typography.labelSmall, color = Color.Gray)
                            Text("₹${itm.price.toInt()} / ${itm.unit}", fontWeight = FontWeight.Bold, color = MaterialTheme.colorScheme.primary)
                        }

                        Column(horizontalAlignment = Alignment.End) {
                            Switch(
                                checked = itm.isAvailable,
                                onCheckedChange = { isChecked ->
                                    val idx = items.indexOf(itm)
                                    if (idx != -1) {
                                        items[idx] = itm.copy(isAvailable = isChecked)
                                    }
                                }
                            )
                            Text(
                                text = if (itm.isAvailable) "In Stock" else "Sold Out",
                                style = MaterialTheme.typography.labelSmall,
                                color = if (itm.isAvailable) Color(0xFF10B981) else Color.Gray,
                                fontWeight = FontWeight.Bold
                            )
                        }
                    }
                }
            }
        }
    }

    if (showAddProductModal) {
        AlertDialog(
            onDismissRequest = { showAddProductModal = false },
            title = { Text("Add Catalog Product", fontWeight = FontWeight.Bold) },
            text = {
                Column(verticalArrangement = Arrangement.spacedBy(10.dp)) {
                    OutlinedTextField(
                        value = newName,
                        onValueChange = { newName = it },
                        label = { Text("Product Name") },
                        modifier = Modifier.fillMaxWidth()
                    )
                    OutlinedTextField(
                        value = newPrice,
                        onValueChange = { newPrice = it },
                        label = { Text("Price (₹)") },
                        modifier = Modifier.fillMaxWidth()
                    )
                    OutlinedTextField(
                        value = newStock,
                        onValueChange = { newStock = it },
                        label = { Text("Stock Quantity") },
                        modifier = Modifier.fillMaxWidth()
                    )
                }
            },
            confirmButton = {
                Button(
                    onClick = {
                        if (newName.isNotBlank()) {
                            items.add(
                                VendorInventoryItem(
                                    id = "p_${items.size + 1}",
                                    vendorId = "v1",
                                    itemName = newName,
                                    category = newCategory,
                                    price = newPrice.toDoubleOrNull() ?: 100.0,
                                    stockQuantity = newStock.toIntOrNull() ?: 10,
                                    unit = "packet",
                                    isAvailable = true
                                )
                            )
                            showAddProductModal = false
                        }
                    }
                ) {
                    Text("Add to Storefront")
                }
            },
            dismissButton = {
                TextButton(onClick = { showAddProductModal = false }) {
                    Text("Cancel")
                }
            }
        )
    }
}
