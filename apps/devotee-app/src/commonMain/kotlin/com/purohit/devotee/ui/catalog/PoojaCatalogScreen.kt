package com.purohit.devotee.ui.catalog

import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyRow
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.purohit.contracts.i18n.AppLanguage
import com.purohit.contracts.i18n.LocalizationManager
import com.purohit.contracts.i18n.StringContract
import com.purohit.devotee.network.PoojaCatalogItem
import com.purohit.devotee.network.PurohitApiClient
import com.purohit.devotee.ui.resilience.ErrorScreen
import com.purohit.devotee.ui.resilience.NoInternetScreen
import com.purohit.devotee.ui.resilience.VedicLoadingScreen
import kotlinx.coroutines.launch

// Discovery matches canonical service names, while the user sees translated titles.
internal fun discoveryServiceNames(items: Collection<PoojaCatalogItem>): String =
    items.joinToString(" + ") { it.title_en.ifBlank { it.title } }

@Composable
fun PoojaCatalogScreen(
    currentLanguage: AppLanguage = AppLanguage.ENGLISH,
    strings: StringContract = LocalizationManager.getStrings(AppLanguage.ENGLISH),
    onSelectPoojaForDispatch: (poojaTitle: String, poojaId: String, totalAmount: Double, discoveryNames: String) -> Unit
) {
    var searchQuery by remember { mutableStateOf("") }
    var selectedCategory by remember { mutableStateOf(strings.catalog.categoryAll) }
    var poojaList by remember { mutableStateOf<List<PoojaCatalogItem>>(emptyList()) }
    var isLoading by remember { mutableStateOf(false) }
    var catalogError by remember { mutableStateOf<String?>(null) }

    val isTelugu = currentLanguage == AppLanguage.TELUGU
    val isHindi = currentLanguage == AppLanguage.HINDI

    // Multi-service selection states
    val selectedPoojas = remember { mutableStateMapOf<String, PoojaCatalogItem>() }
    val samagriSelections = remember { mutableStateMapOf<String, Boolean>() }
    var viewingItemsPooja by remember { mutableStateOf<PoojaCatalogItem?>(null) }

    val coroutineScope = rememberCoroutineScope()

    val categories = remember(currentLanguage) {
        if (isTelugu) {
            listOf("అన్నీ", "శుభకార్యాలు", "సౌభాగ్యం", "విఘ్న నివారణ", "సంస్కారాలు", "ఆరోగ్యం")
        } else if (isHindi) {
            listOf("सभी", "मांगलिक", "समृद्धि", "विघ्नहर्ता", "संस्कार", "स्वास्थ्य")
        } else {
            listOf("All", "Major Occasion", "Prosperity", "Obstacle Removal", "Sanskaras", "Health")
        }
    }

    fun loadCatalog() {
        isLoading = true
        coroutineScope.launch {
            val backendCategory = if (selectedCategory == "అన్నీ" || selectedCategory == "सभी" || selectedCategory == "All") "" else selectedCategory
            val res = PurohitApiClient.fetchCatalog(search = searchQuery, category = backendCategory, lang = currentLanguage.isoCode)
            isLoading = false
            res.onSuccess { items ->
                poojaList = items
                catalogError = null
            }.onFailure { err ->
                catalogError = err.message
                poojaList = emptyList()
            }
        }
    }

    LaunchedEffect(selectedCategory, currentLanguage) {
        loadCatalog()
    }

    // Computations for combined totals
    val totalBasePrice = selectedPoojas.values.sumOf { it.base_price }
    val totalSamagriPrice = selectedPoojas.values.sumOf { pkg ->
        if (samagriSelections[pkg.id] != false) pkg.samagri_kit_price else 0.0
    }
    val combinedDurationMinutes = selectedPoojas.values.sumOf { it.duration_minutes }
    val totalEscrowAmount = totalBasePrice + totalSamagriPrice

    val compactSelection = LocalDensity.current.fontScale >= 1.6f && selectedPoojas.isNotEmpty()
    Column(modifier = Modifier.fillMaxSize().padding(16.dp)) {
        // Leave room for the booking action at large system text sizes. Clearing
        // the selection restores search and category controls.
        if (!compactSelection) {
        // Header
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.SpaceBetween,
            verticalAlignment = Alignment.CenterVertically
        ) {
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = "🪔 " + strings.catalog.catalogTitle,
                    style = MaterialTheme.typography.headlineMedium,
                    fontWeight = FontWeight.Bold,
                    color = MaterialTheme.colorScheme.primary
                )
                Text(
                    text = if (isTelugu) "ఒకటి లేదా అంతకంటే ఎక్కువ పూజలను కలిపి సంపూర్ణ క్రతువుగా ఎంచుకోండి" else if (isHindi) "एक या अधिक पूजा चुनकर अपना अनुष्ठान बंडल बनाएं" else "Select one or multiple pujas to bundle your ceremony",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
            }

            if (selectedPoojas.isNotEmpty()) {
                Surface(
                    shape = CircleShape,
                    color = MaterialTheme.colorScheme.primary
                ) {
                    Text(
                        text = "${selectedPoojas.size}",
                        color = Color.White,
                        fontWeight = FontWeight.Bold,
                        style = MaterialTheme.typography.titleSmall,
                        modifier = Modifier.padding(horizontal = 12.dp, vertical = 6.dp)
                    )
                }
            }
        }

        Spacer(modifier = Modifier.height(12.dp))

        // 1. Search Bar
        OutlinedTextField(
            value = searchQuery,
            onValueChange = {
                searchQuery = it
                loadCatalog()
            },
            placeholder = { Text("🔍 " + strings.catalog.searchPlaceholder) },
            modifier = Modifier.fillMaxWidth(),
            shape = RoundedCornerShape(24.dp),
            singleLine = true
        )

        Spacer(modifier = Modifier.height(10.dp))

        // 2. Category Filter Chips
        LazyRow(
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            modifier = Modifier.fillMaxWidth()
        ) {
            items(categories) { cat ->
                FilterChip(
                    selected = selectedCategory == cat,
                    onClick = { selectedCategory = cat },
                    label = { Text(cat) }
                )
            }
        }

        Spacer(modifier = Modifier.height(12.dp))

        }

        // 3. Grid View (2 Columns)
        if (isLoading && poojaList.isEmpty()) {
            VedicLoadingScreen(
                title = strings.catalog.catalogTitle,
                subtitle = "Fetching consecrated ritual packages & Dakshina...",
                modifier = Modifier.weight(1f)
            )
        } else if (poojaList.isEmpty() && catalogError != null) {
            if (catalogError!!.contains("connect", ignoreCase = true) || catalogError!!.contains("network", ignoreCase = true) || catalogError!!.contains("timeout", ignoreCase = true)) {
                NoInternetScreen(
                    strings = strings,
                    onRetry = { loadCatalog() },
                    serverUrl = PurohitApiClient.baseUrl,
                    modifier = Modifier.weight(1f)
                )
            } else {
                ErrorScreen(
                    errorMessage = catalogError!!,
                    strings = strings,
                    onRetry = { loadCatalog() },
                    modifier = Modifier.weight(1f)
                )
            }
        } else if (poojaList.isEmpty()) {
            Box(modifier = Modifier.weight(1f).fillMaxWidth(), contentAlignment = Alignment.Center) {
                Text(if (isTelugu) "పూజలు కనబడలేదు" else if (isHindi) "कोई पूजा नहीं मिली" else "No poojas found", color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
        } else {
            LazyVerticalGrid(
                columns = GridCells.Adaptive(minSize = 170.dp),
                modifier = Modifier.weight(1f),
                horizontalArrangement = Arrangement.spacedBy(10.dp),
                verticalArrangement = Arrangement.spacedBy(10.dp)
            ) {
                items(poojaList) { pkg ->
                    val isSelected = selectedPoojas.containsKey(pkg.id)
                    val includeSamagri = samagriSelections[pkg.id] != false

                    Card(
                        modifier = Modifier
                            .fillMaxWidth()
                            .clickable {
                                if (isSelected) {
                                    selectedPoojas.remove(pkg.id)
                                    samagriSelections.remove(pkg.id)
                                } else {
                                    selectedPoojas[pkg.id] = pkg
                                    samagriSelections[pkg.id] = true
                                }
                            }
                            .then(
                                if (isSelected) {
                                    Modifier.border(
                                        width = 2.dp,
                                        color = MaterialTheme.colorScheme.primary,
                                        shape = RoundedCornerShape(14.dp)
                                    )
                                } else Modifier
                            ),
                        shape = RoundedCornerShape(14.dp),
                        colors = CardDefaults.cardColors(
                            containerColor = if (isSelected) MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.5f) else MaterialTheme.colorScheme.surfaceVariant
                        ),
                        elevation = CardDefaults.cardElevation(defaultElevation = if (isSelected) 4.dp else 1.dp)
                    ) {
                        Column(modifier = Modifier.padding(12.dp)) {
                            // Icon and Duration Pill
                            Row(
                                modifier = Modifier.fillMaxWidth(),
                                horizontalArrangement = Arrangement.SpaceBetween,
                                verticalAlignment = Alignment.CenterVertically
                            ) {
                                Text(pkg.icon, style = MaterialTheme.typography.headlineSmall)
                                Surface(
                                    color = if (isSelected) MaterialTheme.colorScheme.primary else Color.White.copy(alpha = 0.85f),
                                    shape = RoundedCornerShape(8.dp)
                                ) {
                                    Text(
                                        text = "${pkg.duration_minutes}m",
                                        modifier = Modifier.padding(horizontal = 6.dp, vertical = 2.dp),
                                        style = MaterialTheme.typography.labelSmall,
                                        color = if (isSelected) Color.White else Color.Black,
                                        fontWeight = FontWeight.Bold
                                    )
                                }
                            }

                            Spacer(modifier = Modifier.height(8.dp))

                            Text(
                                text = pkg.title,
                                fontWeight = FontWeight.Bold,
                                style = MaterialTheme.typography.titleSmall,
                                maxLines = 2,
                                overflow = TextOverflow.Ellipsis
                            )

                            Spacer(modifier = Modifier.height(4.dp))

                            Text(
                                text = pkg.category,
                                style = MaterialTheme.typography.labelSmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant
                            )

                            Spacer(modifier = Modifier.height(6.dp))

                            // Base Price
                            Text(
                                text = "₹${pkg.base_price.toInt()}",
                                fontWeight = FontWeight.Bold,
                                style = MaterialTheme.typography.titleMedium,
                                color = MaterialTheme.colorScheme.primary
                            )

                            // Samagri Kit Checkbox (if selected)
                            if (isSelected) {
                                Spacer(modifier = Modifier.height(6.dp))
                                Row(
                                    modifier = Modifier.fillMaxWidth(),
                                    verticalAlignment = Alignment.CenterVertically
                                ) {
                                    Checkbox(
                                        checked = includeSamagri,
                                        onCheckedChange = { checked ->
                                            samagriSelections[pkg.id] = checked
                                        },
                                        modifier = Modifier.size(48.dp)
                                    )
                                    Spacer(modifier = Modifier.width(4.dp))
                                    Text(
                                        text = "+${if (isTelugu) "సామాగ్రి" else if (isHindi) "सामग्री" else "Samagri"} (₹${pkg.samagri_kit_price.toInt()})",
                                        style = MaterialTheme.typography.labelSmall,
                                        fontWeight = FontWeight.Medium
                                    )
                                }
                            }

                            Spacer(modifier = Modifier.height(6.dp))

                            // View Items (Household & Samagri) Button
                            OutlinedButton(
                                onClick = { viewingItemsPooja = pkg },
                                modifier = Modifier.fillMaxWidth().heightIn(min = 48.dp),
                                contentPadding = PaddingValues(0.dp),
                                shape = RoundedCornerShape(8.dp)
                            ) {
                                Text(
                                    "👁️ " + if (isTelugu) "సామాగ్రి జాబితా" else if (isHindi) "सामग्री सूची" else "View Items List",
                                    style = MaterialTheme.typography.labelSmall
                                )
                            }

                            Spacer(modifier = Modifier.height(6.dp))

                            // Add / Added status button
                            Surface(
                                shape = RoundedCornerShape(8.dp),
                                color = if (isSelected) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.surface,
                                modifier = Modifier.fillMaxWidth()
                            ) {
                                Text(
                                    text = if (isSelected) {
                                        if (isTelugu) "✓ క్రతువుకు చేర్చబడింది" else if (isHindi) "✓ पूजा में शामिल" else "✓ Added to Puja Vidhi"
                                    } else {
                                        if (isTelugu) "+ సేవను చేర్చండి" else if (isHindi) "+ पूजा जोड़ें" else "+ Add Service"
                                    },
                                    modifier = Modifier.padding(vertical = 4.dp),
                                    style = MaterialTheme.typography.labelSmall,
                                    color = if (isSelected) Color.White else MaterialTheme.colorScheme.primary,
                                    fontWeight = FontWeight.Bold,
                                    textAlign = androidx.compose.ui.text.style.TextAlign.Center
                                )
                            }
                        }
                    }
                }
            }
        }

        // Multi-Service Selection Action Bar
        if (selectedPoojas.isNotEmpty()) {
            Spacer(modifier = Modifier.height(8.dp))
            Surface(
                modifier = Modifier.fillMaxWidth(),
                tonalElevation = 8.dp,
                shape = RoundedCornerShape(16.dp),
                color = MaterialTheme.colorScheme.surface,
                shadowElevation = 6.dp
            ) {
                Column(modifier = Modifier.padding(14.dp)) {
                    // Header with count & Clear All
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Text(
                            text = if (isTelugu) "⚡ సంకల్ప బండిల్: ${selectedPoojas.size} పూజలు ఎంపికయ్యాయి" else if (isHindi) "⚡ अनुष्ठान बंडल: ${selectedPoojas.size} पूजा चयनित" else "⚡ Bundle: ${selectedPoojas.size} Pooja(s) Selected",
                            fontWeight = FontWeight.Bold,
                            style = MaterialTheme.typography.titleSmall,
                            color = MaterialTheme.colorScheme.primary,
                            modifier = Modifier.weight(1f)
                        )
                        TextButton(
                            onClick = {
                                selectedPoojas.clear()
                                samagriSelections.clear()
                            },
                            contentPadding = PaddingValues(0.dp)
                        ) {
                            Text(if (isTelugu) "అన్నీ తీసివేయి" else if (isHindi) "सभी हटाएं" else "Clear All", style = MaterialTheme.typography.labelSmall)
                        }
                    }

                    // Combined breakdown
                    Column(
                        modifier = Modifier.fillMaxWidth(),
                        verticalArrangement = Arrangement.spacedBy(8.dp)
                    ) {
                        Column {
                            Text(
                                text = "${strings.catalog.durationLabel}: ~${combinedDurationMinutes} ${strings.catalog.durationMinutes} | ${if (isTelugu) "సామాగ్రి" else if (isHindi) "सामग्री" else "Samagri"}: ₹${totalSamagriPrice.toInt()}",
                                style = MaterialTheme.typography.labelSmall,
                                color = MaterialTheme.colorScheme.onSurfaceVariant
                            )
                            Text(
                                text = "${if (isTelugu) "అంచనా మొత్తం" else if (isHindi) "अनुमानित राशि" else "Estimated total"}: ₹${totalEscrowAmount.toInt()}",
                                fontWeight = FontWeight.Bold,
                                style = MaterialTheme.typography.titleLarge,
                                color = MaterialTheme.colorScheme.primary
                            )
                        }

                        Button(
                            onClick = {
                                val combinedTitle = selectedPoojas.values.joinToString(" + ") { it.title }
                                val combinedIds = selectedPoojas.values.joinToString(",") { it.id }
                                onSelectPoojaForDispatch(combinedTitle, combinedIds, totalEscrowAmount, discoveryServiceNames(selectedPoojas.values))
                            },
                            modifier = Modifier.fillMaxWidth().heightIn(min = 48.dp),
                            colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.primary),
                            shape = RoundedCornerShape(10.dp)
                        ) {
                            Text("📍 " + strings.catalog.instantDispatchButton)
                        }
                    }
                }
            }
        }

        // Required Items (Household & Samagri) Viewer Modal
        viewingItemsPooja?.let { pkg ->
            RequiredItemsModal(
                poojaId = pkg.id,
                poojaTitle = pkg.title,
                onDismiss = { viewingItemsPooja = null }
            )
        }
    }
}
