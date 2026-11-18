package com.purohit.priest.network

import io.ktor.client.engine.HttpClientEngine

expect fun createPlatformHttpEngine(): HttpClientEngine
