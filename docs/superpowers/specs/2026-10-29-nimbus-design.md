# Nimbus Phonebook — תוכנית מיועלת למימוש מהיר

## הקשר
התוכנית המקורית (6 סוכנים, שלבים A–F) נכונה בעקרונות: JSON כמקור אמת, ניהול ב-loopback בלבד, XML לקריאה בלבד ב-LAN ושירות Windows. אבל בשביל כלי בהיקף של כ-2,000 שורות היא כבדה מדי. התיקייה `PhonebookManager` ריקה, Go לא מותקן (יש winget ו-git). המטרה: להגיע מהר לבינארי אחד שעובד מול טלפון Yealink אמיתי, בלי לוותר על שמירה אמינה ועל גבולות הגישה.

## ממצאי הבדיקה: מה משתנה ולמה
1. **סוכן מממש אחד במקום שישה, ואחריו סבב QA עצמאי.** הרכיבים צמודים זה לזה (מודל, אחסון, API ו-UI). תיאום חוזים ובעלות קבצים בין 6 סוכנים יעלה יותר מהעבודה עצמה. נשאר בדיקה עצמאית אחת בסוף (subagent או `/code-review`) כדי לשמור על הפרדת QA.
2. **XML לא נכתב לדיסק בכלל.** הוא נוצר בזיכרון מה-snapshot ומוגש דרך `atomic.Pointer`. כך נעלמת בעיית "שתי החלפות קבצים אינן טרנזקציה", מובטח שהטלפון מקבל תמיד מסמך שלם, וב-restart ה-XML נבנה מחדש מה-JSON.
3. **ה-listener של הטלפונים מאזין ב-`0.0.0.0:8088` ולא על IP ספציפי.** שירות שעולה לפני שהרשת מוכנה, או אחרי שה-DHCP שינה כתובת, ייכשל ב-bind. ההגבלה נשארת: חוק חומת אש לתת-הרשת המורשית, ו-handler שמגיש רק `GET/HEAD /phonebook.xml`. הניהול נשאר `127.0.0.1:8765` בלבד.
4. **שירות Windows בלי ספרייה חיצונית.** משתמשים ב-`golang.org/x/sys/windows/svc` וב-`svc/mgr` (ספריות רשמיות של Go, רישיון BSD-3). ההתקנה מתבצעת דרך פקודות של ה-exe עצמו: `install`, `uninstall` ו-`status`. הן רושמות את השירות ב-SCM, מגדירות recovery actions ויוצרות חוק חומת אש ב-`netsh`. מתקין Inno Setup הוא עטיפה דקה ואופציונלית שלא מעכבת את העבודה.
5. **שני פורמטים של Yealink מאחורי הגדרה אחת.** `YealinkIPPhoneDirectory` (DirectoryEntry/Name/Telephone ×3) הוא ברירת המחדל, ו-`YealinkIPPhoneBook` (Menu/Unit Phone1–3) זמין כחלופה. כך שלב A כבר לא חוסם את הפיתוח, ונשאר רק לבחור את הפורמט שמתאים למכשיר.
6. **שלב A מצטמצם לבדיקת עשן מוקדמת.** פקודה `phonebook.exe serve-fixture` מגישה fixture קבוע (עברית, &, +, 0 מוביל). בודקים אותה מול טלפון אמיתי מיד אחרי M2, במקביל להמשך הפיתוח.
7. **מלכודות CSV בעברית שהתוכנית לא כיסתה:**
   - Excel שומר CSV בקידוד Windows-1255. הייבוא מזהה UTF-8 (עם BOM או בלי), ואם הקובץ אינו UTF-8 תקין הוא מפענח cp1255 (`golang.org/x/text`).
   - הייצוא נכתב ב-UTF-8 עם BOM.
   - הגנת נוסחאות: תא שמתחיל ב-`= + - @` מקבל `'` מוביל, והייבוא מסיר `'` מוביל אחד. זה פותר את ההתנגשות עם `+972…`.
   - Excel מוחק אפסים מובילים כשפותחים CSV ישירות. זה ייכנס לתיעוד ולאזהרה במסך ה-preview.
8. **הייבוא משתמש בחוזה הקיים.** `POST /api/import/preview` רק מפענח את הקובץ. ה-UI ממזג או מחליף, והשמירה עוברת דרך `PUT /api/book` הרגיל. אין endpoint נוסף ואין מסלול commit שני.
9. **מה יוצא מגרסה 1:** IPv6 loopback, גרסה ניידת נפרדת (אותו exe עם `run` בקונסול), ייבוא XML כללי (פקודת `import-xml` חד-פעמית רק אם מתקבל קובץ אמיתי) ומסמך החלטות נפרד (README במקומו).
10. **CSRF בפשטות:**
    - בדיקת Host מול `127.0.0.1:port` ו-`localhost:port`, נגד DNS rebinding.
    - Origin חייב להתאים או להיעדר.
    - אסימון אקראי שנוצר בעלייה ומוזרק ל-index.html, ונדרש בכותרת `X-Phonebook-Token` בכל שינוי.
    - בקשות שינוי חייבות `Content-Type: application/json`, ואין CORS.
11. **מגבלות ברירת מחדל עד לאימות במכשיר:** עד 1,000 אנשי קשר, גוף בקשה עד 1MB, CSV עד 2MB, שם עד 64 תווים, מספר עד 32 תווים מהקבוצה `0-9 + * # , ; space - ( )`. כל הערכים ניתנים לשינוי ב-config.
12. **מגבלה לתיעוד:** בשרת RDS עם כמה משתמשים מחוברים, כל משתמש במכונה יכול לגשת ל-loopback.

## מבנה המאגר
```
cmd/phonebook/main.go     פקודות: run | install | uninstall | status | serve-fixture; כניסת שירות
internal/book/            מודל, validation, CSV import/export, זיהוי כפילויות
internal/store/           JSON, כתיבה אטומית, נעילה, revision, גיבויים מתגלגלים (10)
internal/yealink/         encoding/xml לשני הפורמטים
internal/server/          admin + phones handlers, middleware (Host/Origin/token/limits), timeouts
internal/winsvc/          svc handler, mgr install/uninstall, netsh, icacls  (build tag windows)
web/                      index.html, app.js, app.css; RTL, ללא CDN; go:embed
packaging/                README_HE.md (לקוח), SUPPORT_HE.md, phonebook.iss (אופציונלי)
testdata/                 fixtures פיקטיביים
```
התלויות: `golang.org/x/sys` ו-`golang.org/x/text` בלבד, בגרסאות מוצמדות ב-go.mod.

תיקיית הנתונים: `%ProgramData%\NimbusPhonebook\`, ובה `book.json`, `config.json`, `backups\book-r<rev>-<utc>.json` ו-`logs\`. ההרשאות: SYSTEM ו-Administrators בלבד.

## זרימת שמירה (`store.Save`)
1. נעילת mutex. אם expectedRevision שונה מהנוכחית, מוחזר 409.
2. validation. כשל מחזיר 400 עם רשימת שגיאות לפי שורה.
3. נבנה ספר חדש עם `revision+1` ו-`updatedAt`, והאפליקציה מייצרת ממנו XML ובודקת אותו.
4. `book.json` הנוכחי מועתק לגיבוי, עם רוטציה ל-10 גיבויים.
5. נכתב `book.json.tmp` באותה תיקייה, ואחריו `Sync` ו-`Close`.
6. הקובץ מוחלף ב-`MoveFileEx(REPLACE_EXISTING|WRITE_THROUGH)`.
7. רק עכשיו `snapshot.Store({book, xml, etag})`, והתשובה 200.

כשל בכל אחד מהשלבים משאיר את ה-snapshot ואת הקובץ הקודם בלי שינוי ומחזיר 500 עם הודעה ברורה.

**בעלייה:** טוענים `book.json`. אם הוא פגום, נטען הגיבוי התקין האחרון, נרשם log, וה-UI מציג באנר אזהרה. שחזור גיבוי הוא `Save` רגיל של תוכן הגיבוי, ולכן נוצרת revision חדשה.

## API (החוזה המקורי, עם תוספת אחת)
- `GET /api/book`, `PUT /api/book`, `POST /api/import/preview`, `GET /api/export.csv`, `GET /api/backups`, `POST /api/restore {id}`.
- **התוספת:** `GET /api/info` מחזיר את כתובת ה-LAN להעתקה, revision, מועד שמירה אחרון ומצב degraded.
- **הטלפונים:** `GET/HEAD /phonebook.xml` עם `text/xml; charset=utf-8` ו-`ETag`/`Last-Modified`. כל נתיב אחר מחזיר 404.

## אבני דרך (סוכן אחד, ברצף, בדיקות בכל שלב)
- **M0:** התקנת Go דרך `winget install GoLang.Go` (דורשת אישור), `git init`, `go mod init` והצמדת גרסאות.
- **M1:** book, yealink ו-store עם בדיקות:
  - round-trip של עברית, `& < " '`, 0 מוביל ו-`+`;
  - כשל כתיבה מדומה לא משנה את המצב;
  - קריאה מקבילית בזמן שמירה תחת `-race` מחזירה מסמך שלם;
  - 409 ו-400 מוחזרים במקרים הנכונים.
- **M2:** server ו-`run`, עם בדיקות httptest: הטלפונים לא מגיעים ל-`/api`, Host או Origin זרים מקבלים 403, בקשה בלי אסימון מקבלת 403, ומגבלות גודל נאכפות. כאן גם `serve-fixture`, **ובשלב הזה מבקשים ממך לבדוק טלפון אמיתי (שער A).**
- **M3:** ממשק web:
  - חיפוש, עריכה, מחיקה עם אישור, אזהרת `beforeunload`, preview לייבוא, גיבויים ושחזור, העתקת URL;
  - מספרים ב-`dir=ltr`, `textContent` בלבד, ניווט מקלדת;
  - הודעת הצלחה מוצגת רק אחרי 200 מהשרת.
- **M4:** פקודות winsvc (install/uninstall/status), חומת אש, ACL ו-recovery, וכן README_HE, SUPPORT_HE ו-.iss אופציונלי. ההסרה שומרת את ProgramData אלא אם ניתן `--purge`.
- **M5:** ביקורת עצמאית (subagent נפרד או `/code-review`), מטריצת 12 בדיקות הקבלה שמסווגת כל בדיקה כאוטומטית, ידנית ב-Windows או בטלפון אמיתי, ותבנית דוח מכשיר (דגם, קושחה, `features.remote_phonebook.flash_time`, זמן רענון בפועל).

## אימות
- **אוטומטי:** `go vet ./...` ו-`go test -race ./...`.
- **ידני מקומי** (אני מריץ):
  - `phonebook.exe run`;
  - `curl` ל-`127.0.0.1:8765` עובד, ול-`<LAN-IP>:8765` החיבור נדחה;
  - `<LAN-IP>:8088/phonebook.xml` עובד, ו-`/api/book` על אותו פורט מחזיר 404;
  - `curl` עם `Origin: http://evil` מקבל 403;
  - יצירת איש קשר בדפדפן ובדיקה שהוא מופיע ב-XML.
- **שירות** (חלון מנהל, באישורך): `install`, `sc query` מראה Running, `uninstall` משאיר את ProgramData.
- **באחריותך ולא אסמן כעבר:**
  - reboot במכונה נקייה בלי שאף משתמש מחובר;
  - טלפון Yealink אמיתי מציג עברית ומחייג נכון;
  - פיילוט אצל לקוח.

## חסמים ותשומות נדרשות ממך
- אישור להתקנת Go.
- דגם וקושחה של טלפון בדיקה, ואם יש, קובץ XML קיים שעובד. בלעדיהם הפיתוח ממשיך, ושער התאימות נשאר פתוח.
- האם הלקוח מתיר HTTP ב-LAN. HTTPS לא נכלל בגרסה 1 עד שתאימות המכשיר תיבדק.
