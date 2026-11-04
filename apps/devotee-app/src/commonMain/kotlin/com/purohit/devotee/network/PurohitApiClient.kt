package com.purohit.devotee.network

import com.purohit.contracts.store.Vendor
import com.purohit.contracts.store.VendorInventoryItem
import com.purohit.contracts.temple.Temple
import com.purohit.contracts.temple.TempleVipPuja
import io.ktor.client.*
import io.ktor.client.call.*
import io.ktor.client.engine.cio.*
import io.ktor.client.plugins.contentnegotiation.*
import io.ktor.client.request.*
import io.ktor.client.statement.*
import io.ktor.http.*
import io.ktor.serialization.kotlinx.json.*
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json

@Serializable
data class SamagriItemPayload(
    val id: String = "",
    val pooja_id: String = "",
    val item_name: String = "",
    val category: String = "",
    val quantity: String = "",
    val is_mandatory: Boolean = true,
    val estimated_price: Double = 0.0
)

@Serializable
data class BookingMaterialPayload(
    val id: String = "",
    val booking_id: String = "",
    val item_name: String = "",
    val category: String = "",
    val quantity: String = "",
    val is_included_by_priest: Boolean = true,
    val is_fulfilled_by_vendor: Boolean = false
)

@Serializable
data class ChatMessagePayload(
    val id: String = "",
    val room_id: String = "",
    val sender_id: String = "",
    val sender_role: String = "DEVOTEE",
    val message_type: String = "TEXT",
    val content: String = "",
    val attachment_url: String = "",
    val is_read: Boolean = false,
    val created_at: String = ""
)

@Serializable
data class ChatRoomResponse(
    val room_id: String = "",
    val booking_id: String = ""
)

@Serializable
data class BookingDetailPayload(
    val id: String = "",
    val user_id: String = "",
    val priest_id: String = "",
    val devotee_name: String = "",
    val devotee_phone: String = "",
    val priest_name: String = "",
    val priest_phone: String = "",
    val pooja_title: String = "",
    val booking_type: String = "INSTANT_DISPATCH",
    val status: String = "REQUESTED",
    val escrow_status: String = "HELD_IN_ESCROW",
    val total_amount: Double = 0.0,
    val include_samagri_kit: Boolean = true,
    val completion_otp: String = "",
    val event_address: String = "",
    val latitude: Double = 17.3850,
    val longitude: Double = 78.4867,
    val chat_room_id: String = ""
)

@Serializable
data class UpdateBookingMaterialsRequest(
    val materials: List<BookingMaterialPayload>
)

@Serializable
data class DevoteeLoginPayload(
    val phone: String,
    val otp: String,
    val role: String = "devotee"
)

@Serializable
data class DevoteeRegistrationPayload(
    val full_name: String,
    val phone: String,
    val gotram: String = "",
    val nakshatram: String = "",
    val rashi: String = "",
    val city: String = "Hyderabad",
    val consent_agreed: Boolean = true
)

@Serializable
data class AuthResponsePayload(
    val status: String = "",
    val access_token: String = "",
    val token_type: String = "Bearer",
    val user: DevoteeProfilePayload? = null,
    val user_id: String = "",
    val full_name: String = "",
    val message: String = ""
)

@Serializable
data class DevoteeProfilePayload(
    val id: String = "",
    val phone: String = "",
    val full_name: String = "",
    val gotram: String = "",
    val nakshatram: String = "",
    val rashi: String = "",
    val city: String = "Hyderabad",
    val role: String = "devotee",
    val consent_agreed: Boolean = true
)

@Serializable
data class PoojaCatalogItem(
    val id: String,
    val title: String,
    val title_en: String = "",
    val title_te: String = "",
    val title_hi: String = "",
    val category: String,
    val base_price: Double,
    val samagri_kit_price: Double,
    val duration_minutes: Int,
    val is_online_eligible: Boolean,
    val icon: String = "🪔",
    val description: String = "",
    val description_en: String = "",
    val description_te: String = "",
    val description_hi: String = ""
)

@Serializable
data class NearbyPriestItem(
    val id: String,
    val full_name: String,
    val phone: String,
    val sampradaya: String,
    val veda_shakha: String = "",
    val rating_avg: Double,
    val total_completed_poojas: Int,
    val repeat_booking_count: Int,
    val distance_meters: Double,
    val distance_km: Double,
    val eta_minutes: Int,
    val vehicle_type: String = "Two Wheeler",
    val services_offered: List<String> = emptyList(),
    val score: Double = 0.0
)

@Serializable
data class NearbyPriestsResponse(
    val total_priests_online: Int,
    val priests: List<NearbyPriestItem>
)

@Serializable
data class BookingCreatePayload(
    val user_id: String,
    val priest_id: String,
    val pooja_id: String = "1",
    val scheduled_date: String = "Today",
    val slot_time: String = "08:30 AM - 10:30 AM",
    val total_amount: Double,
    val booking_type: String = "INSTANT_DISPATCH",
    val event_address: String = "Plot 42, Sri Rama Nagar, Madhav Nagar, Hyderabad (Landmark: Behind Sri Rama Temple)",
    val devotee_gotram: String = "",
    val devotee_nakshatram: String = "",
    val devotee_rashi: String = "",
    val sankalpam_purpose: String = "Family Prosperity & Health"
)

@Serializable
data class BookingResponsePayload(
    val booking_id: String = "",
    val status: String = "REQUESTED",
    val escrow_status: String = "HELD_IN_ESCROW",
    val completion_otp: String = "",
    val total_amount: Double = 0.0,
    val message: String = ""
)

@Serializable
data class StoreOrderPayload(
    val user_id: String,
    val vendor_id: String,
    val booking_id: String? = null,
    val total_amount: Double,
    val delivery_address: String = "Hyderabad",
    val delivery_slot: String? = "Express Delivery",
    val items: List<StoreOrderItemPayload> = emptyList()
)

@Serializable
data class StoreOrderItemPayload(
    val item_id: String,
    val item_name: String,
    val quantity: Int,
    val unit_price: Double
)

@Serializable
data class StoreOrderResponse(
    val order_id: String = "",
    val status: String = "PLACED",
    val total_amount: Double = 0.0,
    val message: String = ""
)

@Serializable
data class BookingStatusUpdatePayload(
    val booking_id: String,
    val status: String,
    val priest_id: String? = null,
    val completion_otp: String? = null,
    val dakshina_amount: Double = 0.0
)

@Serializable
data class CancellationQuotePayload(
    val booking_id: String = "",
    val current_status: String = "",
    val advance_paid: Double = 0.0,
    val cancellation_fee: Double = 0.0,
    val travel_fee: Double = 0.0,
    val priest_compensation: Double = 0.0,
    val refund_amount: Double = 0.0,
    val is_free_cancellation: Boolean = false,
    val policy_notice: String = ""
)

@Serializable
data class CancelBookingRequestPayload(
    val booking_id: String,
    val cancelled_by_role: String = "devotee",
    val reason_code: String,
    val reason_text: String = ""
)

@Serializable
data class CancelBookingResponsePayload(
    val status: String = "CANCELLED",
    val booking_id: String = "",
    val refund_amount: Double = 0.0,
    val priest_compensation: Double = 0.0,
    val policy_notice: String = "",
    val message: String = ""
)

@Serializable
data class TempleTicketBookingPayload(
    val temple_id: String,
    val vip_puja_id: String,
    val user_id: String = "",
    val devotee_name: String,
    val devotee_phone: String,
    val devotee_gotram: String = "",
    val devotee_nakshatram: String = "",
    val visit_date: String = "2026-11-04",
    val slot_time: String = "07:00 AM",
    val ticket_type: String = "VIP_IN_PERSON",
    val amount_paid: Double
)

@Serializable
data class TempleTicketResponse(
    val ticket_id: String = "",
    val qr_token: String = "",
    val status: String = "CONFIRMED",
    val amount_paid: Double = 0.0,
    val message: String = ""
)

object PurohitApiClient {
    // Connected to live backend via Ngrok tunnel
    var baseUrl: String = "http://10.0.2.2:8085"
    var authToken: String = ""
    var currentProfile: DevoteeProfilePayload? = null

    val httpClient: HttpClient? by lazy {
        try {
            HttpClient(CIO) {
                install(ContentNegotiation) {
                    json(Json {
                        ignoreUnknownKeys = true
                        isLenient = true
                        prettyPrint = true
                        encodeDefaults = true
                    })
                }
            }
        } catch (t: Throwable) {
            null
        }
    }

    suspend fun registerDevotee(
        fullName: String,
        phone: String,
        gotram: String,
        nakshatram: String,
        rashi: String,
        city: String,
        consentAgreed: Boolean
    ): Result<AuthResponsePayload> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.post("$baseUrl/api/v1/auth/register-devotee") {
                contentType(ContentType.Application.Json)
                header("ngrok-skip-browser-warning", "1")
                setBody(
                    DevoteeRegistrationPayload(
                        full_name = fullName,
                        phone = phone,
                        gotram = gotram,
                        nakshatram = nakshatram,
                        rashi = rashi,
                        city = if (city.isNotBlank()) city else "Hyderabad",
                        consent_agreed = true
                    )
                )
            }
            if (response.status.value in 200..299) {
                val authRes = response.body<AuthResponsePayload>()
                if (authRes.access_token.isNotBlank()) {
                    authToken = authRes.access_token
                    currentProfile = authRes.user ?: DevoteeProfilePayload(
                        id = authRes.user_id,
                        phone = phone,
                        full_name = fullName,
                        gotram = gotram,
                        nakshatram = nakshatram,
                        rashi = rashi,
                        city = city,
                        consent_agreed = true
                    )
                }
                Result.success(authRes)
            } else {
                val errText = response.bodyAsText()
                Result.failure(Exception("Server returned ${response.status.value}: $errText"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun login(phone: String, otp: String): Result<AuthResponsePayload> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.post("$baseUrl/api/v1/auth/login") {
                contentType(ContentType.Application.Json)
                header("ngrok-skip-browser-warning", "1")
                setBody(DevoteeLoginPayload(phone, otp, "devotee"))
            }
            if (response.status.value in 200..299) {
                val authRes = response.body<AuthResponsePayload>()
                if (authRes.access_token.isNotBlank()) {
                    authToken = authRes.access_token
                    currentProfile = authRes.user
                }
                Result.success(authRes)
            } else {
                val errText = response.bodyAsText()
                Result.failure(Exception("Server returned ${response.status.value}: $errText"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchProfile(): Result<DevoteeProfilePayload> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.get("$baseUrl/api/v1/user/profile") {
                header("Authorization", "Bearer $authToken")
                header("ngrok-skip-browser-warning", "1")
            }
            if (response.status.value in 200..299) {
                val profile = response.body<DevoteeProfilePayload>()
                currentProfile = profile
                Result.success(profile)
            } else {
                val errText = response.bodyAsText()
                Result.failure(Exception("Server returned ${response.status.value}: $errText"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchCatalog(search: String = "", category: String = "", lang: String = ""): Result<List<PoojaCatalogItem>> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.get("$baseUrl/api/v1/poojas") {
                parameter("search", search)
                parameter("category", category)
                if (lang.isNotEmpty()) {
                    parameter("lang", lang)
                    header("Accept-Language", lang)
                }
                header("ngrok-skip-browser-warning", "1")
            }
            Result.success(response.body<List<PoojaCatalogItem>>())
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchNearbyPriests(lat: Double = 17.3850, lng: Double = 78.4867, service: String = ""): Result<List<NearbyPriestItem>> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.get("$baseUrl/api/v1/priests/nearby") {
                parameter("lat", lat)
                parameter("lng", lng)
                parameter("service", service)
                header("ngrok-skip-browser-warning", "1")
            }
            val res = response.body<NearbyPriestsResponse>()
            Result.success(res.priests)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun createBooking(
        priestId: String,
        poojaId: String = "1",
        totalAmount: Double,
        scheduledDate: String = "Today",
        slotTime: String = "08:30 AM - 10:30 AM",
        gotram: String = "",
        nakshatram: String = "",
        rashi: String = "",
        purpose: String = "Family Prosperity & Health"
    ): Result<BookingResponsePayload> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        val userId = currentProfile?.id ?: "devotee_direct"
        val userGotram = if (gotram.isNotBlank()) gotram else (currentProfile?.gotram ?: "Kashyapa")
        val userNakshatram = if (nakshatram.isNotBlank()) nakshatram else (currentProfile?.nakshatram ?: "Rohini")
        val userRashi = if (rashi.isNotBlank()) rashi else (currentProfile?.rashi ?: "Vrishabha")
        return try {
            val response = client.post("$baseUrl/api/v1/bookings") {
                contentType(ContentType.Application.Json)
                header("Authorization", "Bearer $authToken")
                header("ngrok-skip-browser-warning", "1")
                setBody(
                    BookingCreatePayload(
                        user_id = userId,
                        priest_id = priestId,
                        pooja_id = poojaId,
                        scheduled_date = scheduledDate,
                        slot_time = slotTime,
                        total_amount = totalAmount,
                        devotee_gotram = userGotram,
                        devotee_nakshatram = userNakshatram,
                        devotee_rashi = userRashi,
                        sankalpam_purpose = purpose
                    )
                )
            }
            Result.success(response.body<BookingResponsePayload>())
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchTemples(): Result<List<Temple>> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.get("$baseUrl/api/v1/temples") {
                header("ngrok-skip-browser-warning", "1")
            }
            Result.success(response.body<List<Temple>>())
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchTempleVipPujas(templeId: String): Result<List<TempleVipPuja>> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.get("$baseUrl/api/v1/temples/$templeId/vip-pujas") {
                header("ngrok-skip-browser-warning", "1")
            }
            Result.success(response.body<List<TempleVipPuja>>())
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun bookTempleTicket(
        templeId: String,
        vipPujaId: String,
        amount: Double
    ): Result<TempleTicketResponse> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        val profile = currentProfile
        return try {
            val response = client.post("$baseUrl/api/v1/temples/tickets/book") {
                contentType(ContentType.Application.Json)
                header("Authorization", "Bearer $authToken")
                header("ngrok-skip-browser-warning", "1")
                setBody(
                    TempleTicketBookingPayload(
                        temple_id = templeId,
                        vip_puja_id = vipPujaId,
                        user_id = profile?.id ?: "",
                        devotee_name = profile?.full_name ?: "Devotee",
                        devotee_phone = profile?.phone ?: "+91 99999 99999",
                        devotee_gotram = profile?.gotram ?: "",
                        devotee_nakshatram = profile?.nakshatram ?: "",
                        amount_paid = amount
                    )
                )
            }
            Result.success(response.body<TempleTicketResponse>())
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchVendors(): Result<List<Vendor>> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.get("$baseUrl/api/v1/store/vendors") {
                header("ngrok-skip-browser-warning", "1")
            }
            Result.success(response.body<List<Vendor>>())
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchVendorInventory(vendorId: String): Result<List<VendorInventoryItem>> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.get("$baseUrl/api/v1/store/vendors/$vendorId/inventory") {
                header("ngrok-skip-browser-warning", "1")
            }
            Result.success(response.body<List<VendorInventoryItem>>())
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun createStoreOrder(
        vendorId: String,
        total: Double,
        items: List<StoreOrderItemPayload>
    ): Result<StoreOrderResponse> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.post("$baseUrl/api/v1/store/orders") {
                contentType(ContentType.Application.Json)
                header("Authorization", "Bearer $authToken")
                header("ngrok-skip-browser-warning", "1")
                setBody(
                    StoreOrderPayload(
                        user_id = currentProfile?.id ?: "devotee_store",
                        vendor_id = vendorId,
                        total_amount = total,
                        delivery_address = currentProfile?.city ?: "Hyderabad",
                        items = items
                    )
                )
            }
            Result.success(response.body<StoreOrderResponse>())
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchPoojaSamagri(poojaId: String): Result<List<SamagriItemPayload>> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.get("$baseUrl/api/v1/poojas/$poojaId/samagri") {
                header("ngrok-skip-browser-warning", "1")
            }
            Result.success(response.body<List<SamagriItemPayload>>())
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchBookingMaterials(bookingId: String): Result<List<BookingMaterialPayload>> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.get("$baseUrl/api/v1/bookings/$bookingId/materials") {
                header("ngrok-skip-browser-warning", "1")
            }
            Result.success(response.body<List<BookingMaterialPayload>>())
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchChatMessages(roomId: String): Result<List<ChatMessagePayload>> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.get("$baseUrl/api/v1/chat/messages") {
                parameter("room_id", roomId)
                header("ngrok-skip-browser-warning", "1")
            }
            Result.success(response.body<List<ChatMessagePayload>>())
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun sendChatMessage(
        roomId: String,
        senderId: String,
        senderRole: String,
        content: String
    ): Result<ChatMessagePayload> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.post("$baseUrl/api/v1/chat/messages") {
                contentType(ContentType.Application.Json)
                header("ngrok-skip-browser-warning", "1")
                setBody(
                    ChatMessagePayload(
                        room_id = roomId,
                        sender_id = senderId,
                        sender_role = senderRole,
                        content = content
                    )
                )
            }
            Result.success(response.body<ChatMessagePayload>())
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getChatRoomByBooking(bookingId: String): Result<ChatRoomResponse> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.get("$baseUrl/api/v1/chat/room-by-booking") {
                parameter("booking_id", bookingId)
                header("ngrok-skip-browser-warning", "1")
            }
            Result.success(response.body<ChatRoomResponse>())
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun updateBookingStatus(
        bookingId: String,
        status: String,
        priestId: String? = null,
        otp: String? = null,
        dakshina: Double = 0.0
    ): Result<String> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.post("$baseUrl/api/v1/bookings/status") {
                contentType(ContentType.Application.Json)
                header("ngrok-skip-browser-warning", "1")
                setBody(
                    BookingStatusUpdatePayload(
                        booking_id = bookingId,
                        status = status,
                        priest_id = priestId,
                        completion_otp = otp,
                        dakshina_amount = dakshina
                    )
                )
            }
            Result.success(response.body<String>())
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchBookingDetails(bookingId: String): Result<BookingDetailPayload> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.get("$baseUrl/api/v1/bookings/$bookingId") {
                header("ngrok-skip-browser-warning", "1")
            }
            Result.success(response.body<BookingDetailPayload>())
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchCancellationQuote(bookingId: String): Result<CancellationQuotePayload> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.get("$baseUrl/api/v1/bookings/$bookingId/cancel-quote") {
                header("ngrok-skip-browser-warning", "1")
            }
            Result.success(response.body<CancellationQuotePayload>())
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun cancelBooking(
        bookingId: String,
        reasonCode: String,
        reasonText: String = ""
    ): Result<CancelBookingResponsePayload> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.post("$baseUrl/api/v1/bookings/cancel") {
                contentType(ContentType.Application.Json)
                header("ngrok-skip-browser-warning", "1")
                setBody(
                    CancelBookingRequestPayload(
                        booking_id = bookingId,
                        cancelled_by_role = "devotee",
                        reason_code = reasonCode,
                        reason_text = reasonText
                    )
                )
            }
            Result.success(response.body<CancelBookingResponsePayload>())
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}
