# קובץ התקנה להפצה

הקובץ להפצה הוא **NimbusPhonebook-Setup-1.1.0.exe**. פתחו אותו ב־GitHub ולחצו על Download raw file, או השתמשו בקישור ההורדה:

[הורדת מתקין Windows](https://github.com/isaaceranlevy/PhonebookManager/raw/refs/heads/main/releases/NimbusPhonebook-Setup-1.1.0.exe)

התמיכה מורידה את הקובץ ושולחת אותו ללקוח; הלקוח אינו צריך גישה למאגר GitHub.

- [הוראות התקנה לתמיכה — PDF](../packaging/INSTALL_SUPPORT_HE.pdf)
- [הוראות שימוש ללקוח — PDF](../packaging/USER_GUIDE_HE.pdf)
- [תוצאות בדיקה ומגבלות](../docs/INSTALLER_VERIFICATION.md)

המתקין מיועד ל־Windows 64 ביט ואינו דורש Go, Git או הורדות בזמן ההתקנה. הוא דורש אישור מנהל ובחירת רשת מאושרת, יוצר שירות אוטומטי וקיצור דרך ושומר על הנתונים בהסרה ושדרוג.

**לפני הפצה רחבה:** יש לבצע פיילוט התקנה/שדרוג/הסרה במחשב Windows. המתקין נבנה ונבדק כאן ברמת הקוד והאריזה, אך לא הורץ על Windows. הקובץ אינו חתום דיגיטלית; יש להתאים למדיניות הארגון.

בדיקת שלמות אופציונלית לתמיכה: `Get-FileHash .\NimbusPhonebook-Setup-1.1.0.exe -Algorithm SHA256` והשוואה ל־`SHA256SUMS.txt`.
