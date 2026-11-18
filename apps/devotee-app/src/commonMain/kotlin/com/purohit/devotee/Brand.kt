@file:OptIn(org.jetbrains.compose.resources.ExperimentalResourceApi::class)

package com.purohit.devotee

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.ui.draw.clip
import androidx.compose.runtime.Composable
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.ui.Alignment
import androidx.compose.ui.platform.LocalDensity
import androidx.compose.ui.text.font.FontWeight
import com.purohit.contracts.i18n.AppLanguage
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.FilterQuality
import androidx.compose.ui.unit.IntOffset
import androidx.compose.ui.unit.IntSize
import androidx.compose.ui.unit.dp
import org.jetbrains.compose.resources.imageResource
import puja_vaahini.apps.`devotee-app`.generated.resources.Res
import puja_vaahini.apps.`devotee-app`.generated.resources.reference_sun

/** Displays the rising sun directly from the supplied reference image. */
@Composable
internal fun SunLogo(modifier: Modifier = Modifier) {
    val reference = imageResource(Res.drawable.reference_sun)
    Canvas(modifier = modifier.size(width = 52.dp, height = 36.dp).clip(RoundedCornerShape(6.dp))) {
        drawImage(reference, srcOffset = IntOffset(521, 0), srcSize = IntSize(102, 70),
            dstSize = IntSize(size.width.toInt(), size.height.toInt()), filterQuality = FilterQuality.High)
    }
}

@Composable
internal fun AppHeader(language: AppLanguage, onAppearance: () -> Unit, onLanguage: () -> Unit) {
    val appearanceLabel = when (language) { AppLanguage.TELUGU -> "రూపం"; AppLanguage.HINDI -> "रूप-रंग"; else -> "Appearance" }
    Surface(color = MaterialTheme.colorScheme.primary, modifier = Modifier.fillMaxWidth()) {
        BoxWithConstraints(Modifier.fillMaxWidth().padding(horizontal = 16.dp, vertical = 4.dp)) {
            val compact = maxWidth < 390.dp || LocalDensity.current.fontScale > 1.1f
            @Composable fun Heading() {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    SunLogo()
                    Spacer(Modifier.width(8.dp))
                    Text("Pooja Vahini", color = MaterialTheme.colorScheme.onPrimary,
                        fontWeight = FontWeight.Bold, style = MaterialTheme.typography.titleMedium, maxLines = 1)
                }
            }
            @Composable fun Actions() {
                Row(verticalAlignment = Alignment.CenterVertically) {
                    TextButton(onClick = onAppearance) { Text(appearanceLabel, color = MaterialTheme.colorScheme.onPrimary) }
                    TextButton(onClick = onLanguage) { Text(language.nativeName, color = MaterialTheme.colorScheme.onPrimary) }
                }
            }
            if (compact) {
                Column { Heading(); Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.End) { Actions() } }
            } else {
                Row(Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween, verticalAlignment = Alignment.CenterVertically) {
                    Heading(); Actions()
                }
            }
        }
    }
}
