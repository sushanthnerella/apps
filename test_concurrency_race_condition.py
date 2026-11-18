import time
import sys
import json
import uuid
import concurrent.futures
import urllib.request
import urllib.error

sys.stdout.reconfigure(encoding='utf-8')

BASE_URL = sys.argv[1] if len(sys.argv) > 1 else "http://localhost:8085"

def post_json(endpoint, data, custom_headers=None):
    url = f"{BASE_URL}{endpoint}"
    payload = json.dumps(data).encode('utf-8')
    headers = {'Content-Type': 'application/json'}
    if custom_headers:
        headers.update(custom_headers)
    req = urllib.request.Request(url, data=payload, headers=headers, method="POST")
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

def get_json(endpoint, custom_headers=None):
    url = f"{BASE_URL}{endpoint}"
    headers = {'Content-Type': 'application/json'}
    if custom_headers:
        headers.update(custom_headers)
    req = urllib.request.Request(url, headers=headers, method="GET")
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
        "password": "Password123",
        "gotram": "Bharadwaja",
        "nakshatram": "Anuradha",
        "rashi": "Vrischika",
        "city": "Hyderabad",
        "consent_agreed": True
    })
    assert status == 200, f"Devotee registration failed: {dev_res}"
    devotee_id = dev_res["user_id"]
    devotee_token = dev_res["access_token"]

    status, priest_res = post_json("/api/v1/auth/register-priest", {
        "full_name": f"Priest Test {unique_suffix}",
        "phone": priest_phone,
        "password": "Password123",
        "sampradaya": "Smartha",
        "veda_shakha": "Rigveda",
        "experience_years": 12,
        "consent_agreed": True
    })
    assert status == 200, f"Priest registration failed: {priest_res}"
    priest_id = priest_res["priest_id"]
    priest_token = priest_res["access_token"]
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
    }, custom_headers={"Authorization": f"Bearer {priest_token}"})
    assert status in [200, 201], f"Failed to create slot: {slot_res}"
    slot_id = slot_res["slot_id"]
    print(f"Created Real Priest Slot ID: {slot_id} on {slot_date} at {start_time}")

    num_concurrent_threads = 30
    print(f"Firing {num_concurrent_threads} simultaneous lock requests for slot {slot_id} across threads...")

    results = []
    def attempt_hold(i):
        user_uuid = devotee_id if i == 0 else f"user-{i}-{uuid.uuid4()}"
        code, body = post_json("/api/v1/priests/slots/hold", {
            "slot_id": slot_id,
            "devotee_id": user_uuid,
            "hold_duration_seconds": 60
        })
        return i, code, body, user_uuid

    bench_start = time.perf_counter()
    with concurrent.futures.ThreadPoolExecutor(max_workers=num_concurrent_threads) as executor:
        futures = [executor.submit(attempt_hold, i) for i in range(num_concurrent_threads)]
        for f in concurrent.futures.as_completed(futures):
            results.append(f.result())
    elapsed = (time.perf_counter() - bench_start) * 1000.0

    success_count = sum(1 for _, code, _, _ in results if code == 200)
    conflict_count = sum(1 for _, code, _, _ in results if code == 409)
    other_count = len(results) - success_count - conflict_count

    print(f"⚡ Elapsed: {elapsed:.2f} ms for {num_concurrent_threads} concurrent attempts")
    print(f"📊 Breakdown: {success_count} Acquired (200), {conflict_count} Rejected (409 Conflict), {other_count} Other")

    t1_pass = (success_count == 1 and conflict_count == (num_concurrent_threads - 1))
    if t1_pass:
        print("✅ PASSED: EXACT-ONCE MUTEX GUARANTEED! Slot race condition safely resolved.")
    else:
        print("❌ FAILED: Race condition failed mutex check.")

    # Find the holder of the slot
    held_result = [r for r in results if r[1] == 200][0]
    holding_user_id = held_result[3]

    # Fetch catalog pooja
    _, catalog = get_json("/api/v1/poojas")
    pooja_id = catalog[0]["id"]

    # Verify actual booking creation cannot double-book the slot:
    # Register contender devotee 2
    status, dev2_res = post_json("/api/v1/auth/register-devotee", {
        "full_name": f"Devotee Contender {unique_suffix}",
        "phone": f"+9198766{unique_suffix}",
        "password": "Password123",
        "gotram": "Kashyapa",
        "nakshatram": "Rohini",
        "rashi": "Vrishabha",
        "city": "Hyderabad",
        "consent_agreed": True
    })
    assert status == 200, f"Contender registration failed: {dev2_res}"
    devotee2_token = dev2_res["access_token"]

    # Contender tries to book the held slot -> rejected (409 Conflict)
    status_contender, res_contender = post_json("/api/v1/bookings", {
        "user_id": dev2_res["user_id"],
        "pooja_id": pooja_id,
        "priest_id": priest_id,
        "slot_id": slot_id,
        "event_address": "Banjara Hills Road No 12, Hyderabad",
        "total_amount": 3500.0,
        "booking_type": "SCHEDULED",
        "include_samagri_kit": True
    }, custom_headers={"Authorization": f"Bearer {devotee2_token}"})
    assert status_contender == 409, f"Expected 409 for contender booking held slot, got {status_contender}: {res_contender}"
    print(f"  [OK] Contender booking held slot rejected with HTTP 409 Conflict")

    # If the slot hold was acquired by another synthetic user_uuid, release it so legitimate devotee can book
    if holding_user_id != devotee_id:
        # Acquire hold as devotee_id (or create another slot for devotee_id to test double-booking prevention)
        pass

    # =========================================================================
    # TEST 2: Swiggy-Style Cancellation & Dynamic Refund Calculation
    # =========================================================================
    print("\n" + "="*70)
    print("💰 TEST 2: Cancellation Policy & Dynamic Refund Calculation")
    print("="*70)

    # Create real booking with devotee Bearer token and valid contracts
    status, booking = post_json("/api/v1/bookings", {
        "user_id": devotee_id,
        "pooja_id": pooja_id,
        "priest_id": priest_id,
        "scheduled_date": slot_date,
        "event_address": "Banjara Hills Road No 12, Hyderabad",
        "total_amount": 3500.0,
        "booking_type": "SCHEDULED",
        "include_samagri_kit": True
    }, custom_headers={"Authorization": f"Bearer {devotee_token}"})
    assert status in [200, 201], f"Failed to create booking: {booking}"
    booking_id = booking.get("booking_id") or booking.get("id")
    print(f"Created Booking ID: {booking_id} with Total Amount Rs. 3500.00")

    # Request Cancellation Quote
    status, quote = get_json(f"/api/v1/bookings/{booking_id}/cancel-quote", custom_headers={"Authorization": f"Bearer {devotee_token}"})
    print(f"Cancellation Quote (Status: {quote.get('current_status')}):")
    print(f"  - Advance Paid: Rs. {quote.get('advance_paid')}")
    print(f"  - Cancellation Fee: Rs. {quote.get('cancellation_fee')}")
    print(f"  - Refund Amount: Rs. {quote.get('refund_amount')}")
    print(f"  - Policy Notice: {quote.get('policy_notice')}")

    # Execute Cancellation
    status, cancel_res = post_json("/api/v1/bookings/cancel", {
        "booking_id": booking_id,
        "reason_code": "DEVOTEE_CANCELLED",
        "reason_text": "Date rescheduled due to family event"
    }, custom_headers={"Authorization": f"Bearer {devotee_token}"})
    print(f"Cancel Execution Status: {status}")
    print(f"Cancel Response: {cancel_res}")

    t2_pass = (status == 200 and cancel_res.get("status") == "CANCELLED" and cancel_res.get("refund_amount") == 3500.0)
    if t2_pass:
        print("✅ PASSED: 100% Instant Refund successfully executed under pre-dispatch window!")
    else:
        print("❌ FAILED: Unexpected cancellation response.")

    # =========================================================================
    # TEST 3: Auth Security, Password Verification & JWT Tamper Rejection
    # =========================================================================
    print("\n" + "="*70)
    print("🛡️ TEST 3: Auth Security, Password Verification & JWT Claims")
    print("="*70)

    # 1. Invalid Password rejection
    status, bad_pwd_res = post_json("/api/v1/auth/login", {
        "phone": devotee_phone,
        "password": "WrongPassword999",
        "role": "devotee"
    })
    print(f"Login with Invalid Password: Status {status} (Expected 401)")
    bad_pwd_rejected = (status == 401)

    # 2. Valid Password authentication
    status, good_login_res = post_json("/api/v1/auth/login", {
        "phone": devotee_phone,
        "password": "Password123",
        "role": "devotee"
    })
    print(f"Login with Valid Password: Status {status}")
    jwt_token = good_login_res.get("access_token") or good_login_res.get("token")
    good_login_accepted = (status == 200 and jwt_token is not None)
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

    t3_pass = bad_pwd_rejected and good_login_accepted and (prof_status == 200) and forged_rejected
    if t3_pass:
        print("✅ PASSED: Password validation, JWT issuance, and cryptographic signature verification verified!")
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
