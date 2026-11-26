# Deployment Guide: AWS, per-country compliance, App Store, Play Store

Written 2026-11-17. Companion to [world-ready-plan.md](world-ready-plan.md).

This is an engineering runbook, not legal advice. Each compliance section says what the system must do; have a lawyer in each country confirm before launch. Items marked **VERIFY** were not confirmed from an official source while writing and must be checked on the linked official page.

## 0. What gets deployed

| Component | Source | Port | Notes |
|---|---|---|---|
| API gateway (Go) | `backend/Dockerfile`, `backend/cmd/api-gateway` | 8085 | HTTP + WebSocket (`/ws/webrtc`, chat, tracking). Health: `GET /health` |
| Migrator (Go) | `backend/cmd/migrate` | - | one-off task per deploy |
| Admin bootstrap | `backend/cmd/create-admin` | - | one-off, first deploy only |
| Temple CMS (Next.js 14) | `temple-cms/Dockerfile` | 3000 | |
| PostgreSQL 16 + PostGIS 3.4 | - | 5432 | |
| Redis | - | 6379 | |
| TURN/STUN (coturn) | - | 3478 UDP/TCP, 49152-65535 UDP | needed for virtual pooja video |

Environment variables read by the gateway (from `docker-compose.prod.yml`): `PORT`, `PG_CONN_STR`, `REDIS_ADDR`, `REDIS_PASSWORD`, `JWT_SECRET`, `ALLOWED_ORIGINS`. CMS: `NEXT_PUBLIC_API_BASE_URL`, `NEXT_PUBLIC_API_URL`. SMTP settings for password recovery (see `backend/internal/auth/email.go`).

## 1. Regions and data residency

| Deploy region | AWS region | Countries served | Personal data stored here |
|---|---|---|---|
| `in` | `ap-south-1` Mumbai (DR: `ap-south-2` Hyderabad) | India | all Indian users, priests, bookings, backups, logs |
| `eu` | `eu-west-2` London (DR: `eu-west-1` Ireland) | UK, EU | all UK/EU users |
| `us` | `us-east-1` N. Virginia (DR: `us-west-2`) | USA, Canada | all US/CA users |

Rules that apply to every region:

- Database, backups, snapshots, logs, S3 buckets and DR copies stay inside the region's country or legal bloc. **Never enable cross-region replication from `in` or `eu` to another bloc.**
- Each region has its own KMS keys, JWT secret and database credentials.
- Global services (Route 53, CloudFront, IAM) carry no personal data. Do not put personal data in DNS names, tags or CloudFront logs.
- Start with `in` only. Add a region only when a country in it is ready to launch.

## 2. Compliance by country

### India: DPDP Act 2023 and DPDP Rules 2025

Facts: MeitY notified the DPDP Rules on 13 Nov 2025. Enforcement is phased; the main obligations are due about 18 months later (13 May 2027).

**On data localisation:** the DPDP Act does **not** impose a blanket rule that all Indian personal data must stay in India. It permits transfer abroad except to countries the government restricts, and the Rules allow extra restrictions for Significant Data Fiduciaries. Separate from DPDP, **RBI requires payment system data to be stored only in India.** Our decision: keep all Indian personal data in `ap-south-1` anyway. It satisfies RBI, removes the question for any future restriction, and costs nothing extra.

System requirements:

| Requirement | How we meet it |
|---|---|
| Notice and consent, in plain language, in English and the user's language | Consent screen at registration in English, Telugu, Hindi; store consent text version and timestamp (priests already have `consent_agreed`, `consent_timestamp`; add the same for users) |
| Withdraw consent as easily as giving it | In-app setting |
| Access, correction, erasure requests | In-app "Download my data" and "Delete my account"; admin tool in CMS |
| Erase when purpose is served | Retention job: delete or anonymise inactive accounts and old location history. Set the periods with counsel |
| Children under 18: verifiable parental consent, no tracking or targeted ads | Age gate at registration; block under-18 accounts until a consent flow exists |
| Reasonable security safeguards | Encryption at rest and in transit, access control, logs kept at least 1 year |
| Breach notice | Tell the Data Protection Board and affected users without delay; detailed report to the Board within 72 hours |
| Grievance officer | Name and contact published in app and website |
| Payment data in India only | Indian payment gateway; no payment data in any non-India region |

Also: CERT-In directions require reporting certain cyber incidents within 6 hours and keeping logs for 180 days in India (**VERIFY** current text).

### UK: UK GDPR and Data Protection Act 2018

| Requirement | How we meet it |
|---|---|
| Register with the ICO and pay the data protection fee | Before launch |
| Lawful basis per purpose; religious belief is **special category data** (Article 9) | Booking a pooja reveals religion. Get **explicit consent** at registration and record it |
| DPIA for high-risk processing | Write one: special category data plus location tracking |
| Data subject rights (access, erasure, portability, objection) within 1 month | Same in-app tools as India |
| Breach notice to ICO within 72 hours | Incident runbook (section 12) |
| International transfers need a safeguard | UK data stays in `eu-west-2`. Virtual pooja with an India priest sends name and booking details to India: use the UK IDTA or Addendum, and get the user's consent screen described in the world-ready plan Phase 10 |
| Processor contracts | Sign the AWS DPA (in the AWS customer agreement) and DPAs with payment, email and push providers |
| UK representative if the company has no UK establishment | Appoint one |
| PECR: consent for non-essential cookies and marketing messages | CMS and website cookie banner; marketing opt-in |
| Consumer law: 14-day cooling-off for distance contracts, with exceptions for dated services | State the cancellation policy at checkout; confirm with counsel |
| Priest vetting | Basic DBS check |

### EU: GDPR

Same as UK plus: appoint an EU representative (Article 27), lead supervisory authority, Standard Contractual Clauses for transfers to India. Serving EU users from London relies on the EU's adequacy decision for the UK (**VERIFY** it is still in force at launch; if not, move EU users to `eu-west-1`).

### USA

No single federal privacy law. Meet the strictest state rules and it covers the rest:

| Requirement | How we meet it |
|---|---|
| CCPA/CPRA (California) and similar laws in other states: notice, access, deletion, opt-out of sale/sharing | We do not sell data; say so in the policy. Same in-app access and delete tools |
| Sensitive data (religion, precise location) needs opt-in consent in several states | Same explicit consent screen |
| COPPA: no accounts under 13 without parental consent | Age gate |
| Breach notification laws in every state | Incident runbook |
| Sales tax varies by state | Tax provider through the `TaxCalculator` interface |
| Priest vetting | Background check provider |

### Canada, Australia, Singapore, UAE

Launch virtual pooja only at first. Before in-person launch review PIPEDA (Canada), Privacy Act 1988 (Australia), PDPA (Singapore), PDPL (UAE). UAE has stricter localisation for some sectors; use `me-central-1` if required.

## 3. AWS account setup (from zero)

1. Create the **management account** with a company email alias (not a personal address). Use the company's legal name and GST/VAT number so invoices are correct.
2. Root user: set a strong password, enable hardware MFA, create no access keys, store credentials in the company password manager. Never use root again except for billing-level tasks.
3. Enable **AWS Organizations**, then **IAM Identity Center** (SSO). Every human signs in through SSO with MFA. No long-lived IAM users.
4. Create accounts with **Control Tower** (or manually):

   | Account | Purpose |
   |---|---|
   | management | billing and organisation only |
   | log-archive | CloudTrail and Config logs |
   | security | GuardDuty, Security Hub admin |
   | shared | ECR, CI/CD roles |
   | staging | one region (`ap-south-1`) |
   | prod-in, prod-eu, prod-us | one account per data bloc |

   One production account per bloc makes residency enforceable with a policy instead of with discipline.
5. **Service control policies:** on `prod-in` deny every region except `ap-south-1`, `ap-south-2` and the global endpoints; same idea for `prod-eu` and `prod-us`. Deny leaving the organisation, deny disabling CloudTrail/GuardDuty/Config, deny root access keys.
6. Organisation-wide: CloudTrail (all regions, log file validation, to log-archive), AWS Config, GuardDuty, Security Hub, IAM Access Analyzer.
7. Billing: budgets with email alerts at 50/80/100%, Cost Anomaly Detection, cost allocation tags `env`, `region`, `service`.
8. Support plan: Business for production accounts.
9. Permission sets: `Admin` (2 people, break-glass), `Deployer` (CI role), `ReadOnly`, `DBA`. Review quarterly.

## 4. Network (per production account)

VPC `10.<n>.0.0/16` across **3 Availability Zones**. Use a different `<n>` per region.

| Subnet tier | Per AZ | Holds | Route |
|---|---|---|---|
| public | /24 | ALB, NAT gateways, coturn | internet gateway |
| private-app | /22 | ECS tasks | NAT gateway |
| private-data | /24 | RDS, ElastiCache | no internet route |

- NAT gateway per AZ in production (one in staging).
- VPC endpoints: S3 (gateway), ECR api/dkr, Secrets Manager, CloudWatch Logs, SSM, KMS. Saves NAT cost and keeps traffic private.
- VPC Flow Logs to CloudWatch, 1 year retention.

Security groups (deny by default; reference groups, not IP ranges):

| Group | Inbound |
|---|---|
| `alb` | 443 from `0.0.0.0/0` (or CloudFront prefix list if fronted by CloudFront) |
| `api` | 8085 from `alb` |
| `cms` | 3000 from `alb` |
| `rds` | 5432 from `api` and from `migrate` |
| `redis` | 6379 from `api` |
| `turn` | 3478 UDP+TCP and 49152-65535 UDP from `0.0.0.0/0` |

No SSH anywhere. Use ECS Exec and SSM Session Manager, both logged.

## 5. DNS, certificates, edge

1. Register or transfer the domain to Route 53. One public hosted zone.
2. Names: `in.api.<domain>`, `eu.api.<domain>`, `us.api.<domain>`, `in.cms.<domain>`, `turn-in.<domain>`, plus `api.<domain>` for the region directory.
3. **ACM** certificate per region for `*.api.<domain>` and `*.cms.<domain>`, DNS validation, auto-renew.
4. **AWS WAF** on each ALB: AWS managed rule groups (Common, KnownBadInputs, SQLi, IP reputation), plus rate-based rules: 100 requests / 5 min per IP on `/api/v1/auth/*`, 2000 / 5 min per IP overall. The app already throttles logins; WAF is the outer layer.
5. Shield Standard is automatic.

## 6. Load balancer

One **Application Load Balancer** per region, internet-facing, in the 3 public subnets.

| Setting | Value | Why |
|---|---|---|
| Listener 80 | redirect to 443 | |
| Listener 443 | TLS policy `ELBSecurityPolicy-TLS13-1-2-2021-06`, ACM cert | TLS 1.2+ only |
| Rule: host `*.api.<domain>` | target group `api` | |
| Rule: host `*.cms.<domain>` | target group `cms` | |
| Idle timeout | **3600 s** | WebSockets for video signalling, chat and tracking drop at the default 60 s |
| Target type | `ip` | required for Fargate |
| `api` health check | `GET /health`, 15 s interval, 5 s timeout, 2 healthy / 3 unhealthy, expect 200 | |
| `cms` health check | `GET /`, 20 s interval | |
| Deregistration delay | 60 s for `api` | lets in-flight requests finish |
| Stickiness | off | state is in Postgres and Redis |
| Drop invalid headers | on | |
| Deletion protection | on | |
| Access logs | on, to an S3 bucket in the same region, 1 year lifecycle | |
| HTTP/2 | on | |

**Before running more than one gateway task:** confirm the WebSocket hubs in `backend/cmd/api-gateway/realtime_hubs.go` broadcast through Redis pub/sub and not only in process memory. If they are in-memory, two peers on different tasks will not see each other. Fix that first, or run a single task.

Set `ALLOWED_ORIGINS` to the exact CMS origin for that region. Have the gateway trust `X-Forwarded-For` only from the ALB (check `auth.ExtractClientIP`), or login throttling will key on the ALB's address.

**TURN cannot sit behind an ALB** (UDP). Run coturn on two small EC2 instances (`t4g.small`), one per AZ, each with an Elastic IP, in the public subnets, with a Route 53 record listing both. Use TURN REST API time-limited credentials (`use-auth-secret`), not the static `--user` in the compose file. Set `external-ip` to the Elastic IP and the realm to the region's hostname. Enable TLS on 5349 with a certificate.

## 7. Compute: ECS on Fargate

One ECS cluster per region. Images in ECR (shared account, replicated to each region; images contain no personal data). Enable scan-on-push and immutable tags.

| Service | CPU / memory | Count | Auto scaling |
|---|---|---|---|
| `api-gateway` | 0.5 vCPU / 1 GB | min 2, max 10, spread across AZs | target 60% CPU, and ALB requests per target |
| `temple-cms` | 0.25 vCPU / 0.5 GB | min 2, max 4 | target 60% CPU |
| `migrate` | 0.25 vCPU / 0.5 GB | run-task only | - |

Task settings:

- Secrets injected from **Secrets Manager** by ARN (`PG_CONN_STR`, `REDIS_PASSWORD`, `JWT_SECRET`, SMTP password). Never in the image or plain env.
- `PG_CONN_STR` must include `sslmode=verify-full` and the RDS CA bundle.
- Task role: least privilege (only the S3 prefixes and secrets it needs). Execution role: pull image, read its secrets, write logs.
- Logs: `awslogs` driver to CloudWatch, 1 year retention (India: at least 180 days in-country).
- Read-only root filesystem, non-root user, no privileged mode.
- Deployment: rolling, minimum healthy 100%, maximum 200%, **circuit breaker with rollback on**.
- Enable ECS Exec for debugging (audited through CloudTrail).
- Graceful shutdown: gateway must stop accepting on SIGTERM and finish within the 60 s deregistration delay.

## 8. Database: RDS for PostgreSQL

| Setting | Production value |
|---|---|
| Engine | PostgreSQL 16 |
| Instance | `db.m6g.large` to start (staging `db.t4g.medium`) |
| Deployment | **Multi-AZ** |
| Storage | gp3, 100 GB, autoscaling to 500 GB |
| Encryption | on, customer-managed KMS key for that region |
| Subnet group | private-data subnets |
| Public access | **off** |
| Backups | automated, **35 days** retention, point-in-time recovery |
| Cross-region backup copy | only to the DR region in the same bloc (`ap-south-2` for India) |
| Deletion protection | on |
| Maintenance window | lowest-traffic local night |
| Minor version upgrade | auto |
| Performance Insights | on, 7 days |
| Enhanced monitoring | 60 s |
| Log exports | `postgresql`, `upgrade` to CloudWatch |

Parameter group:

```
rds.force_ssl = 1
log_min_duration_statement = 500
log_connections = 1
log_disconnections = 1
shared_preload_libraries = pg_stat_statements
idle_in_transaction_session_timeout = 60000
```

First-time setup, run from a one-off ECS task or SSM port-forward:

```sql
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE ROLE purohit_app LOGIN PASSWORD '<from Secrets Manager>';
CREATE ROLE purohit_migrate LOGIN PASSWORD '<from Secrets Manager>';
GRANT CONNECT ON DATABASE purohit_db TO purohit_app, purohit_migrate;
```

- The app role gets data read/write only; the migrate role owns the schema. The gateway never connects as the master user.
- PostGIS matters: `backend/internal/db/discovery.go` uses `ST_DWithin` and falls back to slower maths if the extension is missing. Confirm `SELECT postgis_version();` after setup.
- **Migrations:** the compose file mounts `backend/migrations` into Postgres init. RDS has no such hook. Run the `migrate` task before each deploy.
- Connections: put **RDS Proxy** in front once there are more than about 4 gateway tasks.
- Enable Secrets Manager rotation for both roles.
- Restore drill: restore the latest snapshot to a scratch instance every quarter and run the test suite against it.

## 9. Redis: ElastiCache

| Setting | Value |
|---|---|
| Engine | Valkey or Redis OSS 7.x, cluster mode disabled |
| Nodes | `cache.t4g.small`, 1 primary + 1 replica, Multi-AZ, automatic failover |
| Encryption | at rest and in transit (TLS) |
| Auth | auth token from Secrets Manager |
| Subnets | private-data |
| Snapshots | daily, 7 days |

The compose file uses `redis/redis-stack-server`. ElastiCache does **not** include Redis Stack modules (RedisJSON, RediSearch). Core commands (GEO, pub/sub, locks) work. Grep the backend for module commands (`JSON.`, `FT.`) before migrating; if any are used, replace them or use MemoryDB / self-managed Redis.

The gateway must connect with TLS (`rediss://`). Check the Redis client options where `REDIS_ADDR` is read.

## 10. Storage, secrets, keys, email, push

- **S3:** one bucket per purpose per region (uploads, ALB logs, exports). Block Public Access on, SSE-KMS, versioning on, lifecycle rules, bucket policy denying non-TLS requests. No cross-region replication out of the bloc.
- **KMS:** one customer-managed key per region per data class (rds, s3, secrets). Key rotation on.
- **Secrets Manager:** one secret per value, per region. `JWT_SECRET` is at least 32 random bytes and different in every region and environment.
- **Email (password recovery):** Amazon SES in the same region, out of sandbox, with SPF, DKIM and DMARC on the sending domain. Point the existing SMTP settings at the SES SMTP endpoint.
- **Push:** Firebase Cloud Messaging for Android, APNs for iOS (`backend/internal/notify/push.go`). Keep push payloads free of personal data; send an id and let the app fetch details.

## 11. CI/CD

GitHub Actions with **OIDC** to assume a `Deployer` role in each account. No AWS keys in GitHub.

Pipeline per merge to `main`:

1. `go vet`, `go test ./...`, Kotlin tests, CMS lint and build.
2. Build images, tag with the git SHA, push to ECR, wait for the scan; fail on critical findings.
3. Deploy to staging: run `migrate` task, wait for success, update services, wait for stable, run smoke tests (`/health`, login, nearby priests).
4. Manual approval.
5. Deploy production **one region at a time**, `in` first: `migrate`, then services. Stop on first failure. Next region only after 30 minutes healthy.
6. Rollback: redeploy the previous task definition. Migrations must be backward compatible with the previous image (add columns first, remove in a later release).

## 12. Monitoring, alarms, incidents

CloudWatch alarms to an SNS topic that pages on-call:

| Alarm | Threshold |
|---|---|
| ALB 5xx rate | > 1% for 5 min |
| ALB target response time p95 | > 1.5 s for 10 min |
| Unhealthy targets | >= 1 for 3 min |
| ECS running tasks | < desired for 5 min |
| RDS CPU | > 80% for 10 min |
| RDS free storage | < 20% |
| RDS connections | > 80% of max |
| RDS replica/failover event | any |
| Redis memory | > 75% |
| Redis evictions | > 0 |
| WAF blocked requests | sudden spike |
| GuardDuty finding, severity high | any |
| Certificate expiry | < 30 days |

Also: a CloudWatch Synthetics canary per region hitting `/health` and a login every 5 minutes; a status page.

Incident runbook (keep the long form in the team wiki):

1. Contain: revoke credentials, isolate the task or instance, snapshot for evidence.
2. Assess: what data, how many people, which countries.
3. Notify on the clock: India: Data Protection Board and users without delay, detailed report within 72 hours; CERT-In within 6 hours for listed incident types. UK/EU: ICO or lead authority within 72 hours, users if high risk. USA: per state law.
4. Fix, then write a blameless review within 5 working days.

Named roles before launch: incident lead, Grievance Officer (India), data protection contact (UK/EU).

## 13. Go-live checklist (per region)

- [ ] SCP region lock tested: creating a resource outside the bloc fails.
- [ ] RDS is Multi-AZ, encrypted, not public; `rds.force_ssl=1`; PostGIS installed.
- [ ] Restore from backup tested.
- [ ] Failover tested: reboot RDS with failover and kill a gateway task under load; no user-visible outage beyond a short blip.
- [ ] Two gateway tasks in different AZs; WebSocket chat and video work across tasks.
- [ ] TURN reachable on UDP from a mobile network; a video call connects between two phones on different carriers.
- [ ] HTTPS only; SSL Labs grade A; HSTS header on.
- [ ] WAF rate limit triggers on a login flood.
- [ ] No secret in images, env files or the repository (run a secret scanner).
- [ ] Alarms fire to the on-call phone (test one).
- [ ] Load test at 3 times expected peak (`benchmark_load_test.py`).
- [ ] Privacy policy, terms, consent screens, delete-account and data-export flows live.
- [ ] Country compliance section above ticked and signed off by counsel.

## 14. Rough monthly cost per region (estimate, **VERIFY** in the AWS Pricing Calculator)

Fargate (2 api + 2 cms) about $60; RDS `db.m6g.large` Multi-AZ about $300; ElastiCache about $50; ALB about $25; NAT gateways (3) about $110 plus data; TURN instances about $30 plus data transfer for relayed video; WAF, logs, KMS, backups about $50. Around **$600-700 per region** before traffic. Staging at about a third of that with single-AZ and one NAT.

## 15. Google Play Store

Official pages to follow: Play Console Help (`support.google.com/googleplay/android-developer`).

Account:

1. Create a Google account on a company email alias. Register at `play.google.com/console`. One-time fee about **$25** (**VERIFY**).
2. Choose **Organization** account. It requires a **D-U-N-S number** matching the legal entity, plus verified website, phone and email (**VERIFY** current list). Getting a D-U-N-S can take days to weeks; start now.
3. Complete identity and developer verification.
4. A **personal** account created after 13 Nov 2023 must run a closed test with at least **12 testers opted in for 14 continuous days** before it can apply for production. Organisation accounts are exempt. Another reason to use an organisation account.

App preparation (one listing each for devotee, priest, vendor):

1. Unique `applicationId` per app. Set `versionCode` and `versionName` in each `apps/*/build.gradle.kts`.
2. Target the API level Google currently requires for new apps (**VERIFY** the "Target API level requirements" page; it rises every August).
3. Build a release **Android App Bundle**: `./gradlew :apps:devotee-app:bundleRelease`. Enable R8.
4. Enrol in **Play App Signing**. Keep the upload keystore in the password manager and out of git.
5. Production builds point at the region directory URL, not a staging URL. No cleartext HTTP.

Store listing and declarations:

- App name, short and full description, icon 512x512, feature graphic 1024x500, phone screenshots (and 7-inch/10-inch if supported), per language.
- **Privacy policy URL** (public web page).
- **Data safety form:** declare phone number, name, precise location, religious information where collected, messages, device ids; state encryption in transit and that users can request deletion.
- **Account deletion:** apps with account creation must offer deletion in the app **and** a web link where users can request it. Enter that URL in the Data safety section.
- **Permissions:** priest app tracks location while on duty. Background location needs a declaration form, an in-app prominent disclosure before the system prompt, and a short demo video. Use a foreground service of type `location`. The devotee app should request foreground location only.
- Content rating questionnaire, target audience (18+ keeps it out of the Families policy), ads declaration (no ads), government-app declaration (no; even if a government body issues event credentials, declare any official affiliation truthfully and keep proof).
- **Payments:** paying for a real-world service (a priest visit) or a service delivered live by a person must use a non-Play payment method; Play Billing is for digital goods. Virtual pooja is a live one-to-one service; confirm against the Payments policy before launch (**VERIFY**).
- Countries: select only countries whose `countries.is_active` is true.

Release flow: internal testing (team) → closed testing (pilot priests and devotees) → open testing (optional) → production with **staged rollout** 10% → 50% → 100%. Add a pre-launch report check and watch Android vitals (crash and ANR rates) between steps. First review can take several days; plan a week.

## 16. Apple App Store

Official pages: `developer.apple.com/programs/enroll`, App Store Review Guidelines, App Store Connect Help.

Account:

1. Apple Account on a company email alias with two-factor authentication.
2. Enrol in the **Apple Developer Program as an Organization**: needs a **D-U-N-S number** (Apple uses it to verify the legal entity; allow up to 5 business days from D&B, then up to 2 more weeks for Apple to see it), legal entity status (not a sole trade name), a public website on the company domain, and a person with authority to sign. Fee about **$99/year** (**VERIFY**).
3. In App Store Connect: accept agreements, add tax and banking (needed even for free apps if anything is ever sold), add team members with least-privilege roles.

App preparation (`apps/iosApp`):

1. Bundle ids per app; register them, plus capabilities: Push Notifications, Background Modes (location for the priest app, VoIP/audio for calls if used), Associated Domains if using universal links.
2. Automatic signing with a distribution certificate; APNs key (`.p8`) uploaded to the push provider.
3. Build with the Xcode and SDK version Apple currently requires for submissions (**VERIFY**; it changes each spring).
4. `Info.plist` purpose strings, specific and honest: `NSLocationWhenInUseUsageDescription`, `NSLocationAlwaysAndWhenInUseUsageDescription` (priest app only), `NSCameraUsageDescription`, `NSMicrophoneUsageDescription` (virtual pooja), photo library if used.
5. **Privacy manifest** (`PrivacyInfo.xcprivacy`) for the app and every third-party SDK, declaring collected data and required-reason APIs.
6. App Transport Security on; no exceptions in production.
7. Archive in Xcode → upload → **TestFlight**: internal testers immediately; external testers after Beta App Review.

App Store Connect listing:

- Name, subtitle, description, keywords, support URL, marketing URL, **privacy policy URL**, screenshots for the required iPhone sizes (and iPad if supported), per language.
- **App Privacy** ("nutrition label"): same data types as the Play data safety form. Religious belief is sensitive information; declare it.
- Age rating questionnaire. Export compliance: uses standard HTTPS encryption only (exempt), set `ITSAppUsesNonExemptEncryption` to `NO` if that is accurate.
- **Review notes:** give working demo accounts for devotee and priest, explain the priest's background location use, and explain how to see a booking end to end in a region where reviewers have no priests (provide a demo mode or a seeded reviewer city).

Guidelines that commonly cause rejection for this kind of app:

- **5.1.1(v) account deletion:** must be possible inside the app, not only by email.
- **3.1.3(e) / 3.1.3(d):** physical services use non-Apple payment; live **one-to-one** services between two people may use non-Apple payment; one-to-many live services must use in-app purchase. Virtual pooja for one family is one-to-one; a broadcast pooja for many paying viewers would need in-app purchase (**VERIFY** wording at submission time).
- **4.2 minimum functionality** and **2.1 completeness:** no placeholder screens, no dead buttons (for example the `onOpenLiveStream` stub), no test data.
- **5.1.2:** no location collection beyond what the feature needs; ask at the moment of use.
- **1.2 user-generated content:** chat needs report and block.
- Login: if any third-party social login is added, Sign in with Apple must be offered too.

Release: submit for review (usually 1-3 days), choose manual release, then **phased release** over 7 days. Keep the previous build ready for an expedited fix.

## 17. Order to do all this

1. Start D-U-N-S, Play organisation account and Apple organisation enrolment today; they have the longest waits.
2. AWS organisation, accounts, SCPs, logging (section 3).
3. Staging in `ap-south-1` end to end (sections 4-11).
4. `prod-in`, go-live checklist, India compliance sign-off.
5. Closed test on Play and TestFlight with pilot users.
6. Store submissions.
7. Next region only after the world-ready plan phases 1-6 are merged.

## Sources

- [Mondaq: DPDP Rules 2025 notified](https://www.mondaq.com/india/data-protection/1708164/digital-personal-data-protection-rules-2025-notified)
- [AZB & Partners: India's DPDP framework comes into effect](https://www.azbpartners.com/bank/update-indias-digital-personal-data-protection-framework-comes-into-effect/)
- [Bar and Bench: MeitY notifies final DPDP Rules 2025](https://www.barandbench.com/view-point/meity-notifies-final-digital-personal-data-protection-rules-2025)
- [SCC Online: DPDP Rules 2025 key highlights](https://www.scconline.com/blog/post/2025/12/26/digital-personal-data-protection-rules-2025-key-highlights/)
- [Apple: D-U-N-S Number help](https://developer.apple.com/support/D-U-N-S/)
- [Testers Community: Google Play closed testing requirements 2026](https://www.testerscommunity.com/blog/google-play-closed-testing-requirements-2026)
- [Tapcart: Google Play Console 2026 requirements](https://help.tapcart.com/en/articles/13800709-google-play-console-2026-requirements)
