plugins {
    id("com.google.gms.google-services") version "4.6.0" apply false
    alias(libs.plugins.androidApplication) apply false
    alias(libs.plugins.androidLibrary) apply false
    alias(libs.plugins.kotlinMultiplatform) apply false
    alias(libs.plugins.kotlinSerialization) apply false
    alias(libs.plugins.jetbrainsCompose) apply false
}

allprojects {
    repositories {
        google()
        mavenCentral()
    }
}
