package com.purohit.devotee.network

import io.ktor.client.engine.HttpClientEngine
import io.ktor.client.engine.okhttp.OkHttp
import java.util.concurrent.TimeUnit

actual fun createPlatformHttpEngine(): HttpClientEngine {
    return OkHttp.create {
        config {
            connectTimeout(45, TimeUnit.SECONDS)
            readTimeout(45, TimeUnit.SECONDS)
            writeTimeout(45, TimeUnit.SECONDS)
        }
    }
}
