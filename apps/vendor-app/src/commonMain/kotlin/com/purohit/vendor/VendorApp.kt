package com.purohit.vendor

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.purohit.contracts.i18n.AppLanguage
import com.purohit.contracts.i18n.LocalizationManager
import com.purohit.contracts.i18n.StringContract
import com.purohit.vendor.ui.inventory.VendorInventoryScreen
import com.purohit.vendor.ui.orders.VendorOrdersScreen

val LocalVendorStrings = compositionLocalOf<StringContract> {
    LocalizationManager.getStrings(AppLanguage.ENGLISH)
}

enum class VendorScreen {
    ORDERS,     // Tab 1: 📦 Live Orders Feed
    INVENTORY,  // Tab 2: 🏪 Stock & Inventory Catalog
    PROFILE     // Tab 3: ⚙️ Store Profile & Business Hours
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun VendorAppRoot() {
    var currentScreen by remember { mutableStateOf(VendorScreen.ORDERS) }
    var isStoreOpen by remember { mutableStateOf(true) }
    var currentLanguage by remember { mutableStateOf(AppLanguage.TELUGU) } // Regional default
    val strings = remember(currentLanguage) { LocalizationManager.getStrings(currentLanguage) }
    var showLanguageModal by remember { mutableStateOf(false) }

    CompositionLocalProvider(LocalVendorStrings provides strings) {
        MaterialTheme(
            colorScheme = lightColorScheme(
                primary = Color(0xFF0D9488), // Vedic emerald/teal for vendors
                secondary = Color(0xFF0F766E),
                background = Color(0xFFF0FDF4)
            )
        ) {
            Scaffold(
                topBar = {
                    Surface(
                        color = MaterialTheme.colorScheme.primary,
                        modifier = Modifier.fillMaxWidth()
                    ) {
                        Row(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(horizontal = 16.dp, vertical = 10.dp),
                            horizontalArrangement = Arrangement.SpaceBetween,
                            verticalAlignment = Alignment.CenterVertically
                        ) {
                            Column {
                                Text(
                                    text = "🏪 పూజా స్టోర్ వ్యాపారి",
                                    color = Color.White,
                                    fontWeight = FontWeight.Bold,
                                    style = MaterialTheme.typography.titleMedium
                                )
                                Text(
                                    text = "Sri Sai Vedic Pooja Store",
                                    color = Color.White.copy(alpha = 0.8f),
                                    style = MaterialTheme.typography.labelSmall
                                )
                            }
                            Surface(
                                shape = MaterialTheme.shapes.small,
                                color = Color.White.copy(alpha = 0.2f),
                                modifier = Modifier.clickable { showLanguageModal = true }
                            ) {
                                Text(
                                    text = "🌐 ${currentLanguage.nativeName}",
                                    color = Color.White,
                                    fontWeight = FontWeight.Bold,
                                    style = MaterialTheme.typography.labelSmall,
                                    modifier = Modifier.padding(horizontal = 8.dp, vertical = 4.dp)
                                )
                            }
                        }
                    }
                },
                bottomBar = {
                    NavigationBar {
                        NavigationBarItem(
                            selected = currentScreen == VendorScreen.ORDERS,
                            onClick = { currentScreen = VendorScreen.ORDERS },
                            label = { Text(strings.common.tabOrders) },
                            icon = { Text("📦") }
                        )
                        NavigationBarItem(
                            selected = currentScreen == VendorScreen.INVENTORY,
                            onClick = { currentScreen = VendorScreen.INVENTORY },
                            label = { Text(strings.common.tabInventory) },
                            icon = { Text("🏪") }
                        )
                        NavigationBarItem(
                            selected = currentScreen == VendorScreen.PROFILE,
                            onClick = { currentScreen = VendorScreen.PROFILE },
                            label = { Text(strings.common.tabProfile) },
                            icon = { Text("⚙️") }
                        )
                    }
                }
            ) { paddingValues ->
                Box(modifier = Modifier.padding(paddingValues).fillMaxSize()) {
                    when (currentScreen) {
                        VendorScreen.ORDERS -> VendorOrdersScreen(strings = strings)
                        VendorScreen.INVENTORY -> VendorInventoryScreen(strings = strings)
                        VendorScreen.PROFILE -> {
                            Column(
                                modifier = Modifier.fillMaxSize().padding(20.dp),
                                horizontalAlignment = Alignment.CenterHorizontally
                            ) {
                                Card(
                                    modifier = Modifier.fillMaxWidth(),
                                    shape = RoundedCornerShape(16.dp),
                                    colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant)
                                ) {
                                    Column(modifier = Modifier.padding(16.dp)) {
                                        Text(
                                            text = "Sri Sai Vedic Pooja Store & Samagri Center",
                                            fontWeight = FontWeight.Bold,
                                            style = MaterialTheme.typography.titleMedium
                                        )
                                        Text("Owner: Ramesh Kumar Sharma", style = MaterialTheme.typography.bodySmall, color = Color.Gray)
                                        Text("Phone: +91 98490 55443", style = MaterialTheme.typography.bodySmall, color = Color.Gray)
                                        Text("Address: Shop #4, Near Temple Arch, Main Road, Hyderabad", style = MaterialTheme.typography.bodySmall, color = Color.Gray)

                                        Divider(modifier = Modifier.padding(vertical = 12.dp))

                                        Row(
                                            modifier = Modifier.fillMaxWidth(),
                                            horizontalArrangement = Arrangement.SpaceBetween,
                                            verticalAlignment = Alignment.CenterVertically
                                        ) {
                                            Column {
                                                Text(
                                                    text = if (isStoreOpen) "🟢 Storefront OPEN" else "🔴 Storefront CLOSED",
                                                    fontWeight = FontWeight.Bold,
                                                    color = if (isStoreOpen) Color(0xFF10B981) else Color.Red
                                                )
                                                Text("Receiving live devotee orders", style = MaterialTheme.typography.labelSmall)
                                            }
                                            Switch(
                                                checked = isStoreOpen,
                                                onCheckedChange = { isStoreOpen = it }
                                            )
                                        }
                                    }
                                }

                                Spacer(modifier = Modifier.height(20.dp))

                                OutlinedButton(
                                    onClick = { showLanguageModal = true },
                                    modifier = Modifier.fillMaxWidth()
                                ) {
                                    Text("🌐 ${strings.common.selectLanguage} (${currentLanguage.nativeName})")
                                }
                            }
                        }
                    }
                }

                // Language Modal
                if (showLanguageModal) {
                    AlertDialog(
                        onDismissRequest = { showLanguageModal = false },
                        title = { Text(strings.common.selectLanguage, fontWeight = FontWeight.Bold) },
                        text = {
                            Column {
                                AppLanguage.entries.forEach { lang ->
                                    Row(
                                        modifier = Modifier
                                            .fillMaxWidth()
                                            .clickable {
                                                currentLanguage = lang
                                                showLanguageModal = false
                                            }
                                            .padding(vertical = 8.dp),
                                        verticalAlignment = Alignment.CenterVertically
                                    ) {
                                        RadioButton(
                                            selected = currentLanguage == lang,
                                            onClick = {
                                                currentLanguage = lang
                                                showLanguageModal = false
                                            }
                                        )
                                        Spacer(modifier = Modifier.width(8.dp))
                                        Text(
                                            text = "${lang.nativeName} (${lang.englishName})",
                                            fontWeight = if (currentLanguage == lang) FontWeight.Bold else FontWeight.Normal
                                        )
                                    }
                                }
                            }
                        },
                        confirmButton = {
                            OutlinedButton(onClick = { showLanguageModal = false }) {
                                Text(strings.common.closeAction)
                            }
                        }
                    )
                }
            }
        }
    }
}
