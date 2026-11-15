import subprocess
import time
import xml.etree.ElementTree as ET
import sys
import os

sys.stdout.reconfigure(encoding='utf-8')

SCREENSHOT_DIR = r"C:\Users\susha\OneDrive\Documents\ChatGPT\pooja vahini\output\supervised-internal-20261115\screenshots"

def cap(name):
    p = os.path.join(SCREENSHOT_DIR, name)
    subprocess.run(f'adb exec-out screencap -p > "{p}"', shell=True)
    print("Captured", name)

# 1. Appearance -> Calm Modern
subprocess.run(['adb', 'shell', 'input', 'tap', '740', '298'])
time.sleep(1.0)
subprocess.run(['adb', 'shell', 'uiautomator', 'dump', '/data/local/tmp/dump.xml'])
out = subprocess.run(['adb', 'shell', 'cat', '/data/local/tmp/dump.xml'], capture_output=True, text=True, encoding='utf-8').stdout
tree = ET.fromstring(out)
for n in tree.iter('node'):
    if 'calm modern' in n.attrib.get('text', '').lower():
        b = n.attrib.get('bounds')
        import re; m = re.match(r'\[(\d+),(\d+)\]\[(\d+),(\d+)\]', b)
        cx, cy = (int(m.group(1)) + int(m.group(3))) // 2, (int(m.group(2)) + int(m.group(4))) // 2
        subprocess.run(['adb', 'shell', 'input', 'tap', str(cx), str(cy)])
        break
time.sleep(1.0)
cap('10_theme_calm_modern.png')

# 2. Appearance -> Temple Earth
subprocess.run(['adb', 'shell', 'input', 'tap', '740', '298'])
time.sleep(1.0)
subprocess.run(['adb', 'shell', 'uiautomator', 'dump', '/data/local/tmp/dump.xml'])
out = subprocess.run(['adb', 'shell', 'cat', '/data/local/tmp/dump.xml'], capture_output=True, text=True, encoding='utf-8').stdout
tree = ET.fromstring(out)
for n in tree.iter('node'):
    if 'temple earth' in n.attrib.get('text', '').lower():
        b = n.attrib.get('bounds')
        import re; m = re.match(r'\[(\d+),(\d+)\]\[(\d+),(\d+)\]', b)
        cx, cy = (int(m.group(1)) + int(m.group(3))) // 2, (int(m.group(2)) + int(m.group(4))) // 2
        subprocess.run(['adb', 'shell', 'input', 'tap', str(cx), str(cy)])
        break
time.sleep(1.0)
cap('11_theme_temple_earth.png')

# Reset to Royal Heritage
subprocess.run(['adb', 'shell', 'input', 'tap', '740', '298'])
time.sleep(1.0)
subprocess.run(['adb', 'shell', 'uiautomator', 'dump', '/data/local/tmp/dump.xml'])
out = subprocess.run(['adb', 'shell', 'cat', '/data/local/tmp/dump.xml'], capture_output=True, text=True, encoding='utf-8').stdout
tree = ET.fromstring(out)
for n in tree.iter('node'):
    if 'royal heritage' in n.attrib.get('text', '').lower():
        b = n.attrib.get('bounds')
        import re; m = re.match(r'\[(\d+),(\d+)\]\[(\d+),(\d+)\]', b)
        cx, cy = (int(m.group(1)) + int(m.group(3))) // 2, (int(m.group(2)) + int(m.group(4))) // 2
        subprocess.run(['adb', 'shell', 'input', 'tap', str(cx), str(cy)])
        break
time.sleep(1.0)

# 3. Language -> Telugu
subprocess.run(['adb', 'shell', 'input', 'tap', '946', '298'])
time.sleep(1.0)
subprocess.run(['adb', 'shell', 'uiautomator', 'dump', '/data/local/tmp/dump.xml'])
out = subprocess.run(['adb', 'shell', 'cat', '/data/local/tmp/dump.xml'], capture_output=True, text=True, encoding='utf-8').stdout
tree = ET.fromstring(out)
for n in tree.iter('node'):
    if 'తెలుగు' in n.attrib.get('text', ''):
        b = n.attrib.get('bounds')
        import re; m = re.match(r'\[(\d+),(\d+)\]\[(\d+),(\d+)\]', b)
        cx, cy = (int(m.group(1)) + int(m.group(3))) // 2, (int(m.group(2)) + int(m.group(4))) // 2
        subprocess.run(['adb', 'shell', 'input', 'tap', str(cx), str(cy)])
        break
time.sleep(1.0)
# Tap outside or Close
subprocess.run(['adb', 'shell', 'input', 'tap', '540', '1500'])
time.sleep(1.0)
cap('12_telugu_store.png')

# 4. Language -> Hindi
subprocess.run(['adb', 'shell', 'input', 'tap', '946', '298'])
time.sleep(1.0)
subprocess.run(['adb', 'shell', 'uiautomator', 'dump', '/data/local/tmp/dump.xml'])
out = subprocess.run(['adb', 'shell', 'cat', '/data/local/tmp/dump.xml'], capture_output=True, text=True, encoding='utf-8').stdout
tree = ET.fromstring(out)
for n in tree.iter('node'):
    if 'हिन्दी' in n.attrib.get('text', ''):
        b = n.attrib.get('bounds')
        import re; m = re.match(r'\[(\d+),(\d+)\]\[(\d+),(\d+)\]', b)
        cx, cy = (int(m.group(1)) + int(m.group(3))) // 2, (int(m.group(2)) + int(m.group(4))) // 2
        subprocess.run(['adb', 'shell', 'input', 'tap', str(cx), str(cy)])
        break
time.sleep(1.0)
subprocess.run(['adb', 'shell', 'input', 'tap', '540', '1500'])
time.sleep(1.0)
cap('13_hindi_store.png')

# Reset to English
subprocess.run(['adb', 'shell', 'input', 'tap', '946', '298'])
time.sleep(1.0)
subprocess.run(['adb', 'shell', 'uiautomator', 'dump', '/data/local/tmp/dump.xml'])
out = subprocess.run(['adb', 'shell', 'cat', '/data/local/tmp/dump.xml'], capture_output=True, text=True, encoding='utf-8').stdout
tree = ET.fromstring(out)
for n in tree.iter('node'):
    if 'english' in n.attrib.get('text', '').lower():
        b = n.attrib.get('bounds')
        import re; m = re.match(r'\[(\d+),(\d+)\]\[(\d+),(\d+)\]', b)
        cx, cy = (int(m.group(1)) + int(m.group(3))) // 2, (int(m.group(2)) + int(m.group(4))) // 2
        subprocess.run(['adb', 'shell', 'input', 'tap', str(cx), str(cy)])
        break
time.sleep(1.0)
subprocess.run(['adb', 'shell', 'input', 'tap', '540', '1500'])
time.sleep(1.0)
print("Themes and languages cycled successfully.")
