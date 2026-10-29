Unicode true
RequestExecutionLevel admin
!include "MUI2.nsh"
!include "LogicLib.nsh"
!include "FileFunc.nsh"
!include "nsDialogs.nsh"
!include "x64.nsh"

!ifndef VERSION
 !define VERSION "1.1.0"
!endif
Name "Nimbus Phonebook"
OutFile "../bin/NimbusPhonebook-Setup-${VERSION}.exe"
InstallDir "$PROGRAMFILES64\NimbusPhonebook"
InstallDirRegKey HKLM "Software\NimbusPhonebook" "InstallDir"
BrandingText "Nimbus Phonebook"
SetCompressor /SOLID lzma
SetCompressorDictSize 32
ShowInstDetails show
ShowUninstDetails show
VIProductVersion "1.1.0.0"
VIAddVersionKey "ProductName" "Nimbus Phonebook"
VIAddVersionKey "FileDescription" "Nimbus Phonebook offline installer"
VIAddVersionKey "FileVersion" "${VERSION}"
VIAddVersionKey "LegalCopyright" "Nimbus"

Var Subnet
Var AdminPort
Var NetworkControl
Var HadService
Var HadBinary
Var ExitCode
Var CommandOutput
Var SetupDialog

!define MUI_ABORTWARNING
!define MUI_WELCOMEPAGE_TITLE "ברוכים הבאים ל־Nimbus Phonebook"
!define MUI_WELCOMEPAGE_TEXT "ההתקנה תכין את ספר הטלפונים במחשב זה ותפעיל אותו ברקע באופן אוטומטי.$\r$\n$\r$\nבסיום תוכלו לפתוח את קיצור הדרך ולהתחיל לנהל אנשי קשר. התמיכה תחבר את הטלפונים לכתובת הספר פעם אחת.$\r$\n$\r$\nאין צורך בהתקנת תוכנות נוספות."
!insertmacro MUI_PAGE_WELCOME
Page custom NetworkPage NetworkLeave
!insertmacro MUI_PAGE_INSTFILES
!define MUI_FINISHPAGE_TITLE "ספר הטלפונים מוכן"
!define MUI_FINISHPAGE_TEXT "ההתקנה הושלמה והשירות פועל.$\r$\n$\r$\nפתחו את קיצור הדרך 'Nimbus - ספר הטלפונים' בשולחן העבודה.$\r$\n$\r$\nהתמיכה תגדיר פעם אחת את כתובת הספר בטלפונים. המחשב צריך להישאר דלוק וזמין ברשת."
!insertmacro MUI_PAGE_FINISH
!insertmacro MUI_UNPAGE_CONFIRM
!insertmacro MUI_UNPAGE_INSTFILES
!insertmacro MUI_LANGUAGE "Hebrew"

Function .onInit
 ${IfNot} ${RunningX64}
  MessageBox MB_ICONSTOP "נדרשת מערכת Windows בגרסת 64 ביט."
  SetErrorLevel 1
  Quit
 ${EndIf}
 SetRegView 64
 SetShellVarContext all
 ; /D= and legacy registry paths must not redirect a LocalSystem executable into a writable directory.
 StrCmp $INSTDIR "$PROGRAMFILES64\NimbusPhonebook" FixedLocationOK
  MessageBox MB_ICONSTOP "המתקין תומך רק בתיקיית NimbusPhonebook תחת Program Files. פנו לתמיכה." /SD IDOK
  SetErrorLevel 1
  Quit
 FixedLocationOK:
 InitPluginsDir
 SetOutPath "$PLUGINSDIR"
 File /oname=phonebook-helper.exe "../bin/phonebook.exe"
 nsExec::ExecToStack '"$PLUGINSDIR\phonebook-helper.exe" setup-info --output "$PLUGINSDIR\setup.ini"'
 Pop $ExitCode
 Pop $CommandOutput
 ${If} $ExitCode != 0
  MessageBox MB_ICONSTOP "לא ניתן לקרוא את הגדרות ההתקנה. פנו לתמיכה.$\r$\n$CommandOutput" /SD IDOK
  SetErrorLevel 1
  Quit
 ${EndIf}
 ReadINIStr $AdminPort "$PLUGINSDIR\setup.ini" "Setup" "AdminPort"
 ReadINIStr $HadService "$PLUGINSDIR\setup.ini" "Setup" "ServiceExists"
 ${GetParameters} $0
 ClearErrors
 ${GetOptions} $0 "/SUBNET=" $Subnet
 ${If} ${Errors}
  ReadINIStr $Subnet "$PLUGINSDIR\setup.ini" "Setup" "Subnet"
 ${EndIf}
 IfFileExists "$INSTDIR\phonebook-previous.exe" 0 NoPendingRecovery
  MessageBox MB_ICONSTOP "נמצא עותק שחזור מהתקנה קודמת שלא הושלמה. פנו לתמיכה לפני התקנה נוספת." /SD IDOK
  SetErrorLevel 1
  Quit
 NoPendingRecovery:
 StrCpy $HadBinary 0
 IfFileExists "$INSTDIR\phonebook.exe" 0 +2
  StrCpy $HadBinary 1
FunctionEnd

Function NetworkPage
 !insertmacro MUI_HEADER_TEXT "חיבור לרשת הטלפונים" "בחרו פעם אחת את הרשת שאושרה על ידי התמיכה."
 nsDialogs::Create 1018
 Pop $SetupDialog
 ${If} $SetupDialog == error
  Abort
 ${EndIf}
 nsDialogs::SetRTL 1
 ${NSD_CreateLabel} 0 0 100% 30u "הרשת המורשית לגשת לספר הטלפונים:$\r$\nאם מופיעות כמה רשתות, בחרו את רשת הטלפונים בעזרת התמיכה."
 Pop $0
 ${NSD_CreateComboBox} 0 38u 100% 110u ""
 Pop $NetworkControl
 ReadINIStr $1 "$PLUGINSDIR\setup.ini" "Setup" "Count"
 StrCpy $2 0
 ${DoWhile} $2 < $1
  ReadINIStr $3 "$PLUGINSDIR\setup.ini" "Setup" "Network$2"
  ${NSD_CB_AddString} $NetworkControl $3
  IntOp $2 $2 + 1
 ${Loop}
 ${NSD_SetText} $NetworkControl $Subnet
 ${NSD_CreateLabel} 0 75u 100% 42u "דוגמה: 192.168.1.0/24.$\r$\nזיהוי רשת הוא הצעה בלבד; יש להשתמש ברשת שאושרה. התקנה קיימת תשמור על הרשת שכבר הוגדרה."
 Pop $0
 nsDialogs::Show
FunctionEnd

Function SecureInstallDirectory
 nsExec::ExecToStack '"$PLUGINSDIR\phonebook-helper.exe" setup-harden --install-dir "$INSTDIR"'
 Pop $ExitCode
 Pop $CommandOutput
FunctionEnd

Function CheckNetwork
 nsExec::ExecToStack '"$PLUGINSDIR\phonebook-helper.exe" setup-check --install-dir "$INSTDIR" --subnet "$Subnet"'
 Pop $ExitCode
 Pop $CommandOutput
FunctionEnd
Function NetworkLeave
 ${NSD_GetText} $NetworkControl $Subnet
 Call CheckNetwork
 ${If} $ExitCode != 0
  MessageBox MB_ICONSTOP "לא ניתן להמשיך בהתקנה. בדקו את הרשת או פנו לתמיכה.$\r$\n$CommandOutput" /SD IDOK
  Abort
 ${EndIf}
FunctionEnd

; Called only after prepare/copy/finish failed. Never delete data or backups.
Function RollbackInstall
 nsExec::ExecToStack '"$PLUGINSDIR\phonebook-helper.exe" setup-remove --install-dir "$INSTDIR"'
 Pop $ExitCode
 Pop $CommandOutput
 ${If} $ExitCode != 0
  MessageBox MB_ICONSTOP "השחזור לא הושלם. לא נמחקו נתוני הספר. אל תפעילו התקנה נוספת; פנו לתמיכה.$\r$\n$CommandOutput" /SD IDOK
  Return
 ${EndIf}
 ${If} $HadBinary == 1
  ClearErrors
  CopyFiles /SILENT "$INSTDIR\phonebook-previous.exe" "$INSTDIR\phonebook.exe"
  ${If} ${Errors}
   MessageBox MB_ICONSTOP "לא ניתן לשחזר את קובץ התוכנה הקודם. העותק נשמר ב־phonebook-previous.exe. פנו לתמיכה." /SD IDOK
   Return
  ${EndIf}
  ${If} $HadService == 1
   nsExec::ExecToStack '"$INSTDIR\phonebook.exe" install --subnet "$Subnet"'
   Pop $ExitCode
   Pop $CommandOutput
   ${If} $ExitCode != 0
    MessageBox MB_ICONSTOP "הקובץ הקודם שוחזר אך השירות לא הופעל. פנו לתמיכה.$\r$\n$CommandOutput" /SD IDOK
    Return
   ${EndIf}
   nsExec::ExecToStack '"$PLUGINSDIR\phonebook-helper.exe" setup-ready --install-dir "$INSTDIR" --subnet "$Subnet"'
   Pop $ExitCode
   Pop $CommandOutput
   ${If} $ExitCode != 0
    MessageBox MB_ICONSTOP "השירות הקודם לא הגיע למצב תקין. עותק השחזור נשמר; פנו לתמיכה.$\r$\n$CommandOutput" /SD IDOK
    Return
   ${EndIf}
  ${EndIf}
  Delete "$INSTDIR\phonebook-previous.exe"
 ${Else}
  Delete "$INSTDIR\phonebook.exe"
  RMDir "$INSTDIR"
 ${EndIf}
FunctionEnd

Section "Nimbus Phonebook" MainSection
 Call CheckNetwork
 ${If} $ExitCode != 0
  MessageBox MB_ICONSTOP "בדיקת ההתקנה נכשלה. לא הוחלפו קבצים.$\r$\n$CommandOutput" /SD IDOK
  SetErrorLevel 1
  Abort
 ${EndIf}
 ${If} $HadService == 1
 ${AndIf} $HadBinary != 1
  MessageBox MB_ICONSTOP "השירות הקיים אינו תואם לתיקיית ההתקנה. פנו לתמיכה לפני שדרוג." /SD IDOK
  SetErrorLevel 1
  Abort
 ${EndIf}
 CreateDirectory "$INSTDIR"
 Call SecureInstallDirectory
 ${If} $ExitCode != 0
  MessageBox MB_ICONSTOP "לא ניתן לאבטח את תיקיית התוכנה. ההתקנה נעצרה לפני שינוי השירות.$\r$\n$CommandOutput" /SD IDOK
  SetErrorLevel 1
  Abort
 ${EndIf}
 ${If} $HadBinary == 1
  ClearErrors
  CopyFiles /SILENT "$INSTDIR\phonebook.exe" "$INSTDIR\phonebook-previous.exe"
  ${If} ${Errors}
   MessageBox MB_ICONSTOP "לא ניתן לגבות את קובץ התוכנה הקיים. ההתקנה נעצרה לפני שינוי השירות." /SD IDOK
   SetErrorLevel 1
   Abort
  ${EndIf}
 ${EndIf}
 nsExec::ExecToStack '"$PLUGINSDIR\phonebook-helper.exe" setup-prepare --install-dir "$INSTDIR" --subnet "$Subnet"'
 Pop $ExitCode
 Pop $CommandOutput
 ${If} $ExitCode != 0
  DetailPrint "$CommandOutput"
  Call RollbackInstall
  MessageBox MB_ICONSTOP "הכנת השירות נכשלה. פנו לתמיכה ובדקו את יומן ההתקנה." /SD IDOK
  SetErrorLevel 1
  Abort
 ${EndIf}
 ClearErrors
 CopyFiles /SILENT "$PLUGINSDIR\phonebook-helper.exe" "$INSTDIR\phonebook.exe"
 ${If} ${Errors}
  Call RollbackInstall
  MessageBox MB_ICONSTOP "העתקת התוכנה נכשלה. פנו לתמיכה." /SD IDOK
  SetErrorLevel 1
  Abort
 ${EndIf}
 Call SecureInstallDirectory
 ${If} $ExitCode != 0
  Call RollbackInstall
  MessageBox MB_ICONSTOP "לא ניתן לאבטח את קובץ התוכנה. פנו לתמיכה." /SD IDOK
  SetErrorLevel 1
  Abort
 ${EndIf}
 nsExec::ExecToStack '"$INSTDIR\phonebook.exe" setup-finish --install-dir "$INSTDIR" --subnet "$Subnet"'
 Pop $ExitCode
 Pop $CommandOutput
 ${If} $ExitCode != 0
  DetailPrint "$CommandOutput"
  Call RollbackInstall
  MessageBox MB_ICONSTOP "השירות לא הגיע למצב תקין. ההתקנה לא הושלמה; פנו לתמיכה." /SD IDOK
  SetErrorLevel 1
  Abort
 ${EndIf}
 SetOutPath "$INSTDIR"
 File "USER_GUIDE_HE.html"
 File "INSTALL_SUPPORT_HE.html"
 File "THIRD_PARTY_NOTICES.txt"
 WriteINIStr "$INSTDIR\Nimbus Phonebook.url" "InternetShortcut" "URL" "http://127.0.0.1:$AdminPort"
 WriteUninstaller "$INSTDIR\Uninstall.exe"
 Call SecureInstallDirectory
 ${If} $ExitCode != 0
  Call RollbackInstall
  MessageBox MB_ICONSTOP "לא ניתן להשלים את הרשאות קובצי ההתקנה. פנו לתמיכה." /SD IDOK
  SetErrorLevel 1
  Abort
 ${EndIf}
 CreateDirectory "$SMPROGRAMS\Nimbus Phonebook"
 CreateShortcut "$SMPROGRAMS\Nimbus Phonebook\Nimbus - ספר הטלפונים.lnk" "$INSTDIR\Nimbus Phonebook.url"
 CreateShortcut "$SMPROGRAMS\Nimbus Phonebook\מדריך שימוש.lnk" "$INSTDIR\USER_GUIDE_HE.html"
 CreateShortcut "$DESKTOP\Nimbus - ספר הטלפונים.lnk" "$INSTDIR\Nimbus Phonebook.url"
 WriteRegStr HKLM "Software\NimbusPhonebook" "InstallDir" "$INSTDIR"
 WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\NimbusPhonebook" "DisplayName" "Nimbus Phonebook"
 WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\NimbusPhonebook" "DisplayVersion" "${VERSION}"
 WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\NimbusPhonebook" "UninstallString" '$\"$INSTDIR\Uninstall.exe$\"'
 WriteRegStr HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\NimbusPhonebook" "InstallLocation" "$INSTDIR"
 WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\NimbusPhonebook" "NoModify" 1
 WriteRegDWORD HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\NimbusPhonebook" "NoRepair" 1
 Delete "$INSTDIR\phonebook-previous.exe"
 SetErrorLevel 0
SectionEnd

Function un.onInit
 SetRegView 64
 SetShellVarContext all
FunctionEnd
Section "Uninstall"
 nsExec::ExecToStack '"$INSTDIR\phonebook.exe" setup-remove --install-dir "$INSTDIR"'
 Pop $ExitCode
 Pop $CommandOutput
 ${If} $ExitCode != 0
  MessageBox MB_ICONSTOP "לא ניתן לעצור ולהסיר את השירות. הקבצים והנתונים נשמרו. פנו לתמיכה.$\r$\n$CommandOutput" /SD IDOK
  SetErrorLevel 1
  Abort
 ${EndIf}
 Delete "$DESKTOP\Nimbus - ספר הטלפונים.lnk"
 Delete "$SMPROGRAMS\Nimbus Phonebook\Nimbus - ספר הטלפונים.lnk"
 Delete "$SMPROGRAMS\Nimbus Phonebook\מדריך שימוש.lnk"
 RMDir "$SMPROGRAMS\Nimbus Phonebook"
 Delete "$INSTDIR\phonebook.exe"
 Delete "$INSTDIR\Nimbus Phonebook.url"
 Delete "$INSTDIR\USER_GUIDE_HE.html"
 Delete "$INSTDIR\INSTALL_SUPPORT_HE.html"
 Delete "$INSTDIR\THIRD_PARTY_NOTICES.txt"
 Delete "$INSTDIR\Uninstall.exe"
 RMDir "$INSTDIR"
 DeleteRegKey HKLM "Software\NimbusPhonebook"
 DeleteRegKey HKLM "Software\Microsoft\Windows\CurrentVersion\Uninstall\NimbusPhonebook"
 SetErrorLevel 0
SectionEnd
