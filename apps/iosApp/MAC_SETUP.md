# Running Pooja Vahini on iPhone (Mac setup)

This guide is for whoever builds the iOS apps on a Mac. Both iOS apps (Devotee and Priest) use the same Kotlin screens as Android, so the regular booking flow and Pushkaralu run on iOS without separate Swift screens. The Swift code in `DevoteeApp/` and `PriestApp/` is only a small launcher.

The apps talk to Render staging: `https://pooja-vahini-staging.onrender.com`. The first request after the server has been idle can take about a minute.

## 1. One-time installs

| Tool | How | Check |
| --- | --- | --- |
| Xcode | App Store. Open it once and accept the licence. | `xcodebuild -version` |
| Xcode command line tools | `xcode-select --install` | `xcode-select -p` |
| Homebrew | https://brew.sh | `brew --version` |
| JDK 17 | `brew install --cask temurin@17` | `/usr/libexec/java_home -v 17` prints a path |
| XcodeGen | `brew install xcodegen` | `xcodegen --version` |

Use JDK **17**. Newer JDKs may not work with this project's Gradle version.

## 2. Get the code and generate the Xcode project

```bash
cd path/to/the/repo
chmod +x gradlew
cd apps/iosApp
xcodegen generate
open PujaVaahini.xcodeproj
```

Run `xcodegen generate` again whenever `project.yml` changes. Do not edit the generated `.xcodeproj` by hand.

## 3. First build (simulator)

1. In Xcode, pick the **DevoteeApp** scheme and an iPhone simulator (for example iPhone 15).
2. Press **Run** (⌘R).
3. The first build runs Gradle and compiles the Kotlin code for iOS. It can take **10 to 20 minutes** and needs internet. Later builds are much faster.
4. Repeat with the **PriestApp** scheme.

Optional: check the Kotlin part on its own from Terminal before opening Xcode. It shows Kotlin errors more clearly:

```bash
./gradlew :apps:devotee-app:linkDebugFrameworkIosSimulatorArm64 :apps:priest-app:linkDebugFrameworkIosSimulatorArm64
```

## 4. Running on a real iPhone

1. Connect the iPhone and trust the Mac.
2. In Xcode, select the target, open **Signing & Capabilities**, tick **Automatically manage signing** and choose your Apple ID as **Team** (a free Apple ID works for testing).
3. If Xcode says the bundle identifier is taken, change it to something unique, such as `com.yourname.poojavahini.devotee`.
4. On the phone, open Settings, then General, then VPN & Device Management, and trust your developer certificate.
5. A free Apple ID keeps the app working for 7 days, then you rebuild.

## 5. What to test

**Devotee app**
1. Register or log in, choose a pooja, request a priest (the regular booking flow).
2. Change language (English, Telugu, Hindi) and the Appearance theme.
3. Pushkaralu: appears only while an event is live in the CMS. Open the Pushkaralu tab, pick a ghat, pick a priest, send a request, and see the booking code screen.

**Priest app**
1. Log in, go online, accept a request, then move it through Arrived, Start (enter the devotee's booking code for ghat bookings) and Complete.
2. Pushkaralu tab: apply, set prices, and (after CMS approval) start duty at a ghat. iOS asks for location permission. Choose **Allow While Using App**; iOS may later ask to allow **Always**, which keeps location sharing working with the screen locked.

**Location on the simulator:** the simulator has no GPS. In the Simulator menu choose **Features > Location > Custom Location** and enter a point near the ghat (for example 17.0005, 81.8040). For the priest, background location (screen locked, app in background) can only be tested properly on a **real iPhone**.

## 6. Known limitations on iOS

- **No push notifications yet.** Push needs an Apple Developer account (paid, $99/year) with an APNs key added to Firebase. Until then the apps recheck the event config when opened, when brought back to the foreground, and every 10 minutes. CMS changes therefore reach iPhones within about 10 minutes instead of instantly. New booking requests still reach the priest app through its regular polling.
- **Kotlin 1.9.22 and new Xcode versions.** This project uses Kotlin 1.9.22, which was released for Xcode 15. A newer Xcode usually works but prints a compatibility warning. If the build fails in the Kotlin step with an Xcode or SDK version error, install Xcode 15.4 alongside (`brew install xcodes && xcodes install 15.4`), select it with `sudo xcode-select -s /Applications/Xcode-15.4.app`, and build again. Report it either way, because the real fix is upgrading Kotlin.
- Vendor app is out of scope for launch.

## 7. If the build fails, send back

1. A screenshot of the first red error in Xcode's **Report navigator** (the last icon in the left sidebar), expanded.
2. The output of the Terminal command in step 3 (`./gradlew ... linkDebugFrameworkIosSimulatorArm64`), especially lines starting with `e:`.
3. The output of `xcodebuild -version` and `java -version`.

Most first-build errors are small and are fixed in the shared Kotlin code, not in Swift.
