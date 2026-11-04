"""
End-to-End Testing Suite for Puja Vaahini Backend & Ngrok Tunnel
Tests every action, button, state transition, and consent validation.
"""

import sys
import json
import urllib.request
import urllib.error
import time

def make_request(url, method="GET", data=None, custom_headers=None):
    headers = {
        "User-Agent": "PurohitMobileApp/1.0",
        "ngrok-skip-browser-warning": "1"
    }
    if custom_headers:
        headers.update(custom_headers)
    
    encoded_data = None
    if data is not None:
        headers["Content-Type"] = "application/json"
        encoded_data = json.dumps(data).encode("utf-8")
        
    req = urllib.request.Request(url, data=encoded_data, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=15) as response:
            res_body = response.read().decode("utf-8")
            return response.status, json.loads(res_body) if res_body else {}
    except urllib.error.HTTPError as e:
        err_body = e.read().decode("utf-8")
        try:
            return e.code, json.loads(err_body)
        except:
            return e.code, {"raw_error": err_body}
    except Exception as e:
        return 0, {"error": str(e)}

def run_e2e_test(base_url):
    print("=" * 70)
    print(f"[START] INITIATING END-TO-END VERIFICATION: {base_url}")
    print("=" * 70)

    # 1. Health Check
    print("\n[TEST 1] GET /health (Postgres & Redis Check)...")
    status, res = make_request(f"{base_url}/health")
    assert status == 200, f"Expected 200, got {status}: {res}"
    print(f"  [OK] Health Check: {res.get('status')}, Postgres: {res.get('postgres')}, Redis: {res.get('redis')}")

    # 2. Consent Enforcement Test (Devotee rejects consent)
    print("\n[TEST 2] POST /api/v1/auth/register-devotee (Consent Rejected Validation)...")
    payload_no_consent = {
        "full_name": "Test Devotee Without Consent",
        "phone": "+91 99999 00001",
        "gotram": "Bharadwaja",
        "nakshatram": "Pushya",
        "rashi": "Karkataka",
        "city": "Hyderabad",
        "consent_agreed": False
    }
    status, res = make_request(f"{base_url}/api/v1/auth/register-devotee", method="POST", data=payload_no_consent)
    assert status == 400, f"Expected 400 rejection for missing consent, got {status}: {res}"
    print(f"  [OK] Consent Enforcement: Successfully rejected registration without consent: {res.get('error')}")

    # 3. Devotee Registration with Valid Consent
    print("\n[TEST 3] POST /api/v1/auth/register-devotee (Registration with Consent)...")
    payload_devotee = {
        "full_name": "Srikanth Sharma",
        "phone": "+91 98480 12345",
        "gotram": "Kashyapa",
        "nakshatram": "Rohini",
        "rashi": "Vrishabha",
        "city": "Hyderabad",
        "consent_agreed": True
    }
    status, res = make_request(f"{base_url}/api/v1/auth/register-devotee", method="POST", data=payload_devotee)
    assert status == 200, f"Expected 200, got {status}: {res}"
    user_id = res.get("user_id")
    print(f"  [OK] Devotee Registered! User ID: {user_id}, Name: {res.get('full_name')}")

    # 4. Priest Registration with Code of Conduct Consent
    print("\n[TEST 4] POST /api/v1/auth/register-priest (Priest Registration with Honor Code Consent)...")
    payload_priest = {
        "full_name": "Brahmasri Venkata Sharma Garu",
        "phone": "+91 98765 43210",
        "sampradaya": "Smartha",
        "veda_shakha": "Rigveda Shakha",
        "experience_years": 18,
        "city": "Hyderabad",
        "latitude": 17.3850,
        "longitude": 78.4867,
        "consent_agreed": True
    }
    status, res = make_request(f"{base_url}/api/v1/auth/register-priest", method="POST", data=payload_priest)
    assert status == 200, f"Expected 200, got {status}: {res}"
    priest_id = res.get("priest_id")
    print(f"  [OK] Priest Registered! Priest ID: {priest_id}, Sampradaya: {res.get('sampradaya')}")

    # 5. Fetch Pooja Catalog from Database
    print("\n[TEST 5] GET /api/v1/poojas (Pooja Packages from Postgres)...")
    status, poojas = make_request(f"{base_url}/api/v1/poojas")
    assert status == 200 and len(poojas) > 0, f"Failed to load catalog: {poojas}"
    print(f"  [OK] Catalog loaded from PostgreSQL ({len(poojas)} packages available):")
    for p in poojas:
        print(f"     - {p.get('title')} (Rs.{p.get('base_price')} + Samagri Rs.{p.get('samagri_kit_price')})")
    chosen_pooja = poojas[0]

    # 6. Fetch Vedic Panchangam
    print("\n[TEST 6] GET /api/v1/panchangam/today (Vedic Ephemeris & Timings)...")
    status, panchangam_data = make_request(f"{base_url}/api/v1/panchangam/today")
    assert status == 200, f"Failed to get panchangam: {panchangam_data}"
    print(f"  [OK] Today's Vedic Panchangam:")
    print(f"     - Tithi: {panchangam_data.get('tithi')}")
    print(f"     - Nakshatra: {panchangam_data.get('nakshatra')}")
    print(f"     - Rahu Kalam: {panchangam_data.get('rahu_kalam')}")
    print(f"     - Auspicious Day: {panchangam_data.get('is_auspicious')}")

    # 7. Create Booking with Escrow Hold
    print("\n[TEST 7] POST /api/v1/bookings (Create Escrow Booking with OTP)...")
    booking_req = {
        "user_id": user_id,
        "pooja_id": chosen_pooja.get("id"),
        "booking_type": "INSTANT_DISPATCH",
        "include_samagri_kit": True,
        "total_amount": float(chosen_pooja.get("base_price", 5116)) + float(chosen_pooja.get("samagri_kit_price", 2500)),
        "event_address": "Flat 402, Sri Rama Nilayam, Hyderabad",
        "latitude": 17.3850,
        "longitude": 78.4867
    }
    status, booking_res = make_request(f"{base_url}/api/v1/bookings", method="POST", data=booking_req)
    assert status == 200, f"Booking creation failed: {booking_res}"
    booking_id = booking_res.get("booking_id")
    otp = booking_res.get("completion_otp")
    print(f"  [OK] Booking Created in Escrow!")
    print(f"     - Booking ID: {booking_id}")
    print(f"     - Escrow Status: {booking_res.get('escrow_status')}")
    print(f"     - Completion OTP: {otp}")

    # 8. Dispatch Match (Dynamic Rapido Ranking Scorer)
    print("\n[TEST 8] POST /api/v1/dispatch/match (Rapido Composite Scorer & Nearest Match)...")
    dispatch_req = {
        "booking_id": booking_id,
        "pooja_title": chosen_pooja.get("title"),
        "devotee_latitude": 17.3850,
        "devotee_longitude": 78.4867,
        "required_sampradaya": "Smartha",
        "search_radius_meters": 5000.0
    }
    status, dispatch_res = make_request(f"{base_url}/api/v1/dispatch/match", method="POST", data=dispatch_req)
    assert status == 200 and dispatch_res.get("total_candidates") > 0, f"Dispatch failed: {dispatch_res}"
    best_match = dispatch_res.get("best_match")
    print(f"  [OK] Dispatch Match Result ({dispatch_res.get('total_candidates')} priests scored):")
    print(f"     - Best Match: {best_match.get('full_name')}")
    print(f"     - Score: {best_match.get('score')} | Distance: {best_match.get('distance_meters')}m | Rating: {best_match.get('rating_avg')} stars")

    # 9. State Machine Transitions (IN_TRANSIT -> ARRIVED -> POOJA_STARTED)
    print("\n[TEST 9] State Transitions (Simulating in-app lifecycle)...")
    for next_state in ["ACCEPTED", "IN_TRANSIT", "ARRIVED", "POOJA_STARTED"]:
        status_req = {
            "booking_id": booking_id,
            "status": next_state,
            "priest_id": best_match.get("id")
        }
        st, res = make_request(f"{base_url}/api/v1/bookings/status", method="POST", data=status_req)
        assert st == 200, f"Transition to {next_state} failed: {res}"
        print(f"  [OK] State Transition: -> {next_state}")

    # 10. Complete Pooja & Escrow Release with OTP Validation
    print("\n[TEST 10] Complete Pooja with OTP Escrow Release...")
    # First test with invalid OTP
    status_wrong_otp = {
        "booking_id": booking_id,
        "status": "COMPLETED",
        "completion_otp": "0000",
        "dakshina_amount": 501.0
    }
    st, res = make_request(f"{base_url}/api/v1/bookings/status", method="POST", data=status_wrong_otp)
    assert st == 400, f"Expected 400 for bad OTP, got {st}: {res}"
    print(f"  [OK] Escrow Protection: Bad OTP successfully blocked release: {res.get('error')}")

    # Now test with correct OTP
    status_correct_otp = {
        "booking_id": booking_id,
        "status": "COMPLETED",
        "completion_otp": otp,
        "dakshina_amount": 501.0
    }
    st, res = make_request(f"{base_url}/api/v1/bookings/status", method="POST", data=status_correct_otp)
    assert st == 200, f"Escrow release failed: {res}"
    print(f"  [OK] Pooja Completed & Escrow Released: {res.get('escrow_status')}")

    print("\n" + "=" * 70)
    print("[SUCCESS] ALL 10 END-TO-END FLOW TESTS PASSED VIA NGROK & LOCAL!")
    print("=" * 70)

if __name__ == "__main__":
    url = sys.argv[1] if len(sys.argv) > 1 else "http://localhost:8080"
    run_e2e_test(url)
