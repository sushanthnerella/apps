# One-Click Android Emulator Launcher & App Runner
Write-Host "==========================================================" -ForegroundColor Cyan
Write-Host "   Puja Vaahini - Emulator & Apps Runner" -ForegroundColor Yellow
Write-Host "==========================================================" -ForegroundColor Cyan

$sdkPath = "$env:LOCALAPPDATA\Android\Sdk"
$emulatorExe = "$sdkPath\emulator\emulator.exe"
$adbExe = "$sdkPath\platform-tools\adb.exe"

if (-not (Test-Path $emulatorExe)) {
    Write-Host "❌ Android SDK Emulator not found at: $emulatorExe" -ForegroundColor Red
    exit 1
}

# 1. Check if emulator is already running
$device = & $adbExe devices | Where-Object { $_ -match "emulator-\d+\s+device" }

if (-not $device) {
    Write-Host "📱 Starting Android Emulator (Medium_Phone)..." -ForegroundColor Green
    Start-Process -FilePath $emulatorExe -ArgumentList "-avd", "Medium_Phone"
    Write-Host "⏳ Waiting for emulator to boot up (this may take 20-30 seconds)..." -ForegroundColor Yellow
    & $adbExe wait-for-device
    
    # Wait until boot completed
    do {
        Start-Sleep -Seconds 3
        $boot = (& $adbExe shell getprop sys.boot_completed).Trim()
    } while ($boot -ne "1")
    Write-Host "✅ Emulator is online and ready!" -ForegroundColor Green
} else {
    Write-Host "✅ Emulator is already running!" -ForegroundColor Green
}

# 2. Install Devotee App
$devoteeApk = "E:\New folder (5)\apps\devotee-app\build\outputs\apk\debug\devotee-app-debug.apk"
Write-Host "`n📦 Installing Devotee App..." -ForegroundColor Cyan
& $adbExe install -r $devoteeApk

# 3. Install Priest App
$priestApk = "E:\New folder (5)\apps\priest-app\build\outputs\apk\debug\priest-app-debug.apk"
Write-Host "`n📦 Installing Priest App..." -ForegroundColor Cyan
& $adbExe install -r $priestApk

# 4. Launch Devotee App
Write-Host "`n🚀 Launching Devotee App on Emulator..." -ForegroundColor Green
& $adbExe shell am start -n com.purohit.devotee/.MainActivity

Write-Host "`n🎉 Done! Both apps are installed on your emulator." -ForegroundColor Yellow
Write-Host "   - Devotee App: Launched on screen" -ForegroundColor White
Write-Host "   - Priest App: Available in the app drawer (or run: adb shell am start -n com.purohit.priest/.MainActivity)" -ForegroundColor White
