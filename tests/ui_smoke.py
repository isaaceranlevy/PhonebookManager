"""Run against a clean temporary instance on port 8765. Requires Playwright."""
import json, urllib.request, tempfile, subprocess, time, atexit, socket, os
from pathlib import Path
from playwright.sync_api import sync_playwright
root=Path(__file__).resolve().parents[1]
(root/'bin').mkdir(exist_ok=True)
for port in (8765,8088):
 with socket.socket() as sock:
  if sock.connect_ex(('127.0.0.1',port))==0: raise RuntimeError(f'Port {port} is already in use; stop the existing test instance')
data=tempfile.TemporaryDirectory(prefix='nimbus-browser-')
process=subprocess.Popen([os.environ.get('UI_BINARY',str(root/'bin/phonebook')),'run','--data-dir',data.name],stderr=subprocess.PIPE)
def cleanup():
 process.terminate()
 try: process.wait(timeout=15)
 except subprocess.TimeoutExpired: process.kill();process.wait()
 data.cleanup()
atexit.register(cleanup)
for attempt in range(100):
 if process.poll() is not None: raise RuntimeError(process.stderr.read().decode())
 try:
  urllib.request.urlopen('http://127.0.0.1:8765/api/book',timeout=.3).close();break
 except OSError: time.sleep(.1)
else: raise RuntimeError('Application did not become ready')
with sync_playwright() as p:
 browser=p.chromium.launch(executable_path=os.environ.get('CHROMIUM_PATH','/usr/bin/chromium'),args=['--no-sandbox'])
 context=browser.new_context(viewport={'width':1440,'height':1000})
 page=context.new_page()
 errors=[]
 page.on('pageerror',lambda e:errors.append(str(e)))
 page.goto('http://127.0.0.1:8765')
 page.get_by_role('button',name='איש קשר חדש',exact=True).click()
 page.get_by_label('שם מלא').fill('דנה & כהן')
 page.get_by_label('טלפון ראשי').fill('0012345')
 page.get_by_label('טלפון נוסף').fill('+972501234567')
 page.get_by_role('button',name='הוספה לספר',exact=True).click()
 with page.expect_download() as download_event:
  page.locator('a[download]').click()
 exported=Path(download_event.value.path()).read_text(encoding='utf-8-sig')
 assert 'דנה & כהן' in exported and '0012345' in exported, 'Export must preserve unsaved draft'
 page.get_by_role('button',name='שמירת שינויים',exact=True).click()
 page.get_by_text('השינויים נשמרו בהצלחה',exact=True).wait_for()
 page.reload()
 page.get_by_text('דנה & כהן',exact=True).wait_for()
 page.get_by_label('חיפוש אנשי קשר').fill('לא נמצא')
 page.get_by_text('לא נמצאו תוצאות',exact=True).wait_for()
 page.get_by_label('חיפוש אנשי קשר').fill('')
 page.get_by_role('button',name='עריכת דנה & כהן').click()
 page.get_by_label('שם מלא').fill('דנה לוי')
 page.get_by_role('button',name='עדכון איש קשר',exact=True).click()
 page.get_by_role('button',name='שמירת שינויים',exact=True).click()
 page.get_by_text('השינויים נשמרו בהצלחה',exact=True).wait_for()
 page.get_by_role('button',name='ייבוא CSV',exact=True).click()
 page.locator('#csv-file').set_input_files({'name':'test.csv','mimeType':'text/csv','buffer':'name,phone1,phone2,phone3\nישראל,+97254321,00123,\n'.encode()})
 page.get_by_role('button',name='מיזוג עם הספר',exact=True).wait_for()
 page.get_by_role('button',name='מיזוג עם הספר',exact=True).click()
 page.get_by_role('button',name='שמירת שינויים',exact=True).click()
 page.get_by_text('השינויים נשמרו בהצלחה',exact=True).wait_for()
 xml=urllib.request.urlopen('http://127.0.0.1:8088/phonebook.xml').read().decode()
 assert 'ישראל' in xml and '0012345' in xml and '+97254321' in xml
 page.screenshot(path=str(root/'bin/desktop.png'),full_page=True)
 page.get_by_role('button',name='גיבויים ושחזור',exact=True).click()
 page.locator('#backup-list button').first.click()
 page.get_by_role('button',name='אישור שחזור',exact=True).click()
 page.get_by_text('הגיבוי שוחזר ונשמר',exact=True).wait_for()
 page.locator('#backups-dialog .dialog-footer button').click()
 page.get_by_role('button',name='מחיקת דנה לוי').click()
 page.get_by_role('button',name='אישור מחיקה',exact=True).click()
 page.get_by_role('button',name='שמירת שינויים',exact=True).click()
 page.get_by_text('השינויים נשמרו בהצלחה',exact=True).wait_for()
 page.reload()
 page.get_by_text('הספר שלכם מתחיל כאן',exact=True).wait_for()
 page.set_viewport_size({'width':390,'height':844})
 page.screenshot(path=str(root/'bin/mobile.png'),full_page=True)
 assert page.evaluate('document.documentElement.scrollWidth <= window.innerWidth'), 'horizontal overflow'
 # Two independent pages reproduce a real optimistic-concurrency conflict.
 second=browser.new_page()
 second.goto('http://127.0.0.1:8765')
 second.get_by_role('button',name='איש קשר חדש',exact=True).click()
 second.get_by_label('שם מלא').fill('נשמר בחלון שני')
 second.get_by_label('טלפון ראשי').fill('01234')
 second.get_by_role('button',name='הוספה לספר',exact=True).click()
 second.get_by_role('button',name='שמירת שינויים',exact=True).click()
 second.get_by_text('השינויים נשמרו בהצלחה',exact=True).wait_for()
 page.get_by_role('button',name='איש קשר חדש',exact=True).click()
 page.get_by_label('שם מלא').fill('טיוטה לשימור')
 page.get_by_label('טלפון ראשי').fill('+9720001')
 page.get_by_role('button',name='הוספה לספר',exact=True).click()
 # Closing a dirty tab must warn and allow cancellation.
 def dismiss_unload(dialog): dialog.dismiss()
 page.on('dialog',dismiss_unload)
 with page.expect_event('dialog') as warning:
  page.close(run_before_unload=True)
 assert warning.value.type=='beforeunload'
 assert not page.is_closed()
 page.remove_listener('dialog',dismiss_unload)
 page.get_by_role('button',name='שמירת שינויים',exact=True).click()
 page.get_by_text('הספר עודכן בחלון אחר.',exact=False).wait_for()
 with page.expect_download() as conflict_download:
  page.locator('a[download]').click()
 draft=Path(conflict_download.value.path()).read_text(encoding='utf-8-sig')
 assert 'טיוטה לשימור' in draft and '+9720001' in draft and 'נשמר בחלון שני' not in draft
 page.get_by_role('button',name='טעינה מחדש',exact=True).click()
 page.get_by_role('button',name='אישור טעינה',exact=True).click()
 page.get_by_text('נשמר בחלון שני',exact=True).wait_for()
 assert page.get_by_text('טיוטה לשימור',exact=True).count()==0
 second.close()
 assert not errors, errors
 print('PASS: create, persist/reload, search, edit, CSV merge, XML, backup restore, delete, mobile, beforeunload, two-tab 409 and draft export; no JavaScript errors')
 browser.close()
