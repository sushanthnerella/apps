package com.purohit.vendor.ui.orders

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
import com.purohit.contracts.store.StoreOrder
import com.purohit.contracts.store.StoreOrderItem

@Composable
fun VendorOrdersScreen(
    strings: StringContract
) {
    val orders = remember {
        mutableStateListOf(
            StoreOrder(
                id = "ord_101",
                userId = "u1",
                vendorId = "v1",
                bookingId = "bk_001",
                status = "PLACED",
                totalAmount = 630.0,
                deliveryAddress = "Flat 402, Sri Rama Nilayam, Hyderabad",
                deliverySlot = "Within 45 mins",
                items = listOf(
                    StoreOrderItem("item_1", "p1", "Pure Gir Cow Ghee (500ml)", 1, 480.0, 480.0),
                    StoreOrderItem("item_2", "p2", "Fresh Jasmine Flowers", 1, 150.0, 150.0)
                ),
                createdAt = "10 mins ago"
            ),
            StoreOrder(
                id = "ord_102",
                userId = "u2",
                vendorId = "v1",
                bookingId = "bk_002",
                status = "PACKED",
                totalAmount = 430.0,
                deliveryAddress = "House 12, Temple View Enclave, Hyderabad",
                deliverySlot = "Evening 05:00 PM",
                items = listOf(
                    StoreOrderItem("item_3", "p4", "Vedic Havan Wood (Sami)", 1, 250.0, 250.0),
                    StoreOrderItem("item_4", "p5", "Pure Bhimseni Camphor", 1, 180.0, 180.0)
                ),
                createdAt = "25 mins ago"
            )
        )
    }

    var selectedOrder by remember { mutableStateOf<StoreOrder?>(null) }

    Column(modifier = Modifier.fillMaxSize().padding(16.dp)) {
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            Column {
                Text(
                    text = "📦 " + strings.common.tabOrders,
                    style = MaterialTheme.typography.headlineMedium,
                    fontWeight = FontWeight.Bold,
                    color = MaterialTheme.colorScheme.primary
                )
                Text(
                    text = "Live incoming orders for pooja samagri kits & fresh flowers",
                    style = MaterialTheme.typography.bodySmall,
                    color = Color.Gray
                )
            }
            Surface(
                color = Color(0xFF10B981).copy(alpha = 0.15f),
                shape = RoundedCornerShape(8.dp)
            ) {
                Text(
                    text = "${orders.size} Active",
                    color = Color(0xFF047857),
                    fontWeight = FontWeight.Bold,
                    modifier = Modifier.padding(horizontal = 10.dp, vertical = 4.dp),
                    style = MaterialTheme.typography.labelMedium
                )
            }
        }

        Spacer(modifier = Modifier.height(14.dp))

        LazyColumn(
            verticalArrangement = Arrangement.spacedBy(12.dp),
            modifier = Modifier.weight(1f)
        ) {
            items(orders) { order ->
                Card(
                    modifier = Modifier.fillMaxWidth(),
                    shape = RoundedCornerShape(14.dp),
                    colors = CardDefaults.cardColors(
                        containerColor = if (order.status == "PLACED") Color(0xFFFEF3C7) else MaterialTheme.colorScheme.surfaceVariant
                    )
                ) {
                    Column(modifier = Modifier.padding(16.dp)) {
                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.SpaceBetween,
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            Text(
                                text = "Order #${order.id.takeLast(4).uppercase()}",
                                style = MaterialTheme.typography.titleMedium,
                                fontWeight = FontWeight.Bold
                            )
                            Surface(
                                color = when (order.status) {
                                    "PLACED" -> Color(0xFFF59E0B)
                                    "PACKED" -> Color(0xFF3B82F6)
                                    "DISPATCHED" -> Color(0xFF8B5CF6)
                                    else -> Color(0xFF10B981)
                                },
                                shape = RoundedCornerShape(6.dp)
                            ) {
                                Text(
                                    text = order.status,
                                    color = Color.White,
                                    fontWeight = FontWeight.Bold,
                                    style = MaterialTheme.typography.labelSmall,
                                    modifier = Modifier.padding(horizontal = 8.dp, vertical = 2.dp)
                                )
                            }
                        }

                        Spacer(modifier = Modifier.height(6.dp))

                        Text("📍 ${order.deliveryAddress}", style = MaterialTheme.typography.bodySmall, color = Color.DarkGray)
                        Text("⏰ Slot: ${order.deliverySlot ?: "Immediate"}", style = MaterialTheme.typography.bodySmall, color = Color.Gray)

                        Spacer(modifier = Modifier.height(8.dp))

                        // Items list
                        order.items.forEach { it ->
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween
                            ) {
                                Text("• ${it.itemName} (x${it.quantity})", style = MaterialTheme.typography.bodySmall)
                                Text("₹${it.subtotal.toInt()}", style = MaterialTheme.typography.bodySmall, fontWeight = FontWeight.SemiBold)
                            }
                        }

                        Divider(modifier = Modifier.padding(vertical = 8.dp))

                        Row(
                            modifier = Modifier.fillMaxWidth(),
                            horizontalArrangement = Arrangement.SpaceBetween,
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            Text(
                                text = "Total: ₹${order.totalAmount.toInt()}",
                                style = MaterialTheme.typography.titleMedium,
                                fontWeight = FontWeight.Bold,
                                color = MaterialTheme.colorScheme.primary
                            )

                            // Status Stepper Action
                            when (order.status) {
                                "PLACED" -> {
                                    Button(
                                        onClick = {
                                            val idx = orders.indexOf(order)
                                            if (idx != -1) {
                                                orders[idx] = order.copy(status = "PACKED")
                                            }
                                        },
                                        colors = ButtonDefaults.buttonColors(containerColor = Color(0xFF3B82F6))
                                    ) {
                                        Text("Mark Packed")
                                    }
                                }
                                "PACKED" -> {
                                    Button(
                                        onClick = {
                                            val idx = orders.indexOf(order)
                                            if (idx != -1) {
                                                orders[idx] = order.copy(status = "DISPATCHED")
                                            }
                                        },
                                        colors = ButtonDefaults.buttonColors(containerColor = Color(0xFF8B5CF6))
                                    ) {
                                        Text("Dispatch")
                                    }
                                }
                                "DISPATCHED" -> {
                                    Button(
                                        onClick = {
                                            val idx = orders.indexOf(order)
                                            if (idx != -1) {
                                                orders[idx] = order.copy(status = "DELIVERED")
                                            }
                                        },
                                        colors = ButtonDefaults.buttonColors(containerColor = Color(0xFF10B981))
                                    ) {
                                        Text("Mark Delivered")
                                    }
                                }
                                "DELIVERED" -> {
                                    Text("✓ Complete", fontWeight = FontWeight.Bold, color = Color(0xFF10B981))
                                }
                            }
                        }
                    }
                }
            }
        }
    }
}
