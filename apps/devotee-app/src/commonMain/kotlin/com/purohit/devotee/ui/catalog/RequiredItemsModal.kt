package com.purohit.devotee.ui.catalog

import androidx.compose.foundation.background
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
import com.purohit.devotee.network.PurohitApiClient
import com.purohit.devotee.network.SamagriItemPayload

@Composable
fun RequiredItemsModal(
    poojaId: String,
    poojaTitle: String,
    onDismiss: () -> Unit
) {
    var items by remember { mutableStateOf<List<SamagriItemPayload>>(emptyList()) }
    var isLoading by remember { mutableStateOf(true) }

    LaunchedEffect(poojaId) {
        isLoading = true
        val res = PurohitApiClient.fetchPoojaSamagri(poojaId)
        isLoading = false
        res.onSuccess { list ->
            items = list
        }.onFailure {
            // Fallback default list
            items = listOf(
                SamagriItemPayload(item_name = "Brass / Copper Puja Plates (Thalis)", category = "Devotee Household Items", quantity = "3 nos"),
                SamagriItemPayload(item_name = "Brass Panchapatra & Udharani", category = "Devotee Household Items", quantity = "1 set"),
                SamagriItemPayload(item_name = "Brass Deepam / Diyas & Pure Cow Ghee", category = "Devotee Household Items", quantity = "2 nos, 500g"),
                SamagriItemPayload(item_name = "Matchbox & Cotton Wicks (Vattulu)", category = "Devotee Household Items", quantity = "1 pack"),
                SamagriItemPayload(item_name = "Sitting Mats (Asanam)", category = "Devotee Household Items", quantity = "3 nos"),
                SamagriItemPayload(item_name = "Fresh Fruits (Bananas, etc.) & Flowers", category = "Devotee Household Items", quantity = "Variety, 1 garland"),
                SamagriItemPayload(item_name = "Fresh Betel Leaves & Supari (Tamalapakulu)", category = "Devotee Household Items", quantity = "25 leaves"),
                SamagriItemPayload(item_name = "Haldi (Pasupu) & Kumkuma", category = "Sacred Ritual Samagri", quantity = "100g each"),
                SamagriItemPayload(item_name = "Akshatulu & Gandham Paste", category = "Sacred Ritual Samagri", quantity = "1 box"),
                SamagriItemPayload(item_name = "Dhoop Cones & Karpooram (Camphor)", category = "Sacred Ritual Samagri", quantity = "1 pack"),
                SamagriItemPayload(item_name = "Navadhanya Kit (9 Sacred Cosmic Grains)", category = "Sacred Ritual Samagri", quantity = "1 kit"),
                SamagriItemPayload(item_name = "Homa Samidhalu & Purnahuti Dry Coconut", category = "Sacred Ritual Samagri", quantity = "2 bundles, 2 nos")
            )
        }
    }

    val householdItems = items.filter { it.category == "Devotee Household Items" }
    val sacredItems = items.filter { it.category != "Devotee Household Items" }

    AlertDialog(
        onDismissRequest = onDismiss,
        title = {
            Column {
                Text(
                    text = "📋 Required Items & Samagri",
                    fontWeight = FontWeight.Bold,
                    style = MaterialTheme.typography.titleMedium,
                    color = MaterialTheme.colorScheme.primary
                )
                Text(
                    text = poojaTitle,
                    style = MaterialTheme.typography.bodySmall,
                    color = Color.Gray
                )
            }
        },
        text = {
            if (isLoading) {
                Box(modifier = Modifier.fillMaxWidth().height(200.dp), contentAlignment = Alignment.Center) {
                    CircularProgressIndicator()
                }
            } else {
                LazyColumn(
                    modifier = Modifier.fillMaxWidth().heightIn(max = 420.dp),
                    verticalArrangement = Arrangement.spacedBy(10.dp)
                ) {
                    // Explanatory Guidance Banner
                    item {
                        Surface(
                            shape = RoundedCornerShape(10.dp),
                            color = Color(0xFFFEF3C7)
                        ) {
                            Text(
                                text = "💡 Guidance: Household essentials (plates, deepam, mats, fruits) should be kept ready at home. When you include '+Samagri', the Purohit brings all Consecrated Vedic items.",
                                style = MaterialTheme.typography.bodySmall,
                                color = Color(0xFF78350F),
                                modifier = Modifier.padding(10.dp)
                            )
                        }
                    }

                    // Section 1: Devotee Household Items
                    item {
                        Text(
                            text = "🏡 Items to Arrange at Home (${householdItems.size}):",
                            fontWeight = FontWeight.Bold,
                            style = MaterialTheme.typography.titleSmall,
                            color = Color(0xFFB45309)
                        )
                    }

                    items(householdItems) { item ->
                        Surface(
                            shape = RoundedCornerShape(8.dp),
                            color = Color(0xFFFFFBEB),
                            modifier = Modifier.fillMaxWidth()
                        ) {
                            Row(
                                modifier = Modifier.padding(horizontal = 10.dp, vertical = 6.dp).fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween,
                                verticalAlignment = Alignment.CenterVertically
                            ) {
                                Text(
                                    text = "• ${item.item_name}",
                                    style = MaterialTheme.typography.bodySmall,
                                    fontWeight = FontWeight.Medium,
                                    modifier = Modifier.weight(1f)
                                )
                                Text(
                                    text = item.quantity,
                                    style = MaterialTheme.typography.labelSmall,
                                    color = Color(0xFFB45309),
                                    fontWeight = FontWeight.Bold
                                )
                            }
                        }
                    }

                    // Section 2: Sacred Ritual Samagri
                    item {
                        Spacer(modifier = Modifier.height(4.dp))
                        Text(
                            text = "🪔 Sacred Samagri (Purohit Kit - ${sacredItems.size} Items):",
                            fontWeight = FontWeight.Bold,
                            style = MaterialTheme.typography.titleSmall,
                            color = Color(0xFF15803D)
                        )
                    }

                    items(sacredItems) { item ->
                        Surface(
                            shape = RoundedCornerShape(8.dp),
                            color = Color(0xFFF0FDF4),
                            modifier = Modifier.fillMaxWidth()
                        ) {
                            Row(
                                modifier = Modifier.padding(horizontal = 10.dp, vertical = 6.dp).fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween,
                                verticalAlignment = Alignment.CenterVertically
                            ) {
                                Text(
                                    text = "• ${item.item_name}",
                                    style = MaterialTheme.typography.bodySmall,
                                    fontWeight = FontWeight.Medium,
                                    modifier = Modifier.weight(1f)
                                )
                                Text(
                                    text = item.quantity,
                                    style = MaterialTheme.typography.labelSmall,
                                    color = Color(0xFF15803D),
                                    fontWeight = FontWeight.Bold
                                )
                            }
                        }
                    }
                }
            }
        },
        confirmButton = {
            Button(onClick = onDismiss) {
                Text("Got It")
            }
        }
    )
}
