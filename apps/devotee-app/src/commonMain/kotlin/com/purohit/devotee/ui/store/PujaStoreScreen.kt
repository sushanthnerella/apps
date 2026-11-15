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
import com.purohit.contracts.i18n.AppLanguage
import com.purohit.contracts.i18n.StringContract
import com.purohit.contracts.idempotency.IdempotencyKeyGenerator
import com.purohit.contracts.store.StoreOrderItemDraft
import com.purohit.contracts.store.Vendor
import com.purohit.contracts.store.VendorInventoryItem
import com.purohit.devotee.LocalAppLanguage
import com.purohit.devotee.location.rememberLocationAccess
import com.purohit.devotee.network.PurohitApiClient
import com.purohit.devotee.network.StoreOrderItemPayload
import kotlinx.coroutines.launch

@Composable
fun PujaStoreScreen(
    strings: StringContract,
    onOrderPlaced: (vendorName: String, total: Double) -> Unit
) {
    val language = LocalAppLanguage.current
    fun tr(en: String, te: String, hi: String) = when (language) {
        AppLanguage.TELUGU -> te
        AppLanguage.HINDI -> hi
        else -> en
    }

    var isLoading by remember { mutableStateOf(true) }
    var storeError by remember { mutableStateOf<String?>(null) }
    var vendors by remember { mutableStateOf<List<Vendor>>(emptyList()) }
    var selectedVendor by remember { mutableStateOf<Vendor?>(null) }
    var products by remember { mutableStateOf<List<VendorInventoryItem>>(emptyList()) }
    val cart = remember { mutableStateMapOf<String, Int>() }
    val coroutineScope = rememberCoroutineScope()
    val locationAccess = rememberLocationAccess()
    var isPlacingOrder by remember { mutableStateOf(false) }
    var orderIntentKey by remember { mutableStateOf(IdempotencyKeyGenerator.generate()) }

    fun loadVendors() {
        isLoading = true
        storeError = null
        locationAccess.request { location ->
            val coordinates = location.getOrNull()
            if (coordinates == null) {
                vendors = emptyList()
                selectedVendor = null
                products = emptyList()
                storeError = tr(
                    "Allow location access to find nearby stores, then retry.",
                    "సమీప దుకాణాల కోసం లొకేషన్ అనుమతించి మళ్లీ ప్రయత్నించండి.",
                    "आस-पास की दुकानें खोजने के लिए स्थान की अनुमति दें और फिर प्रयास करें।"
                )
                isLoading = false
                return@request
            }
            coroutineScope.launch {
            val res = PurohitApiClient.fetchVendors(coordinates.latitude, coordinates.longitude)
            res.onSuccess { list ->
                vendors = list
                selectedVendor = list.firstOrNull()
                storeError = null
            }.onFailure {
                vendors = emptyList()
                selectedVendor = null
                products = emptyList()
                storeError = tr(
                    "Could not connect to store service. Please check your connection and retry.",
                    "దుకాణాల వివరాలు పొందలేకపోయాము. మీ కనెక్షన్‌ని తనిఖీ చేసి మళ్లీ ప్రయత్నించండి.",
                    "स्टोर सेवा से कनेक्ट नहीं हो सका। कृपया अपना कनेक्शन जाँचें और पुनः प्रयास करें।"
                )
            }
            isLoading = false
            }
        }
    }

    LaunchedEffect(Unit) {
        loadVendors()
    }

    LaunchedEffect(selectedVendor) {
        val v = selectedVendor
        if (v != null) {
            isLoading = true
            storeError = null
            val res = PurohitApiClient.fetchVendorInventory(v.id)
            res.onSuccess {
                products = it
            }.onFailure {
                products = emptyList()
                storeError = tr(
                    "Could not load products for this store. Please retry.",
                    "ఈ దుకాణం సామాగ్రి వివరాలు పొందలేకపోయాము. మళ్లీ ప్రయత్నించండి.",
                    "इस स्टोर की सामग्री लोड नहीं हो सकी। कृपया पुनः प्रयास करें।"
                )
            }
            isLoading = false
        } else {
            products = emptyList()
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
            color = MaterialTheme.colorScheme.onSurfaceVariant
        )

        Spacer(modifier = Modifier.height(12.dp))

        if (isLoading) {
            Spacer(modifier = Modifier.height(24.dp))
            LinearProgressIndicator(modifier = Modifier.fillMaxWidth())
            Spacer(modifier = Modifier.height(12.dp))
            Text(
                text = strings.common.loadingState,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.align(Alignment.CenterHorizontally)
            )
        } else if (storeError != null) {
            Spacer(modifier = Modifier.height(16.dp))
            Card(
                modifier = Modifier.fillMaxWidth(),
                colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.errorContainer)
            ) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Text(
                        text = storeError!!,
                        color = MaterialTheme.colorScheme.onErrorContainer,
                        style = MaterialTheme.typography.bodyMedium
                    )
                    Spacer(modifier = Modifier.height(12.dp))
                    Button(onClick = { loadVendors() }) {
                        Text(tr("Retry", "మళ్లీ ప్రయత్నించండి", "पुनः प्रयास करें"))
                    }
                }
            }
        } else if (vendors.isEmpty()) {
            Spacer(modifier = Modifier.height(16.dp))
            Card(
                modifier = Modifier.fillMaxWidth(),
                shape = RoundedCornerShape(16.dp),
                colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant)
            ) {
                Column(
                    modifier = Modifier.padding(24.dp),
                    horizontalAlignment = Alignment.CenterHorizontally
                ) {
                    Text(
                        text = "🏪",
                        style = MaterialTheme.typography.headlineLarge
                    )
                    Spacer(modifier = Modifier.height(12.dp))
                    Text(
                        text = tr("No Local Stores Available", "స్థానిక దుకాణాలు అందుబాటులో లేవు", "कोई स्थानीय स्टोर उपलब्ध नहीं है"),
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold,
                        color = MaterialTheme.colorScheme.onSurface
                    )
                    Spacer(modifier = Modifier.height(8.dp))
                    Text(
                        text = tr(
                            "Verified local vendor stores and samagri delivery are not yet active in this location for the free pilot. Ritual materials can be arranged directly with your assigned Purohit.",
                            "ఉచిత పైలట్ సమయంలో ఈ ప్రాంతంలో స్థానిక పూజా దుకాణాలు మరియు డెలివరీ అందుబాటులో లేవు. మీ పురోహితుల ద్వారా పూజా సామాగ్రిని నేరుగా సమకూర్చుకోవచ్చు.",
                            "निःशुल्क पायलट में इस स्थान पर स्थानीय पूजा सामग्री स्टोर अभी सक्रिय नहीं हैं। पूजा सामग्री की व्यवस्था अपने पुरोहित से सीधे कर सकते हैं।"
                        ),
                        style = MaterialTheme.typography.bodySmall,
                        color = MaterialTheme.colorScheme.onSurfaceVariant
                    )
                    Spacer(modifier = Modifier.height(16.dp))
                    OutlinedButton(
                        onClick = { loadVendors() },
                        modifier = Modifier.heightIn(min = 48.dp)
                    ) {
                        Text(tr("Refresh Stores", "స్టోర్లను రీఫ్రెష్ చేయండి", "स्टोर रीफ़्रेश करें"))
                    }
                }
            }
        } else {
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
                                Text("📍 ${v.address}", style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
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

            if (products.isEmpty()) {
                Text(
                    text = tr("No products listed for this store.", "ఈ దుకాణంలో వస్తువులు అందుబాటులో లేవు.", "इस स्टोर के लिए कोई उत्पाद उपलब्ध नहीं है।"),
                    color = MaterialTheme.colorScheme.onSurfaceVariant,
                    style = MaterialTheme.typography.bodyMedium
                )
            } else {
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
                                    Text(
                                        "${item.category} • ₹${item.price.toInt()} / ${item.unit}",
                                        style = MaterialTheme.typography.bodySmall,
                                        color = MaterialTheme.colorScheme.onSurfaceVariant
                                    )
                                }

                                // Add / Quantity Controls
                                if (quantity == 0) {
                                    OutlinedButton(
                                        onClick = { cart[item.id] = 1 },
                                        shape = RoundedCornerShape(8.dp),
                                        modifier = Modifier.heightIn(min = 48.dp)
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
                            Text(tr("Delivery Total:", "డెలివరీ మొత్తం:", "डिलीवरी कुल:"), style = MaterialTheme.typography.labelSmall)
                            Text("₹${cartTotal.toInt()}", fontWeight = FontWeight.Bold, style = MaterialTheme.typography.titleLarge)
                        }
                        Button(
                            onClick = {
                                selectedVendor?.let { v ->
                                    coroutineScope.launch {
                                        isPlacingOrder = true
                                        val itemsPayload = cart.map { (itemId, qty) ->
                                            StoreOrderItemDraft(
                                                inventory_item_id = itemId,
                                                quantity = qty
                                            )
                                        }
                                        val res = PurohitApiClient.createStoreOrder(
                                            vendorId = v.id,
                                            items = itemsPayload,
                                            idempotencyKey = orderIntentKey
                                        )
                                        isPlacingOrder = false
                                        if (res.isSuccess) {
                                            val finalTotal = if (res.getOrNull()?.total_amount != null && res.getOrNull()!!.total_amount > 0.0) res.getOrNull()!!.total_amount else cartTotal
                                            onOrderPlaced(v.shopName, finalTotal)
                                            cart.clear()
                                            orderIntentKey = IdempotencyKeyGenerator.generate()
                                        }
                                    }
                                }
                            },
                            colors = ButtonDefaults.buttonColors(containerColor = Color(0xFF059669)),
                            enabled = !isPlacingOrder,
                            modifier = Modifier.heightIn(min = 48.dp)
                        ) {
                            Text(if (isPlacingOrder) strings.common.loadingState else tr("Place Order", "ఆర్డర్ చేయండి", "ऑर्डर दें"))
                        }
                    }
                }
            }
        }
    }
}
