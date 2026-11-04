rootProject.name = "puja-vaahini"

pluginManagement {
    repositories {
        google()
        mavenCentral()
        gradlePluginPortal()
    }
}

dependencyResolutionManagement {
    repositories {
        google()
        mavenCentral()
    }
}

include(":apps:common-contracts")
include(":apps:devotee-app")
include(":apps:priest-app")
include(":apps:vendor-app")
