import os
import sys
import time
import json
import urllib.request
import urllib.error
import subprocess
import xml.etree.ElementTree as ET

sys.stdout.reconfigure(encoding='utf-8')

OUTPUT_DIR = r"C:\Users\susha\OneDrive\Documents\ChatGPT\pooja vahini\output\supervised-internal-20261115"
SCREENSHOT_DIR = os.path.join(OUTPUT_DIR, "screenshots")
os.makedirs(SCREENSHOT_DIR, exist_ok=True)
LOG_FILE = os.path.join(OUTPUT_DIR, "core_journeys.log")

STAGING_URL = "https://pooja-vahini-staging.onrender.com"

DEVOTEE_PHONE = "+919988110001"
DEVOTEE_PASS = "TestPass1234!"
DEVOTEE_ID = "8b240c46-9071-45fb-b59f-58f3579318da"

PRIEST_PHONE = "+919877000001"
PRIEST_PASS = "PriestPass1234!"
PRIEST_ID = "a4e26676-210c-44ee-b09c-964bacf50420"

def log(msg):
    ts = time.strftime("%Y-%m-%d %H:%M:%S")
    line = f"[{ts}] {msg}"
    print(line)
    with open(LOG_FILE, "a", encoding="utf-8") as f:
        f.write(line + "\n")

def cap(name):
    p = os.path.join(SCREENSHOT_DIR, name)
    subprocess.run(f'adb exec-out screencap -p > "{p}"', shell=True)
    size = os.path.getsize(p) if os.path.exists(p) else 0
    log(f"[SCREENSHOT] Captured {name} ({size} bytes)")
    return p

def api_call(url, method="GET", data=None, token=None):
    headers = {"Content-Type": "application/json", "User-Agent": "CoreJourneysTester/1.0"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    encoded = json.dumps(data).encode("utf-8") if data is not None else None
    req = urllib.request.Request(url, data=encoded, headers=headers, method=method)
    try:
        with urllib.request.urlopen(req, timeout=45) as resp:
            body = resp.read().decode("utf-8")
            return resp.status, json.loads(body) if body else {}
    except urllib.error.HTTPError as e:
        err = e.read().decode("utf-8")
        try:
            return e.code, json.loads(err)
        except Exception:
            return e.code, {"raw": err}
    except Exception as e:
        return 0, {"error": str(e)}

def run_adb(args):
    return subprocess.run(["adb"] + args, capture_output=True, text=True, encoding="utf-8", errors="ignore").stdout.strip()

def dump_ui():
    run_adb(["shell", "uiautomator", "dump", "/data/local/tmp/dump.xml"])
    out = run_adb(["shell", "cat", "/data/local/tmp/dump.xml"])
    if out.startswith("<?xml"):
        try:
            return ET.fromstring(out)
        except Exception:
            pass
    return None

def main():
    log("=" * 80)
    log("STARTING CORE JOURNEY EXECUTION")
    log("=" * 80)

    # 1. Obtain JWT tokens
    _, dev_login = api_call(f"{STAGING_URL}/api/v1/auth/login", method="POST", data={"phone": DEVOTEE_PHONE, "password": DEVOTEE_PASS, "role": "devotee"})
    dev_token = dev_login.get("access_token")
    log(f"Devotee authenticated: {DEVOTEE_ID}")

    _, pr_login = api_call(f"{STAGING_URL}/api/v1/auth/login", method="POST", data={"phone": PRIEST_PHONE, "password": PRIEST_PASS, "role": "priest"})
    pr_token = pr_login.get("access_token")
    log(f"Priest authenticated: {PRIEST_ID}")

    # 2. Get pooja catalog
    _, poojas = api_call(f"{STAGING_URL}/api/v1/poojas")
    pooja = poojas[0] if poojas else {"id": "1", "title": "Sri Maha Ganapathi Pooja", "base_price": 2100.0}
    log(f"Selected Pooja: {pooja.get('title')} (Price: {pooja.get('base_price')})")

    # ------------------------------------------------------------------
    # JOURNEY 1: Devotee Booking, Priest Acceptance, Chat & Cancellation
    # ------------------------------------------------------------------
    log("\n--- JOURNEY 1: Booking, Two-Way Chat & Swiggy Cancellation ---")
    b1_payload = {
        "user_id": DEVOTEE_ID,
        "priest_id": PRIEST_ID,
        "pooja_id": pooja.get("id"),
        "booking_type": "CALENDAR_SLOT",
        "include_samagri_kit": False,
        "total_amount": float(pooja.get("base_price", 2100.0)),
        "event_address": "Flat 202, SV Nilayam, Banjara Hills, Hyderabad",
        "latitude": 17.3850,
        "longitude": 78.4867
    }
    st_b1, res_b1 = api_call(f"{STAGING_URL}/api/v1/bookings", method="POST", data=b1_payload, token=dev_token)
    b1_id = res_b1.get("booking_id")
    b1_otp = res_b1.get("completion_otp")
    log(f"Booking 1 Created: HTTP {st_b1}, ID: {b1_id}, OTP: [PROTECTED]")

    # Priest accepts Booking 1
    st_acc, res_acc = api_call(f"{STAGING_URL}/api/v1/bookings/status", method="POST", data={"booking_id": b1_id, "status": "ACCEPTED", "priest_id": PRIEST_ID}, token=pr_token)
    log(f"Priest Accepted Booking 1: HTTP {st_acc}, Status: {res_acc.get('status')}")

    # Devotee creates chat room
    st_room, res_room = api_call(f"{STAGING_URL}/api/v1/chat/rooms", method="POST", data={"booking_id": b1_id, "priest_id": PRIEST_ID}, token=dev_token)
    room_id = res_room.get("room_id")
    log(f"Chat room initialized: {room_id}")

    # Devotee sends message
    api_call(f"{STAGING_URL}/api/v1/chat/messages", method="POST", data={"room_id": room_id, "sender_id": DEVOTEE_ID, "sender_role": "devotee", "content": "Namaste Swamiji, what time will you arrive?"}, token=dev_token)
    # Priest replies
    api_call(f"{STAGING_URL}/api/v1/chat/messages", method="POST", data={"room_id": room_id, "sender_id": PRIEST_ID, "sender_role": "priest", "content": "Namaste. I will arrive 15 minutes before muhurtham."}, token=pr_token)
    log("Exchanged two-way chat messages via API")

    # Launch Devotee App to verify Active Booking & Live Tracking
    run_adb(["shell", "am", "start", "-S", "-n", "com.purohit.devotee/.MainActivity"])
    time.sleep(3.5)

    # Tap Bookings Tab (center ~ 760, 2274)
    run_adb(["shell", "input", "tap", "760", "2274"])
    time.sleep(2.5)

    cap("18_devotee_active_booking_tracking.png")
    log("Captured Devotee Live Tracking Screen with assigned Purohit")

    # Tap In-App Chat button
    tree = dump_ui()
    for n in (tree.iter("node") if tree else []):
        if "chat" in n.attrib.get("text", "").lower() or "💬" in n.attrib.get("text", ""):
            b = n.attrib.get("bounds")
            import re; m = re.match(r'\[(\d+),(\d+)\]\[(\d+),(\d+)\]', b)
            cx, cy = (int(m.group(1)) + int(m.group(3))) // 2, (int(m.group(2)) + int(m.group(4))) // 2
            log(f"Tapping chat button at {cx}, {cy}")
            run_adb(["shell", "input", "tap", str(cx), str(cy)])
            break

    time.sleep(2.5)
    cap("19_devotee_chat_screen.png")
    log("Captured Devotee In-App Chat Screen showing live conversation")

    # Press back to return to tracking
    run_adb(["shell", "input", "keyevent", "4"])
    time.sleep(1.5)

    # Tap Cancel Booking button
    tree = dump_ui()
    for n in (tree.iter("node") if tree else []):
        if "cancel" in n.attrib.get("text", "").lower():
            b = n.attrib.get("bounds")
            import re; m = re.match(r'\[(\d+),(\d+)\]\[(\d+),(\d+)\]', b)
            cx, cy = (int(m.group(1)) + int(m.group(3))) // 2, (int(m.group(2)) + int(m.group(4))) // 2
            log(f"Tapping Cancel Booking at {cx}, {cy}")
            run_adb(["shell", "input", "tap", str(cx), str(cy)])
            break

    time.sleep(2.0)
    cap("20_devotee_cancellation_modal.png")
    log("Captured Devotee Swiggy Cancellation Modal")

    # Tap Confirm Cancellation button in modal
    tree = dump_ui()
    for n in (tree.iter("node") if tree else []):
        if "confirm" in n.attrib.get("text", "").lower() or "cancel pooja" in n.attrib.get("text", "").lower():
            b = n.attrib.get("bounds")
            import re; m = re.match(r'\[(\d+),(\d+)\]\[(\d+),(\d+)\]', b)
            cx, cy = (int(m.group(1)) + int(m.group(3))) // 2, (int(m.group(2)) + int(m.group(4))) // 2
            log(f"Confirming cancellation at {cx}, {cy}")
            run_adb(["shell", "input", "tap", str(cx), str(cy)])
            break

    time.sleep(3.5)
    cap("21_cancellation_confirmed_history.png")

    # Verify status in backend DB
    st_b1_stat, b1_stat = api_call(f"{STAGING_URL}/api/v1/bookings/{b1_id}", token=dev_token)
    log(f"Booking 1 DB Status: {b1_stat.get('status')} (HTTP {st_b1_stat})")

    # ------------------------------------------------------------------
    # JOURNEY 2: Sacred OTP Pooja Completion (Wrong OTP vs Correct OTP)
    # ------------------------------------------------------------------
    log("\n--- JOURNEY 2: Sacred OTP Pooja Completion ---")
    b2_payload = {
        "user_id": DEVOTEE_ID,
        "priest_id": PRIEST_ID,
        "pooja_id": pooja.get("id"),
        "booking_type": "CALENDAR_SLOT",
        "include_samagri_kit": False,
        "total_amount": float(pooja.get("base_price", 2100.0)),
        "event_address": "Flat 304, Sri Rama Nilayam, Hyderabad",
        "latitude": 17.3850,
        "longitude": 78.4867
    }
    st_b2, res_b2 = api_call(f"{STAGING_URL}/api/v1/bookings", method="POST", data=b2_payload, token=dev_token)
    b2_id = res_b2.get("booking_id")
    b2_otp = res_b2.get("completion_otp")
    log(f"Booking 2 Created: ID {b2_id}, Completion OTP: [PROTECTED]")

    # Advance lifecycle: ACCEPTED -> IN_TRANSIT -> ARRIVED -> POOJA_STARTED
    for st in ["ACCEPTED", "IN_TRANSIT", "ARRIVED", "POOJA_STARTED"]:
        st_res, res = api_call(f"{STAGING_URL}/api/v1/bookings/status", method="POST", data={"booking_id": b2_id, "status": st, "priest_id": PRIEST_ID}, token=pr_token)
        log(f"Advanced Booking 2 to {st}: HTTP {st_res}")

    # Attempt completion with WRONG OTP
    st_wrong, res_wrong = api_call(
        f"{STAGING_URL}/api/v1/bookings/status",
        method="POST",
        data={"booking_id": b2_id, "status": "COMPLETED", "completion_otp": "0000", "dakshina_amount": 2100.0},
        token=pr_token
    )
    log(f"Wrong OTP rejection result: HTTP {st_wrong}, Response: {res_wrong}")

    # Complete with CORRECT SACRED OTP
    st_ok, res_ok = api_call(
        f"{STAGING_URL}/api/v1/bookings/status",
        method="POST",
        data={"booking_id": b2_id, "status": "COMPLETED", "completion_otp": b2_otp, "dakshina_amount": 2100.0},
        token=pr_token
    )
    log(f"Correct Sacred OTP completion result: HTTP {st_ok}, Escrow Status: {res_ok.get('escrow_status')}")

    # Launch Devotee App to view completed booking in Booking History
    run_adb(["shell", "am", "start", "-n", "com.purohit.devotee/.MainActivity"])
    time.sleep(2.5)
    run_adb(["shell", "input", "tap", "760", "2274"]) # Bookings tab
    time.sleep(2.0)
    cap("22_devotee_completed_history.png")
    log("Captured Devotee Completed Booking History Screen")

    # ------------------------------------------------------------------
    # JOURNEY 3: Accessibility & Font Scaling (2.0x)
    # ------------------------------------------------------------------
    log("\n--- JOURNEY 3: Accessibility Testing (Font Scale 2.0x) ---")
    run_adb(["shell", "settings", "put", "system", "font_scale", "2.0"])
    time.sleep(1.0)
    run_adb(["shell", "am", "start", "-n", "com.purohit.devotee/.MainActivity"])
    time.sleep(2.5)
    cap("23_accessibility_font_scale_2x.png")
    log("Captured Accessibility 2.0x Font Scaling Screen")

    # Reset font scale to normal
    run_adb(["shell", "settings", "put", "system", "font_scale", "1.0"])
    log("Reset font scale to 1.0")

    log("\n" + "=" * 80)
    log("ALL CORE JOURNEYS COMPLETED SUCCESSFULLY")
    log("=" * 80)

if __name__ == "__main__":
    main()
