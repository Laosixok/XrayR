"""Offline installer checks: all downloads and service operations are mocked."""
import hashlib
import pathlib
import subprocess
import tempfile
import zipfile

script = pathlib.Path(__file__).resolve().parents[1] / 'install.sh'
with tempfile.TemporaryDirectory(prefix='xrayr-install-test-') as temp:
    root = pathlib.Path(temp)
    archive = root / 'XrayR-linux-amd64.zip'
    with zipfile.ZipFile(archive, 'w') as z:
        z.writestr('XrayR', '#!/bin/sh\necho "XrayR 1.0.0"\n')
        z.writestr('config_sspanel_metron_ws_tls.yml.example', 'template: ws-tls\n')
        z.writestr('config_sspanel_metron_protocols.yml.example', 'template: protocols\n')
    checksum = hashlib.sha256(archive.read_bytes()).hexdigest()
    (root / 'SHA256SUMS').write_text(checksum + '  ' + archive.name + '\n')
    harness = r'''
source "$1"
fixture="$2"
install_dir="$fixture/bin"
config_dir="$fixture/config"
unit_dir="$fixture/units"
bin_link="$fixture/links/XrayR"
version=1.0.0
asset=XrayR-linux-amd64.zip
template=protocols
curl() { local url="$4"; cp "$fixture/${url##*/}" "$6"; }
systemctl() { if [[ "$1" == is-active ]]; then return 1; fi; echo "$*" >> "$fixture/service-calls"; }
install_release
'''
    def run():
        return subprocess.run(['bash', '-c', harness, 'test', str(script), str(root)], capture_output=True, text=True)
    first = run()
    assert first.returncode == 0, first.stderr
    config = root / 'config/config.yml'
    assert config.exists(), (first.stdout, first.stderr, list(root.rglob("*")))
    assert config.read_text() == 'template: protocols\n'
    config.write_text('existing: keep\n')
    unit = root / 'units/XrayR.service'
    unit.write_text('existing unit\n')
    second = run()
    assert second.returncode == 0, second.stderr
    assert config.read_text() == 'existing: keep\n'
    assert unit.read_text() == 'existing unit\n'
    assert 'restart' not in (root / 'service-calls').read_text()
    (root / 'bin/XrayR').write_text('previous program\n')
    rollback_harness = harness.replace('if [[ "$1" == is-active ]]; then return 1; fi;', 'if [[ "$1" == is-active ]]; then return 0; fi; if [[ "$1" == restart && ! -e "$fixture/restart-failed" ]]; then touch "$fixture/restart-failed"; return 1; fi;')
    rollback = subprocess.run(['bash', '-c', rollback_harness, 'test', str(script), str(root)], capture_output=True, text=True)
    assert rollback.returncode != 0
    assert (root / 'bin/XrayR').read_text() == 'previous program\n'
    (root / 'SHA256SUMS').write_text('0' * 64 + '  ' + archive.name + '\n')
    before = (root / 'bin/XrayR').read_bytes()
    rejected = run()
    assert rejected.returncode != 0
    assert (root / 'bin/XrayR').read_bytes() == before
print('Installer checks passed: install, config/unit preservation, idle service, checksum rejection.')
