package com.purohit.priest.network

import io.ktor.client.*
import io.ktor.client.call.*
import io.ktor.client.engine.cio.*
import io.ktor.client.plugins.contentnegotiation.*
import io.ktor.client.request.*
import io.ktor.http.*
import io.ktor.serialization.kotlinx.json.*
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json

@Serializable
data class PriestRegPayload(
    val full_name: String,
    val phone: String,
    val sampradaya: String,
    val veda_shakha: String,
    val experience_years: Int,
    val city: String,
    val latitude: Double = 17.3850,
    val longitude: Double = 78.4867,
    val consent_agreed: Boolean
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
    val scheduled_date: String = "Today",
    val slot_time: String = "08:30 AM - 10:30 AM",
    val devotee_gotram: String = "Kashyapa",
    val devotee_nakshatram: String = "Rohini",
    val devotee_rashi: String = "Vrishabha",
    val sankalpam_purpose: String = "Family Prosperity & Health",
    val chat_room_id: String = "",
    val created_at: String = ""
)

@Serializable
data class PriestDirectoryItem(
    val id: String = "",
    val full_name: String = "",
    val phone: String = "",
    val sampradaya: String = "",
    val veda_shakha: String = "",
    val rating_avg: Double = 5.0,
    val total_completed_poojas: Int = 0,
    val repeat_booking_count: Int = 0,
    val latitude: Double = 17.3850,
    val longitude: Double = 78.4867,
    val vehicle_type: String = "Two Wheeler"
)

@Serializable
data class PriestSlotPayload(
    val id: String = "",
    val priest_id: String = "",
    val slot_date: String = "",
    val start_time: String = "",
    val end_time: String = "",
    val is_booked: Boolean = false,
    val is_held: Boolean = false
)

@Serializable
data class PriestReferralPayload(
    val booking_id: String,
    val referring_priest_id: String,
    val referred_priest_id: String,
    val reason: String
)

@Serializable
data class PriestCustomServicePayload(
    val id: String = "",
    val priest_id: String = "",
    val title: String = "",
    val title_en: String = "",
    val title_te: String = "",
    val title_hi: String = "",
    val category: String = "Special Ritual",
    val dakshina_fee: Double = 5116.0,
    val samagri_kit_price: Double = 1500.0,
    val duration_minutes: Int = 90,
    val is_online_eligible: Boolean = true,
    val samagri_checklist: List<String> = emptyList(),
    val description: String = "",
    val description_en: String = "",
    val description_te: String = "",
    val description_hi: String = "",
    val is_active: Boolean = true,
    val created_at: String = ""
)

@Serializable
data class CreateCustomServicePayload(
    val priest_id: String,
    val title: String,
    val title_en: String = "",
    val title_te: String = "",
    val title_hi: String = "",
    val category: String = "Special Ritual",
    val dakshina_fee: Double,
    val samagri_kit_price: Double,
    val duration_minutes: Int,
    val is_online_eligible: Boolean = true,
    val samagri_checklist: List<String> = emptyList(),
    val description: String
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
data class UpdateBookingMaterialsRequest(
    val materials: List<BookingMaterialPayload>
)

@Serializable
data class ChatMessagePayload(
    val id: String = "",
    val room_id: String = "",
    val sender_id: String = "",
    val sender_role: String = "PRIEST",
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

object PurohitApiClient {
    // Connected to live backend via Ngrok tunnel
    var baseUrl: String = "http://10.0.2.2:8085"

    val httpClient: HttpClient? by lazy {
        try {
            HttpClient(CIO) {
                install(ContentNegotiation) {
                    json(Json {
                        ignoreUnknownKeys = true
                        isLenient = true
                    })
                }
            }
        } catch (t: Throwable) {
            null
        }
    }

    suspend fun registerPriest(
        fullName: String,
        phone: String,
        sampradaya: String,
        vedaShakha: String,
        experienceYears: Int,
        city: String,
        consentAgreed: Boolean
    ): Result<String> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.post("$baseUrl/api/v1/auth/register-priest") {
                contentType(ContentType.Application.Json)
                header("ngrok-skip-browser-warning", "1")
                setBody(
                    PriestRegPayload(
                        full_name = fullName,
                        phone = phone,
                        sampradaya = sampradaya,
                        veda_shakha = vedaShakha,
                        experience_years = experienceYears,
                        city = city,
                        consent_agreed = consentAgreed
                    )
                )
            }
            Result.success(response.body<String>())
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

    suspend fun updateBookingMaterials(
        bookingId: String,
        materials: List<BookingMaterialPayload>
    ): Result<String> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.post("$baseUrl/api/v1/bookings/$bookingId/materials") {
                contentType(ContentType.Application.Json)
                header("ngrok-skip-browser-warning", "1")
                setBody(UpdateBookingMaterialsRequest(materials = materials))
            }
            Result.success(response.body<String>())
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
        senderRole: String = "PRIEST",
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

    suspend fun fetchPriestsList(): Result<List<PriestDirectoryItem>> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.get("$baseUrl/api/v1/priests") {
                header("ngrok-skip-browser-warning", "1")
            }
            Result.success(response.body<List<PriestDirectoryItem>>())
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchPriestSlots(priestId: String, date: String = ""): Result<List<PriestSlotPayload>> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.get("$baseUrl/api/v1/priests/slots") {
                parameter("priest_id", priestId)
                if (date.isNotEmpty()) {
                    parameter("date", date)
                }
                header("ngrok-skip-browser-warning", "1")
            }
            Result.success(response.body<List<PriestSlotPayload>>())
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun referBooking(
        bookingId: String,
        referringPriestId: String,
        referredPriestId: String,
        reason: String
    ): Result<String> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.post("$baseUrl/api/v1/bookings/refer") {
                contentType(ContentType.Application.Json)
                header("ngrok-skip-browser-warning", "1")
                setBody(
                    PriestReferralPayload(
                        booking_id = bookingId,
                        referring_priest_id = referringPriestId,
                        referred_priest_id = referredPriestId,
                        reason = reason
                    )
                )
            }
            Result.success(response.body<String>())
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchCustomServices(priestId: String = "", lang: String = ""): Result<List<PriestCustomServicePayload>> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.get("$baseUrl/api/v1/priests/custom-services") {
                if (priestId.isNotEmpty()) parameter("priest_id", priestId)
                if (lang.isNotEmpty()) {
                    parameter("lang", lang)
                    header("Accept-Language", lang)
                }
                header("ngrok-skip-browser-warning", "1")
            }
            Result.success(response.body<List<PriestCustomServicePayload>>())
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun createCustomService(payload: CreateCustomServicePayload): Result<String> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.post("$baseUrl/api/v1/priests/custom-services") {
                contentType(ContentType.Application.Json)
                header("ngrok-skip-browser-warning", "1")
                setBody(payload)
            }
            Result.success(response.body<String>())
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}
