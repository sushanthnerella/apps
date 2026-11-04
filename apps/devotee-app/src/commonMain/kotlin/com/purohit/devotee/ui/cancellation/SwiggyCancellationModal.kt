package com.purohit.devotee.ui.cancellation

import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.unit.dp
import com.purohit.contracts.cancellation.CancellationQuote
import com.purohit.contracts.i18n.StringContract

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SwiggyCancellationModal(
    quote: CancellationQuote,
    strings: StringContract,
    onConfirmCancel: (reason: String) -> Unit,
    onRescheduleInstead: () -> Unit,
    onDismiss: () -> Unit
) {
    var selectedReason by remember { mutableStateOf(strings.cancellation.reasonPriestDelayed) }

    val reasons = remember(strings) {
        listOf(
            strings.cancellation.reasonPriestDelayed,
            strings.cancellation.reasonFamilyPlansChanged,
            strings.cancellation.reasonAlternateArrangedLocally,
            strings.cancellation.reasonWrongPoojaOrDate,
            strings.cancellation.reasonOther
        )
    }

    ModalBottomSheet(
        onDismissRequest = onDismiss,
        shape = RoundedCornerShape(topStart = 24.dp, topEnd = 24.dp)
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(20.dp)
                .verticalScroll(rememberScrollState())
        ) {
            // Header
            Text(
                text = strings.cancellation.cancelBookingTitle,
                style = MaterialTheme.typography.headlineSmall,
                fontWeight = FontWeight.Bold,
                color = MaterialTheme.colorScheme.error
            )
            Spacer(modifier = Modifier.height(4.dp))
            Text(
                text = strings.cancellation.cancelWarningHeader,
                style = MaterialTheme.typography.bodyMedium,
                color = Color.DarkGray
            )

            Spacer(modifier = Modifier.height(14.dp))

            // Policy Notice Banner
            Surface(
                modifier = Modifier.fillMaxWidth(),
                color = if (quote.isFreeCancellation) Color(0xFFD1FAE5) else Color(0xFFFEF3C7),
                shape = RoundedCornerShape(12.dp)
            ) {
                Column(modifier = Modifier.padding(12.dp)) {
                    Text(
                        text = if (quote.isFreeCancellation) "✓ Free Cancellation Policy" else "⚠️ Nominal Travel Fee Applies",
                        fontWeight = FontWeight.Bold,
                        color = if (quote.isFreeCancellation) Color(0xFF065F46) else Color(0xFF92400E)
                    )
                    Spacer(modifier = Modifier.height(4.dp))
                    Text(
                        text = quote.policyNotice,
                        style = MaterialTheme.typography.bodySmall,
                        color = Color.Black
                    )
                }
            }

            Spacer(modifier = Modifier.height(16.dp))

            // Retention Rescue Hook
            Surface(
                modifier = Modifier.fillMaxWidth(),
                color = MaterialTheme.colorScheme.primaryContainer.copy(alpha = 0.3f),
                shape = RoundedCornerShape(12.dp)
            ) {
                Column(modifier = Modifier.padding(12.dp)) {
                    Text(
                        text = "🗓️ " + strings.cancellation.retentionRescueTitle,
                        fontWeight = FontWeight.Bold,
                        style = MaterialTheme.typography.titleSmall
                    )
                    Spacer(modifier = Modifier.height(2.dp))
                    Text(
                        text = strings.cancellation.retentionRescueDescription,
                        style = MaterialTheme.typography.bodySmall
                    )
                    Spacer(modifier = Modifier.height(8.dp))
                    OutlinedButton(
                        onClick = onRescheduleInstead,
                        modifier = Modifier.fillMaxWidth(),
                        shape = RoundedCornerShape(8.dp)
                    ) {
                        Text(strings.cancellation.rescheduleInsteadCTA)
                    }
                }
            }

            Spacer(modifier = Modifier.height(16.dp))

            // Mandatory Reason Selector
            Text(
                text = strings.cancellation.reasonLabel,
                fontWeight = FontWeight.Bold,
                style = MaterialTheme.typography.titleSmall
            )
            Spacer(modifier = Modifier.height(8.dp))
            reasons.forEach { r ->
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .clickable { selectedReason = r }
                        .padding(vertical = 4.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    RadioButton(
                        selected = selectedReason == r,
                        onClick = { selectedReason = r }
                    )
                    Spacer(modifier = Modifier.width(8.dp))
                    Text(r, style = MaterialTheme.typography.bodyMedium)
                }
            }

            Spacer(modifier = Modifier.height(14.dp))

            // Refund Summary Box
            Card(
                modifier = Modifier.fillMaxWidth(),
                shape = RoundedCornerShape(12.dp),
                colors = CardDefaults.cardColors(containerColor = MaterialTheme.colorScheme.surfaceVariant)
            ) {
                Column(modifier = Modifier.padding(14.dp)) {
                    Text(
                        text = strings.cancellation.refundSummaryTitle,
                        fontWeight = FontWeight.Bold,
                        style = MaterialTheme.typography.titleSmall
                    )
                    Spacer(modifier = Modifier.height(8.dp))
                    Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                        Text(strings.cancellation.advanceDakshinaPaid, color = Color.Gray)
                        Text("₹${quote.advancePaid.toInt()}", fontWeight = FontWeight.SemiBold)
                    }
                    if (quote.cancellationFee > 0) {
                        Spacer(modifier = Modifier.height(4.dp))
                        Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                            Text(strings.cancellation.cancellationFee, color = MaterialTheme.colorScheme.error)
                            Text("- ₹${quote.cancellationFee.toInt()}", color = MaterialTheme.colorScheme.error, fontWeight = FontWeight.SemiBold)
                        }
                    }
                    Divider(modifier = Modifier.padding(vertical = 8.dp))
                    Row(modifier = Modifier.fillMaxWidth(), horizontalArrangement = Arrangement.SpaceBetween) {
                        Text(strings.cancellation.totalRefundAmount, fontWeight = FontWeight.Bold)
                        Text("₹${quote.refundAmount.toInt()}", fontWeight = FontWeight.Bold, color = Color(0xFF059669), style = MaterialTheme.typography.titleMedium)
                    }
                }
            }

            Spacer(modifier = Modifier.height(16.dp))

            // Action Buttons
            Button(
                onClick = { onConfirmCancel(selectedReason) },
                modifier = Modifier.fillMaxWidth().height(50.dp),
                colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.error),
                shape = RoundedCornerShape(12.dp)
            ) {
                Text(
                    text = strings.cancellation.confirmCancelCTA,
                    fontWeight = FontWeight.Bold
                )
            }

            Spacer(modifier = Modifier.height(8.dp))

            TextButton(
                onClick = onDismiss,
                modifier = Modifier.fillMaxWidth()
            ) {
                Text(strings.common.cancelAction)
            }
        }
    }
}
