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
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import com.purohit.contracts.i18n.AppLanguage
import com.purohit.contracts.i18n.LocalizationManager
import com.purohit.contracts.i18n.StringContract
import com.purohit.devotee.network.PoojaCatalogItem
import com.purohit.devotee.network.PurohitApiClient
import kotlinx.coroutines.launch

@Composable
fun PoojaCatalogScreen(
    currentLanguage: AppLanguage = AppLanguage.ENGLISH,
    strings: StringContract = LocalizationManager.getStrings(AppLanguage.ENGLISH),
    onSelectPoojaForDispatch: (poojaTitle: String, poojaId: String, totalAmount: Double) -> Unit
) {
    var searchQuery by remember { mutableStateOf("") }
    var selectedCategory by remember { mutableStateOf(strings.catalog.categoryAll) }
    var poojaList by remember { mutableStateOf<List<PoojaCatalogItem>>(emptyList()) }
    var isLoading by remember { mutableStateOf(false) }

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
            }.onFailure {
                poojaList = if (isTelugu) {
                    listOf(
                        PoojaCatalogItem("1", "గృహప్రవేశం (నూతన గృహ ప్రవేశ మహోత్సవం)", category = "శుభకార్యాలు", base_price = 5116.0, samagri_kit_price = 2500.0, duration_minutes = 180, is_online_eligible = false, icon = "🏡", description = "వాస్తు శాంతి, నవగ్రహ మరియు గణపతి హోమములతో కూడిన క్రతువు"),
                        PoojaCatalogItem("2", "శ్రీ సత్యనారాయణ స్వామి వ్రతం", category = "సౌభాగ్యం", base_price = 2516.0, samagri_kit_price = 1200.0, duration_minutes = 120, is_online_eligible = true, icon = "✨", description = "పంచాధ్యాయ వ్రత కథ, సత్యదేవుని పూజ"),
                        PoojaCatalogItem("3", "మహా గణపతి హోమం / యాగానుష్ఠానం", category = "విఘ్న నివారణ", base_price = 3116.0, samagri_kit_price = 1800.0, duration_minutes = 90, is_online_eligible = true, icon = "🕉️", description = "సర్వ విఘ్న నివారణకు చేయు పవిత్ర హోమము"),
                        PoojaCatalogItem("4", "నామకరణం & ఆయుష్య హోమం", category = "సంస్కారాలు", base_price = 2116.0, samagri_kit_price = 1100.0, duration_minutes = 90, is_online_eligible = true, icon = "👶", description = "శిశువునకు జన్మనక్షత్ర నామకరణం & దీర్ఘాయుష్షు"),
                        PoojaCatalogItem("5", "మహా మృత్యుంజయ హోమం / ఆయుర్వృద్ధి శాంతి", category = "ఆరోగ్యం", base_price = 4116.0, samagri_kit_price = 2100.0, duration_minutes = 120, is_online_eligible = true, icon = "🔱", description = "సంపూర్ణ ఆరోగ్యానికి చేయు మహత్తర హోమము"),
                        PoojaCatalogItem("6", "నవగ్రహ శాంతి హోమం & దోష నివారణ", category = "దోష నివారణలు", base_price = 3516.0, samagri_kit_price = 1600.0, duration_minutes = 100, is_online_eligible = true, icon = "☀️", description = "నవగ్రహ దేవతా అనుగ్రహం & జాతక దోష నివారణ")
                    )
                } else if (isHindi) {
                    listOf(
                        PoojaCatalogItem("1", "गृहप्रवेश पूजा एवं वास्तु शांति हवन", category = "मांगलिक", base_price = 5116.0, samagri_kit_price = 2500.0, duration_minutes = 180, is_online_eligible = false, icon = "🏡", description = "वास्तु शांति एवं नवग्रह हवन सहित गृहप्रवेश पूजन"),
                        PoojaCatalogItem("2", "श्री सत्यनारायण स्वामी व्रत कथा", category = "समृद्धि", base_price = 2516.0, samagri_kit_price = 1200.0, duration_minutes = 120, is_online_eligible = true, icon = "✨", description = "पंच-अध्याय कथा एवं सपाद महाप्रसाद भोग"),
                        PoojaCatalogItem("3", "महा गणपति हवन / विघ्नहर्ता अनुष्ठान", category = "विघ्नहर्ता", base_price = 3116.0, samagri_kit_price = 1800.0, duration_minutes = 90, is_online_eligible = true, icon = "🕉️", description = "सर्व विघ्न बाधा निवारण हेतु वैदिक महाहवन"),
                        PoojaCatalogItem("4", "नामकरण संस्कार एवं आयुष्य हवन", category = "संस्कार", base_price = 2116.0, samagri_kit_price = 1100.0, duration_minutes = 90, is_online_eligible = true, icon = "👶", description = "शिशु का नामकरण एवं आयुष्य होम"),
                        PoojaCatalogItem("5", "महा मृत्युंजय हवन / आरोग्य शांति", category = "स्वास्थ्य", base_price = 4116.0, samagri_kit_price = 2100.0, duration_minutes = 120, is_online_eligible = true, icon = "🔱", description = "आरोग्य लाभ एवं अकाल मृत्यु निवारक महाहवन"),
                        PoojaCatalogItem("6", "नवग्रह शांति हवन एवं दोष निवारण", category = "दोष निवारण", base_price = 3516.0, samagri_kit_price = 1600.0, duration_minutes = 100, is_online_eligible = true, icon = "☀️", description = "नवग्रहों की प्रसन्नता एवं ग्रह शांति हवन")
                    )
                } else {
                    listOf(
                        PoojaCatalogItem("1", "Gruhapravesam (House Warming)", category = "Major Occasion", base_price = 5116.0, samagri_kit_price = 2500.0, duration_minutes = 180, is_online_eligible = false, icon = "🏡", description = "House warming ritual"),
                        PoojaCatalogItem("2", "Sri Satyanarayana Swamy Vratam", category = "Prosperity", base_price = 2516.0, samagri_kit_price = 1200.0, duration_minutes = 120, is_online_eligible = true, icon = "✨", description = "Prosperity vratam"),
                        PoojaCatalogItem("3", "Maha Ganapathi Homa / Yagam", category = "Obstacle Removal", base_price = 3116.0, samagri_kit_price = 1800.0, duration_minutes = 90, is_online_eligible = true, icon = "🕉️", description = "Obstacle removal yagam"),
                        PoojaCatalogItem("4", "Namakaranam & Ayushya Homa", category = "Sanskaras", base_price = 2116.0, samagri_kit_price = 1100.0, duration_minutes = 90, is_online_eligible = true, icon = "👶", description = "Baby naming ceremony"),
                        PoojaCatalogItem("5", "Mrityunjaya Homa / Shanti", category = "Health", base_price = 4116.0, samagri_kit_price = 2100.0, duration_minutes = 120, is_online_eligible = true, icon = "🔱", description = "Health & healing homa"),
                        PoojaCatalogItem("6", "Navagraha Shanti Homa", category = "Astrological Remedies", base_price = 3516.0, samagri_kit_price = 1600.0, duration_minutes = 100, is_online_eligible = true, icon = "☀️", description = "Planetary dosha shanti")
                    )
                }
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

    Column(modifier = Modifier.fillMaxSize().padding(16.dp)) {
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

        // 3. Grid View (2 Columns)
        if (isLoading) {
            Box(modifier = Modifier.weight(1f).fillMaxWidth(), contentAlignment = Alignment.Center) {
                CircularProgressIndicator()
            }
        } else {
            LazyVerticalGrid(
                columns = GridCells.Fixed(2),
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
                                color = Color.Gray
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
                                        modifier = Modifier.size(24.dp)
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
                                modifier = Modifier.fillMaxWidth().height(30.dp),
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
                            color = MaterialTheme.colorScheme.primary
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
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Column {
                            Text(
                                text = "${strings.catalog.durationLabel}: ~${combinedDurationMinutes} ${strings.catalog.durationMinutes} | ${if (isTelugu) "సామాగ్రి" else if (isHindi) "सामग्री" else "Samagri"}: ₹${totalSamagriPrice.toInt()}",
                                style = MaterialTheme.typography.labelSmall,
                                color = Color.Gray
                            )
                            Text(
                                text = "${if (isTelugu) "మొత్తం ఎస్క్రో దక్షిణ" else if (isHindi) "कुल एस्क्रो दक्षिणा" else "Total Escrow"}: ₹${totalEscrowAmount.toInt()}",
                                fontWeight = FontWeight.Bold,
                                style = MaterialTheme.typography.titleLarge,
                                color = MaterialTheme.colorScheme.primary
                            )
                        }

                        Button(
                            onClick = {
                                val combinedTitle = selectedPoojas.values.joinToString(" + ") { it.title }
                                val combinedIds = selectedPoojas.values.joinToString(",") { it.id }
                                onSelectPoojaForDispatch(combinedTitle, combinedIds, totalEscrowAmount)
                            },
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
