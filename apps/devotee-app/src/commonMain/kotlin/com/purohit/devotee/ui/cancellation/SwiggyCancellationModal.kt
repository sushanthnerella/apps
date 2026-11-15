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
import com.purohit.contracts.i18n.AppLanguage
import com.purohit.devotee.LocalAppLanguage

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SwiggyCancellationModal(
    quote: CancellationQuote,
    strings: StringContract,
    onConfirmCancel: (reason: String) -> Unit,
    isSubmitting: Boolean = false,
    onDismiss: () -> Unit
) {
    val language = LocalAppLanguage.current
    fun tr(en: String, te: String, hi: String) = when (language) {
        AppLanguage.TELUGU -> te
        AppLanguage.HINDI -> hi
        else -> en
    }
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
        onDismissRequest = { if (!isSubmitting) onDismiss() },
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
                color = MaterialTheme.colorScheme.onSurfaceVariant
            )

            Spacer(modifier = Modifier.height(14.dp))

            // Policy Notice Banner
            Surface(
                modifier = Modifier.fillMaxWidth(),
                color = if (quote.isFreeCancellation) Color(0xFFD1FAE5) else MaterialTheme.colorScheme.surfaceVariant,
                shape = RoundedCornerShape(12.dp)
            ) {
                Column(modifier = Modifier.padding(12.dp)) {
                    Text(
                        text = tr("Cancellation", "రద్దు", "रद्द करना"),
                        fontWeight = FontWeight.Bold,
                        color = if (quote.isFreeCancellation) Color(0xFF065F46) else Color(0xFF92400E)
                    )
                    Spacer(modifier = Modifier.height(4.dp))
                    Text(
                        text = tr("No online payment was collected. Confirm any priest or materials charges directly.", "ఆన్‌లైన్ చెల్లింపు వసూలు చేయలేదు. పురోహితులు లేదా సామగ్రి ఛార్జీలను నేరుగా నిర్ధారించుకోండి.", "कोई ऑनलाइन भुगतान नहीं लिया गया। पुरोहित या सामग्री का शुल्क सीधे सुनिश्चित करें।"),
                        style = MaterialTheme.typography.bodySmall,
                        color = Color.Black
                    )
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

            // The free pilot does not collect online payments. The backend quote is
            // calculated from the booking estimate, so it is not evidence of a payment.
            Spacer(modifier = Modifier.height(16.dp))

            // Action Buttons
            Button(
                onClick = { onConfirmCancel(selectedReason) },
                enabled = !isSubmitting,
                modifier = Modifier.fillMaxWidth().heightIn(min = 50.dp),
                colors = ButtonDefaults.buttonColors(containerColor = MaterialTheme.colorScheme.error),
                shape = RoundedCornerShape(12.dp)
            ) {
                Text(
                    text = if (isSubmitting) strings.common.loadingState else strings.cancellation.confirmCancelCTA,
                    fontWeight = FontWeight.Bold
                )
            }

            Spacer(modifier = Modifier.height(8.dp))

            TextButton(
                onClick = onDismiss,
                enabled = !isSubmitting,
                modifier = Modifier.fillMaxWidth()
            ) {
                Text(strings.common.cancelAction)
            }
        }
    }
}
