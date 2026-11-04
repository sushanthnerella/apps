package com.purohit.devotee.ui.store

import androidx.compose.foundation.clickable
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
import com.purohit.contracts.store.Vendor
import com.purohit.contracts.store.VendorInventoryItem
import com.purohit.devotee.network.PurohitApiClient
import com.purohit.devotee.network.StoreOrderItemPayload
import kotlinx.coroutines.launch

@Composable
fun PujaStoreScreen(
    strings: StringContract,
    onOrderPlaced: (vendorName: String, total: Double) -> Unit
) {
    var vendors by remember { mutableStateOf<List<Vendor>>(emptyList()) }
    var selectedVendor by remember { mutableStateOf<Vendor?>(null) }
    var products by remember { mutableStateOf<List<VendorInventoryItem>>(emptyList()) }
    val cart = remember { mutableStateMapOf<String, Int>() }
    val coroutineScope = rememberCoroutineScope()
    var isPlacingOrder by remember { mutableStateOf(false) }

    LaunchedEffect(Unit) {
        val res = PurohitApiClient.fetchVendors()
        res.onSuccess {
            vendors = it
            if (it.isNotEmpty()) {
                selectedVendor = it[0]
            }
        }.onFailure {
            // Fallback list if network issue
            val fallback = listOf(
                Vendor(
                    id = "v1",
                    shopName = "Sri Sai Vedic Pooja Store & Samagri Center",
                    ownerName = "Ramesh Kumar Sharma",
                    phone = "+91 98490 55443",
                    address = "Shop #4, Near Temple Arch, Main Road, Hyderabad",
                    ratingAvg = 4.95,
                    isOpen = true,
                    distanceKm = 1.2
                )
            )
            vendors = fallback
            selectedVendor = fallback[0]
        }
    }

    LaunchedEffect(selectedVendor) {
        selectedVendor?.let { v ->
            val res = PurohitApiClient.fetchVendorInventory(v.id)
            res.onSuccess {
                products = it
            }.onFailure {
                products = listOf(
                    VendorInventoryItem("p1", v.id, "Pure Gir Cow Ghee (500ml)", "Ghee & Oils", 480.0, 50, "jar", "", true),
                    VendorInventoryItem("p2", v.id, "Fresh Jasmine & Marigold Garlands", "Fresh Offerings", 150.0, 30, "garland", "", true),
                    VendorInventoryItem("p3", v.id, "Organic Betel Leaves (25 Nos)", "Fresh Offerings", 60.0, 100, "bunch", "", true),
                    VendorInventoryItem("p4", v.id, "Vedic Havan Wood (Sami, Palasha)", "Havan Samagri", 250.0, 20, "bundle", "", true),
                    VendorInventoryItem("p5", v.id, "Pure Bhimseni Camphor (Karpuram)", "Dry Essentials", 180.0, 40, "box", "", true)
                )
            }
        }
    }

    val cartTotal = products.sumOf { p -> (cart[p.id] ?: 0) * p.price }

    Column(modifier = Modifier.fillMaxSize().padding(16.dp)) {
        Text(
            text = "🛒 " + strings.common.tabPujaStore,
            style = MaterialTheme.typography.headlineMedium,
            fontWeight = FontWeight.Bold,
            color = MaterialTheme.colorScheme.primary
        )
        Text(
            text = strings.samagri.storeVendorDeliveryNotice,
            style = MaterialTheme.typography.bodySmall,
            color = Color.Gray
        )

        Spacer(modifier = Modifier.height(12.dp))

        // Vendor Selector Card
        selectedVendor?.let { v ->
            Card(
                modifier = Modifier.fillMaxWidth(),
                shape = RoundedCornerShape(12.dp),
                colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant)
            ) {
                Column(modifier = Modifier.padding(12.dp)) {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Column(modifier = Modifier.weight(1f)) {
                            Text(v.shopName, fontWeight = FontWeight.Bold)
                            Text("Prop: ${v.ownerName} • ${v.phone}", style = MaterialTheme.typography.bodySmall)
                            Text("📍 ${v.address}", style = MaterialTheme.typography.bodySmall, color = Color.Gray)
                        }
                        Surface(
                            color = Color(0xFFDCFCE7),
                            shape = RoundedCornerShape(8.dp)
                        ) {
                            Text(
                                text = "★ ${v.ratingAvg}",
                                color = Color(0xFF15803D),
                                fontWeight = FontWeight.Bold,
                                modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp),
                                style = MaterialTheme.typography.labelSmall
                            )
                        }
                    }
                }
            }
        }

        Spacer(modifier = Modifier.height(16.dp))

        Text(
            text = strings.samagri.checklistTitle,
            fontWeight = FontWeight.Bold,
            style = MaterialTheme.typography.titleMedium
        )

        Spacer(modifier = Modifier.height(8.dp))

        // Product Inventory List
        LazyColumn(
            modifier = Modifier.weight(1f),
            verticalArrangement = Arrangement.spacedBy(8.dp)
        ) {
            items(products) { item ->
                val quantity = cart[item.id] ?: 0
                Card(
                    modifier = Modifier.fillMaxWidth(),
                    shape = RoundedCornerShape(10.dp)
                ) {
                    Row(
                        modifier = Modifier.padding(12.dp).fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Column(modifier = Modifier.weight(1f)) {
                            Text(item.itemName, fontWeight = FontWeight.Bold)
                            Text("${item.category} • ₹${item.price.toInt()} / ${item.unit}", style = MaterialTheme.typography.bodySmall, color = Color.Gray)
                        }

                        // Add / Quantity Controls
                        if (quantity == 0) {
                            OutlinedButton(
                                onClick = { cart[item.id] = 1 },
                                shape = RoundedCornerShape(8.dp)
                            ) {
                                Text("+ Add")
                            }
                        } else {
                            Row(verticalAlignment = Alignment.CenterVertically) {
                                FilledTonalIconButton(
                                    onClick = {
                                        if (quantity > 1) cart[item.id] = quantity - 1
                                        else cart.remove(item.id)
                                    }
                                ) {
                                    Text("-", fontWeight = FontWeight.Bold)
                                }
                                Text(
                                    text = "$quantity",
                                    fontWeight = FontWeight.Bold,
                                    modifier = Modifier.padding(horizontal = 8.dp)
                                )
                                FilledTonalIconButton(
                                    onClick = { cart[item.id] = quantity + 1 }
                                ) {
                                    Text("+", fontWeight = FontWeight.Bold)
                                }
                            }
                        }
                    }
                }
            }
        }

        // Bottom Cart Checkout Bar
        if (cartTotal > 0) {
            Spacer(modifier = Modifier.height(12.dp))
            Card(
                modifier = Modifier.fillMaxWidth(),
                shape = RoundedCornerShape(16.dp),
                colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.primaryContainer)
            ) {
                Row(
                    modifier = Modifier.padding(16.dp).fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Column {
                        Text("Doorstep Delivery Total:", style = MaterialTheme.typography.labelSmall)
                        Text("₹${cartTotal.toInt()}", fontWeight = FontWeight.Bold, style = MaterialTheme.typography.titleLarge)
                    }
                    Button(
                        onClick = {
                            selectedVendor?.let { v ->
                                coroutineScope.launch {
                                    isPlacingOrder = true
                                    val itemsPayload = cart.map { (itemId, qty) ->
                                        val product = products.find { it.id == itemId }
                                        StoreOrderItemPayload(
                                            item_id = itemId,
                                            item_name = product?.itemName ?: "Pooja Item",
                                            quantity = qty,
                                            unit_price = product?.price ?: 100.0
                                        )
                                    }
                                    PurohitApiClient.createStoreOrder(v.id, cartTotal, itemsPayload)
                                    isPlacingOrder = false
                                    onOrderPlaced(v.shopName, cartTotal)
                                    cart.clear()
                                }
                            }
                        },
                        colors = ButtonDefaults.buttonColors(containerColor = Color(0xFF059669)),
                        enabled = !isPlacingOrder
                    ) {
                        Text(if (isPlacingOrder) "Submitting..." else "Place Order")
                    }
                }
            }
        }
    }
}
