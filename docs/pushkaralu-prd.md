# Pooja Vahini Pushkaralu PRD v2

Nov 11, 2026 · @TONY STARK

## Purpose and decisions

Pushkaralu ships as an event mode inside the existing Priest App, Devotee App and temple-cms, targeting production launch on 1 December 2026, with external alpha first. This v2 merges PRD v1.0 and the government accreditation amendment, and records owner decisions taken on 11 November 2026.

| Decision | Answer | Consequence |
| --- | --- | --- |
| Pilot vs Pushkaralu | Run both in parallel on one backend | Hyderabad home-puja pilot (pilot-contract v1.2.0) keeps going; shared booking core must serve both |
| Government issuer | In talks, not signed | Build the workflow; government-issued cards stay disabled until written authorisation. Pooja Vahini cards only until then |
| Payment | Online gateway at launch | Gateway, webhooks, refunds become launch-blocking; supersedes v1.0 direct-payment fallback |
| CMS | temple-cms (Next.js) in this repo | Needs login, roles, jurisdiction scoping before any Pushkaralu module |

Precedence where documents conflict: this v2, then the amendment, then PRD v1.0, then the pilot contract. The pushkaralu.com website stays information-only and links into the booking apps; it stores no bookings, payments or codes.

## Current codebase status

Of 25 defects found in the earlier review, 13 are fixed, 6 partial and 6 open (working tree checked 11 November 2026). No Pushkaralu, event, accreditation, ghat or payment-gateway code exists yet.

| Area | Status | Remaining work |
| --- | --- | --- |
| Unauthenticated write routes (translations, samagri, custom services, store orders, temple tickets, dispatch) | Open | Add auth and role checks to every route listed; /ws/webrtc already fixed |
| Admin token fallback (booking\_handlers.go:106) | Open | Replace with real admin accounts, session version and expiry |
| temple-cms login and API URL | Partial (14 Nov) | Login cookie, middleware and session route exist; API_BASE_URL env used but falls back to localhost:8085; role enforcement not verified |
| Gate scanner fake "verified" on error | Open | Show "Unable to verify"; drop invented names |
| Temple ticket double check-in | Open | Single conditional UPDATE |
| Idempotency keys and money as float | Open | Idempotency-Key on all creates; integer paise |
| Push and SMS | Partial (14 Nov) | FCM sender exists (backend/internal/notify/push.go), end-to-end delivery not verified; SMS deferred |
| Vendor app | Open (deferred) | Placeholder screens only; out of launch scope |
| CreateBooking side effects | Partial | Slot hold now bound; material and chat inserts still ignore errors outside the transaction |
| /dispatch/match fakes | Partial | Remove test request and fake priest fallbacks |
| Fixed-interval polling | Partial | Nearby priests (2 s) and priest app (2.5 s) still poll without backoff |
| Priest foreground location service | Partial | Permissions declared; service not implemented |
| docker-compose TURN relay ports | Partial | Prod compose has gateway and CMS; TURN relay range still unpublished |
| CORS, WebSocket origins, crash, OTP, secrets, phone exposure, FSM, PostGIS discovery, cleartext, token storage, ngrok URL, iOS hosts | Fixed | None |

## Launch scope for 1 December 2026

Launch includes only features that work end to end with failure handling and CMS support; everything else is explicitly deferred.

| Scope | Items |
| --- | --- |
| In launch | CMS login and roles; event config, toggle and daily refresh; priest application and Pooja Vahini verification; Pooja Vahini event ID card with live QR status; priest services and prices with booking snapshot; duty sessions and 2-minute location; ghat-based map and list discovery; immediate and scheduled booking requests; booking code at meeting; completion and reviews; online payment and refunds; push notifications; support cases; audit log; emergency stop control |
| Built, switched off | Government accreditation workflow, government CMS workspace, government integration API (enabled per authority after signed approval) |
| Conditional | Saved priests (My Family Priest), event guide content, family booking share link |
| Deferred | Vendor app and Puja Store, live E-Pooja video, colleague referral, Rapido-style 45-second auto-dispatch, dynamic pricing, crowd prediction, SMS/WhatsApp, chat beyond existing booking chat |

## Functional requirements

Requirement IDs follow PRD v1.0; only changes and launch-critical rules are restated here. The server enforces every eligibility rule; a stale app config never overrides it.

| ID | Requirement | Launch rule |
| --- | --- | --- |
| EVT-001..005 | Event record, lifecycle Draft > Scheduled > Active > Paused > Ended > Archived, toggle, config refresh | Config fetched on launch and on resume if older than 24 h; urgent refresh via push data message; disabling never cancels bookings; CMS shows affected active bookings before disable |
| PRI-001..003 | Priest participation application and status | States: Not applied, Draft, Submitted, Changes requested, Approved, Rejected, Suspended; reasons mandatory for negative outcomes |
| ID-001..004 | Pooja Vahini event card, digital + printable + public link | Card says Pooja Vahini is issuer, never implies government accreditation; QR page reads live status; failure shows "Unable to verify current status" |
| SRV-001..004 | Priest services from CMS templates, moderation | Price in paise; booking stores agreement snapshot; staff never silently change a price |
| DUT-001..002 | Start/End duty, Available / Performing / Break | Ending duty with active booking warns and preserves it |
| LOC-001..005 | 2-minute location while on duty | Each point has capture time and accuracy; older points never overwrite newer; fresh up to 5 min, delayed 5-15 min, stale over 15 min and excluded from Available now; collection stops at End duty |
| DEV-001..004 | Ghat selection, map + list, filters | Works without GPS; public map shows approximate location or service point only |
| BKG-001..007 | Request, accept, pay, confirm, meet, complete | Immediate request expires in 5 min, scheduled in 2 h or cutoff; idempotency key on every create; acceptance rechecks availability in one transaction; booking code verified before In progress |
| PAY-001..003 | Online payment | See Payments section |
| CMS-001..012 | Event ops in temple-cms | All modules behind login with role and event scoping; every change audited |
| NTF | Push + in-app notifications | No precise location or codes on lock screen; duplicates suppressed |

## Government accreditation

The workflow is built and tested for launch but stays disabled per authority until that authority signs written approval of branding, card template and identifier rules. Talks are ongoing as of 11 November 2026.

- Two separate concepts: Pooja Vahini profile verification, and government event accreditation. The first never grants the second.
- Flow: priest applies; Pooja Vahini staff check completeness (only if delegated); government staff request corrections, approve or reject with reasons; approval allows issuance; priest gets digital and printable card; government staff can suspend, revoke, expire or replace.
- Pooja Vahini admins never hold government approval rights by default. Any delegation is configured per authority and audited.
- Card fields: issuing authority, jurisdiction, event, validity, priest name and photo, accreditation ID, approved ghats or zones, QR, authority template, live status.
- Access surfaces are separate: public QR page (status only, no documents, contact or precise location); government workspace inside temple-cms; versioned integration API.
- Scoping: every government user and API client is bound to authority + jurisdiction + event + role. No query can return users outside that scope.
- API: one client per integrating system, scoped tokens, rotation and revocation, pagination, filters, rate limits, standard error body, idempotent approve and issue, signed webhooks for status changes, sandbox environment, audit of sensitive reads.
- Location: government viewing of on-duty location is a separate permission, off by default, enabled only under an approved operating policy that fixes purpose, precision, retention and priest notice.
- Alpha uses test cards visibly marked TEST. If no authority signs before launch, the product must not present government accreditation as live.

## Payments

Launch takes full online payment after priest acceptance; a booking is Confirmed only after the gateway webhook reports capture. No payment code exists today, so this is the largest single launch risk.

- Flow: Requested > priest accepts > Accepted, awaiting payment (payment window, proposed 15 min) > devotee pays > webhook > Confirmed. Unpaid in window: payment expired, slot released.
- Payment status lives in its own table and never overwrites booking status.
- All amounts are integer paise with currency INR.
- Webhooks are signature-verified and idempotent on gateway event ID; the client callback alone never confirms.
- Refunds follow the published cancellation policy, are issued through the gateway API, and are tracked to completion.
- Priest payout: settlement timing and method is an open decision; launch can hold funds and settle in batch.
- Platform fee shown to devotee before payment and to priest before acceptance.
- Handle failed, pending, duplicate, refunded and disputed payments; reconciliation report in CMS Finance.
- Gateway provider and merchant account approval are prerequisites; without them by about 20 November, launch must drop to free bookings.

## System design

Keep one Go gateway and extend it; do not split services before launch. dispatch-service, webrtc-signaling and mock\_server.py are retired from the launch path.

&#91;embedded content: target architecture · 5 clients, 1 gateway, 5 backing services\]

New tables, all keyed by event\_id where event-specific:

| Table | Holds |
| --- | --- |
| events, event\_regions, ghats, meeting\_points | Event config, lifecycle, booking switch, config version |
| event\_participations, verification\_records, documents | Priest applications, reviewer, reasons; documents in private object storage |
| credentials | Card ID, issuer (Pooja Vahini or authority), status, validity, replaced\_by |
| authorities, jurisdictions, staff\_grants, api\_clients | Government tenancy and scoped permissions |
| ritual\_templates, priest\_services, agreement\_snapshots | Listings in paise and the frozen terms per booking |
| duty\_sessions, location\_points | Duty start/end and timestamped points with accuracy; TTL by retention policy |
| payments, refunds, webhook\_events | Payment lifecycle separate from booking status |
| support\_cases, notifications, audit\_log | Ops, delivery state, append-only audit |

Key design rules:

- Location ingest: priest app posts every 2 minutes; server rejects points older than the latest stored; latest point kept in Redis GEO for discovery, history in Postgres partitioned by day.
- Discovery: PostGIS or Redis GEO radius query filtered by event, ghat, freshness and duty status; never load all priests.
- Booking accept: one transaction with SELECT FOR UPDATE on the priest's overlapping window; unique constraint prevents two Confirmed bookings in the same slot.
- Config: GET /v1/events/config returns version + ETag; apps cache it; FCM data message forces refresh.
- QR: card URL holds an opaque random ID, not personal data; page renders live status from the server and is cacheable for at most 60 s.
- Android duty tracking runs as a foreground service with a visible notification; it stops at End duty.

## Production readiness and launch gates

Launch on 1 December requires every gate below to have evidence; any unmet gate either blocks launch or removes its feature from scope.

- [ ] Every non-public route requires auth; role, event and jurisdiction scoping tested with cross-tenant negative tests
- [ ] CORS and WebSocket origins restricted to known app and CMS origins
- [ ] Secrets only from a secret store; no default DB password; Redis password set; no ports exposed publicly except the load balancer
- [ ] Concurrent accept, cancel and duplicate submit produce one outcome (automated race tests)
- [ ] Card issue, revoke and QR status verified end to end, including printed card after revocation
- [ ] Location freshness and End duty stop verified on real Android devices with battery restrictions
- [ ] Payment success, failure, duplicate webhook and refund tested in gateway test mode
- [ ] Monitoring: API error rate, latency, booking failures, webhook failures, location ingest lag, push failures, each with an alert owner
- [ ] Daily backup plus a timed restore drill
- [ ] Rollback for API, CMS and app config documented and rehearsed
- [ ] Load test at 2x the agreed peak estimate
- [ ] External security review of auth, documents, APIs and public QR page
- [ ] Privacy, retention, payment and cancellation policies approved and published
- [ ] Support rota, runbooks and incident process staffed for event dates
- [ ] Emergency control stops new bookings or issuance without deleting records
- [ ] Alpha completed with invited priests, devotees and (if available) government reviewers; zero open launch-blocking defects

## Delivery plan and risks

The 1 December date leaves 19 days; the plan only fits if scope cuts are decided by 20 November, when the merchant account must be approved.

&#91;embedded content\]

Dates are proposed, not committed; each bar needs an owner.

| Risk | Likelihood | Fallback |
| --- | --- | --- |
| Merchant account or gateway not ready by 20 Nov | High | Launch with free bookings; payments in next release |
| No government authority signs | High | Launch with Pooja Vahini cards only; government workflow stays off |
| Android background limits stop 2-min updates | Medium | Foreground service; stale priests drop out of Available now |
| Alpha finds blocking defects with 2 days left | Medium | Go/no-go on 30 Nov may move launch or cut features |
| Parallel Hyderabad pilot competes for the same engineers | High | Freeze pilot scope until after launch |
| Support team not staffed for event days | Medium | Emergency stop control; limit ghats at launch |

## Open decisions

Items 1 to 6 block implementation design; the rest block release.

1. Which Pushkaralu (river), regions and ghats launch first, and event dates?
2. Expected peak: priests on duty, devotees, bookings per hour?
3. Payment gateway provider, and is the merchant account approved?
4. Platform fee, settlement timing, cancellation, refund and no-show policy?
5. Mandatory verification documents, and who approves and handles appeals?
6. Languages at launch: English, Telugu, Hindi only?
7. Which authority, jurisdiction and card template if government approval arrives; who in Pooja Vahini owns that relationship?
8. Location retention period and the public vs confirmed-devotee precision rule.
9. Auto-completion rule when the devotee does not confirm.
10. Support staffing and hours during the event.
11. Minimum supported app versions and forced-update policy.
12. If scope must shrink, which goes first: online payments or the government workflow?
