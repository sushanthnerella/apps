package com.purohit.devotee.network

import com.purohit.contracts.idempotency.IdempotencyKeyGenerator

import com.purohit.contracts.store.CreateStoreOrderRequest
import com.purohit.contracts.store.StoreOrderItemDraft
import com.purohit.contracts.store.Vendor
import com.purohit.contracts.store.VendorInventoryItem
import com.purohit.contracts.temple.Temple
import com.purohit.contracts.temple.TempleVipPuja
import com.purohit.contracts.event.EventConfig
import com.purohit.contracts.event.EventConfigResponse
import com.purohit.contracts.event.CreateEventBookingRequest
import com.purohit.contracts.event.EventBookingResponse
import com.purohit.contracts.event.GhatAvailability
import com.purohit.contracts.event.GhatPriest
import io.ktor.client.*
import io.ktor.client.plugins.websocket.*
import io.ktor.websocket.*
import kotlinx.coroutines.*
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.flow
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

/** Result of one config fetch: [notModified] means keep the cached config. */
data class EventConfigFetch(val event: EventConfig?, val etag: String?, val notModified: Boolean)

/** A refused event API call; [code] is the server's error code (see EventErrorCodes). */
class EventApiException(val httpStatus: Int, val code: String) : Exception("HTTP $httpStatus $code")

internal fun eventErrorCode(body: String): String =
    Regex("\"error\"\\s*:\\s*\"([a-z_]+)\"").find(body)?.groupValues?.get(1) ?: ""

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
    val scheduled_date: String = "",
    val slot_time: String = "",
    val latitude: Double = 17.3850,
    val longitude: Double = 78.4867,
    val chat_room_id: String = "",
    val cancellation_reason: String = ""
)

@Serializable
data class BookingOtpPayload(
    val booking_id: String = "",
    val completion_otp: String = "",
    val status: String = ""
)

@Serializable
data class UpdateBookingMaterialsRequest(
    val materials: List<BookingMaterialPayload>
)

@Serializable
data class DevoteeLoginPayload(
    val phone: String,
    val otp: String = "",
    val password: String = "",
    val role: String = "devotee"
)

@Serializable
data class ResetPasswordPayload(
    val phone: String,
    val otp: String = "",
    val new_password: String,
    val role: String = "devotee"
)

@Serializable
data class RecoveryStartPayload(
    val email: String = "",
    val phone: String = "",
    val role: String = "devotee"
)

@Serializable
data class RecoveryStartResponse(
    val status: String = "",
    val message: String = "",
    val cooldown_seconds: Int = 60
)

@Serializable
data class RecoveryConfirmPayload(
    val phone: String,
    val role: String = "devotee",
    val code: String,
    val new_password: String,
    val confirm_password: String
)

@Serializable
data class RecoveryConfirmResponse(
    val status: String = "",
    val message: String = ""
)

@Serializable
data class DevoteeRegistrationPayload(
    val full_name: String,
    val phone: String,
    val password: String = "",
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
data class DevoteeProfilePayload(
    val id: String = "",
    val phone: String = "",
    val full_name: String = "",
    val gotram: String = "",
    val nakshatram: String = "",
    val rashi: String = "",
    val city: String = "Hyderabad",
    val role: String = "devotee",
    val consent_agreed: Boolean = true,
    val email: String = "",
    val email_verified: Boolean = false
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
    val phone: String = "",
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
    val slot_id: String? = null,
    val hold_token: String? = null,
    val scheduled_date: String = "Today",
    val slot_time: String = "08:30 AM - 10:30 AM",
    val total_amount: Double,
    val total_amount_minor: Long = (total_amount * 100).toLong(),
    val currency: String = "INR",
    val booking_type: String = "INSTANT_DISPATCH",
    val event_address: String = "",
    val devotee_gotram: String = "",
    val devotee_nakshatram: String = "",
    val devotee_rashi: String = "",
    val sankalpam_purpose: String = "Family Prosperity & Health"
)

@Serializable
data class SlotHoldPayload(val slot_id: String, val devotee_id: String, val hold_duration_seconds: Int)

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
    val total_amount_minor: Long = (total_amount * 100).toLong(),
    val currency: String = "INR",
    val delivery_address: String = "",
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
    val total_amount_minor: Long = 0L,
    val currency: String = "INR",
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
    val amount_paid: Double,
    val amount_paid_minor: Long = (amount_paid * 100).toLong(),
    val currency: String = "INR"
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
    // Configured via centralized BackendConfig (EMULATOR / LOCAL / PILOT)
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
    var currentProfile: DevoteeProfilePayload? = null

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
                    header("User-Agent", "PujaVaahiniDevotee/1.0")
                }
            }
        } catch (t: Throwable) {
            throw IllegalStateException("Unable to initialize HTTP client", t)
        }
    }

    /** Reconnect with bounded backoff; reload history after every connection to fill gaps. */
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
            catch (e: Exception) { /* Retain visible messages while reconnecting. */ }
            delay(backoff + kotlin.random.Random.nextLong(500))
            backoff = (backoff * 2).coerceAtMost(30000L)
        }
    }

    suspend fun registerDevotee(
        fullName: String,
        phone: String,
        password: String = "",
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
                setBody(
                    DevoteeRegistrationPayload(
                        full_name = fullName,
                        phone = phone,
                        password = password,
                        gotram = gotram,
                        nakshatram = nakshatram,
                        rashi = rashi,
                        city = city.trim(),
                        consent_agreed = consentAgreed
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
                if (response.status.value == 401) authToken = ""
                val errText = response.bodyAsText()
                Result.failure(Exception("Server returned ${response.status.value}: $errText"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun login(phone: String, otp: String = "", password: String = ""): Result<AuthResponsePayload> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.post("$baseUrl/api/v1/auth/login") {
                contentType(ContentType.Application.Json)
                setBody(DevoteeLoginPayload(phone = phone, otp = otp, password = password, role = "devotee"))
            }
            if (response.status.value in 200..299) {
                val authRes = response.body<AuthResponsePayload>()
                if (authRes.access_token.isNotBlank()) {
                    authToken = authRes.access_token
                    currentProfile = authRes.user
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

    suspend fun requestPasswordReset(email: String, role: String = "devotee"): Result<RecoveryStartResponse> {
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

    suspend fun startRecovery(phone: String = "", email: String = "", role: String = "devotee"): Result<RecoveryStartResponse> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.post("$baseUrl/api/v1/auth/recovery/start") {
                contentType(ContentType.Application.Json)
                setBody(RecoveryStartPayload(email = email, phone = phone, role = role))
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

    suspend fun confirmRecovery(
        phone: String,
        role: String = "devotee",
        code: String,
        newPassword: String,
        confirmPassword: String = newPassword
    ): Result<RecoveryConfirmResponse> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.post("$baseUrl/api/v1/auth/recovery/confirm") {
                contentType(ContentType.Application.Json)
                setBody(
                    RecoveryConfirmPayload(
                        phone = phone,
                        role = role,
                        code = code,
                        new_password = newPassword,
                        confirm_password = confirmPassword
                    )
                )
            }
            if (response.status.value in 200..299) {
                Result.success(response.body<RecoveryConfirmResponse>())
            } else {
                val errText = response.bodyAsText()
                Result.failure(Exception(errText))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun resetPassword(phone: String, otp: String = "", newPassword: String): Result<String> {
        return confirmRecovery(
            phone = phone,
            role = "devotee",
            code = otp,
            newPassword = newPassword,
            confirmPassword = newPassword
        ).map { it.message }
    }

    suspend fun fetchDevoteeBookings(userId: String): Result<List<BookingDetailPayload>> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.get("$baseUrl/api/v1/bookings") {
                parameter("user_id", userId)
                header("Authorization", "Bearer $authToken")
            }
            if (response.status.value !in 200..299) return Result.failure(Exception("Unable to load bookings (HTTP ${response.status.value})"))
            Result.success(response.body<List<BookingDetailPayload>>())
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchProfile(): Result<DevoteeProfilePayload> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.get("$baseUrl/api/v1/user/profile") {
                header("Authorization", "Bearer $authToken")
            }
            if (response.status.value in 200..299) {
                val profile = response.body<DevoteeProfilePayload>()
                currentProfile = profile
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
            }
            Result.success(response.body<List<PoojaCatalogItem>>())
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchNearbyPriests(lat: Double, lng: Double, service: String = ""): Result<List<NearbyPriestItem>> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.get("$baseUrl/api/v1/priests/nearby") {
                parameter("lat", lat)
                parameter("lng", lng)
                parameter("service", service)
            }
            if (response.status.value !in 200..299) return Result.failure(Exception("Unable to load nearby priests"))
            if (response.status.value !in 200..299) return Result.failure(Exception("Unable to load nearby priests"))
            val res = response.body<NearbyPriestsResponse>()
            Result.success(res.priests)
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
            }
            if (response.status.value !in 200..299) return Result.failure(Exception("Unable to load slots"))
            val raw = response.bodyAsText()
            Result.success(if (raw.trim() == "null") emptyList() else Json { ignoreUnknownKeys = true }.decodeFromString<List<PriestSlotPayload>>(raw))
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

    suspend fun holdPriestSlot(slotId: String, durationSeconds: Int = 600, idempotencyKey: String): Result<Boolean> {
        val userId = currentProfile?.id ?: "devotee_direct"
        return postWithConflictRetry(
            url = "$baseUrl/api/v1/priests/slots/hold",
            idempotencyKey = idempotencyKey,
            body = SlotHoldPayload(slotId, userId, durationSeconds),
            parseResponse = { true }
        )
    }

    suspend fun createBooking(
        priestId: String,
        poojaId: String = "1",
        totalAmount: Double,
        scheduledDate: String = "Today",
        slotTime: String = "08:30 AM - 10:30 AM",
        slotId: String? = null,
        eventAddress: String,
        gotram: String = "",
        nakshatram: String = "",
        rashi: String = "",
        purpose: String = "Family Prosperity & Health",
        idempotencyKey: String
    ): Result<BookingResponsePayload> {
        val userId = currentProfile?.id ?: "devotee_direct"
        val userGotram = if (gotram.isNotBlank()) gotram else (currentProfile?.gotram ?: "")
        val userNakshatram = if (nakshatram.isNotBlank()) nakshatram else (currentProfile?.nakshatram ?: "")
        val userRashi = if (rashi.isNotBlank()) rashi else (currentProfile?.rashi ?: "")
        return postWithConflictRetry(
            url = "$baseUrl/api/v1/bookings",
            idempotencyKey = idempotencyKey,
            body = BookingCreatePayload(
                user_id = userId,
                priest_id = priestId,
                pooja_id = poojaId,
                slot_id = slotId,
                event_address = eventAddress,
                scheduled_date = scheduledDate,
                slot_time = slotTime,
                total_amount = totalAmount,
                devotee_gotram = userGotram,
                devotee_nakshatram = userNakshatram,
                devotee_rashi = userRashi,
                sankalpam_purpose = purpose
            ),
            parseResponse = { it.body<BookingResponsePayload>() }
        )
    }

    /** Fetches live event config. Sends the last ETag so an unchanged config costs a 304 with no body. */
    suspend fun fetchEventConfig(etag: String?): Result<EventConfigFetch> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.get("$baseUrl/api/v1/events/config") {
                if (!etag.isNullOrBlank()) header(HttpHeaders.IfNoneMatch, etag)
            }
            when (response.status) {
                HttpStatusCode.NotModified -> Result.success(EventConfigFetch(null, etag, notModified = true))
                HttpStatusCode.OK -> Result.success(
                    EventConfigFetch(response.body<EventConfigResponse>().event, response.headers[HttpHeaders.ETag], notModified = false)
                )
                else -> Result.failure(Exception("Event config unavailable (${response.status.value})"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchGhatAvailability(eventId: String): Result<List<GhatAvailability>> = eventCall {
        it.get("$baseUrl/api/v1/events/$eventId/ghats/availability")
    }

    suspend fun fetchGhatPriests(eventId: String, ghatId: String): Result<List<GhatPriest>> = eventCall {
        it.get("$baseUrl/api/v1/events/$eventId/ghats/$ghatId/priests") { header("Authorization", "Bearer $authToken") }
    }

    /** Creates a ghat booking. The same [idempotencyKey] must be reused when retrying the same tap. */
    suspend fun createEventBooking(eventId: String, request: CreateEventBookingRequest, idempotencyKey: String): Result<EventBookingResponse> = eventCall {
        it.post("$baseUrl/api/v1/events/$eventId/bookings") {
            header("Authorization", "Bearer $authToken")
            header("Idempotency-Key", idempotencyKey)
            contentType(ContentType.Application.Json)
            setBody(request)
        }
    }

    private suspend inline fun <reified T> eventCall(crossinline send: suspend (HttpClient) -> HttpResponse): Result<T> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = send(client)
            if (response.status.isSuccess()) Result.success(response.body<T>())
            else Result.failure(EventApiException(response.status.value, eventErrorCode(response.bodyAsText())))
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchTemples(): Result<List<Temple>> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.get("$baseUrl/api/v1/temples") {
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
            }
            Result.success(response.body<List<TempleVipPuja>>())
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun bookTempleTicket(
        templeId: String,
        vipPujaId: String,
        amount: Double,
        idempotencyKey: String
    ): Result<TempleTicketResponse> {
        val profile = currentProfile
        return postWithConflictRetry(
            url = "$baseUrl/api/v1/temples/tickets/book",
            idempotencyKey = idempotencyKey,
            body = TempleTicketBookingPayload(
                temple_id = templeId,
                vip_puja_id = vipPujaId,
                user_id = profile?.id ?: "",
                devotee_name = profile?.full_name ?: "Devotee",
                devotee_phone = profile?.phone ?: "+91 99999 99999",
                devotee_gotram = profile?.gotram ?: "",
                devotee_nakshatram = profile?.nakshatram ?: "",
                amount_paid = amount
            ),
            parseResponse = { it.body<TempleTicketResponse>() }
        )
    }

    suspend fun fetchVendors(latitude: Double, longitude: Double): Result<List<Vendor>> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.get("$baseUrl/api/v1/store/vendors") {
                parameter("lat", latitude)
                parameter("lng", longitude)
            }
            val items = response.body<List<Vendor>?>() ?: emptyList()
            Result.success(items)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchVendorInventory(vendorId: String): Result<List<VendorInventoryItem>> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.get("$baseUrl/api/v1/store/vendors/$vendorId/inventory") {
            }
            val items = response.body<List<VendorInventoryItem>?>() ?: emptyList()
            Result.success(items)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun createStoreOrder(
        vendorId: String,
        items: List<StoreOrderItemDraft>,
        idempotencyKey: String
    ): Result<StoreOrderResponse> {
        return postWithConflictRetry(
            url = "$baseUrl/api/v1/store/orders",
            idempotencyKey = idempotencyKey,
            body = CreateStoreOrderRequest(
                userId = currentProfile?.id ?: "devotee_store",
                vendorId = vendorId,
                deliveryAddress = currentProfile?.city.orEmpty(),
                items = items
            ),
            parseResponse = { it.body<StoreOrderResponse>() }
        )
    }

    suspend fun fetchPoojaSamagri(poojaId: String): Result<List<SamagriItemPayload>> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.get("$baseUrl/api/v1/poojas/$poojaId/samagri") {
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
                header("Authorization", "Bearer $authToken")
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
        senderRole: String,
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
                header("Authorization", "Bearer $authToken")
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
            if (response.status.value in 200..299) {
                Result.success(response.body<String>())
            } else {
                val errBody = try { response.body<String>() } catch (_: Exception) { "" }
                Result.failure(Exception("HTTP ${response.status.value}: $errBody"))
            }
        } catch (e: Exception) {
            Result.failure(e)
        }
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

    suspend fun fetchBookingOtp(bookingId: String): Result<BookingOtpPayload> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.get("$baseUrl/api/v1/bookings/$bookingId/otp") {
                header("Authorization", "Bearer $authToken")
            }
            Result.success(response.body<BookingOtpPayload>())
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun fetchCancellationQuote(bookingId: String): Result<CancellationQuotePayload> {
        val client = httpClient ?: return Result.failure(Exception("HTTP client engine unavailable"))
        return try {
            val response = client.get("$baseUrl/api/v1/bookings/$bookingId/cancel-quote") {
                header("Authorization", "Bearer $authToken")
            }
            Result.success(response.body<CancellationQuotePayload>())
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    suspend fun cancelBooking(
        bookingId: String,
        reasonCode: String,
        reasonText: String = "",
        idempotencyKey: String
    ): Result<CancelBookingResponsePayload> {
        return postWithConflictRetry(
            url = "$baseUrl/api/v1/bookings/cancel",
            idempotencyKey = idempotencyKey,
            body = CancelBookingRequestPayload(
                booking_id = bookingId,
                cancelled_by_role = "devotee",
                reason_code = reasonCode,
                reason_text = reasonText
            ),
            parseResponse = { it.body<CancelBookingResponsePayload>() }
        )
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
}
