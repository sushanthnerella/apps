package com.purohit.contracts.config

/**
 * Authoritative backend configuration for Puja Vaahini client applications.
 * Separates local development, local emulator testing, and remote pilot/staging endpoints.
 * Enforces HTTPS for remote staging/pilot environments and prevents silent fallback to 10.0.2.2.
 */
object BackendConfig {
    /**
     * Local workstation gateway host port (desktop JVM / direct host loopback)
     */
    const val LOCAL_HOST_URL = "http://localhost:8085"

    /**
     * Standard Android emulator alias for host machine loopback
     */
    const val EMULATOR_HOST_URL = "http://10.0.2.2:8085"

    /**
     * Supported environment names
     */
    const val ENV_EMULATOR = "EMULATOR"
    const val ENV_LOCAL = "LOCAL"
    const val ENV_STAGING = "STAGING"
    const val ENV_PILOT = "PILOT"

    /**
     * Active environment selector:
     * - "EMULATOR": for Android emulators accessing local backend (default for local debug builds)
     * - "LOCAL": for desktop JVM / direct host loopback
     * - "STAGING" / "PILOT": for remote staging devices connecting via verified HTTPS URL
     */
    var currentEnvironment: String = ENV_STAGING

    /**
     * Remote staging/pilot HTTPS endpoint supplied by deployment owner / build configuration.
     * Must be explicitly configured when targeting STAGING or PILOT environments.
     */
    private var stagingHttpsUrl: String? = "https://pooja-vahini-staging.onrender.com"

    /**
     * Optional manual override URL for debugging or dynamic configuration
     */
    var customOverrideUrl: String? = null

    /**
     * Configures the remote staging/pilot HTTPS URL.
     * Enforces that remote endpoints must use HTTPS.
     */
    fun setStagingUrl(url: String) {
        val trimmed = url.trim()
        require(trimmed.startsWith("https://", ignoreCase = true)) {
            "Staging endpoint must use HTTPS scheme for security (got '$url')"
        }
        stagingHttpsUrl = trimmed
    }

    /**
     * Retrieves the currently configured staging URL (if any).
     */
    fun getStagingUrl(): String? = stagingHttpsUrl

    /**
     * Clears the configured staging URL (for testing or resets).
     */
    fun clearStagingUrl() {
        stagingHttpsUrl = null
    }

    /**
     * Dynamic base URL resolution based on active environment or manual override.
     * Throws IllegalStateException if STAGING/PILOT is selected without a configured HTTPS staging URL,
     * ensuring staging builds never silently fall back to emulator 10.0.2.2 or outdated tunnels.
     */
    val activeBaseUrl: String
        get() {
            customOverrideUrl?.let { override ->
                val env = currentEnvironment.uppercase()
                if (env == ENV_STAGING || env == ENV_PILOT) {
                    check(override.startsWith("https://", ignoreCase = true)) {
                        "Staging override URL must use HTTPS (got '$override')"
                    }
                }
                return override
            }

            return when (val env = currentEnvironment.uppercase()) {
                ENV_EMULATOR -> EMULATOR_HOST_URL
                ENV_LOCAL -> LOCAL_HOST_URL
                ENV_STAGING, ENV_PILOT -> {
                    stagingHttpsUrl ?: error(
                        "Remote $env environment requires an explicit HTTPS staging URL. " +
                        "Call BackendConfig.setStagingUrl(\"https://<your-railway-app>.up.railway.app\") before making network calls."
                    )
                }
                else -> error("Unknown environment '$env'. Expected EMULATOR, LOCAL, STAGING, or PILOT.")
            }
        }
}
