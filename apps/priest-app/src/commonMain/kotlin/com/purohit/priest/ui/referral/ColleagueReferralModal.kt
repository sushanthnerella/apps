package com.purohit.priest.ui.referral

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
import com.purohit.contracts.i18n.StringContract
import com.purohit.priest.network.PriestDirectoryItem
import com.purohit.priest.network.PurohitApiClient
import kotlinx.coroutines.launch

data class TrustedColleague(
    val id: String,
    val name: String,
    val sampradaya: String,
    val vedaShakha: String,
    val experienceYears: Int,
    val ratingAvg: Double,
    val isSlotFree: Boolean = true
)

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun ColleagueReferralModal(
    bookingId: String = "f312db8f-1d71-4cab-b5ba-77fbead70c71",
    referringPriestId: String = "6e7d277c-4ee2-469f-96ac-331517389762",
    bookingTitle: String,
    scheduledSlot: String = "Tomorrow (Nov 4) • Abhijit Muhurtham (11:45 AM)",
    strings: StringContract,
    onReferralConfirmed: (colleagueId: String, colleagueName: String, reason: String) -> Unit,
    onDismiss: () -> Unit
) {
    val coroutineScope = rememberCoroutineScope()
    var colleagues by remember { mutableStateOf<List<TrustedColleague>>(emptyList()) }
    var loading by remember { mutableStateOf(true) }
    var directoryError by remember { mutableStateOf<String?>(null) }

    LaunchedEffect(Unit) {
        val res = PurohitApiClient.fetchPriestsList()
        res.onSuccess { dirList ->
            run {
                colleagues = dirList.filter { it.id != referringPriestId }.map { p ->
                    TrustedColleague(
                        id = p.id,
                        name = p.full_name,
                        sampradaya = p.sampradaya,
                        vedaShakha = p.veda_shakha,
                        experienceYears = 0,
                        ratingAvg = p.rating_avg,
                        isSlotFree = false
                    )
                }
            }
        }.onFailure { directoryError = "Could not load colleagues. Please reopen to retry." }
        loading = false
    }

    var selectedColleague by remember { mutableStateOf<TrustedColleague?>(null) }
    LaunchedEffect(colleagues) {
        if (selectedColleague == null && colleagues.isNotEmpty()) {
            selectedColleague = colleagues.first()
        }
    }

    var reasonText by remember { mutableStateOf("") }
    var isSubmitting by remember { mutableStateOf(false) }
    var statusMessage by remember { mutableStateOf<String?>(null) }

    ModalBottomSheet(
        onDismissRequest = onDismiss,
        shape = RoundedCornerShape(topStart = 24.dp, topEnd = 24.dp)
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(20.dp)
        ) {
            Text(
                text = "🤝 Delegate Booking to Colleague Purohit",
                style = MaterialTheme.typography.titleLarge,
                fontWeight = FontWeight.Bold,
                color = MaterialTheme.colorScheme.primary
            )
            Spacer(modifier = Modifier.height(4.dp))
            Text(
                text = "Request reassignment of '$bookingTitle' to a colleague.",
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )

            Spacer(modifier = Modifier.height(8.dp))

            Surface(
                shape = RoundedCornerShape(8.dp),
                color = MaterialTheme.colorScheme.surfaceVariant,
                modifier = Modifier.fillMaxWidth()
            ) {
                Row(
                    modifier = Modifier.padding(10.dp).fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text("⏰ Slot to Reassign:", fontWeight = FontWeight.Bold, style = MaterialTheme.typography.labelSmall)
                    Text(scheduledSlot, fontWeight = FontWeight.Bold, color = MaterialTheme.colorScheme.primary, style = MaterialTheme.typography.labelSmall)
                }
            }

            Spacer(modifier = Modifier.height(14.dp))

            Text("Select a colleague:", fontWeight = FontWeight.Bold, style = MaterialTheme.typography.titleSmall)

            Spacer(modifier = Modifier.height(8.dp))

            if (loading) LinearProgressIndicator(modifier = Modifier.fillMaxWidth())
            else if (directoryError != null) Text(directoryError!!, color = MaterialTheme.colorScheme.error)
            else if (colleagues.isEmpty()) Text("No colleagues available", color = MaterialTheme.colorScheme.onSurfaceVariant)
            colleagues.forEach { c ->
                val isSelected = selectedColleague?.id == c.id
                Card(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(vertical = 4.dp)
                        .clickable { selectedColleague = c },
                    shape = RoundedCornerShape(10.dp),
                    colors = CardDefaults.cardColors(
                        containerColor = if (isSelected) MaterialTheme.colorScheme.primaryContainer else MaterialTheme.colorScheme.surfaceVariant
                    )
                ) {
                    Row(
                        modifier = Modifier.padding(12.dp).fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Column {
                            Row(verticalAlignment = Alignment.CenterVertically) {
                                Text(c.name, fontWeight = FontWeight.Bold)

                            }
                            Text("${c.vedaShakha} • ${c.sampradaya}", style = MaterialTheme.typography.labelSmall, color = MaterialTheme.colorScheme.onSurfaceVariant)
                        }
                        Text("⭐ ${c.ratingAvg}", fontWeight = FontWeight.Bold, color = MaterialTheme.colorScheme.primary)
                    }
                }
            }

            Spacer(modifier = Modifier.height(12.dp))

            OutlinedTextField(
                value = reasonText,
                onValueChange = { reasonText = it },
                label = { Text("Reason for Colleague Delegation") },
                modifier = Modifier.fillMaxWidth()
            )

            statusMessage?.let { msg ->
                Spacer(modifier = Modifier.height(6.dp))
                Text(msg, color = MaterialTheme.colorScheme.primary, style = MaterialTheme.typography.bodySmall)
            }

            Spacer(modifier = Modifier.height(14.dp))

            Button(
                onClick = {
                    selectedColleague?.let { c ->
                        isSubmitting = true
                        coroutineScope.launch {
                            val res = PurohitApiClient.referBooking(
                                bookingId = bookingId,
                                referringPriestId = referringPriestId,
                                referredPriestId = c.id,
                                reason = reasonText
                            )
                            isSubmitting = false
                            res.onSuccess {
                                onReferralConfirmed(c.id, c.name, reasonText)
                            }.onFailure {
                                statusMessage = "Could not reassign the booking. Please try again."
                            }
                        }
                    }
                },
                enabled = !isSubmitting && selectedColleague != null,
                modifier = Modifier.fillMaxWidth().height(48.dp),
                shape = RoundedCornerShape(10.dp)
            ) {
                Text(if (isSubmitting) "Reassigning..." else "Confirm reassignment")
            }

            Spacer(modifier = Modifier.height(8.dp))

            TextButton(
                onClick = onDismiss,
                modifier = Modifier.fillMaxWidth()
            ) {
                Text("Cancel")
            }
        }
    }
}
