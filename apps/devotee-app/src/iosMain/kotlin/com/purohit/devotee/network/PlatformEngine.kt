package com.purohit.devotee.network

import io.ktor.client.engine.HttpClientEngine
import io.ktor.client.engine.darwin.Darwin

actual fun createPlatformHttpEngine(): HttpClientEngine {
    return Darwin.create()
}
