package com.purohit.priest.ui.studio
import androidx.compose.foundation.layout.*
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.purohit.priest.LocalPriestStrings
@Composable
fun BroadcasterStudioScreen() {
    Column(Modifier.fillMaxSize().padding(24.dp)) {
        Text(LocalPriestStrings.current.common.videoUnavailable)
    }
}
