# Purohit Platform - iOS Application Wrapper

This folder contains the native iOS SwiftUI wrapper hosting the Compose Multiplatform UI (`DevoteeApp` and `PriestApp`).

## 🛠️ How to Run in Xcode (macOS)
1. Ensure Xcode is installed.
2. Build the shared Kotlin Multiplatform framework:
   ```bash
   ./gradlew :apps:devotee-app:embedAndSignAppleFrameworkForXcode
   ```
   or for Priest App:
   ```bash
   ./gradlew :apps:priest-app:embedAndSignAppleFrameworkForXcode
   ```
3. Open `iosApp.xcodeproj` or import this directory into Xcode.
4. Select an iOS Simulator (e.g. iPhone 15 Pro) and click **Run (⌘R)**.
