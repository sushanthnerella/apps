# iOS hosts

Step-by-step instructions for building and testing on a Mac: [MAC_SETUP.md](MAC_SETUP.md).

On macOS with Xcode and XcodeGen installed, run `xcodegen generate` in this directory and open `PujaVaahini.xcodeproj`.
Select DevoteeApp, PriestApp or VendorApp. Each target links its own Kotlin framework and invokes the matching Gradle build task. Set your signing team in Xcode for device builds.

The legacy `iosApp/` directory is not included in the generated project. Only the three role-specific directories are compiled.
Windows cannot compile or validate Apple frameworks. These hosts require a macOS build and device test before release.

Specification reference: https://github.com/yonaskolb/XcodeGen/blob/master/Docs/ProjectSpec.md
