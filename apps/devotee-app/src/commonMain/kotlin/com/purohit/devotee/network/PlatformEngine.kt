package com.purohit.devotee.network

import io.ktor.client.engine.HttpClientEngine

expect fun createPlatformHttpEngine(): HttpClientEngine
