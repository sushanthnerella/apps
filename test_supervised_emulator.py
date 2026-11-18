import subprocess
import time
import os
import sys
import xml.etree.ElementTree as ET
import requests

SCREENSHOT_DIR = r"C:\Users\susha\OneDrive\Documents\ChatGPT\pooja vahini\output\supervised-internal-20261115\screenshots"
LOG_DIR = r"C:\Users\susha\OneDrive\Documents\ChatGPT\pooja vahini\output\supervised-internal-20261115"
os.makedirs(SCREENSHOT_DIR, exist_ok=True)

STAGING_URL = "https://pooja-vahini-staging.onrender.com"

# Generate unique timestamps for fresh test accounts
TIMESTAMP = int(time.time())
DEVOTEE_PHONE = f"+91998811{TIMESTAMP % 10000:04d}"
DEVOTEE_NAME = f"Devotee Pilot {TIMESTAMP % 10000}"
DEVOTEE_PASS = "TestPass1234!"

PRIEST_PHONE = f"+91987766{TIMESTAMP % 10000:04d}"
PRIEST_NAME = f"Pandit Sharma {TIMESTAMP % 10000}"
PRIEST_PASS = "PriestPass1234!"

results = []

def record_result(category, name, status, details=""):
    results.append({
        "category": category,
        "name": name,
        "status": status,
        "details": details
    })
    print(f"[{status}] {category} - {name}: {details}")

def run_adb(args):
    cmd = ["adb"] + args
    res = subprocess.run(cmd, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, encoding="utf-8", errors="ignore")
    return res.stdout.strip()

import sys
if hasattr(sys.stdout, 'reconfigure'):
    sys.stdout.reconfigure(encoding='utf-8', errors='replace')

def capture_screenshot(filename):
    path = os.path.join(SCREENSHOT_DIR, filename)
    subprocess.run(f'adb exec-out screencap -p > "{path}"', shell=True)
    size = os.path.getsize(path) if os.path.exists(path) else 0
    print(f"[SCREENSHOT] Captured: {filename} ({size} bytes)")
    return path

def dump_ui():
    run_adb(["shell", "uiautomator", "dump", "/data/local/tmp/dump.xml"])
    xml_data = run_adb(["shell", "cat", "/data/local/tmp/dump.xml"])
    if not xml_data.startswith("<?xml"):
        time.sleep(1)
        run_adb(["shell", "uiautomator", "dump", "/data/local/tmp/dump.xml"])
        xml_data = run_adb(["shell", "cat", "/data/local/tmp/dump.xml"])
    try:
        return ET.fromstring(xml_data)
    except Exception as e:
        print(f"UI dump error: {e}")
        return None

def find_node(root, predicate):
    if root is None:
        return None
    for node in root.iter("node"):
        if predicate(node):
            return node
    return None

def find_all_nodes(root, predicate):
    if root is None:
        return []
    return [node for node in root.iter("node") if predicate(node)]

def get_node_bounds(node):
    bounds = node.attrib.get("bounds", "")
    import re
    m = re.match(r"\[(\d+),(\d+)\]\[(\d+),(\d+)\]", bounds)
    if m:
        return tuple(map(int, m.groups()))
    return None

def tap_node(node):
    b = get_node_bounds(node)
    if b:
        cx = (b[0] + b[2]) // 2
        cy = (b[1] + b[3]) // 2
        run_adb(["shell", "input", "tap", str(cx), str(cy)])
        time.sleep(0.8)
        return True
    return False

def tap_text(root, text_substr):
    node = find_node(root, lambda n: text_substr.lower() in n.attrib.get("text", "").lower() or text_substr.lower() in n.attrib.get("content-desc", "").lower())
    if node:
        return tap_node(node)
    return False

def hide_keyboard():
    run_adb(["shell", "input", "keyevent", "111"])
    time.sleep(0.5)

def input_text(text):
    escaped = text.replace(" ", "%s")
    run_adb(["shell", "input", "text", escaped])
    time.sleep(0.5)

def clear_input(chars=40):
    for _ in range(chars):
        run_adb(["shell", "input", "keyevent", "67"]) # KEYCODE_DEL
    time.sleep(0.3)

def wait_for_text(text_substr, timeout=10):
    t0 = time.time()
    while time.time() - t0 < timeout:
        root = dump_ui()
        if root:
            node = find_node(root, lambda n: text_substr.lower() in n.attrib.get("text", "").lower())
            if node:
                return root, node
        time.sleep(1)
    return None, None

print(f"Test run target: Devotee={DEVOTEE_PHONE}, Priest={PRIEST_PHONE}")
def test_part_a_password_recovery():
    print("\n--- Part A: Password Recovery Verification (Goal 4) ---")
    run_adb(["shell", "pm", "clear", "com.purohit.devotee"])
    run_adb(["shell", "pm", "grant", "com.purohit.devotee", "android.permission.POST_NOTIFICATIONS"])
    run_adb(["shell", "pm", "grant", "com.purohit.devotee", "android.permission.ACCESS_FINE_LOCATION"])
    run_adb(["shell", "am", "start", "-n", "com.purohit.devotee/.MainActivity"])
    time.sleep(3)

    root = dump_ui()
    capture_screenshot("01_devotee_login_screen.png")

    # Tap Forgot Password
    forgot_btn = find_node(root, lambda n: "forgot" in n.attrib.get("text", "").lower())
    if not forgot_btn:
        record_result("Recovery", "Forgot Password Button", "FAIL", "Forgot password button not found")
        return
    tap_node(forgot_btn)
    time.sleep(1)

    root = dump_ui()
    # Check dialog title
    if not find_node(root, lambda n: "reset password" in n.attrib.get("text", "").lower()):
        record_result("Recovery", "Reset Password Dialog", "FAIL", "Dialog not opened")
        return

    # Enter test email
    email_field = find_node(root, lambda n: n.attrib.get("class") == "android.widget.EditText")
    if email_field:
        tap_node(email_field)
        input_text("pilot.test@example.com")
        hide_keyboard()

    root = dump_ui()
    send_btn = find_node(root, lambda n: "send reset link" in n.attrib.get("text", "").lower())
    if send_btn:
        tap_node(send_btn)
        time.sleep(3) # wait for server 503 response

    root = dump_ui()
    capture_screenshot("02_devotee_password_recovery_unconfigured.png")
    err_node = find_node(root, lambda n: "currently unconfigured" in n.attrib.get("text", "").lower() or "unavailable" in n.attrib.get("text", "").lower())
    if err_node:
        record_result("Recovery", "Password Recovery Unconfigured Notice", "PASS", f"Honest error displayed: {err_node.attrib.get('text')}")
    else:
        record_result("Recovery", "Password Recovery Unconfigured Notice", "FAIL", "Unconfigured notice not found")

    # Dismiss dialog
    cancel_btn = find_node(root, lambda n: "cancel" in n.attrib.get("text", "").lower())
    if cancel_btn:
        tap_node(cancel_btn)
    else:
        run_adb(["shell", "input", "keyevent", "4"])
    time.sleep(1)

def test_part_b_registration_and_login():
    print("\n--- Part B: Devotee Registration, Duplicate Rejection & Login (Goal 2) ---")
    root = dump_ui()
    reg_btn = find_node(root, lambda n: "register" in n.attrib.get("text", "").lower())
    if reg_btn:
        tap_node(reg_btn)
        time.sleep(1.5)

    root = dump_ui()
    # Registration screen
    edit_texts = find_all_nodes(root, lambda n: n.attrib.get("class") == "android.widget.EditText")
    print(f"Found {len(edit_texts)} input fields on registration screen.")

    # 1. Test validation failure (empty submit)
    # Scroll to Register button
    run_adb(["shell", "input", "swipe", "500", "1800", "500", "600"])
    time.sleep(1)
    root = dump_ui()
    sub_btn = find_node(root, lambda n: "register" in n.attrib.get("text", "").lower())
    if sub_btn:
        tap_node(sub_btn)
        time.sleep(1)
        root = dump_ui()
        val_err = find_node(root, lambda n: "enter your name" in n.attrib.get("text", "").lower() or "phone" in n.attrib.get("text", "").lower())
        if val_err:
            record_result("Registration", "Blank Submission Validation", "PASS", f"Validation error shown: {val_err.attrib.get('text')}")
        else:
            record_result("Registration", "Blank Submission Validation", "FAIL", "No validation error shown on blank submit")

    # Scroll back up to fill details
    run_adb(["shell", "input", "swipe", "500", "600", "500", "1800"])
    time.sleep(1)
    root = dump_ui()
    edit_texts = find_all_nodes(root, lambda n: n.attrib.get("class") == "android.widget.EditText")

    if len(edit_texts) >= 3:
        # Full Name
        tap_node(edit_texts[0])
        input_text(DEVOTEE_NAME)
        hide_keyboard()

        # Phone
        tap_node(edit_texts[1])
        input_text(DEVOTEE_PHONE)
        hide_keyboard()

        # Password
        tap_node(edit_texts[2])
        input_text(DEVOTEE_PASS)
        hide_keyboard()

    # Fill optional personal details if visible
    if len(edit_texts) >= 5:
        tap_node(edit_texts[3])
        input_text("Kashyapa")
        hide_keyboard()
        tap_node(edit_texts[4])
        input_text("Rohini")
        hide_keyboard()

    # Scroll down to consent and submit
    run_adb(["shell", "input", "swipe", "500", "1800", "500", "600"])
    time.sleep(1)
    root = dump_ui()

    # Attempt register WITHOUT consent
    sub_btn = find_node(root, lambda n: "register" in n.attrib.get("text", "").lower() and n.attrib.get("class") == "android.widget.TextView")
    if sub_btn:
        tap_node(sub_btn)
        time.sleep(1)
        root = dump_ui()
        capture_screenshot("03_devotee_reg_consent_required.png")
        consent_err = find_node(root, lambda n: "consent" in n.attrib.get("text", "").lower() or "mandatory" in n.attrib.get("text", "").lower())
        if consent_err:
            record_result("Registration", "Mandatory Consent Validation", "PASS", f"Consent error shown: {consent_err.attrib.get('text')}")
        else:
            record_result("Registration", "Mandatory Consent Validation", "FAIL", "Missing consent was not flagged")

    # Check the consent checkbox
    root = dump_ui()
    checkbox = find_node(root, lambda n: n.attrib.get("class") == "android.widget.CheckBox" or "agree to the consent" in n.attrib.get("text", "").lower())
    if checkbox:
        tap_node(checkbox)
        time.sleep(0.5)

    # Submit fresh registration
    root = dump_ui()
    sub_btn = find_node(root, lambda n: "register" in n.attrib.get("text", "").lower() and n.attrib.get("class") == "android.widget.TextView")
    if sub_btn:
        tap_node(sub_btn)
        time.sleep(3)

    root = dump_ui()
    capture_screenshot("04_devotee_fresh_registration_success.png")
    # Devotee registration logs in or redirects to login/catalog
    logged_in = find_node(root, lambda n: "puja" in n.attrib.get("text", "").lower() or "services" in n.attrib.get("text", "").lower() or "bookings" in n.attrib.get("text", "").lower())
    if logged_in:
        record_result("Registration", "Fresh Devotee Registration", "PASS", f"Registered account {DEVOTEE_PHONE} successfully")
    else:
        record_result("Registration", "Fresh Devotee Registration", "FAIL", "Did not navigate to main screen after registration")

    # Test Duplicate Registration:
    print("\n--- Duplicate Registration Test ---")
    run_adb(["shell", "pm", "clear", "com.purohit.devotee"])
    run_adb(["shell", "pm", "grant", "com.purohit.devotee", "android.permission.POST_NOTIFICATIONS"])
    run_adb(["shell", "pm", "grant", "com.purohit.devotee", "android.permission.ACCESS_FINE_LOCATION"])
    run_adb(["shell", "am", "start", "-n", "com.purohit.devotee/.MainActivity"])
    time.sleep(3)

    root = dump_ui()
    reg_btn = find_node(root, lambda n: "register" in n.attrib.get("text", "").lower())
    if reg_btn:
        tap_node(reg_btn)
        time.sleep(1.5)

    root = dump_ui()
    edit_texts = find_all_nodes(root, lambda n: n.attrib.get("class") == "android.widget.EditText")
    if len(edit_texts) >= 3:
        tap_node(edit_texts[0])
        input_text("Duplicate Test User")
        hide_keyboard()
        tap_node(edit_texts[1])
        input_text(DEVOTEE_PHONE) # EXACT SAME PHONE
        hide_keyboard()
        tap_node(edit_texts[2])
        input_text(DEVOTEE_PASS)
        hide_keyboard()

    run_adb(["shell", "input", "swipe", "500", "1800", "500", "600"])
    time.sleep(1)
    root = dump_ui()
    checkbox = find_node(root, lambda n: n.attrib.get("class") == "android.widget.CheckBox" or "agree to the consent" in n.attrib.get("text", "").lower())
    if checkbox:
        tap_node(checkbox)
        time.sleep(0.5)

    root = dump_ui()
    sub_btn = find_node(root, lambda n: "register" in n.attrib.get("text", "").lower() and n.attrib.get("class") == "android.widget.TextView")
    if sub_btn:
        tap_node(sub_btn)
        time.sleep(3)

    root = dump_ui()
    capture_screenshot("05_devotee_duplicate_registration_error.png")
    dup_err = find_node(root, lambda n: "already exists" in n.attrib.get("text", "").lower() or "please log in" in n.attrib.get("text", "").lower())
    if dup_err:
        record_result("Registration", "Duplicate Account Rejection", "PASS", f"Honest duplicate error shown: {dup_err.attrib.get('text')}")
    else:
        record_result("Registration", "Duplicate Account Rejection", "FAIL", "Duplicate error not shown")

if __name__ == "__main__":
    test_part_a_password_recovery()
    test_part_b_registration_and_login()
