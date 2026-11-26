# World-Ready Plan (multi-country, multi-currency, multi-region, virtual pooja)

Status: PLAN ONLY. Nothing here is implemented.
Written: 2026-11-17, against branch `feat/pushkaralu-event-mode` (commit `2c6e7d7` plus uncommitted working tree).

**Do not start before the Pushkaralu launch on 1 Dec 2026.** Phases 1-5 touch money and auth paths.

## How to use this document (for an AI or human implementer)

- Work one phase at a time, in the order given in [Order of work](#order-of-work). Each phase is one PR.
- Every phase lists: goal, exact files and lines to change, the change, and the tests that must pass.
- Line numbers were read on 2026-11-17. If a line has drifted, search for the quoted string next to it; the string is the real anchor.
- Write the tests listed for a phase in the same PR. A phase is done only when its tests pass and the "Done when" grep returns nothing.
- Do not add anything not listed. Do not convert currencies, do not move money between currencies, do not fork code per country.
- Backend tests: `cd backend && go test ./...`. App tests: `./gradlew :apps:common-contracts:allTests :apps:devotee-app:allTests :apps:priest-app:allTests`.

## Facts about the code today

| Area | Today | Where |
|---|---|---|
| Priest discovery | Already global. Pure lat/long radius query, no country logic | `backend/internal/db/discovery.go:14-139` |
| Radius | Default 50 km, hard cap 100 km, dispatch default 5 km | `discovery.go:21-26`, `backend/cmd/api-gateway/main.go:286`, `main.go:352` |
| Currency | `"INR"` hardcoded in 8 backend files and as default in 9 Kotlin model files | see Phase 2 |
| Money storage | Minor units (`*_minor` BIGINT) already exist next to legacy float columns | migration `000012_money_minor_units.sql` |
| Phone | Normaliser assumes India for any number without `+` | `backend/internal/auth/phone.go:46-59` |
| Login | Phone + password. Recovery is by email. There is no SMS OTP provider | `main.go:154`, `backend/internal/db/recovery.go` |
| Timezone | IST hardcoded on backend (1 place) and apps (3 places) | see Phase 4 |
| Payments | No payment gateway code. Bookings use `escrow_status`, events use `PAY_AT_GHAT` | `backend/internal/db/event_ops.go:575,583` |
| Virtual pooja | Partly built: catalog flag, temple video-call puja, WebRTC signalling, TURN server. Devotee entry point is a stub | see Phase 7 |
| Deploy | One Docker Compose stack: Postgres+PostGIS, Redis, coturn, api-gateway, temple-cms | `docker-compose.prod.yml` |
| JWT | HS256, claims: user_id, phone, role, full_name, session_version | `backend/internal/auth/jwt.go:87-94` |
| Migrations | Last is `000018_event_participation_and_bookings.sql`. Next free number: `000019` | `backend/migrations/` |

## Product positioning (why this order)

Competitors abroad (Panditji On Way, 99Pandit, SmartPuja) are India-run listing sites; Sri Mandir and VAMA sell remote temple pujas to NRIs. Nobody found does live "priest near me" dispatch outside India.

We offer **both**:

1. **In-person** priest at home, matched by location. Needs local priest supply in each country.
2. **Virtual pooja** over video (same thing Sri Mandir/VAMA sell). Needs no local supply: a priest in India can serve a devotee in the UK from day one.

So virtual pooja is the launch product in every new country, and in-person follows as local priests onboard. That is why Phase 7 (virtual) comes before multi-region and before local payments.

## Order of work

| Phase | What | Depends on |
|---|---|---|
| 1 | Country config table and `country_code` on accounts and bookings | - |
| 2 | Currency: remove every hardcoded `INR` and `₹` | 1 |
| 3 | Phone: stop assuming `+91` | 1 |
| 4 | Timezone: remove hardcoded IST | 1 |
| 5 | Discovery radius and refund fees per country | 1 |
| 6 | Provider interfaces (payment, KYC, tax, notification) | 1 |
| 7 | Virtual pooja, cross-country | 2, 3, 4 |
| 8 | FX display rates | 2, 7 |
| 9 | Payments (real gateway per country) | 6 |
| 10 | Multi-region deploy | 1-6 |
| 11 | Service extraction (payments, notifications, KYC) | 6, 9 |
| 12 | Per-country compliance checklist (non-code) | chosen launch country |

---

## Phase 1: Country config

**Goal:** one row per country drives currency, dial code, timezone, radius, fees and providers. Every account and booking knows its country.

### New migration `backend/migrations/000019_countries.sql`

```sql
CREATE TABLE IF NOT EXISTS countries (
    code              CHAR(2) PRIMARY KEY,           -- ISO 3166-1 alpha-2: IN, GB, US
    name              TEXT NOT NULL,
    currency          CHAR(3) NOT NULL,              -- ISO 4217: INR, GBP, USD
    currency_exponent SMALLINT NOT NULL DEFAULT 2,   -- JPY 0, KWD 3
    currency_symbol   TEXT NOT NULL,                 -- ₹, £, $
    dial_code         TEXT NOT NULL,                 -- +91, +44, +1
    national_number_length SMALLINT[] NOT NULL,      -- {10} for IN, {10} for GB (after trunk 0), {10} for US
    trunk_prefix      TEXT NOT NULL DEFAULT '',      -- '0' for IN and GB, '' for US
    default_timezone  TEXT NOT NULL,                 -- IANA: Asia/Kolkata, Europe/London
    default_radius_m  INTEGER NOT NULL DEFAULT 50000,
    max_radius_m      INTEGER NOT NULL DEFAULT 100000,
    travel_fee_minor  BIGINT NOT NULL DEFAULT 15000, -- in-transit cancellation fee
    max_price_minor   BIGINT NOT NULL DEFAULT 10000000,
    payment_provider  TEXT NOT NULL DEFAULT 'none',
    kyc_provider      TEXT NOT NULL DEFAULT 'manual',
    tax_scheme        TEXT NOT NULL DEFAULT 'none',
    region            TEXT NOT NULL DEFAULT 'in',    -- deploy region, used in Phase 10
    is_active         BOOLEAN NOT NULL DEFAULT FALSE
);

INSERT INTO countries (code, name, currency, currency_symbol, dial_code, national_number_length, trunk_prefix, default_timezone, is_active)
VALUES ('IN', 'India', 'INR', '₹', '+91', '{10}', '0', 'Asia/Kolkata', TRUE)
ON CONFLICT (code) DO NOTHING;

ALTER TABLE users    ADD COLUMN IF NOT EXISTS country_code CHAR(2) NOT NULL DEFAULT 'IN' REFERENCES countries(code);
ALTER TABLE priests  ADD COLUMN IF NOT EXISTS country_code CHAR(2) NOT NULL DEFAULT 'IN' REFERENCES countries(code);
ALTER TABLE vendors  ADD COLUMN IF NOT EXISTS country_code CHAR(2) NOT NULL DEFAULT 'IN' REFERENCES countries(code);
ALTER TABLE temples  ADD COLUMN IF NOT EXISTS country_code CHAR(2) NOT NULL DEFAULT 'IN' REFERENCES countries(code);
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS country_code CHAR(2) NOT NULL DEFAULT 'IN' REFERENCES countries(code);
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS currency     CHAR(3) NOT NULL DEFAULT 'INR';
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS timezone     TEXT    NOT NULL DEFAULT 'Asia/Kolkata';
ALTER TABLE events   ADD COLUMN IF NOT EXISTS country_code CHAR(2) NOT NULL DEFAULT 'IN' REFERENCES countries(code);
ALTER TABLE events   ADD COLUMN IF NOT EXISTS timezone     TEXT    NOT NULL DEFAULT 'Asia/Kolkata';
```

Table names above are confirmed against migrations `000001`, `000003` and `000017`. Also add `currency CHAR(3) NOT NULL DEFAULT 'INR'` to every table that holds a price: `pooja_catalog`, `priest_custom_services`, `pooja_samagri_items`, `vendor_inventory`, `store_orders`, `store_order_items`, `temple_vip_pujas`, `temple_vip_tickets`, `event_priest_services`.

The default `'IN'`/`'INR'` backfills every existing row correctly, since all current data is Indian.

### New package `backend/internal/country/country.go`

- `type Country struct` mirroring the table.
- `func Load(ctx, db) error` reads all rows into a package-level `map[string]Country` at startup. Call it in `main.go` right after storage is created (near `main.go:150`, before `mux.NewRouter()`).
- `func Get(code string) (Country, bool)`.
- `func MustGet(code string) Country` returns the `IN` row when code is unknown (keeps legacy rows safe).
- `func ByDialCode(e164 string) (Country, bool)` longest-prefix match on `dial_code`.

No interface, no cache invalidation. Countries change rarely; restart reloads.

### Other edits

| File | Line | Change |
|---|---|---|
| `backend/internal/models/entities.go` | 18-23 (`LoginRequest`) | no change; country is derived from stored account |
| `backend/cmd/api-gateway/main.go` | 443, 500 (register devotee / priest) | accept `country_code` in the registration body; validate with `country.Get`, reject inactive countries with 422 `unsupported_country`; store it |
| `backend/internal/auth/jwt.go` | 87-94 (`CustomClaims`) | add `Country string \`json:"country"\``; 97 (`GenerateToken`) add `country` parameter |
| `backend/internal/auth/jwt.go` callers | `main.go:444`, `main.go:501`, `backend/internal/db/admin.go:74`, `backend/internal/db/db.go:241` | pass the account's country; admin passes `""` |
| `backend/internal/auth/jwt.go` | 138 (`ValidateToken`) | tokens with empty `country` are treated as `IN` so existing sessions keep working |
| new route in `main.go` | - | `GET /api/v1/countries` (public): returns active countries (code, name, currency, symbol, exponent, dial_code, default_timezone). Apps use it for the country picker |
| `backend/internal/db/migrate/migrator.go` | 538 (`reqTables`) | add `"countries"` |

### Tests (Phase 1)

`backend/internal/country/country_test.go`
- `TestGet_KnownCountry`: IN returns INR, `+91`, `Asia/Kolkata`.
- `TestMustGet_UnknownFallsBackToIN`.
- `TestByDialCode_LongestPrefix`: `+1` vs `+1242` style overlap resolves to the longer match; `+447911123456` resolves to GB.
- `TestByDialCode_NoMatch`: returns false.

`backend/internal/auth/jwt_test.go` (extend)
- `TestToken_CarriesCountry`: generate with `GB`, validate, claim equals `GB`.
- `TestToken_MissingCountryDefaultsIN`: hand-built legacy token without the claim validates and reports `IN`.

`backend/cmd/api-gateway/` (new `country_routes_test.go`)
- `TestListCountries_OnlyActive`.
- `TestRegister_UnsupportedCountry_422`.
- `TestRegister_InactiveCountry_422`.
- `TestRegister_NoCountry_DefaultsIN` (old app versions keep working).

`backend/internal/db/migrate/migrator_test.go` (extend)
- `TestMigration19_BackfillsIN`: existing users, priests, bookings all have `country_code='IN'`, `currency='INR'` after migrating.
- `TestMigration19_Idempotent`: running twice does not fail.

**Done when:** all above pass and existing suite is still green.

---

## Phase 2: Currency

**Goal:** currency always comes from the row (or its country), never from a literal.

### Backend: every hardcoded `"INR"`

| File | Line | Current | Change to |
|---|---|---|---|
| `backend/internal/cancellation/refund.go` | 11 | `CalculateRefundQuoteMinor(bookingID, status string, totalAmountMinor, dakshinaAmountMinor int64)` | add parameter `c country.Country` |
| `backend/internal/cancellation/refund.go` | 16 | `Currency: "INR"` | `Currency: c.Currency` |
| `backend/internal/cancellation/refund.go` | 17, 30, 44-47, 60-62 | `/ 100.0` | divide by `10^c.CurrencyExponent` (helper `c.Major(minor)`) |
| `backend/internal/cancellation/refund.go` | 78 (`CalculateRefundQuote`) | float wrapper | add same parameter, pass through |
| `backend/internal/db/cancellation.go` | 51, 145 (the two calls to `CalculateRefundQuoteMinor`) | - | load `bookings.country_code`, pass `country.MustGet(code)` |
| `backend/internal/db/custom_services.go` | 242, 271 | `item.Currency = "INR"` | select `currency` column in the two queries (lines 228-233 and 255-264) and scan it |
| `backend/internal/db/samagri.go` | 37, 81 | `item.Currency = "INR"` | select and scan `currency` |
| `backend/internal/db/samagri.go` | 43-51, 88-96 | default kit with INR prices | move to a `defaultSamagri(countryCode)` function; return the INR list only for `IN`, empty list otherwise |
| `backend/internal/db/temples.go` | 112 | `Currency: "INR"` | currency of the `temple_vip_pujas` row being booked |
| `backend/internal/db/temples.go` | 155 | `ticket.Currency = "INR"` | add `currency` to the `RETURNING` list at 139-143 and scan it |
| `backend/internal/db/temples.go` | 81-86 | `* 100` and `/ 100.0` | use exponent of the puja's country |
| `backend/internal/db/vendors.go` | 42 | `item.Currency = "INR"` | select and scan `currency` (query at 21-26) |
| `backend/internal/db/event_ops.go` | 574 | `"currency": "INR"` | currency of the event service row (same row `price` comes from) |
| `backend/internal/db/event_ops.go` | 585 | `float64(price)/100` | exponent-aware |
| `backend/cmd/api-gateway/main.go` | 862 | `"currency": "INR"` | return currency from `CreateStoreOrder` (called at 842); add it as a third return value |
| `backend/cmd/api-gateway/main.go` | 863 | `float64(totalMinor) / 100.0` | exponent-aware |

**Write rule:** when a priest, vendor or temple creates a priced row, set `currency = country.MustGet(owner.country_code).Currency`. The client must not choose the currency. Edit the insert at `custom_services.go:203-212` and the equivalents in `vendors.go`, `temples.go`, `event_ops.go:305-330` (`SetPriestServices`).

**Guard rule:** a booking's currency must equal the priest's currency. Add the check in `backend/internal/db/db.go` around line 732 (booking insert); return `ErrCurrencyMismatch` (HTTP 422).

**Store order rule:** all items in one store order must share one currency. Add to `CreateStoreOrder`; reuse the existing 422 block at `main.go:844-853`.

### Apps: formatting

`apps/common-contracts/src/commonMain/kotlin/com/purohit/contracts/money/MoneyFormatting.kt` (whole file, 23 lines). Replace with:

```kotlin
fun formatMoney(minorUnits: Long, currency: String = "INR"): String
```

- Symbol and exponent from a small in-file map seeded for `INR`, `GBP`, `USD`, `EUR`, `CAD`, `AUD`, `SGD`, `AED`, and overridable from the `/api/v1/countries` response.
- Unknown currency: render `"<CODE> <amount>"`, exponent 2.
- Keep `formatMinorToRupees(minor)` and `formatAmountToRupees(...)` as one-line wrappers calling `formatMoney(minor, "INR")` so the 6 current call sites in `EventBookingScreens.kt` (lines 108, 185, 225, 346) keep compiling, then migrate them to pass the row's currency.

Inline `₹` to replace with `formatMoney(<minor>, <row currency>)`:

| File | Lines |
|---|---|
| `apps/devotee-app/src/commonMain/kotlin/com/purohit/devotee/ui/dispatch/RapidoNearbyPriestsScreen.kt` | 271, 421, 523 |
| `apps/devotee-app/.../ui/catalog/PoojaCatalogScreen.kt` | 281, 303, 394, 399 |
| `apps/devotee-app/.../ui/store/PujaStoreScreen.kt` | 272, 330 |
| `apps/devotee-app/.../ui/temple/TempleDirectoryScreen.kt` | 203 |
| `apps/devotee-app/.../ui/tracking/LiveTrackingScreen.kt` | 74 |
| `apps/priest-app/.../ui/dispatch/RapidoIncomingAlertModal.kt` | 215 |
| `apps/priest-app/.../ui/duty/DutyToggleScreen.kt` | 195 |
| `apps/priest-app/.../ui/services/PriestServicesScreen.kt` | 162, 167, 201, 207 |
| `apps/priest-app/.../ui/pushkaralu/PriestEventScreen.kt` | 128 (`prefix = { Text("₹") }` becomes the priest's currency symbol) |

These lines use `.toInt()` on float prices (for example `pkg.base_price.toInt()`), which drops decimals. Switch each to the `*_minor` field while editing.

Strings:

| File | Line | Change |
|---|---|---|
| `apps/common-contracts/.../i18n/locales/en/EnStrings.kt` | 85 | `"+ Rs. %s included in Dakshina"` becomes `"+ %s included in Dakshina"`; caller passes `formatMoney(...)` |
| same | 144 | `"...travel fee of Rs. 150 applies..."` becomes a template with `{fee}`; fee comes from the cancellation quote |
| same, `te/TeStrings.kt`, `hi/HiStrings.kt` | 290 | `priceInRupees` becomes `priceLabel` = "Price" (rename in `i18n/models/EventStrings.kt:72` too); symbol shown by the field prefix |
| `te/TeStrings.kt`, `hi/HiStrings.kt` | matching lines for 85 and 144 | same template change |

Backend text with the same problem: `refund.go:35` comment and `refund.go:48` (`"...Rs. 150 applies..."`). Build the notice from `c.TravelFeeMinor` and `c.CurrencySymbol` (Phase 5 moves the fee itself).

Model defaults: leave `val currency: String = "INR"` in the Kotlin models (`TempleModels.kt:30,54,74`, `StoreModels.kt:33,45,62`, `CustomServiceModels.kt:15,32`, `SamagriModels.kt:15`, `CancellationModels.kt:19,41`, `BookingModels.kt:49,72`, devotee `PurohitApiClient.kt:279,319,338,397`). The default only applies when an old backend omits the field, and old backends are India-only.

### Temple CMS

| File | Lines | Change |
|---|---|---|
| `temple-cms/app/reconciliation/page.tsx` | 31, 75, 80, 85, 129-132 | add one helper `formatMoney(minor, currency)` using `Intl.NumberFormat(undefined, {style:'currency', currency})`; replace `₹{x.toLocaleString('en-IN')}` and the `(INR)` CSV headers |
| `temple-cms/app/samagri-catalog/page.tsx` | 246, 315 | same helper |
| `temple-cms/app/vip-pujas/page.tsx` | 117, 178, 252 | same helper |
| `temple-cms/app/services-translation/page.tsx` | 237 | same helper |
| `temple-cms/app/page.tsx` | 8, 14-17 | hardcoded demo figures; replace with API data or leave and mark as demo |

### Tests (Phase 2)

`backend/internal/cancellation/refund_test.go` (extend)
- `TestRefund_UsesCountryCurrency`: GB country gives `Currency == "GBP"`.
- `TestRefund_ZeroExponentCurrency`: JPY (exponent 0), total 5000 minor gives `AdvancePaid == 5000.0`, not 50.0.
- `TestRefund_ThreeExponentCurrency`: KWD (exponent 3), 12345 minor gives 12.345.
- `TestRefund_INRUnchanged`: every existing expectation still holds for IN (regression).
- `TestRefund_InTransit_FeeLargerThanTotal`: refund clamps to 0, never negative, in a non-INR currency.

`backend/internal/db/` (new `currency_test.go`, DB-backed like `temples_test.go`)
- `TestCustomService_CurrencyFromPriestCountry`: GB priest creates a service; row and API response say GBP even if the request body says INR.
- `TestBooking_CurrencyMismatch_Rejected`: booking with currency different from the priest's fails with `ErrCurrencyMismatch`.
- `TestBooking_StoresCurrencyAndCountry`.
- `TestStoreOrder_MixedCurrencies_Rejected`.
- `TestStoreOrder_ReturnsVendorCurrency`.
- `TestTempleTicket_CurrencyFromPuja`.
- `TestVerifyTempleTicket_ReturnsStoredCurrency`.
- `TestSamagriDefaults_OnlyForIN`: GB pooja with no samagri rows returns empty list, not INR defaults.
- `TestEventBookingSnapshot_CurrencyFromService`: `agreement_snapshot.currency` matches the service row.

`apps/common-contracts/src/commonTest/kotlin/com/purohit/contracts/ResilienceAndMoneyTest.kt` (extend; existing INR cases at lines 48-54 must still pass)
- `formatMoney(12345, "GBP") == "£123.45"`
- `formatMoney(15000, "USD") == "$150"`
- `formatMoney(5000, "JPY") == "¥5000"` (exponent 0)
- `formatMoney(12345, "KWD")` uses 3 decimals
- `formatMoney(100, "XXX") == "XXX 1"` (unknown code)
- `formatMoney(-12345, "GBP")` shows the minus once and correct digits
- `formatMoney(0, "GBP") == "£0"`

`temple-cms`: one unit test for the helper covering INR, GBP, JPY.

**Done when:** these return no results outside tests, seeds and migrations:

```bash
grep -rn '"INR"' backend --include=*.go --exclude=*_test.go
```

```bash
grep -rnE '₹|Rs\.' apps --include=*.kt --exclude-dir=commonTest
```

(Allowed leftovers: the symbol map in `MoneyFormatting.kt`, the Kotlin model defaults listed above, and the `IN` branch of `defaultSamagri`.)

---

## Phase 3: Phone numbers

**Goal:** numbers are stored as E.164 for the right country. A UK number must never become `+91...`.

**The bug today:** `backend/internal/auth/phone.go:52-54` turns any 11-digit number starting with `0` into `+91...`. UK mobile `07911123456` becomes `+917911123456`. Line 47-48 does the same for any 10-digit number (US numbers are 10 digits).

### Changes

`backend/internal/auth/phone.go`
- Line 11: new signature `NormalizePhone(raw string, defaultCountry country.Country) (string, error)`.
- Lines 26-39 (input starts with `+`): keep, but require the prefix to match an active country via `country.ByDialCode`; otherwise `unsupported country code`. Remove the dead branch at 34-37.
- Lines 46-59 (no `+`): strip `defaultCountry.TrunkPrefix` if present, check length is in `defaultCountry.NationalNumberLength`, prepend `defaultCountry.DialCode`. Delete the three India-only branches.
- Line 66 (`PhoneLookupVariants`): add the same parameter. Keep the legacy spaced variants at lines 77-90, but only when the canonical number is `+91` (already guarded at line 77).

Callers to update (pass the country from the request body's `country_code`, defaulting to `IN` when absent):

| File | Lines |
|---|---|
| `backend/internal/auth/throttler.go` | 67 |
| `backend/cmd/api-gateway/main.go` | 443, 500 |
| `backend/internal/db/db.go` | 154, 160, 475, 480, 539, 544 |
| `backend/internal/db/recovery.go` | 103, 122, 440, 450 |

Login needs the country before the account is known, so add optional `country_code` to `LoginRequest` (`backend/internal/models/entities.go:18-23`), `ResetPasswordRequest` (25-30), `StartRecoveryRequest` (32-36), `ConfirmRecoveryRequest` (43-50). If the phone already starts with `+`, `country_code` is ignored.

`main.go:163` builds the throttle key from the raw phone before normalising; make sure the key uses the country-aware canonical form so `07911123456` (GB) and `+447911123456` share one bucket.

Apps:
- Add a country picker (dial code dropdown) in front of the phone field in `apps/devotee-app/.../ui/auth/DevoteeLoginScreen.kt:128`, `DevoteeRegistrationScreen.kt:86`, `apps/priest-app/.../ui/auth/PriestRegistrationScreen.kt:186,316`. Options from `GET /api/v1/countries`. Default from device locale region, fall back to `IN`.
- The app sends the full `+<dial><number>` string plus `country_code`.
- `phonePlaceholder` (`EnStrings.kt:13`, `TeStrings.kt:13`, `HiStrings.kt:13`): change to a number without dial code (`"98765 43210"`), since the picker shows the code.
- `apps/devotee-app/.../network/PurohitApiClient.kt:901`: `profile?.phone ?: "+91 99999 99999"` sends a fake Indian number when the profile is missing. Fail the request instead.

### Tests (Phase 3)

`backend/internal/auth/phone_test.go` (extend; keep all 12 existing India cases, passing `IN`)
- `GB`, `"07911 123456"` gives `+447911123456`.
- `GB`, `"7911123456"` gives `+447911123456`.
- `GB`, `"+44 7911 123456"` gives `+447911123456`.
- `US`, `"(415) 555-2671"` gives `+14155552671`.
- `IN`, `"09876543210"` gives `+919876543210` (regression).
- `IN`, `"9876543210"` gives `+919876543210` (regression).
- Default `IN` but input `"+447911123456"` gives `+447911123456` (explicit `+` wins).
- `GB`, `"12345"` gives error (too short).
- `GB`, `"079111234567890"` gives error (too long).
- `"+999123456789"` gives error `unsupported country code`.
- `"+91abc"` gives error; error text does not contain the raw number (uses `MaskPhone`).
- Empty string gives error.
- **The bug itself:** `GB`, `"07911123456"` must NOT start with `+91`.

`PhoneLookupVariants`
- GB number returns the canonical form and raw input only; no `+91` variants.
- IN number still returns all legacy spaced variants (regression).

`backend/internal/auth/throttler_test.go` (extend)
- `"07911123456"`+GB and `"+447911123456"` produce the same throttle key.
- `"9876543210"`+IN and `"9876543210"`+US produce different keys.

`backend/internal/db/auth_test.go` (extend)
- Register GB devotee with national format, log in with E.164 format: succeeds.
- Same digits registered under IN and US are two separate accounts.
- Legacy row stored as `"+91 98765 43210"` still logs in (regression).

**Done when:**

```bash
grep -rn '"+91"' backend --include=*.go --exclude=*_test.go
```

returns only the legacy-variant block in `phone.go`.

---

## Phase 4: Timezone

**Goal:** store UTC, display in the timezone of where the pooja happens.

| File | Line | Current | Change |
|---|---|---|---|
| `backend/internal/db/event_ops.go` | 577 | `ist := time.FixedZone("IST", 5*3600+1800)` | `loc, err := time.LoadLocation(eventTimezone)`; read `events.timezone` in the same query that loads the event; on error fall back to UTC and log |
| `backend/internal/db/event_ops.go` | 586 | `scheduled.In(ist).Format(...)` twice | `scheduled.In(loc)` |
| `backend/internal/db/event_ops.go` | 579-584 | booking insert | also write `timezone`, `country_code`, `currency` |
| `backend/internal/db/db.go` | ~732 | normal booking insert | write `timezone` = priest's country `default_timezone` (in-person) or see Phase 7 (virtual) |
| `backend/Dockerfile` | - | - | make sure `tzdata` is in the final image, or add `import _ "time/tzdata"` in `backend/cmd/api-gateway/main.go`. Without it `LoadLocation` fails in a scratch/alpine image |
| `apps/devotee-app/.../ui/pushkaralu/EventBookingScreens.kt` | 40 | `internal val IST = TimeZone.of("Asia/Kolkata")` | remove the constant; take `TimeZone.of(event.timezone)` (add `timezone` to the event model in `apps/common-contracts/.../event/EventModels.kt`, default `"Asia/Kolkata"`). Used at lines 132, 137, 138, 139, 205 |
| `apps/devotee-app/.../ui/dispatch/RapidoNearbyPriestsScreen.kt` | 536 | `TimeZone.of("Asia/Kolkata")` | `TimeZone.currentSystemDefault()` (in-person: devotee and priest are in the same place) |
| `apps/priest-app/.../ui/schedule/PriestScheduleScreen.kt` | 89 | `TimeZone.of("Asia/Kolkata")` | `TimeZone.currentSystemDefault()` |
| `apps/priest-app/.../ui/schedule/PriestScheduleScreen.kt` | 58-59 | labels say "24-hour IST" | drop "IST"; show the zone's short name |

API responses for bookings and slots should include `timezone` so the app can render the pooja's local time even when the viewer is elsewhere (matters for virtual pooja).

### Tests (Phase 4)

`backend/internal/db/` (new or in `event_ops` tests)
- `TestEventBooking_FormatsInEventTimezone`: event in `Europe/London`, scheduled `2027-07-01T09:00:00Z` gives `slot_time == "10:00 AM"` (BST), `scheduled_date == "01 Jul 2027"`.
- `TestEventBooking_ISTUnchanged`: `Asia/Kolkata` event, `2027-07-01T04:30:00Z` gives `"10:00 AM"` (regression).
- `TestEventBooking_DateRollsOver`: `America/Los_Angeles`, `2027-07-02T03:00:00Z` gives date `01 Jul 2027`.
- `TestEventBooking_DSTBoundary`: `Europe/London` on 2027-03-28 (clocks go forward): a 00:30Z and a 01:30Z booking format as `12:30 AM` and `02:30 AM`.
- `TestEventBooking_BadTimezoneFallsBackUTC`: `events.timezone = 'Not/AZone'` does not fail the booking.
- `TestBooking_StoresTimezone`.
- Add a test that `time.LoadLocation("Europe/London")` succeeds (catches a missing tzdata in CI image).

`apps/devotee-app/src/commonTest/kotlin/com/purohit/devotee/event/EventBookingLogicTest.kt` (extend; existing IST cases at lines 25-36 must pass with `timezone = "Asia/Kolkata"`)
- Slots for a `Europe/London` event are hourly in London local time and inside the event window.
- Slot list on a DST-change day has no duplicate and no missing hour beyond the real gap.
- Event with missing `timezone` field behaves as `Asia/Kolkata`.

**Done when:**

```bash
grep -rnE 'Asia/Kolkata|FixedZone\("IST"' backend apps --include=*.go --include=*.kt --exclude=*_test.go --exclude-dir=commonTest
```

returns only model defaults.

---

## Phase 5: Radius and fees per country

| File | Line | Current | Change |
|---|---|---|---|
| `backend/internal/db/discovery.go` | 14 | `GetNearbyPriests(devLat, devLon, radiusMeters float64, serviceFilter string)` | add `c country.Country` |
| `backend/internal/db/discovery.go` | 21-23 | cap `100000` | `c.MaxRadiusM` |
| `backend/internal/db/discovery.go` | 24-26 | default `50000.0` | `c.DefaultRadiusM` |
| `backend/internal/db/discovery.go` | 31-34 | query | add `AND country_code = $4` so a devotee near a border does not get priests priced in another currency |
| `backend/cmd/api-gateway/main.go` | 286 | `radius := 50000.0` | `0` (let storage apply the country default) |
| `backend/cmd/api-gateway/main.go` | 291, 356 | calls | pass country from JWT claims |
| `backend/cmd/api-gateway/main.go` | 352 | dispatch default `5000.0` | keep; it is a deliberate tighter default for instant dispatch |
| `backend/cmd/api-gateway/vendor_location.go` | 16 | `radius := 10000.0` | keep default; add country filter in `backend/internal/db/vendors_geo.go` |
| `backend/internal/cancellation/refund.go` | 35 | `travelAllowanceMinor := int64(15000)` | `c.TravelFeeMinor` |
| `backend/internal/cancellation/refund.go` | 48 | notice text with `Rs. 150` | build from `c` |
| `backend/internal/db/event_ops.go` | 325 | `svc.PriceMinor > 10000000` | `> c.MaxPriceMinor` |

Suggested first values: GB `default_radius_m=80000`, `max_radius_m=250000`, `travel_fee_minor=1500` (£15). US `default_radius_m=80000`, `max_radius_m=300000`, `travel_fee_minor=2000`.

ETA at `discovery.go:85` (`DistanceKm * 3.0` minutes) is a city-traffic guess. Leave it; note it reads high for motorway distances.

### Tests (Phase 5)

`backend/cmd/api-gateway/location_routes_test.go` and a new `backend/internal/db/discovery_test.go`
- `TestNearby_UsesCountryDefaultRadius`: GB devotee, no radius given, priest 70 km away is returned; same setup for IN is not returned.
- `TestNearby_ClampsToCountryMax`: request 900 km in GB is clamped to 250 km.
- `TestNearby_ExcludesOtherCountry`: IN priest 5 km from a GB devotee (synthetic) is not returned.
- `TestNearby_INBehaviourUnchanged`: 50 km default, 100 km cap (regression).
- `TestNearby_InvalidCoords_StillRejected` (regression for `discovery.go:18`).
- `TestNearby_NoPostGISPath_SameResults`: run the fallback SQL branch (`discovery.go:40-46`) with the country filter.
- `TestNearby_AntimeridianAndPoles`: lon 179.9 vs -179.9 are near each other; lat 89 does not error.
- `TestVendorsNearby_ExcludesOtherCountry`.

`refund_test.go`
- `TestRefund_InTransit_UsesCountryTravelFee`: GB gives 1500, IN gives 15000.
- `TestRefund_PolicyNotice_ShowsLocalCurrency`: GB notice contains `£15`, does not contain `Rs`.

`event_ops` tests
- `TestSetPriestServices_PriceCapPerCountry`.

---

## Phase 6: Provider interfaces

**Goal:** country differences live behind four small interfaces inside the gateway. No new services yet.

New package `backend/internal/providers/`:

```go
type PaymentProvider interface {
    CreateCharge(ctx context.Context, req ChargeRequest) (ChargeResult, error)   // amount_minor + currency + idempotency key
    Refund(ctx context.Context, chargeID string, amountMinor int64) error        // always original currency
    VerifyWebhook(r *http.Request) (Event, error)
}
type KYCProvider interface {
    Start(ctx context.Context, priestID string) (KYCSession, error)
    Status(ctx context.Context, sessionID string) (KYCStatus, error)
}
type TaxCalculator interface {
    Calculate(amountMinor int64, c country.Country) TaxBreakdown
}
type Notifier interface {
    Send(ctx context.Context, to Recipient, tmpl string, data map[string]string) error
}
```

- `func PaymentFor(c country.Country) PaymentProvider` and siblings pick by `c.PaymentProvider`, `c.KYCProvider`, `c.TaxScheme`.
- Ship one implementation each now: `nonePayment` (returns `ErrPaymentsNotEnabled`; today's pay-at-venue flow), `manualKYC` (admin approves in CMS, matching today's behaviour), `noTax` (zero), and wrap the existing `backend/internal/notify/push.go` as `Notifier`.
- Do not add a second implementation until Phase 9. An interface is justified here only because Phase 9 adds two real ones.

### Tests (Phase 6)

`backend/internal/providers/providers_test.go`
- `TestPaymentFor_UnknownProviderFallsBackToNone`.
- `TestNonePayment_ReturnsNotEnabled`.
- `TestNoTax_ZeroForAllCountries`.
- `TestManualKYC_PendingUntilAdminApproves`.
- A fake `PaymentProvider` used in a booking test proves the booking code calls `CreateCharge` with the booking's currency and an idempotency key.

---

## Phase 7: Virtual pooja (cross-country)

**Goal:** a devotee anywhere books a priest anywhere for a pooja over video. This is the Sri Mandir / VAMA style offer and the first product in every new country.

### What already exists

| Piece | Where |
|---|---|
| Catalog flag `is_online_eligible` | `backend/migrations/000001_init_schema.sql:49`, `000003_comprehensive_ecosystem.sql:31`, `backend/internal/models/entities.go:292,315`, `backend/internal/db/db.go:294-375`, `custom_services.go:59,92,203,212` |
| Temple video-call puja `is_online_video_call` | `000003_comprehensive_ecosystem.sql:175,233`, `entities.go:502`, `backend/internal/db/temples.go:45-72` |
| `webrtc_rooms` table | `000001_init_schema.sql:76-77` |
| Signalling WebSocket `/ws/webrtc` | `backend/cmd/api-gateway/main.go:532-580`, `realtime_hubs.go:37-53` |
| Standalone signalling binary | `backend/cmd/webrtc-signaling/main.go` |
| TURN server | `docker-compose.prod.yml` (`coturn` service) |
| App model flags | `apps/common-contracts/.../booking/BookingModels.kt:51`, `services/CustomServiceModels.kt:17,34`, devotee `PurohitApiClient.kt:236`, priest `PurohitApiClient.kt:203,224` |
| Devotee entry point | stub: `apps/devotee-app/.../DevoteeApp.kt:354` (`onOpenLiveStream = {}`), `ui/tracking/LiveTrackingScreen.kt:31` |

### What is missing

1. **Booking mode.** Migration `000020_virtual_pooja.sql`:
   ```sql
   ALTER TABLE bookings ADD COLUMN IF NOT EXISTS mode TEXT NOT NULL DEFAULT 'IN_PERSON'
       CHECK (mode IN ('IN_PERSON','VIRTUAL'));
   ALTER TABLE bookings ADD COLUMN IF NOT EXISTS devotee_country_code CHAR(2);
   ALTER TABLE bookings ADD COLUMN IF NOT EXISTS devotee_timezone TEXT;
   ALTER TABLE priests  ADD COLUMN IF NOT EXISTS offers_virtual BOOLEAN NOT NULL DEFAULT FALSE;
   ALTER TABLE priests  ADD COLUMN IF NOT EXISTS languages TEXT[] NOT NULL DEFAULT '{}';
   ```
   Add `Mode` next to `BookingType` in `entities.go:162,187`. Default `IN_PERSON` in `db.go:668-670`.

2. **Virtual discovery.** New function `GetVirtualPriests(serviceFilter, language, sampradaya string, limit, offset int)` in `backend/internal/db/discovery.go`. No radius, no location, no country filter. Filter: `offers_virtual = true`, service is `is_online_eligible`, optional language and sampradaya. Sort by rating then completed count. New route `GET /api/v1/priests/virtual` next to the nearby route at `main.go:277`. Response includes the priest's `currency` and `timezone`. Never return the priest's coordinates on this route.

3. **Virtual booking rules** in the booking create path (`db.go:~668-732`):
   - Always scheduled; reject instant dispatch with `mode=VIRTUAL` (422).
   - No address, latitude or longitude required.
   - Pooja or service must be `is_online_eligible`; priest must have `offers_virtual`.
   - `country_code`, `currency`, `timezone` are the **priest's** (the priest is paid in their own currency). Also store `devotee_country_code` and `devotee_timezone`.
   - `scheduled_at` is UTC. Must be at least the existing minimum lead ahead.
   - No travel fee on cancellation: in `refund.go`, `IN_TRANSIT` and `ARRIVED` do not occur for virtual; define the virtual policy as free until 2 hours before start, then 25% of dakshina retained.
   - Skip the country filter and the currency-equals-devotee assumption; the Phase 2 guard (booking currency equals priest currency) still applies.

4. **Booking state machine.** Virtual bookings skip `IN_TRANSIT` and `ARRIVED`: `ACCEPTED` goes to `POOJA_STARTED` when both peers join the room. Update the FSM (states at `backend/internal/models/entities.go:7-16`, transition table in `backend/internal/lifecycle/fsm.go`, handlers in `backend/cmd/api-gateway/booking_handlers.go`) to allow that edge only when `mode = VIRTUAL`, and to reject `IN_TRANSIT`/`ARRIVED` for virtual.

5. **Room access.** `/ws/webrtc` (`main.go:533`) must only admit the booking's devotee and priest, only for a `VIRTUAL` booking in `ACCEPTED` or `POOJA_STARTED`, and only from 15 minutes before `scheduled_at` until 30 minutes after scheduled end. Create the `webrtc_rooms` row when the priest accepts. Issue short-lived TURN credentials per session instead of the static `--user` in `docker-compose.prod.yml`.

6. **Completion.** The in-person completion OTP (`completion_otp`) is shown by the devotee to the priest face to face. For virtual, show it on the devotee's call screen and let the priest type it, or auto-complete when the call lasted at least 50% of `duration_minutes` and the devotee confirms. Pick one; the OTP on screen reuses existing code.

7. **Apps.**
   - Devotee: "Virtual pooja" tab on the catalog (`PoojaCatalogScreen.kt`) listing online-eligible poojas, priest list from the new route with language filter, date and time picker shown in the **devotee's** local time with the priest's local time underneath, price in the priest's currency with an approximate local-currency line (Phase 8).
   - Wire `onOpenLiveStream` at `DevoteeApp.kt:354` to a call screen. Priest app gets the matching "Join" button on the booking.
   - Priest: toggle "I offer virtual poojas" and language multi-select in profile; weekly availability already lives in `PriestScheduleScreen.kt`.
   - Reminders 24 hours and 15 minutes before, in each party's own timezone, through `Notifier`.

8. **Temple virtual pujas.** `temple_vip_pujas.is_online_video_call` already models this. Apply the same currency, timezone and room-access rules to tickets where it is true (`temples.go:75-127`). Optional later: prasad shipping (address capture, courier, customs) as a separate order type. Not in this phase.

9. **Recording and consent.** Do not record calls by default. If recording is added, both sides must consent on screen first.

### Tests (Phase 7)

`backend/internal/db/discovery_test.go`
- `TestVirtualPriests_IgnoresDistance`: IN priest is returned to a GB devotee.
- `TestVirtualPriests_OnlyOffersVirtual`.
- `TestVirtualPriests_OnlyOnlineEligibleService`.
- `TestVirtualPriests_LanguageFilter`.
- `TestVirtualPriests_NoCoordinatesInResponse`.
- `TestVirtualPriests_Pagination`.

Booking (`backend/internal/db/booking_regression_test.go`, `backend/cmd/api-gateway/booking_fsm_test.go`, `booking_authorization_test.go`)
- `TestVirtualBooking_UsesPriestCurrencyAndTimezone`: GB devotee, IN priest gives `currency=INR`, `timezone=Asia/Kolkata`, `devotee_country_code=GB`.
- `TestVirtualBooking_NoAddressRequired`.
- `TestVirtualBooking_RejectsInstantDispatch`.
- `TestVirtualBooking_RejectsNonEligiblePooja`.
- `TestVirtualBooking_RejectsPriestNotOfferingVirtual`.
- `TestVirtualBooking_RejectsPastOrTooSoon`.
- `TestVirtualBooking_SlotConflict`: priest already booked for an overlapping in-person or virtual slot is rejected (extend `slot_binding_test.go`).
- `TestVirtualFSM_AcceptedToStarted_Allowed`.
- `TestVirtualFSM_InTransit_Rejected` and `TestVirtualFSM_Arrived_Rejected`.
- `TestInPersonFSM_Unchanged`: `ACCEPTED` straight to `POOJA_STARTED` is still rejected for `IN_PERSON` (regression; add to `booking_fsm_adversarial_matrix_test.go`).
- `TestInPersonBooking_ModeDefaults`: request without `mode` is `IN_PERSON`.

Refund (`refund_test.go`)
- `TestVirtualRefund_FreeBeforeCutoff`.
- `TestVirtualRefund_AfterCutoff_25PercentRetained`.
- `TestVirtualRefund_NoTravelFee`.

Room access (new `backend/cmd/api-gateway/webrtc_access_test.go`)
- Devotee of the booking joins: allowed.
- Priest of the booking joins: allowed.
- Any other user: 403.
- No token: 401.
- In-person booking id: 403.
- 20 minutes before start: rejected; 10 minutes before: allowed.
- After window end: rejected.
- Cancelled booking: rejected.
- Third peer joining a full room: rejected.

Completion
- `TestVirtualCompletion_CorrectOTP`, `TestVirtualCompletion_WrongOTP_Throttled` (extend `completion_security_test.go`).

Apps
- Slot time shown for a `2027-07-01T04:30:00Z` booking: `10:00` for an `Asia/Kolkata` viewer, `05:30` for `Europe/London`.
- Booking a virtual pooja does not ask for location permission.

---

## Phase 8: FX rates (display and reporting only)

**Rule:** a booking is priced, charged, refunded and paid out in the priest's currency. We never convert funds. FX is used to show "≈ £42" and to total revenue in the CMS.

Migration `000021_fx_rates.sql`:

```sql
CREATE TABLE IF NOT EXISTS fx_rates (
    base       CHAR(3) NOT NULL,
    quote      CHAR(3) NOT NULL,
    rate       NUMERIC(18,8) NOT NULL CHECK (rate > 0),
    fetched_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (base, quote)
);
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS display_currency CHAR(3);
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS display_fx_rate NUMERIC(18,8);
```

New package `backend/internal/fx/`:
- `Convert(amountMinor int64, from, to country-or-currency, rate) int64` with exponent handling and round-half-up.
- `Rate(ctx, base, quote) (rate, fetchedAt, error)`; derive cross rates through USD; same currency returns 1.
- Daily fetch job from one rate provider (key from env). On failure keep the previous rate.
- A rate older than 72 hours is "stale": the API returns `approx_amount_minor: null` and the app hides the approximate line. Never block a booking on FX.

API: wherever a price is returned to a viewer whose country currency differs, add `approx_amount_minor`, `approx_currency`, `fx_fetched_at`. Store `display_currency` and `display_fx_rate` on the booking so the receipt can reproduce what the devotee saw.

App: second line under the price, "≈ £42.10, charged in INR. Your bank sets the final rate."

CMS: `temple-cms/app/reconciliation/page.tsx` gets a "report in" currency selector; totals are converted with the stored daily rate and labelled approximate. Per-currency subtotals stay exact.

### Tests (Phase 8)

`backend/internal/fx/fx_test.go`
- `TestConvert_INRtoGBP`: 500000 minor INR at 0.0095 gives 4750 minor GBP.
- `TestConvert_ExponentMismatch`: INR (2) to JPY (0) and GBP (2) to KWD (3).
- `TestConvert_RoundHalfUp`.
- `TestConvert_SameCurrency_Identity`.
- `TestConvert_ZeroAmount`.
- `TestConvert_LargeAmount_NoOverflow` (10^12 minor).
- `TestRate_CrossViaUSD`.
- `TestRate_Missing_ReturnsError`.
- `TestRate_Stale_FlaggedAfter72h`.
- `TestFetchJob_ProviderDown_KeepsOldRate`.
- `TestFetchJob_RejectsZeroOrNegativeRate`.

Booking and refund
- `TestBooking_StoresDisplayFxSnapshot`.
- `TestRefund_IgnoresFx`: rate changes after booking; refund amount and currency equal the original charge exactly.
- `TestBooking_SucceedsWhenFxStale`: approx fields are null, booking is created.

---

## Phase 9: Payments

**Goal:** real online payment per country behind `PaymentProvider` (Phase 6).

- India: an Indian gateway (UPI, cards). Elsewhere: Stripe (Connect for priest payouts).
- The gateway is selected by the **charge currency's country** (the priest's), because that is where funds settle. A UK devotee paying an Indian priest in INR pays through the India provider's international card support; if that is unavailable, virtual cross-border bookings stay "pay priest directly" until a marketplace-of-record decision is made. Record that decision here before building.
- Never store card data. Use hosted fields or redirect.
- Every charge and refund carries an idempotency key; reuse `backend/internal/db/idempotency.go`.
- Webhooks: verify signature, be idempotent on event id, and only move `escrow_status` forward.
- Refund amount comes from `cancellation.CalculateRefundQuoteMinor`, always in the original currency.
- Add `payment_provider`, `payment_charge_id`, `payment_status` to `bookings`, `store_orders`, `temple_vip_tickets`.

### Tests (Phase 9)

With a fake provider and each provider's sandbox:
- Charge success moves escrow to held; charge failure leaves the booking unpaid and retryable.
- Duplicate request with the same idempotency key creates one charge (extend `idempotency_safety_test.go`).
- Webhook with bad signature: 400, no state change.
- Same webhook delivered twice: one state change.
- Webhook arriving before the API response: final state is correct.
- Out-of-order webhooks (refund before capture): state never moves backward.
- Full refund, partial refund, refund larger than charge (rejected).
- Refund in original currency after the FX rate moved.
- Zero-exponent currency charge sends the right integer to the provider.
- Booking cancelled while the charge is pending: charge is voided or refunded exactly once (extend `booking_rollback_test.go`).
- Provider timeout: booking is not marked paid; reconciliation job fixes it later.
- Country with `payment_provider='none'` returns `ErrPaymentsNotEnabled` and the pay-at-venue flow still works.

---

## Phase 10: Multi-region deploy

**Goal:** one full stack per region; personal data stays in its region.

Regions: `in` (Mumbai), `eu` (London; UK and EU), `us` (N. Virginia). `countries.region` maps each country to one.

1. **Same images, one env file per region.** Parameterise `docker-compose.prod.yml`: add `REGION` and `REGION_PUBLIC_URL` to the `api-gateway` environment block (after `ALLOWED_ORIGINS`), and a distinct `JWT_SECRET` per region. Replace the static coturn `--realm=purohit.local` and `--user` with per-region values.
2. **Region lookup.** New tiny stateless service `backend/cmd/region-directory/` (or a static JSON on a CDN): `GET /v1/region?country=GB` returns `{"region":"eu","base_url":"https://eu.api..."}`. It stores no personal data.
3. **Apps.** `apps/devotee-app/.../network/PurohitApiClient.kt` and the priest/vendor equivalents take the base URL from the lookup at login or registration and persist it, instead of a compile-time URL.
4. **Tokens.** Add `Region` to `CustomClaims` (`jwt.go:87`). `ValidateToken` (`jwt.go:138`) rejects a token whose region differs from `REGION`, except on the cross-region virtual routes below. Different secrets per region already make foreign tokens fail; the claim gives a clear error.
5. **Cross-region virtual pooja.** The booking lives in the **priest's** region. The devotee's app calls the priest's region for that booking. Mechanism: the home region issues a short-lived (10 minute) "guest" token signed with a shared inter-region key, accepted only by `/api/v1/priests/virtual`, virtual booking create/read/cancel, chat for that booking, and `/ws/webrtc`. The priest's region stores a minimal shadow user (id, display name, country, timezone; no phone, no email). The devotee must consent to this transfer on first cross-region booking.
6. **Migrations.** `backend/cmd/migrate` runs against every region in the deploy pipeline, one region at a time, stop on first failure. All migrations must be backward compatible with the previous app version, because regions are briefly on different versions.
7. **Catalog.** `pooja_catalog` master is edited in the CMS and pushed to all regions; regional price rows stay local.
8. **CMS.** Region switcher; an admin session is per region.
9. **Backups, monitoring, health checks** per region (`/health` already exists in `health_routes.go`).
10. **Known limit:** a user who moves country creates a new account in the new region. Build account transfer only when requested.

### Tests (Phase 10)

- `TestToken_WrongRegion_Rejected`.
- `TestToken_LegacyNoRegion_AcceptedInIN` (existing sessions survive).
- `TestGuestToken_OnlyVirtualRoutes`: guest token gets 403 on nearby discovery, store, profile.
- `TestGuestToken_Expired_Rejected`; `TestGuestToken_WrongKey_Rejected`.
- `TestShadowUser_HasNoPhoneOrEmail`.
- `TestRegionDirectory_KnownCountry`, `_UnknownCountry_404`, `_InactiveCountry_404`.
- App: base URL comes from lookup; lookup failure shows a retry, not a crash; stored base URL survives restart.
- Deploy pipeline dry run: migration failing in region 2 stops the rollout and leaves region 1 serving.
- Restore-from-backup drill per region.
- Data residency check: after creating a GB devotee, a query in the `in` database finds no row with that phone.
- End to end: GB devotee (eu) books IN priest (in) virtually, both join the call, booking completes, refund path works.

---

## Phase 11: Service extraction

Countries differ by data and providers, not by booking logic. So there are **no per-country forks of the backend**. Only three parts move out of the monolith, each because it has its own secrets, scaling or compliance scope:

| Service | Why separate | Source today |
|---|---|---|
| `payments-service` | Payment secrets and webhooks; smallest audit scope | `providers.PaymentProvider` from Phase 6 and 9 |
| `notifications-service` | Bursty; per-country templates and senders | `backend/internal/notify/push.go` |
| `kyc-service` | Holds identity documents; strictest access | `providers.KYCProvider` |

Pattern for each: new `backend/cmd/<name>/main.go` (same layout as `backend/cmd/dispatch-service` and `backend/cmd/webrtc-signaling`), the gateway keeps the same Go interface but the implementation becomes an HTTP client, service-to-service auth by shared secret per region, new service added to `docker-compose.prod.yml`. Core (discovery, booking FSM, events, catalog, chat) stays in `api-gateway`.

Do this only after Phase 9 is live and stable. Extracting earlier means debugging a network hop and a payment integration at the same time.

### Tests (Phase 11)

- Contract tests: the HTTP client implementation passes the same test suite as the in-process implementation from Phase 6 and 9.
- Service down: gateway returns 503 for payment actions, bookings without payment still work, nothing is double-charged on retry.
- Timeout plus retry uses the same idempotency key.
- Service-to-service call without the secret: 401.
- Notification service down: booking flow is unaffected; notification is retried.
- KYC document endpoints are unreachable with a devotee or priest token.

---

## Phase 12: Per-country launch checklist (non-code)

Tick all before setting `countries.is_active = TRUE`.

- [ ] Legal entity or merchant-of-record arrangement for the country.
- [ ] Payment provider account live in that currency; payout to priests tested with a real account.
- [ ] Tax: registration threshold checked (UK VAT, US state sales tax, GST); `tax_scheme` set; invoice format approved.
- [ ] Privacy: UK/EU GDPR (privacy policy, lawful basis, data-processing agreements, deletion and export request path, breach process, region `eu`); US state laws as applicable.
- [ ] Priest vetting: UK DBS check, US background check; right-to-work where relevant; `kyc_provider` set.
- [ ] Terms of service and cancellation policy reviewed for local consumer law (UK 14-day cooling-off rules for distance contracts).
- [ ] App Store and Play Store listings for the region; age rating; payment rules for digital vs physical services checked.
- [ ] Language strings reviewed; currency and date formats checked on device.
- [ ] Support hours and contact for that timezone.
- [ ] Catalog and samagri rows created in local currency (no INR defaults leak through; see `TestSamagriDefaults_OnlyForIN`).
- [ ] At least N verified priests onboarded (in-person) or virtual-only launch explicitly chosen.
- [ ] `countries` row values reviewed: radius, travel fee, max price, timezone, dial code, number lengths.

---

## Cross-cutting regression suite

Run after every phase. India behaviour must not change.

- Full existing Go suite and Kotlin suite green.
- Root-level Python end-to-end scripts still pass against a local stack: `test_e2e_flow.py`, `test_e2e_comprehensive_flow.py`, `test_concurrency_race_condition.py`.
- Pushkaralu event flow: IN devotee books an event ritual; snapshot shows INR; date and time in IST; pay-at-ghat unchanged.
- An app build from before the change (no `country_code`, no `mode`, no `timezone` in requests) can still register, log in, discover priests and book in India.
- An old JWT (no `country`, no `region` claim) is still accepted in the `in` region.

## Explicitly out of scope

- Converting or holding funds in multiple currencies ourselves.
- Per-country copies of the backend or apps.
- A global multi-master database.
- Account transfer between regions.
- Prasad shipping abroad.
- Call recording.
