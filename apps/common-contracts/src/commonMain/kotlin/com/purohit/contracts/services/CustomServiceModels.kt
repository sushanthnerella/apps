package com.purohit.contracts.services

import kotlinx.serialization.Serializable

@Serializable
data class CustomServiceItem(
    val id: String,
    val priestId: String,
    val title: String,
    val category: String = "Special Ritual",
    val dakshinaFee: Double,
    val samagriKitPrice: Double = 0.0,
    val durationMinutes: Int = 90,
    val isOnlineEligible: Boolean = true,
    val samagriChecklist: List<String> = emptyList(),
    val description: String = "",
    val isActive: Boolean = true
)

@Serializable
data class CreateCustomServiceRequest(
    val priestId: String,
    val title: String,
    val category: String = "Special Ritual",
    val dakshinaFee: Double,
    val samagriKitPrice: Double = 0.0,
    val durationMinutes: Int = 90,
    val isOnlineEligible: Boolean = true,
    val samagriChecklist: List<String> = emptyList(),
    val description: String = ""
)
