"""Static packaging invariants supplement (do not replace) native Windows acceptance."""
from pathlib import Path
import unittest
root=Path(__file__).resolve().parents[1]
class InstallerContract(unittest.TestCase):
 def setUp(self): self.script=(root/'packaging/phonebook.nsi').read_text(encoding='utf-8-sig')
 def test_elevates_and_checks_x64(self):
  self.assertIn('RequestExecutionLevel admin',self.script)
  self.assertIn('${RunningX64}',self.script)
 def test_fixed_protected_destination(self):
  self.assertIn('StrCmp $INSTDIR "$PROGRAMFILES64\\NimbusPhonebook" FixedLocationOK',self.script)
  self.assertIn('setup-harden',self.script)
 def test_runtime_readiness_before_success(self):
  self.assertLess(self.script.index('setup-finish'),self.script.index('WriteUninstaller'))
  self.assertIn('Call RollbackInstall',self.script)
 def test_uninstall_stops_service_before_deleting_binary(self):
  uninstall=self.script.split('Section "Uninstall"',1)[1]
  self.assertLess(uninstall.index('setup-remove'),uninstall.index('Delete "$INSTDIR\\phonebook.exe"'))
  self.assertIn('Abort',uninstall)
 def test_preserves_data_and_network_scope(self):
  self.assertNotIn('--purge',self.script)
  self.assertNotIn('RMDir /r',self.script)
  self.assertIn('setup-check',self.script)
  self.assertIn('/SUBNET=',self.script)
 def test_offline_payload_and_shortcuts(self):
  self.assertIn('File /oname=phonebook-helper.exe',self.script)
  self.assertIn('http://127.0.0.1:$AdminPort',self.script)
  self.assertNotIn('inetc::',self.script)
  self.assertNotIn('NSISdl::',self.script)
if __name__=='__main__': unittest.main()
