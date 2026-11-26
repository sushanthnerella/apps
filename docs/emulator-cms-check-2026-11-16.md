# Emulator and CMS smoke test — 2026-11-16

Environment: Codex_Puja_Test_35 Android emulator; rebuilt debug APKs; local API on 8085, Next.js CMS on 3000, isolated PostgreSQL database pv_ui_20261116. Local-test APKs temporarily enabled emulator HTTP; all source endpoint/manifest settings were restored afterward. No staging records or real notifications used.

PASS — CMS browser UI:
- Admin login; event creation as Draft; transition to Scheduled and Active.
- Ghat and ritual creation; hide/show switches.
- Priest approval; participation and ritual changes visible in Recent changes.
- Booking and card-issuance switches.
- Emergency stop sets both switches off; existing event request remains REQUESTED.

PASS — Devotee emulator UI:
- Login and live event strip.
- Ghat list and availability; priest list and price.
- Request screen; scheduled date/time choices render.
- Immediate request creates a real local booking and shows booking code/countdown/price.
- Screenshots visually checked for these views; no obvious clipping at default size.

FAIL — Priest emulator UI:
- Login works and scheduled-event Pushkaralu tab is visible.
- Opening Pushkaralu crashes the app: java.lang.ArrayIndexOutOfBoundsException, length=10240; index=-5, in androidx.compose.runtime.SlotTableKt.key during recomposition.
- Saved Android crash log. Root cause is not yet established; no crash repair attempted in this smoke-test pass.
- Because the priest screen crashed, its application, prices and duty data were prepared through local API/SQL fixtures to continue CMS/devotee testing. These priest UI actions are NOT claimed as passing.

Not tested: full priest lifecycle, scheduled booking submission, location permissions/background updates, all languages/font sizes, iOS.

Evidence: C:/Users/susha/OneDrive/Documents/ChatGPT/pooja vahini/output/ui-20261116 (devotee-flow.png, cms-approved-active.png, cms-emergency-stop.png, priest-crash.txt). Do not publish cms-session.json; it contains the temporary local browser session.

## Priest crash recheck — 2026-11-17
Rebuilt and installed the updated priest debug APK against the isolated local backend. Approved participation and price loaded successfully. Tapped Save prices, navigated to Duty and back, force-stopped/relaunched, and reopened Pushkaralu. The screen remained functional; no new Android crash-log entries. Screenshot: output/ui-20261116/priest-fixed-20261117.png in the handoff workspace. This is a focused crash regression check, not a complete duty/application lifecycle test. Temporary local HTTP/endpoint settings were restored.
