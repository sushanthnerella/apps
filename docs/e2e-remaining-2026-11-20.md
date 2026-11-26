# Remaining flow check — 20 November 2026

Environment: Android 35 emulator, staging backend c554ec1, installed devotee and priest debug APKs. Original Android 36 emulator repeatedly froze; Android 35 needed a cold start. Test accounts only.

| Flow | Result | Observation |
|---|---|---|
| Normal booking | PASS (core path) | Published 21 Nov 09:00–18:00; devotee selected slot and sent request; priest accepted, moved through transit, arrival, start, and completed with correct code. Devotee history displayed Completed. Earlier step button availability and wrong code were not rechecked. |
| Scheduled event booking | PARTIAL PASS | Devotee scheduled 21 Nov 09:00; priest alert showed date/time, accepted; devotee showed Booking confirmed. Cancelled afterwards; database confirmed CANCELLED. Devotee confirmation did not show the scheduled date/time. An API-only scheduled create/list/accept/cancel also succeeded. |
| Immediate expiry | PARTIAL PASS | Request created through staging API; priest alert appeared with countdown, then closed after expiry. Database showed CANCELLED and PAY_AT_GHAT. Devotee booking history displayed Cancelled; the live devotee expiry message was not observed. |
| Break | PASS | Priest selected Break; devotee ghat displayed 0 available. Restored Available and verified database. |
| CMS controls | NOT TESTED | Awaiting CMS admin action; booking remained enabled at last check. |
| Remaining paths | NOT TESTED | Normal cancellation and decline, event busy/overlap, emergency stop, languages, large font, Firebase and payment. |

Issue: scheduled booking confirmation screen omits the selected date and time even though the priest alert and backend contain them. Expected the scheduled date and time on the devotee confirmation. Expired booking history says Cancelled; live expiry wording not checked. No app crash was found in the inspected crash buffer. Test bookings were completed or cancelled; one test availability slot was published. Screenshots in output/remaining-flows/.
