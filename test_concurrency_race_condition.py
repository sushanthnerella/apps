import time
import sys
import json
import uuid
import concurrent.futures
import urllib.request
import urllib.error

sys.stdout.reconfigure(encoding='utf-8')

BASE_URL = sys.argv[1] if len(sys.argv) > 1 else "http://localhost:8085"

def post_json(endpoint, data):
    url = f"{BASE_URL}{endpoint}"
    payload = json.dumps(data).encode('utf-8')
    req = urllib.request.Request(url, data=payload, headers={'Content-Type': 'application/json'}, method="POST")
    try:
        with urllib.request.urlopen(req, timeout=5) as res:
            return res.status, json.loads(res.read().decode('utf-8'))
    except urllib.error.HTTPError as e:
        body = e.read().decode('utf-8')
        try:
            parsed = json.loads(body)
        except:
            parsed = {"error": body}
        return e.code, parsed
    except Exception as e:
        return 0, {"error": str(e)}

def get_json(endpoint):
    url = f"{BASE_URL}{endpoint}"
    req = urllib.request.Request(url, headers={'Content-Type': 'application/json'}, method="GET")
    try:
        with urllib.request.urlopen(req, timeout=5) as res:
            return res.status, json.loads(res.read().decode('utf-8'))
    except urllib.error.HTTPError as e:
        body = e.read().decode('utf-8')
        try:
            parsed = json.loads(body)
        except:
            parsed = {"error": body}
        return e.code, parsed
    except Exception as e:
        return 0, {"error": str(e)}

def run_concurrency_and_security_tests():
    print("=" * 70)
    print("⚡ PUJA VAAHINI: HIGH-CONCURRENCY, RACE CONDITION & SECURITY RESILIENCE")
    print(f"Target Gateway: {BASE_URL}")
    print("=" * 70)

    # Setup: Register unique test devotee & priest
    unique_suffix = str(int(time.time()))[-6:]
    devotee_phone = f"+9198765{unique_suffix}"
    priest_phone = f"+9191234{unique_suffix}"

    print(f"\n[SETUP] Registering Devotee ({devotee_phone}) and Priest ({priest_phone})...")
    status, dev_res = post_json("/api/v1/auth/register-devotee", {
        "full_name": f"Devotee Test {unique_suffix}",
        "phone": devotee_phone,
        "gotram": "Bharadwaja",
        "nakshatram": "Anuradha",
        "rashi": "Vrischika",
        "city": "Hyderabad",
        "consent_agreed": True
    })
    assert status == 200, f"Devotee registration failed: {dev_res}"
    devotee_id = dev_res["user_id"]

    status, priest_res = post_json("/api/v1/auth/register-priest", {
        "full_name": f"Priest Test {unique_suffix}",
        "phone": priest_phone,
        "vedic_tradition": "Rigveda",
        "experience_years": 12,
        "consent_agreed": True
    })
    assert status == 200, f"Priest registration failed: {priest_res}"
    priest_id = priest_res["priest_id"]
    print(f"  [OK] Devotee ID: {devotee_id} | Priest ID: {priest_id}")

    # =========================================================================
    # TEST 1: High-Concurrency Slot Double-Booking Race Condition
    # =========================================================================
    print("\n" + "="*70)
    print("🔒 TEST 1: High-Concurrency Slot Double-Booking Race Condition (Redis Mutex)")
    print("="*70)

    slot_date = "2026-12-30"
    minute_val = int(unique_suffix[-2:]) % 50
    start_time = f"09:{minute_val:02d}"
    end_time = f"10:{minute_val:02d}"

    status, slot_res = post_json("/api/v1/priests/slots", {
        "priest_id": priest_id,
        "slot_date": slot_date,
        "start_time": start_time,
        "end_time": end_time
    })
    assert status in [200, 201], f"Failed to create slot: {slot_res}"
    slot_id = slot_res["slot_id"]
    print(f"Created Real Priest Slot ID: {slot_id} on {slot_date} at {start_time}")

    num_concurrent_threads = 30
    print(f"Firing {num_concurrent_threads} simultaneous lock requests for slot {slot_id} across threads...")

    results = []
    def attempt_hold(i):
        user_uuid = f"user-{i}-{uuid.uuid4()}"
        code, body = post_json("/api/v1/priests/slots/hold", {
            "slot_id": slot_id,
            "devotee_id": user_uuid,
            "hold_duration_seconds": 60
        })
        return i, code, body

    bench_start = time.perf_counter()
    with concurrent.futures.ThreadPoolExecutor(max_workers=num_concurrent_threads) as executor:
        futures = [executor.submit(attempt_hold, i) for i in range(num_concurrent_threads)]
        for f in concurrent.futures.as_completed(futures):
            results.append(f.result())
    elapsed = (time.perf_counter() - bench_start) * 1000.0

    success_count = sum(1 for _, code, _ in results if code == 200)
    conflict_count = sum(1 for _, code, _ in results if code == 409)
    other_count = len(results) - success_count - conflict_count

    print(f"⚡ Elapsed: {elapsed:.2f} ms for {num_concurrent_threads} concurrent attempts")
    print(f"📊 Breakdown: {success_count} Acquired (200), {conflict_count} Rejected (409 Conflict), {other_count} Other")

    t1_pass = (success_count == 1 and conflict_count == (num_concurrent_threads - 1))
    if t1_pass:
        print("✅ PASSED: EXACT-ONCE MUTEX GUARANTEED! Slot race condition safely resolved.")
    else:
        print("❌ FAILED: Race condition failed mutex check.")

    # =========================================================================
    # TEST 2: Swiggy-Style Cancellation & Dynamic Refund Calculation
    # =========================================================================
    print("\n" + "="*70)
    print("💰 TEST 2: Cancellation Policy & Dynamic Refund Calculation")
    print("="*70)

    # Fetch catalog pooja
    _, catalog = get_json("/api/v1/poojas")
    pooja_id = catalog[0]["id"]

    # Create real booking
    status, booking = post_json("/api/v1/bookings", {
        "devotee_id": devotee_id,
        "pooja_id": pooja_id,
        "priest_id": priest_id,
        "scheduled_date": "2026-12-30T09:00:00Z",
        "location": "Banjara Hills Road No 12, Hyderabad",
        "total_amount": 3500.0,
        "include_samagri": True
    })
    assert status in [200, 201], f"Failed to create booking: {booking}"
    booking_id = booking.get("booking_id") or booking.get("id")
    print(f"Created Booking ID: {booking_id} with Total Amount Rs. 3500.00")

    # Request Cancellation Quote
    status, quote = get_json(f"/api/v1/bookings/{booking_id}/cancel-quote")
    print(f"Cancellation Quote (Status: {quote.get('current_status')}):")
    print(f"  - Advance Paid: Rs. {quote.get('advance_paid')}")
    print(f"  - Cancellation Fee: Rs. {quote.get('cancellation_fee')}")
    print(f"  - Refund Amount: Rs. {quote.get('refund_amount')}")
    print(f"  - Policy Notice: {quote.get('policy_notice')}")

    # Execute Cancellation
    status, cancel_res = post_json("/api/v1/bookings/cancel", {
        "booking_id": booking_id,
        "cancelled_by": "DEVOTEE",
        "reason": "Date rescheduled due to family event"
    })
    print(f"Cancel Execution Status: {status}")
    print(f"Cancel Response: {cancel_res}")

    t2_pass = (status == 200 and cancel_res.get("status") == "CANCELLED" and cancel_res.get("refund_amount") == 3500.0)
    if t2_pass:
        print("✅ PASSED: 100% Instant Refund successfully executed under pre-dispatch window!")
    else:
        print("❌ FAILED: Unexpected cancellation response.")

    # =========================================================================
    # TEST 3: Auth Security, OTP Validation & JWT Tamper Rejection
    # =========================================================================
    print("\n" + "="*70)
    print("🛡️ TEST 3: Auth Security, OTP Verification & JWT Claims")
    print("="*70)

    # 1. Invalid OTP rejection
    status, bad_otp_res = post_json("/api/v1/auth/login", {
        "phone": devotee_phone,
        "otp": "999999",
        "role": "devotee"
    })
    print(f"Login with Invalid OTP ('999999'): Status {status} (Expected 401)")
    bad_otp_rejected = (status == 401)

    # 2. Valid OTP authentication
    status, good_otp_res = post_json("/api/v1/auth/login", {
        "phone": devotee_phone,
        "otp": "1234",
        "role": "devotee"
    })
    print(f"Login with Valid OTP ('1234'): Status {status}")
    jwt_token = good_otp_res.get("access_token") or good_otp_res.get("token")
    good_otp_accepted = (status == 200 and jwt_token is not None)
    print(f"  [OK] Obtained JWT Access Token: {jwt_token[:20]}...")

    # 3. Access protected profile with valid JWT
    req_profile = urllib.request.Request(
        f"{BASE_URL}/api/v1/user/profile",
        headers={"Authorization": f"Bearer {jwt_token}"},
        method="GET"
    )
    with urllib.request.urlopen(req_profile) as res:
        prof_body = json.loads(res.read().decode('utf-8'))
        prof_status = res.status
    print(f"Access Protected Profile with JWT: Status {prof_status} | Devotee: {prof_body.get('full_name')}")

    # 4. Access protected profile with forged / invalid JWT
    forged_token = jwt_token[:-6] + "tamper"
    req_forged = urllib.request.Request(
        f"{BASE_URL}/api/v1/user/profile",
        headers={"Authorization": f"Bearer {forged_token}"},
        method="GET"
    )
    try:
        urllib.request.urlopen(req_forged)
        forged_rejected = False
    except urllib.error.HTTPError as e:
        forged_rejected = (e.code == 401)
        print(f"Access with Tampered JWT: Status {e.code} (Correctly Rejected Unauthorized)")

    t3_pass = bad_otp_rejected and good_otp_accepted and (prof_status == 200) and forged_rejected
    if t3_pass:
        print("✅ PASSED: OTP validation, JWT issuance, and cryptographic signature verification verified!")
    else:
        print("❌ FAILED: Security test checks did not all pass.")

    # =========================================================================
    # SUMMARY
    # =========================================================================
    print("\n" + "="*70)
    print("🏆 FINAL RESILIENCE & INTEGRITY VERIFICATION SUMMARY")
    print("="*70)
    print(f"1. High-Concurrency Slot Mutex (30 threads):       {'PASSED [OK]' if t1_pass else 'FAILED [X]'}")
    print(f"2. Swiggy Cancellation & 100% Dynamic Refund:      {'PASSED [OK]' if t2_pass else 'FAILED [X]'}")
    print(f"3. Auth OTP Security & JWT Signature Integrity:     {'PASSED [OK]' if t3_pass else 'FAILED [X]'}")
    print("="*70)

    if t1_pass and t2_pass and t3_pass:
        print("\n🎉 ALL ADVANCED CONCURRENCY & RESILIENCE TESTS PASSED ZERO DEFECTS!")
        return 0
    else:
        print("\n❌ SOME TESTS FAILED.")
        return 1

if __name__ == "__main__":
    exit_code = run_concurrency_and_security_tests()
    sys.exit(exit_code)
