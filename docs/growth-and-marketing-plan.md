# Growth and Performance Marketing Plan

Written 2026-11-17. Covers India (tier 1, 2, 3) and overseas.

**Read this first:** every rupee figure below is a **planning assumption to start from, not a measured benchmark.** Nobody has run ads for this app yet. Run the small tests in section 6, replace the assumptions with your own numbers in 4 weeks, then scale. The formulas are the durable part.

## 1. The business in one line

A two-sided marketplace. A booking needs a priest nearby (or online) **and** a devotee. Ads can buy devotees; ads cannot buy priests cheaply. So in every city: **supply first, demand second.** Spending on devotee ads in a city with no priests burns money and earns one-star reviews.

Liquidity rule before any devotee ads in a city:

| Mode | Minimum supply |
|---|---|
| In-person | 15 verified active priests per city zone, covering the top 5 poojas, 80% of requests accepted within 30 minutes |
| Virtual | 30 priests offering virtual, across Telugu, Hindi, Tamil, Kannada, English |

## 2. Unit economics (fill in your real numbers)

Definitions:

- **AOV** = average booking value (dakshina + samagri).
- **Take rate** = platform's share of AOV.
- **Contribution per booking** = AOV × take rate − payment fee − support cost − refunds.
- **Bookings per customer per year** = repeat rate.
- **LTV (12 months)** = contribution per booking × bookings per year.
- **CAC** = total acquisition spend ÷ new customers who completed a first booking (not installs, not sign-ups).

Rules:

1. **Maximum CAC = LTV ÷ 3.**
2. **Payback under 6 months** on contribution.
3. Track **blended CAC** (all spend ÷ all new customers, including organic) and **paid CAC** (ad spend ÷ customers from ads). Decisions on channels use paid CAC; the health of the business uses blended.

Worked example (assumed numbers, replace them):

| | India tier 1 | India tier 2/3 | Overseas virtual | Overseas in-person |
|---|---|---|---|---|
| AOV | ₹3,500 | ₹2,000 | ₹4,000 (£38) | £150 |
| Take rate | 15% | 15% | 20% | 15% |
| Contribution per booking | ₹450 | ₹250 | ₹700 | £19 |
| Bookings per year | 2.5 | 2.5 | 3 | 2 |
| LTV 12 months | ₹1,125 | ₹625 | ₹2,100 | £38 |
| **Max CAC (LTV ÷ 3)** | **₹375** | **₹210** | **₹700** | **£13** |

What this shows:

- At these assumptions, paid ads alone will struggle in tier 2/3. Growth there must be mostly referral, priest-led and temple-led.
- Overseas virtual has the best room for paid ads. It also needs no local priests. Start overseas there.
- The three levers that raise allowed CAC: higher AOV (samagri kit, packages), higher take rate, more repeat bookings (festival calendar, annual rituals, reminders). Repeat is the biggest.

## 3. The funnel and what to measure

| Step | Target to aim for | If below target |
|---|---|---|
| Ad click → install | 20-30% | fix store listing, screenshots, ratings |
| Install → registration | 50%+ | shorten sign-up, ask location later |
| Registration → sees priests | 80%+ | supply gap in that area |
| Sees priests → booking request | 10-15% | price clarity, priest profiles, trust signals |
| Request → priest accepts | 80%+ | supply or notification problem |
| Accepted → completed | 90%+ | cancellations; check reasons |
| Completed → second booking in 90 days | 25%+ | reminders, festival prompts |

Install cost × the inverse of these rates = CAC. Example: ₹40 per install × (1 ÷ 0.5 ÷ 0.8 ÷ 0.12 ÷ 0.8 ÷ 0.9) ≈ ₹1,160 per first booking. That is 3 times over the tier 1 ceiling, which is why **funnel fixes come before budget increases.** Doubling request rate from 12% to 24% halves CAC; no ad optimisation does that.

**Tracking needed before spending anything:** Firebase Analytics events for each funnel step, install attribution (Google Play Install Referrer plus a mobile measurement partner once spend passes about ₹3 lakh/month), a `city`, `source`, `campaign` field stored on the user at registration, and a weekly cohort sheet. Send "first booking completed" back to Google and Meta as the conversion event, so their bidding optimises for bookings and not installs.

## 4. Channels

Ranked by expected cost per first booking, cheapest first.

| # | Channel | Works because | Cost model |
|---|---|---|---|
| 1 | **Priest-led acquisition** | Every priest already has 50-300 families. Give each priest a QR code and link; their families book through the app | ₹50-100 credit to the priest per new family's first booking |
| 2 | **Devotee referral** | Poojas are social; relatives ask "who was your priest?" | ₹100 off for both sides, paid only after a completed booking |
| 3 | **Temples and events** | Trust transfer. Pushkaralu is the flagship: lakhs of pilgrims with a need on the spot | Standees, QR at ghats, volunteers; cost is staff and print |
| 4 | **WhatsApp** | Default channel in tier 2/3. Booking confirmations, muhurtham reminders, festival broadcasts to opted-in users, community groups | Per-conversation fees; low |
| 5 | **Google Search** | Highest intent: "pandit for griha pravesh near me", "satyanarayan pooja pandit Hyderabad" | Cost per click; only pay for demand that already exists |
| 6 | **SEO and Google Business Profile** | Same queries, free. One page per city × pooja, in English and the local language | Content time |
| 7 | **YouTube and Instagram/Facebook** | Creates demand around festivals and life events; regional-language video works in tier 2/3 | Cost per install; needs retargeting to convert |
| 8 | **Apartment societies, wedding planners, builders, event halls** | Griha pravesh and weddings are high AOV and predictable | Commission per booking |
| 9 | **Regional creators and astrologers** | Trust in local language | Fixed fee or per booking; test small |

Avoid for now: TV, print, hoardings (except at Pushkaralu sites), broad "awareness" campaigns, and any ad in a city below the liquidity rule.

## 5. Plan by market

### Tier 1: Hyderabad, Mumbai, Delhi NCR, Bengaluru, Chennai, Pune

- Launch one city at a time. **Hyderabad first** (home base, Telugu priests, Pushkaralu spillover).
- People search here. Lead with Google Search and SEO, then Meta/YouTube retargeting of visitors and installers.
- Geo-target ads to the zones where priests are live, radius 10-15 km. Expand zone by zone.
- Message: verified priests, fixed transparent price, on-time, samagri included, same sampradaya and language.
- Life events drive the highest AOV: griha pravesh, weddings, naming, upanayanam. Partner with builders and societies.
- Mix target after month 3: 40% of new customers from referral and priests, 35% search and SEO, 25% social.

### Tier 2 and 3: Vijayawada, Rajahmundry, Guntur, Warangal, Tirupati, Visakhapatnam, then other states

- Paid CAC ceiling is low. **Lead with priests, temples, WhatsApp and local-language video.** Paid ads are a top-up.
- Many families already have a family priest. Do not fight that; onboard that priest. The app becomes his booking and payment tool, and his families come with him.
- Creative entirely in Telugu (later Hindi, Tamil, Kannada, Marathi). Voice and video over text. Show a real local priest.
- Offer phone booking help (a missed-call number) for users not comfortable with apps.
- Low-end Android phones and slow data: keep the app small and fast.
- **Pushkaralu is the tier 2/3 engine.** Every pilgrim who books a ritual at a ghat is a registered user from a tier 2/3 town. After the event, convert them: "the same verified priests, at your home", with a festival-timed offer.

### Overseas: UK, USA, Canada, Australia, Singapore, UAE

- **Phase A, virtual pooja only.** Priests are in India, so supply is ready on day one. Target Indian diaspora by language (Telugu, Hindi, Tamil, Gujarati) and interest on Meta and YouTube, and search queries like "online pooja", "pandit in London", "e-puja for griha pravesh".
- Strong hooks: festivals away from home, pooja for parents back in India, pooja at a specific Indian temple, family joining from several countries on one call.
- Community channels cost little and convert well: temple committees, Telugu/Tamil/Gujarati associations (TANA, ATA, local samaj groups), university Indian societies, diaspora WhatsApp and Facebook groups, Indian grocery stores.
- **Phase B, in-person**, only in cities with dense diaspora and after 15 local priests are onboarded: London (Harrow, Wembley, Southall), Leicester, Birmingham; New Jersey, Bay Area, Dallas, Chicago; Toronto/Brampton; Sydney, Melbourne. At the assumed £13 ceiling, paid ads barely work for in-person abroad; rely on community and priest networks.
- Schedule ads by local time zone and local festival dates. Price in local currency.

## 6. How much to spend

Budget follows proof, not ambition.

| Stage | Duration | Monthly ad spend | Goal | Move on when |
|---|---|---|---|---|
| 0. Pushkaralu pilot | Dec 2026 onward | near zero (on-ground only) | registered users, completed event bookings, reviews | app stable, rating 4.3+ |
| 1. Test | 4 weeks | **₹50,000-₹75,000** total, Hyderabad only | find real cost per install, funnel rates, CAC per channel | you have measured numbers for section 2 and 3 |
| 2. Fix | 4-6 weeks | ₹50,000 | get funnel to targets; referral and priest QR live | paid CAC ≤ max CAC on at least one channel |
| 3. Scale city 1 | 3 months | ₹2-4 lakh | grow while CAC holds | CAC stable 4 weeks, repeat rate known |
| 4. Add cities | ongoing | ₹1.5-3 lakh per tier 1 city, ₹30,000-75,000 per tier 2/3 city | repeat the playbook | each city passes the liquidity rule first |
| 5. Overseas virtual test | 4 weeks | **£1,500-£2,000** (UK first) | measure CAC for virtual | CAC ≤ max CAC |

Stage 1 split: 50% Google Search, 30% Meta (Instagram + Facebook), 20% YouTube. Run at least 3 creatives and 2 audiences per channel; kill the worst half every week.

Scaling rules:

1. Raise a campaign's budget by at most **20-30% per week**, and only if its CAC stayed under the ceiling for 2 weeks.
2. If CAC rises above the ceiling for 2 weeks, cut that campaign's budget by half and look at the funnel, not the ad.
3. Never let paid spend exceed **30% of gross platform revenue** once past stage 3. Keep at least 40% of new customers coming from non-paid sources.
4. Pause devotee ads in any zone where acceptance within 30 minutes falls below 80%. Spend that week on priest onboarding instead.
5. Festival calendar: raise budgets 2-3 weeks before Sankranti, Ugadi, Sravana masam, Vinayaka Chavithi, Dasara, Diwali, Karthika masam and the wedding and griha pravesh muhurtham seasons. Demand is highest and cheapest then.

## 7. Keeping CAC down over time

- **Reviews and rating:** ask after every completed pooja. A 4.5+ store rating lowers install cost on every channel.
- **Repeat engine:** save family details and gotram once; remind on birthdays, death anniversaries (tithi-based), annual vratams; "book the same priest again" in one tap.
- **Samagri kit and packages:** raise AOV without raising CAC.
- **Retargeting:** registered but not booked is the cheapest audience you will ever have. WhatsApp and push first, paid retargeting second.
- **Creative refresh:** every 2-3 weeks; regional language; real priests and real homes.
- **Priest quality:** late or no-show priests destroy repeat rate. Track on-time rate and rating per priest; remove the bottom.
- **Cancellation and refund clarity** on the booking screen increases request rate.

## 8. Weekly dashboard (one sheet)

Per city and per channel: spend, installs, registrations, booking requests, completed first bookings, **paid CAC, blended CAC**, AOV, contribution, repeat rate by monthly cohort, priest count active, acceptance rate, on-time rate, store rating, refund rate.

Three numbers for the founder every Monday: blended CAC vs max CAC, completed bookings, 90-day repeat rate.

## 9. Compliance for marketing

- India (DPDP): marketing messages only to users who opted in; easy opt-out; WhatsApp Business policy opt-in.
- UK/EU: religion is special category data. **Do not build ad audiences from users' religious data or upload customer lists for lookalikes without explicit consent.** Target by language, location and interest instead. Cookie consent on the website.
- No guaranteed-outcome claims ("this pooja will cure/guarantee..."). Ad platforms and consumer law both reject them.
- Referral credits: state terms; pay only on completed bookings to block fraud; cap per user.

## 10. Risks

| Risk | Guard |
|---|---|
| Priests and devotees take the next booking offline | Make the app worth it for the priest: calendar, payments, reminders, new customers; loyalty credit for devotees |
| Ads outrun supply | Liquidity rule and rule 4 in section 6 |
| Referral fraud | Completed-booking trigger, device and payment checks |
| Seasonality | Budget by festival calendar; hold cash for peaks |
| Trust incident with a priest | Verification, ratings, fast support, removal |
