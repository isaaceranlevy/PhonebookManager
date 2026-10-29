"""Local runtime validation of both fixture formats and actual listener boundaries."""
import http.client, json, subprocess, time, urllib.request, urllib.parse, xml.etree.ElementTree as ET
from pathlib import Path
root=Path(__file__).resolve().parents[1]
for fmt,tag in [('directory','YealinkIPPhoneDirectory'),('phonebook','YealinkIPPhoneBook')]:
 p=subprocess.Popen([str(root/'bin/phonebook'),'serve-fixture','--format',fmt],stderr=subprocess.PIPE)
 try:
  for attempt in range(100):
   if p.poll() is not None: raise RuntimeError(p.stderr.read().decode())
   try:
    info=json.load(urllib.request.urlopen('http://127.0.0.1:8765/api/info',timeout=.2));break
   except OSError: time.sleep(.1)
  else: raise RuntimeError('fixture startup timed out')
  result=urllib.request.urlopen('http://127.0.0.1:8088/phonebook.xml').read()
  tree=ET.fromstring(result)
  assert tree.tag==tag
  assert '0012345' in result.decode() and '+972501234567' in result.decode() and 'בדיקת עברית' in result.decode()
  for url in info['urls']:
   host=urllib.parse.urlsplit(url).hostname
   c=http.client.HTTPConnection(host,8765,timeout=1)
   try:
    c.request('GET','/');c.getresponse()
   except (ConnectionRefusedError,TimeoutError,OSError): pass
   else: raise AssertionError('admin reachable on LAN')
   finally: c.close()
   c=http.client.HTTPConnection(host,8088,timeout=2)
   c.request('GET','/phonebook.xml');r=c.getresponse();assert r.status==200;assert ET.fromstring(r.read()).tag==tag
   c.request('GET','/api/book');r=c.getresponse();assert r.status==404;r.read();c.close()
  print(f'PASS {fmt}: fixture content, real HTTP XML, LAN XML/API isolation, loopback-only admin ({len(info["urls"])} LAN interfaces)')
 finally:
  p.terminate()
  try: p.wait(timeout=15)
  except subprocess.TimeoutExpired: p.kill();p.wait();raise
  assert p.returncode==0, p.stderr.read().decode()
