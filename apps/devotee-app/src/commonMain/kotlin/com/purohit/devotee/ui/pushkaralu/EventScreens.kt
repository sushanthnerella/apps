package com.purohit.devotee.ui.pushkaralu

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
import androidx.compose.ui.semantics.heading
import androidx.compose.ui.semantics.semantics
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.purohit.contracts.event.EventConfig
import com.purohit.contracts.event.EventStatus
import com.purohit.contracts.event.Ghat
import com.purohit.contracts.i18n.AppLanguage
import com.purohit.contracts.i18n.models.EventStrings
import com.purohit.devotee.location.DeviceCoordinates
import com.purohit.devotee.location.rememberLocationAccess
import kotlinx.datetime.Clock
import kotlinx.datetime.Instant
import kotlin.math.*

/** "Day X of Y" for a live event, or null when dates cannot be read. Day is clamped into the event window. */
internal fun eventDay(event: EventConfig, now: Instant = Clock.System.now()): Pair<Int, Int>? = try {
    val start = Instant.parse(event.startsAt)
    val end = Instant.parse(event.endsAt)
    val total = ceil((end - start).inWholeMinutes / 1440.0).toInt().coerceAtLeast(1)
    val day = ((now - start).inWholeDays + 1).toInt().coerceIn(1, total)
    day to total
} catch (e: Exception) {
    null
}

internal fun distanceKm(a: DeviceCoordinates, lat: Double, lng: Double): Double {
    val r = 6371.0
    val dLat = (lat - a.latitude) * PI / 180
    val dLng = (lng - a.longitude) * PI / 180
    val h = sin(dLat / 2).pow(2) + cos(a.latitude * PI / 180) * cos(lat * PI / 180) * sin(dLng / 2).pow(2)
    return 2 * r * asin(sqrt(h))
}

/** Ghats filtered by query (any language) and sorted by distance when a location is known, else by CMS order. */
internal fun visibleGhats(ghats: List<Ghat>, query: String, here: DeviceCoordinates?): List<Pair<Ghat, Double?>> {
    val q = query.trim()
    return ghats.filter { it.isActive }
        .filter { q.isEmpty() || listOf(it.name, it.nameTe, it.nameHi).any { n -> n.contains(q, ignoreCase = true) } }
        .map { g -> g to if (here != null && g.latitude != null && g.longitude != null) distanceKm(here, g.latitude!!, g.longitude!!) else null }
        .sortedWith(compareBy<Pair<Ghat, Double?>> { it.second ?: Double.MAX_VALUE }.thenBy { it.first.sortOrder }.thenBy { it.first.name })
}

@Composable
private fun StatusBadge(event: EventConfig, strings: EventStrings) {
    val label = if (event.status == EventStatus.PAUSED) strings.pausedBadge else eventDay(event)?.let { (d, t) ->
        strings.liveDayTemplate.replace("{day}", d.toString()).replace("{total}", t.toString())
    } ?: return
    Surface(color = MaterialTheme.colorScheme.secondaryContainer, contentColor = MaterialTheme.colorScheme.onSecondaryContainer, shape = RoundedCornerShape(50)) {
        Text(label, style = MaterialTheme.typography.labelMedium, fontWeight = FontWeight.Bold, modifier = Modifier.padding(horizontal = 10.dp, vertical = 4.dp))
    }
}

/** Compact strip above the catalog. Kept to one card so large text sizes do not push catalog actions off-screen. */
@Composable
fun EventStrip(event: EventConfig, language: AppLanguage, strings: EventStrings, onOpen: () -> Unit) {
    Surface(
        color = MaterialTheme.colorScheme.primary,
        contentColor = MaterialTheme.colorScheme.onPrimary,
        shape = RoundedCornerShape(14.dp),
        modifier = Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 8.dp).clickable(onClick = onOpen)
    ) {
        Row(Modifier.padding(12.dp).heightIn(min = 48.dp), verticalAlignment = Alignment.CenterVertically, horizontalArrangement = Arrangement.spacedBy(12.dp)) {
            Column(Modifier.weight(1f), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                Text(event.displayName(language), style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
                StatusBadge(event, strings)
            }
            Text(if (event.acceptsBookings) strings.bookPriestAtGhat else strings.tabEvent, style = MaterialTheme.typography.labelLarge, fontWeight = FontWeight.Bold)
        }
    }
}

/** Pushkaralu tab: event hero and the ghat list from the CMS. */
@OptIn(ExperimentalLayoutApi::class)
@Composable
fun PushkaraluScreen(event: EventConfig?, language: AppLanguage, strings: EventStrings, onOpenGhat: (Ghat) -> Unit) {
    if (event == null || !event.isLive) {
        Box(Modifier.fillMaxSize().padding(24.dp), contentAlignment = Alignment.Center) {
            Text(strings.eventEndedNotice, style = MaterialTheme.typography.bodyLarge)
        }
        return
    }
    var query by remember { mutableStateOf("") }
    var here by remember { mutableStateOf<DeviceCoordinates?>(null) }
    val location = rememberLocationAccess()
    val ghats = remember(event.ghats, query, here) { visibleGhats(event.ghats, query, here) }
    var counts by remember { mutableStateOf<Map<String, Int>>(emptyMap()) }
    LaunchedEffect(event.id) {
        while (true) {
            com.purohit.devotee.network.PurohitApiClient.fetchGhatAvailability(event.id)
                .onSuccess { list -> counts = list.associate { it.ghatId to it.availableCount } }
            kotlinx.coroutines.delay(30_000)
        }
    }

    LazyColumn(Modifier.fillMaxSize(), contentPadding = PaddingValues(16.dp), verticalArrangement = Arrangement.spacedBy(12.dp)) {
        item {
            Surface(color = MaterialTheme.colorScheme.primary, contentColor = MaterialTheme.colorScheme.onPrimary, shape = RoundedCornerShape(16.dp)) {
                Column(Modifier.fillMaxWidth().padding(18.dp), verticalArrangement = Arrangement.spacedBy(8.dp)) {
                    StatusBadge(event, strings)
                    Text(event.displayName(language), style = MaterialTheme.typography.headlineSmall, modifier = Modifier.semantics { heading() })
                    if (language != AppLanguage.TELUGU && event.nameTe.isNotBlank()) Text(event.nameTe, style = MaterialTheme.typography.bodyLarge)
                    Text(strings.bannerDescription, style = MaterialTheme.typography.bodyMedium)
                    if (!event.acceptsBookings) Text(strings.bookingsPaused, style = MaterialTheme.typography.bodyMedium, fontWeight = FontWeight.Bold)
                }
            }
        }
        item {
            Text(strings.chooseGhatTitle, style = MaterialTheme.typography.titleLarge, color = MaterialTheme.colorScheme.primary, modifier = Modifier.semantics { heading() })
        }
        item {
            OutlinedTextField(
                value = query, onValueChange = { query = it }, singleLine = true,
                label = { Text(strings.searchGhatPlaceholder) },
                shape = RoundedCornerShape(24.dp), modifier = Modifier.fillMaxWidth()
            )
        }
        item {
            FlowRow(horizontalArrangement = Arrangement.spacedBy(8.dp), verticalArrangement = Arrangement.spacedBy(4.dp)) {
                FilterChip(
                    selected = here != null,
                    onClick = { if (here != null) here = null else location.request { r -> r.onSuccess { here = it } } },
                    label = { Text(if (here != null) strings.sortByName else strings.sortByNearest) },
                    modifier = Modifier.heightIn(min = 48.dp)
                )
                Text(strings.locationOptional, style = MaterialTheme.typography.bodySmall, color = MaterialTheme.colorScheme.onSurfaceVariant, modifier = Modifier.align(Alignment.CenterVertically))
            }
        }
        if (ghats.isEmpty()) {
            item {
                Text(if (event.ghats.none { it.isActive }) strings.noGhatsConfigured else strings.noGhatsMatch, color = MaterialTheme.colorScheme.onSurfaceVariant)
            }
        }
        items(ghats, key = { it.first.id }) { (ghat, km) ->
            Card(onClick = { onOpenGhat(ghat) }, colors = CardDefaults.cardColors(containerColor = Color.White), shape = RoundedCornerShape(14.dp), modifier = Modifier.fillMaxWidth()) {
                Column(Modifier.padding(14.dp).heightIn(min = 48.dp), verticalArrangement = Arrangement.spacedBy(2.dp)) {
                    Text(ghat.displayName(language), style = MaterialTheme.typography.titleMedium, fontWeight = FontWeight.Bold)
                    Text(strings.availableNowTemplate.replace("{count}", (counts[ghat.id] ?: 0).toString()),
                        style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.primary, fontWeight = FontWeight.SemiBold)
                    val secondary = listOfNotNull(
                        ghat.nameTe.takeIf { language != AppLanguage.TELUGU && it.isNotBlank() },
                        km?.let { "${(it * 10).roundToInt() / 10.0} km" }
                    ).joinToString(" · ")
                    if (secondary.isNotEmpty()) Text(secondary, style = MaterialTheme.typography.bodyMedium, color = MaterialTheme.colorScheme.onSurfaceVariant)
                }
            }
        }
    }
}
