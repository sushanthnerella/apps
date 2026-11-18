package com.purohit.priest.network

import io.ktor.client.plugins.websocket.*
import io.ktor.websocket.*
import kotlinx.coroutines.*
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.flow

import com.purohit.contracts.idempotency.IdempotencyKeyGenerator
import com.purohit.contracts.event.*
import io.ktor.client.*
import io.ktor.client.call.*
import io.ktor.client.plugins.contentnegotiation.*
import io.ktor.client.plugins.defaultRequest
import io.ktor.client.plugins.HttpTimeout
import io.ktor.client.request.*
import io.ktor.client.statement.*
import io.ktor.http.*
import io.ktor.serialization.kotlinx.json.*
import kotlinx.serialization.Serializable
import kotlinx.serialization.json.Json

@Serializable
data class PriestRegPayload(
    val full_name: String,
    val phone: String,
    val password: String = "",
    val sampradaya: String,
    val veda_shakha: String,
    val experience_years: Int,
    val city: String,
    val latitude: Double = 17.3850,
    val longitude: Double = 78.4867,
    val consent_agreed: Boolean
)

@Serializable
data class PriestLoginPayload(
    val phone: String,
    val otp: String = "",
    val password: String = "",
    val role: String = "priest"
)

@Serializable
data class PriestProfilePayload(
    val id: String = "",
    val phone: String = "",
    val full_name: String = "",
    val sampradaya: String = "",
    val veda_shakha: String = "",
    val experience_years: Int = 0,
    val city: String = "Hyderabad",
    val email: String = "",
    val email_verified: Boolean = false,
    val role: String = "priest"
)

@Serializable
data class PriestDutyStatusPayload(
    val is_online: Boolean = false,
    val location_confirmed: Boolean = false
)

@Serializable
data class PriestLocationPayload(
    val latitude: Double,
    val longitude: Double,
    val captured_at: String? = null,
    val accuracy_m: Double? = null
)

@Serializable
data class RecoveryStartPayload(
    val email: String = "",
    val phone: String = "",
    val role: String = "priest"
)

@Serializable
data class RecoveryStartResponse(
    val status: String = "",
    val message: String = "",
    val cooldown_seconds: Int = 60
)

@Serializable
data class UpdateEmailRequest(
    val email: String
)

@Serializable
data class UpdateEmailResponse(
    val status: String = "",
    val message: String = "",
    val email: String = "",
    val email_verified: Boolean = false
)

@Serializable
data class AuthResponsePayload(
    val status: String = "",
    val access_token: String = "",
    val token_type: String = "Bearer",
    val user: PriestProfilePayload? = null,
    val user_id: String = "",
    val full_name: String = "",
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
data class CreatePriestSlotPayload(
    val priest_id: String,
    val slot_date: String,
    val start_time: String,
    val end_time: String
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
internal data class ChatSendPayload(
    val room_id: String,
    val sender_id: String,
    val sender_role: String,
    val content: String
)

@Serializable
data class ChatRoomResponse(
    val room_id: String = "",
    val booking_id: String = ""
)

object PurohitApiClient {
    val baseUrl: String get() = com.purohit.contracts.config.BackendConfig.activeBaseUrl

    private var sessionVault: com.purohit.contracts.session.SessionVault? = null
    fun initializeSession(vault: com.purohit.contracts.session.SessionVault) {
        sessionVault = vault
        authToken = vault.getToken().orEmpty()
    }

    var authToken: String = ""
        set(value) {
            if (value.isBlank()) sessionVault?.clear() else sessionVault?.saveToken(value)
            field = value
        }

    var currentPriestProfile: PriestProfilePayload? = null

    val httpClient: HttpClient? by lazy {
        try {
            HttpClient(createPlatformHttpEngine()) {
                install(HttpTimeout) {
                    requestTimeoutMillis = 60_000L
                    connectTimeoutMillis = 45_000L
                    socketTimeoutMillis = 45_000L
                }
                install(WebSockets)
                install(ContentNegotiation) {
                    json(Json {
                        ignoreUnknownKeys = true
                        isLenient = true
                        prettyPrint = true
                        encodeDefaults = true
                    })
                }
                defaultRequest {
                    header("User-Agent", "PujaVaahiniPriest/1.0")
                }
            }
        } catch (t: Throwable) {
            null
        }
    }

    fun generateIdempotencyKey(): String = IdempotencyKeyGenerator.generate()

    suspend fun registerPriest(
        fullName: String,
        phone: String,
        password: String = "",
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
                setBody(
                    PriestRegPayload(
                        full_name = fullName,
                        phone = phone,
                        password = password,
                        sampradaya = sampradaya,
                        veda_shakha = vedaShakha,
                        experience_years = experienceYears,
                        city = city,
                        consent_agreed = consentAgreed
                    )
                )
            }
            if (response.status.value in 200..299) {
                Result.success(response.body<String>())
            } else {
                val err = response.bodyAsText()
                Result.failure(Exception("Registration failed (HTTP ${response.status.value}): $err"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun loginPriest(phone: String, otp: String = "", password: String = ""): Result<AuthResponsePayload> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.post("$baseUrl/api/v1/auth/login") {
                contentType(ContentType.Application.Json)
                setBody(PriestLoginPayload(phone = phone, otp = otp, password = password, role = "priest"))
            }
            if (response.status.value in 200..299) {
                val authRes = response.body<AuthResponsePayload>()
                if (authRes.access_token.isNotBlank()) {
                    authToken = authRes.access_token
                    currentPriestProfile = authRes.user
                }
                Result.success(authRes)
            } else {
                if (response.status.value == 401) authToken = ""
                val errText = response.bodyAsText()
                Result.failure(Exception("Server returned ${response.status.value}: $errText"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchPriestProfile(): Result<PriestProfilePayload> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.get("$baseUrl/api/v1/user/profile") {
                header("Authorization", "Bearer $authToken")
            }
            if (response.status.value in 200..299) {
                val profile = response.body<PriestProfilePayload>()
                currentPriestProfile = profile
                Result.success(profile)
            } else {
                if (response.status.value == 401) authToken = ""
                val errText = response.bodyAsText()
                Result.failure(Exception("Server returned ${response.status.value}: $errText"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchPriestBookings(priestId: String, status: String = ""): Result<List<BookingDetailPayload>> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.get("$baseUrl/api/v1/bookings") {
                parameter("priest_id", priestId)
                if (status.isNotBlank()) parameter("status", status)
                header("Authorization", "Bearer $authToken")
            }
            if (response.status.value !in 200..299) {
                return Result.failure(Exception("Unable to load bookings (HTTP ${response.status.value})"))
            }
            Result.success(response.body<List<BookingDetailPayload>>())
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun getDutyStatus(): Result<PriestDutyStatusPayload> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.get("$baseUrl/api/v1/priests/me/duty") {
                header("Authorization", "Bearer $authToken")
            }
            if (response.status.value in 200..299) {
                Result.success(response.body<PriestDutyStatusPayload>())
            } else {
                Result.failure(Exception("Failed to get duty status: HTTP ${response.status.value}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun setDutyStatus(isOnline: Boolean): Result<Boolean> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.put("$baseUrl/api/v1/priests/me/duty") {
                contentType(ContentType.Application.Json)
                header("Authorization", "Bearer $authToken")
                setBody(mapOf("is_online" to isOnline))
            }
            if (response.status.value in 200..299) {
                val status = response.body<PriestDutyStatusPayload>()
                Result.success(status.is_online)
            } else {
                Result.failure(Exception("Failed to update duty status: HTTP ${response.status.value}"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun requestPasswordReset(email: String = "", role: String = "priest"): Result<RecoveryStartResponse> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.post("$baseUrl/api/v1/auth/recovery/start") {
                contentType(ContentType.Application.Json)
                setBody(RecoveryStartPayload(email = email, role = role))
            }
            if (response.status.value in 200..299) {
                Result.success(response.body<RecoveryStartResponse>())
            } else {
                val errText = response.bodyAsText()
                Result.failure(Exception(errText))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun requestEmailVerification(email: String): Result<UpdateEmailResponse> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.post("$baseUrl/api/v1/user/email") {
                contentType(ContentType.Application.Json)
                header("Authorization", "Bearer $authToken")
                setBody(UpdateEmailRequest(email = email))
            }
            if (response.status.value in 200..299) {
                Result.success(response.body<UpdateEmailResponse>())
            } else {
                val errText = response.bodyAsText()
                Result.failure(Exception(errText))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun updateLocation(latitude: Double, longitude: Double, capturedAt: String? = null, accuracyM: Double? = null): Result<Unit> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.put("$baseUrl/api/v1/priests/me/location") {
                header("Authorization", "Bearer $authToken")
                contentType(ContentType.Application.Json)
                setBody(PriestLocationPayload(latitude, longitude, capturedAt, accuracyM))
            }
            if (response.status.value in 200..299) Result.success(Unit) else Result.failure(Exception("Unable to update location"))
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun registerDeviceToken(token: String, platform: String = "android"): Result<Boolean> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.post("$baseUrl/api/v1/devices") {
                contentType(ContentType.Application.Json)
                header("Authorization", "Bearer $authToken")
                setBody(mapOf("device_token" to token, "platform" to platform))
            }
            if (response.status.value in 200..299) Result.success(true) else Result.failure(Exception("Failed to register device token"))
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    private suspend fun <T> postWithConflictRetry(
        url: String,
        idempotencyKey: String,
        body: Any,
        parseResponse: suspend (HttpResponse) -> T
    ): Result<T> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        var attempts = 0
        while (attempts < 3) {
            attempts++
            try {
                val response = client.post(url) {
                    contentType(ContentType.Application.Json)
                    header("Authorization", "Bearer $authToken")
                    header("Idempotency-Key", idempotencyKey)
                    setBody(body)
                }
                if (response.status.value in 200..299) {
                    return Result.success(parseResponse(response))
                }
                if (response.status.value == 409 && attempts < 3) {
                    val retryAfter = response.headers["Retry-After"]?.toLongOrNull() ?: 1L
                    delay(retryAfter.coerceAtLeast(1L) * 1000L)
                    continue
                }
                val errBody = try { response.bodyAsText() } catch (_: Exception) { "" }
                return Result.failure(Exception("HTTP ${response.status.value}: $errBody"))
            } catch (e: Exception) {
                return Result.failure(e)
            }
        }
        return Result.failure(Exception("Operation failed after retries"))
    }

    suspend fun updateBookingStatus(
        bookingId: String,
        status: String,
        priestId: String? = null,
        otp: String? = null,
        dakshina: Double = 0.0,
        idempotencyKey: String = generateIdempotencyKey()
    ): Result<String> {
        return postWithConflictRetry(
            url = "$baseUrl/api/v1/bookings/status",
            idempotencyKey = idempotencyKey,
            body = BookingStatusUpdatePayload(
                booking_id = bookingId,
                status = status,
                priest_id = priestId,
                completion_otp = otp,
                dakshina_amount = dakshina
            ),
            parseResponse = { it.body<String>() }
        )
    }

    suspend fun fetchBookingDetails(bookingId: String): Result<BookingDetailPayload> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.get("$baseUrl/api/v1/bookings/$bookingId") {
                header("Authorization", "Bearer $authToken")
            }
            Result.success(response.body<BookingDetailPayload>())
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    // ----- Event (Pushkaralu) -----

    suspend fun fetchEventConfig(): Result<EventConfig?> =
        eventCall<EventConfigResponse> { it.get("$baseUrl/api/v1/events/config") }.map { it.event }

    suspend fun fetchParticipation(eventId: String): Result<Participation> = eventCall {
        it.get("$baseUrl/api/v1/events/$eventId/participation/me") { header("Authorization", "Bearer $authToken") }
    }

    suspend fun applyForEvent(eventId: String): Result<Participation> = eventCall {
        it.post("$baseUrl/api/v1/events/$eventId/participation/me") { header("Authorization", "Bearer $authToken") }
    }

    suspend fun setEventServices(eventId: String, request: SetServicesRequest): Result<Participation> = eventCall {
        it.put("$baseUrl/api/v1/events/$eventId/services/me") {
            header("Authorization", "Bearer $authToken"); contentType(ContentType.Application.Json); setBody(request)
        }
    }

    suspend fun setEventDuty(eventId: String, request: DutyRequest): Result<DutyState> = eventCall {
        it.put("$baseUrl/api/v1/events/$eventId/duty/me") {
            header("Authorization", "Bearer $authToken"); contentType(ContentType.Application.Json); setBody(request)
        }
    }

    suspend fun endEventDuty(eventId: String): Result<Unit> = eventCall<Map<String, Boolean>> {
        it.delete("$baseUrl/api/v1/events/$eventId/duty/me") { header("Authorization", "Bearer $authToken") }
    }.map { }

    private suspend inline fun <reified T> eventCall(crossinline send: suspend (HttpClient) -> HttpResponse): Result<T> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = send(client)
            if (response.status.isSuccess()) Result.success(response.body<T>())
            else Result.failure(Exception(Regex("\"error\"\\s*:\\s*\"([a-z_]+)\"").find(response.bodyAsText())?.groupValues?.get(1) ?: "HTTP ${response.status.value}"))
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchBookingMaterials(bookingId: String): Result<List<BookingMaterialPayload>> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.get("$baseUrl/api/v1/bookings/$bookingId/materials") {
                header("Authorization", "Bearer $authToken")
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
                header("Authorization", "Bearer $authToken")
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
                header("Authorization", "Bearer $authToken")
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
                header("Authorization", "Bearer $authToken")
                setBody(
                    ChatSendPayload(
                        room_id = roomId,
                        sender_id = senderId,
                        sender_role = senderRole,
                        content = content
                    )
                )
            }
            if (response.status.value !in 200..299) return Result.failure(Exception("Unable to send message (HTTP ${response.status.value})"))
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
                header("Authorization", "Bearer $authToken")
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
                header("Authorization", "Bearer $authToken")
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
                header("Authorization", "Bearer $authToken")
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
                header("Authorization", "Bearer $authToken")
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
                header("Authorization", "Bearer $authToken")
            }
            Result.success(response.body<List<PriestCustomServicePayload>>())
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun createCustomService(
        payload: CreateCustomServicePayload,
        idempotencyKey: String = generateIdempotencyKey()
    ): Result<String> {
        return postWithConflictRetry(
            url = "$baseUrl/api/v1/priests/custom-services",
            idempotencyKey = idempotencyKey,
            body = payload,
            parseResponse = { it.body<String>() }
        )
    }

    fun chatUpdates(roomId: String): Flow<List<ChatMessagePayload>> = flow {
        var backoff = 1000L
        while (currentCoroutineContext().isActive && authToken.isNotBlank()) {
            try {
                val socketUrl = baseUrl.replaceFirst("https://", "wss://").replaceFirst("http://", "ws://")
                httpClient!!.webSocket(request = {
                    url("$socketUrl/ws/chat")
                    parameter("roomId", roomId)
                    header("Authorization", "Bearer $authToken")
                }) {
                    emit(fetchChatMessages(roomId).getOrThrow())
                    backoff = 1000L
                    for (frame in incoming) {
                        if (frame is Frame.Text) emit(fetchChatMessages(roomId).getOrThrow())
                    }
                }
            } catch (e: CancellationException) { throw e }
            catch (e: Exception) { /* Retain visible messages while reconnecting */ }
            delay(backoff + kotlin.random.Random.nextLong(500))
            backoff = (backoff * 2).coerceAtMost(30000L)
        }
    }

    suspend fun createPriestSlot(priestId: String, slotDate: String, startTime: String, endTime: String): Result<String> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.post("$baseUrl/api/v1/priests/slots") {
                contentType(ContentType.Application.Json)
                header("Authorization", "Bearer $authToken")
                setBody(CreatePriestSlotPayload(priestId, slotDate, startTime, endTime))
            }
            if (response.status.value in 200..299) Result.success(response.bodyAsText())
            else Result.failure(Exception("Failed to create slot (HTTP ${response.status.value})"))
        } catch (e: Exception) {
            Result.failure(e)
        }
    }
}
