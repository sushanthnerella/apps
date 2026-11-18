package com.purohit.devotee.ui.webrtc
import androidx.compose.foundation.layout.*
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.purohit.devotee.LocalAppStrings
@Composable
fun EPoojaViewerScreen() {
    Column(Modifier.fillMaxSize().padding(24.dp)) {
        Text(LocalAppStrings.current.common.videoUnavailable)
    }
}
