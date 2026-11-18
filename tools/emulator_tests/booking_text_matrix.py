#!/usr/bin/env python3
"""Repeatable Android catalog-to-booking-screen checks; does not create bookings."""
import argparse, json, os, re, subprocess, sys, time, xml.etree.ElementTree as ET
from pathlib import Path

PACKAGE = 'com.purohit.devotee'
LABELS = {
 'en': {'language':'English (English)', 'add':'+ Add Service', 'dispatch':'Request Instant Radar Dispatch', 'address':'Pooja address'},
 'te': {'language':'తెలుగు (Telugu)', 'add':'+ సేవను చేర్చండి', 'dispatch':'తక్షణ పురోహితుల రాడార్', 'address':'పూజ చిరునామా'},
 'hi': {'language':'हिन्दी (Hindi)', 'add':'+ पूजा जोड़ें', 'dispatch':'तात्कालिक पुरोहित रडार', 'address':'पूजा का पता'},
}
def adb(*args, raw=False):
 p=subprocess.run(['adb',*map(str,args)],capture_output=True,timeout=45,check=True)
 return p.stdout if raw else p.stdout.decode('utf-8','replace').strip()
def dump():
 for attempt in range(6):
  try:
   adb('shell','rm','-f','/sdcard/pv-regression.xml')
   adb('shell','uiautomator','dump','/sdcard/pv-regression.xml')
   data=adb('exec-out','cat','/sdcard/pv-regression.xml',raw=True)
   return ET.fromstring(data)
  except (ET.ParseError, subprocess.CalledProcessError):
   time.sleep(1)
 raise AssertionError('could not read emulator UI hierarchy')
def find(root, label):
 for node in root.iter('node'):
  if label in node.attrib.get('text','') or label in node.attrib.get('content-desc',''):
   bounds=node.attrib.get('bounds','')
   m=re.fullmatch(r'\[(\d+),(\d+)\]\[(\d+),(\d+)\]',bounds)
   if m:
    l,t,r,b=map(int,m.groups())
    if r>l and b>t: return ((l+r)//2,(t+b)//2)
 return None
def wait(label, seconds=20):
 end=time.monotonic()+seconds
 while time.monotonic()<end:
  node=find(dump(),label)
  if node:return node
 return None
def tap(point): adb('shell','input','tap',*point)
def choose_language(lang):
 # The restored session can briefly show the login screen before loading the catalog.
 node=None
 for _ in range(8):
  root=dump()
  node=next((find(root,x) for x in ('English','తెలుగు','हिन्दी') if find(root,x)),None)
  if node: break
  time.sleep(1)
 if not node: raise AssertionError('language selector unavailable; sign in and open the catalog first')
 tap(node)
 p=wait(LABELS[lang]['language'],8)
 if not p: raise AssertionError('language menu option missing')
 tap(p)
def run_case(scale,lang):
 adb('shell','settings','put','system','font_scale',scale)
 adb('shell','am','force-stop',PACKAGE)
 adb('shell','monkey','-p',PACKAGE,'1')
 if not wait('Pooja Vahini',35): raise AssertionError('app did not open; sign in to the test account first')
 choose_language(lang)
 cfg=LABELS[lang]
 # Grid can be short at high font scales; reveal an Add Service control.
 add=None
 for _ in range(8):
  add=wait(cfg['add'],3)
  if add: break
  adb('shell','input','swipe','500','1900','500','1350','350')
 if not add: raise AssertionError('service selection action not found')
 tap(add)
 action=wait(cfg['dispatch'],8)
 if not action: raise AssertionError('booking action is not visible/reachable after selecting service')
 tap(action)
 # Wait for the live discovery request to finish and expose the address input.
 if not wait(cfg['address'], 35):
  raise AssertionError('booking address screen did not appear')

def main():
 ap=argparse.ArgumentParser(description=__doc__)
 ap.add_argument('--scales',nargs='+',default=['1.0','1.3','1.6','2.0'])
 ap.add_argument('--languages',nargs='+',choices=list(LABELS),default=list(LABELS))
 ap.add_argument('--output',default='output/emulator-booking-matrix.json')
 args=ap.parse_args(); original=adb('shell','settings','get','system','font_scale'); results=[]
 try:
  for scale in args.scales:
   for lang in args.languages:
    try: run_case(scale,lang); item={'scale':scale,'language':lang,'result':'PASS'}
    except Exception as e: item={'scale':scale,'language':lang,'result':'FAIL','reason':str(e)}
    results.append(item); print(json.dumps(item,ensure_ascii=False),flush=True)
 finally:
  adb('shell','settings','put','system','font_scale',original if original not in ('null','') else '1.0')
  out=Path(args.output); out.parent.mkdir(parents=True,exist_ok=True); out.write_text(json.dumps(results,ensure_ascii=False,indent=2),encoding='utf-8')
 return 1 if any(r['result']!='PASS' for r in results) else 0
if __name__=='__main__': sys.exit(main())
