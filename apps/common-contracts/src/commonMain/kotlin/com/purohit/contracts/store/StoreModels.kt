package com.purohit.contracts.store

import kotlinx.serialization.Serializable

@Serializable
data class Vendor(
    val id: String,
    val shopName: String,
    val ownerName: String,
    val phone: String,
    val profilePictureUrl: String = "",
    val address: String,
    val city: String = "Hyderabad",
    val latitude: Double = 17.3850,
    val longitude: Double = 78.4867,
    val ratingAvg: Double = 4.90,
    val isOpen: Boolean = true,
    val distanceKm: Double = 0.0
)

@Serializable
data class VendorInventoryItem(
    val id: String,
    val vendorId: String,
    val itemName: String,
    val category: String = "Pooja Samagri",
    val price: Double,
    val stockQuantity: Int = 100,
    val unit: String = "packet",
    val imageUrl: String = "",
    val isAvailable: Boolean = true
)

@Serializable
data class StoreOrder(
    val id: String,
    val userId: String,
    val vendorId: String,
    val bookingId: String? = null,
    val status: String = "PLACED", // 'PLACED', 'PACKED', 'DISPATCHED', 'DELIVERED', 'CANCELLED'
    val totalAmount: Double,
    val deliveryAddress: String,
    val deliverySlot: String? = null,
    val items: List<StoreOrderItem> = emptyList(),
    val createdAt: String = ""
)

@Serializable
data class StoreOrderItem(
    val id: String = "",
    val inventoryItemId: String,
    val itemName: String,
    val quantity: Int = 1,
    val unitPrice: Double,
    val subtotal: Double
)

@Serializable
data class CreateStoreOrderRequest(
    val userId: String,
    val vendorId: String,
    val bookingId: String? = null,
    val deliveryAddress: String,
    val deliverySlot: String? = null,
    val items: List<StoreOrderItem>
)
