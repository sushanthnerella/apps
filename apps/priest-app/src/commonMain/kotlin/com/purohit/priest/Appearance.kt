package com.purohit.priest

import androidx.compose.foundation.selection.selectable
import androidx.compose.foundation.selection.selectableGroup
import androidx.compose.ui.Alignment
import androidx.compose.ui.semantics.Role
import androidx.compose.foundation.verticalScroll
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import com.purohit.contracts.i18n.AppLanguage

internal enum class Appearance { ROYAL, CALM, EARTH }
internal fun appearanceFromStoredValue(value: String?): Appearance = Appearance.entries.firstOrNull { it.name == value } ?: Appearance.ROYAL
internal expect fun motionAllowed(): Boolean
internal expect fun readAppearance(): String?
internal expect fun writeAppearance(value: String)
internal expect fun readPref(key: String): String?
internal expect fun writePref(key: String, value: String)

internal fun appearanceScheme(value: Appearance): ColorScheme = when (value) {
    Appearance.ROYAL -> lightColorScheme(primary = Color(0xFF74162C), onPrimary = Color.White, primaryContainer = Color(0xFFF6DCE0), onPrimaryContainer = Color(0xFF4B081A), secondary = Color(0xFF805500), secondaryContainer = Color(0xFFFFE4A3), onSecondaryContainer = Color(0xFF3D2800), background = Color(0xFFFFF8EF), surface = Color(0xFFFFF8EF), surfaceVariant = Color(0xFFF3E5D2), onSurfaceVariant = Color(0xFF50443B))
    Appearance.CALM -> lightColorScheme(primary = Color(0xFF005451), onPrimary = Color.White, primaryContainer = Color(0xFFC8EAE2), onPrimaryContainer = Color(0xFF003B39), secondary = Color(0xFF9A4900), secondaryContainer = Color(0xFFFFDCC0), onSecondaryContainer = Color(0xFF3C1800), background = Color(0xFFF8FAF7), surface = Color(0xFFF8FAF7), surfaceVariant = Color(0xFFDCEAE5), onSurfaceVariant = Color(0xFF384B48))
    Appearance.EARTH -> lightColorScheme(primary = Color(0xFF244D2A), onPrimary = Color.White, primaryContainer = Color(0xFFD6EACD), onPrimaryContainer = Color(0xFF102D15), secondary = Color(0xFF964421), secondaryContainer = Color(0xFFFFDBCC), onSecondaryContainer = Color(0xFF3A1204), background = Color(0xFFFFF8ED), surface = Color(0xFFFFF8ED), surfaceVariant = Color(0xFFF0E5D2), onSurfaceVariant = Color(0xFF4C493B))
}

internal fun appTypography(): Typography {
    val defaults = Typography()
    return defaults.copy(
        headlineLarge = defaults.headlineLarge.copy(fontFamily = FontFamily.Serif, fontWeight = FontWeight.Bold),
        headlineMedium = defaults.headlineMedium.copy(fontFamily = FontFamily.Serif, fontWeight = FontWeight.Bold),
        headlineSmall = defaults.headlineSmall.copy(fontFamily = FontFamily.Serif, fontWeight = FontWeight.Bold),
        titleLarge = defaults.titleLarge.copy(fontFamily = FontFamily.Serif, fontWeight = FontWeight.SemiBold)
    )
}

@Composable
internal fun AppearanceDialog(current: Appearance, language: AppLanguage, onDismiss: () -> Unit, onSelect: (Appearance) -> Unit) {
    val title = when (language) { AppLanguage.TELUGU -> "రూపం"; AppLanguage.HINDI -> "रूप-रंग"; else -> "Appearance" }
    val close = when (language) { AppLanguage.TELUGU -> "మూసివేయి"; AppLanguage.HINDI -> "बंद करें"; else -> "Close" }
    AlertDialog(onDismissRequest = onDismiss, title = { Text(title) }, text = {
        Column(modifier = Modifier.heightIn(max = 420.dp).verticalScroll(rememberScrollState()).selectableGroup(), verticalArrangement = Arrangement.spacedBy(8.dp)) {
            Appearance.entries.forEach { option ->
                val scheme = appearanceScheme(option)
                val label = when (language) {
                    AppLanguage.TELUGU -> when (option) { Appearance.ROYAL -> "రాజస వారసత్వం"; Appearance.CALM -> "ప్రశాంత ఆధునికం"; Appearance.EARTH -> "ఆలయ భూమి" }
                    AppLanguage.HINDI -> when (option) { Appearance.ROYAL -> "शाही विरासत"; Appearance.CALM -> "शांत आधुनिक"; Appearance.EARTH -> "मंदिर धरती" }
                    else -> when (option) { Appearance.ROYAL -> "Royal Heritage"; Appearance.CALM -> "Calm Modern"; Appearance.EARTH -> "Temple Earth" }
                }
                Surface(shape = RoundedCornerShape(12.dp), color = scheme.surfaceVariant, modifier = Modifier.fillMaxWidth().selectable(selected = current == option, role = Role.RadioButton, onClick = { onSelect(option) })) {
                    Row(Modifier.padding(12.dp), horizontalArrangement = Arrangement.spacedBy(12.dp), verticalAlignment = Alignment.CenterVertically) {
                        Column(verticalArrangement = Arrangement.spacedBy(3.dp)) {
                            Surface(color = scheme.primary, shape = RoundedCornerShape(4.dp), modifier = Modifier.size(width = 36.dp, height = 14.dp)) {}
                            Row(horizontalArrangement = Arrangement.spacedBy(3.dp)) {
                                Surface(color = scheme.secondary, shape = RoundedCornerShape(4.dp), modifier = Modifier.size(width = 16.dp, height = 14.dp)) {}
                                Surface(color = scheme.background, shape = RoundedCornerShape(4.dp), modifier = Modifier.size(width = 17.dp, height = 14.dp)) {}
                            }
                        }
                        Text(label, color = scheme.onSurfaceVariant, modifier = Modifier.weight(1f))
                        RadioButton(selected = current == option, onClick = null)
                    }
                }
            }
        }
    }, confirmButton = { TextButton(onClick = onDismiss) { Text(close) } })
}
