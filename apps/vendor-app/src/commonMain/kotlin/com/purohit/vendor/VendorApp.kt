package com.purohit.vendor
import androidx.compose.foundation.layout.*
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.purohit.contracts.i18n.AppLanguage
import com.purohit.contracts.i18n.LocalizationManager
import com.purohit.contracts.i18n.StringContract
val LocalVendorStrings = compositionLocalOf<StringContract> { LocalizationManager.getStrings(AppLanguage.ENGLISH) }
@Composable
fun VendorAppRoot() {
    var language by remember { mutableStateOf(AppLanguage.TELUGU) }
    val strings = LocalizationManager.getStrings(language)
    CompositionLocalProvider(LocalVendorStrings provides strings) {
        MaterialTheme {
            Column(Modifier.fillMaxSize().padding(24.dp)) {
                Text(strings.common.tabPujaStore, style = MaterialTheme.typography.headlineMedium)
                Text(strings.common.featureUnavailable)
                AppLanguage.entries.forEach { candidate ->
                    TextButton(onClick = { language = candidate }) { Text(candidate.nativeName) }
                }
            }
        }
    }
}
