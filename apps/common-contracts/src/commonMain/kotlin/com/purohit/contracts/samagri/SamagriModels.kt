package com.purohit.contracts.samagri

import kotlinx.serialization.Serializable

@Serializable
data class SamagriItem(
    val id: String,
    val poojaId: String,
    val itemName: String,
    val category: String, // 'Fresh Items', 'Dry Essentials', 'Havan Samagri', 'Utensils'
    val quantity: String,
    val isMandatory: Boolean = true,
    val estimatedPrice: Double = 0.0
)

@Serializable
data class BookingMaterial(
    val id: String = "",
    val bookingId: String,
    val itemName: String,
    val category: String,
    val quantity: String,
    val isIncludedByPriest: Boolean = true,
    val isFulfilledByVendor: Boolean = false,
    val vendorOrderId: String? = null
)

@Serializable
data class UpdateBookingMaterialsRequest(
    val bookingId: String,
    val priestKitOptIn: Boolean,
    val missingItemsToOrder: List<String> = emptyList()
)
