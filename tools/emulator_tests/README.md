# Android booking text-size regression

Run from the repository root with one Android emulator connected and the Devotee debug app already signed in to a labeled staging test account:

```powershell
python tools/emulator_tests/booking_text_matrix.py
```

By default the script checks English, Telugu and Hindi at Android font scales 1.0, 1.3, 1.6 and 2.0. It selects a service and verifies that the booking action is visible and opens the priest-discovery/address screen. It accesses the backend configured in the installed app; for this project the current debug app targets Render staging. Confirm the target and signed-in account before running. It does not create, accept, cancel or pay for a booking.

A smaller run is useful while fixing a failure:

```powershell
python tools/emulator_tests/booking_text_matrix.py --scales 2.0 --languages te hi
```

Each case is reported as PASS or FAIL; any failure produces a nonzero exit status. The JSON report is written to `output/emulator-booking-matrix.json` by default. The script restores the original Android font scale. It force-stops and reopens the Devotee app, so do not run it with an active booking or unsaved work. It does not install or build an APK: build and install the intended version first. It requires Python 3, `adb` on PATH, one online emulator, network access and an already authenticated test account. It does not take screenshots or store credentials.

This is a focused regression check, not an exhaustive UI or accessibility audit. It does not verify the entire booking lifecycle, Priest app, every device/font combination, touch target size, content clipping, contrast, TalkBack speech, keyboard navigation or translations. Continue to perform those checks separately. Test failure can also indicate an unavailable service, expired session, missing UI hierarchy or changed labels; inspect the failure reason instead of treating those cases as passes.
