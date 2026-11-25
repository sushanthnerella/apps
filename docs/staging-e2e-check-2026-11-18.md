# Staging end-to-end check — 2026-11-18

Environment: Pooja_Pilot_Test Android emulator (GPS fixed at Rajahmundry), debug APKs built 2026-11-18 with the location fix, staging backend (pooja-vahini-staging.onrender.com), CMS operated by the owner. Test accounts: "Emulator Test Devotee" and "Emulator Test Priest" (credentials in the gitignored `.staging-test-devotee.env`).

## PASS — location fix
- Devotee dispatch screen and Puja Store get a location with no "Could not get a current location" error.
- Not covered: indoor phone with no GPS signal (emulator always has a GPS fix). Check on a real device.

## PASS — normal priest booking
- Priest registers, logs in, goes online, publishes availability (19/11, 09:00–18:00).
- Devotee adds Namakaranam + Samagri (₹3216), sees the priest and slot, sends the request.
- Priest gets the request alert with sankalpam details, accepts, then travel → arrived → puja started.
- Devotee sees each status and the completion code; priest enters it and the booking completes.

## PASS — Pushkaralu event booking
- Priest app shows the Pushkaralu tab; screen opens without crashing; apply → "under review"; price ₹501 saved.
- CMS approval → priest sees "Approved" and an ACTIVE event card; starts ghat duty (Available).
- Devotee Pushkaralu tab: LIVE banner, ghat list, sort by nearest (location OK), priest with ₹501.
- Devotee sends an immediate request and gets booking code 9177; priest accepts, enters 9177 to start, completes.
- Devotee sees Requested → Accepted → Started → Completed.

## Issues found (not fixed)
1. Devotee app loses its login after the app is force-stopped; switching apps also resets in-progress selections to the Services screen.
2. Priest "1. travel" button stays enabled after arrival / during the puja (the status might be able to go backwards; not tapped).
3. Devotee can still cancel a booking while the puja is in progress.
4. Priest booking-code dialog at event start reuses the completion-code text ("ask after the puja", "confirm completion").
5. Accept countdown mismatch: priest popup shows ~30s, devotee shows "Expires in 5:00".
6. Event bookings use the home-visit stages (travel/arrived) even though the devotee meets the priest at the ghat.
7. Priest app opens in Telugu on an English device.
8. CMS data: ghat "suskfnk" shows 6726 km from Rajahmundry (coordinates wrong); the ritual's Telugu name shows the event name ("గోదావరి పుష్కరాలు") instead of the ritual.

Not tested: scheduled event booking, priest reject/expiry, CMS booking/card switches and emergency stop on staging, vendor store order (no vendors on staging), iOS.
