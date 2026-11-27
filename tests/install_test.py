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
    (root / 'XrayR.sh').write_text((script.parent / 'XrayR.sh').read_text())
    checksum = hashlib.sha256(archive.read_bytes()).hexdigest()
    (root / 'SHA256SUMS').write_text(checksum + '  ' + archive.name + '\n')
    harness = r'''
source "$1"
fixture="$2"
install_dir="$fixture/bin"
config_dir="$fixture/config"
unit_dir="$fixture/units"
bin_link="$fixture/links/XrayR"
legacy_bin_dir="$fixture/legacy-bin"
sleep() { :; }
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
    assert (root / 'links/xrayr').is_symlink()
    assert (root / 'links/XrayR').resolve() == (root / 'bin/manager.sh').resolve()
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
    # A fwcloud 0.9.0 installation: preserve all local configuration while
    # migrating the one incompatible CLI flag and both old manager paths.
    legacy = root / 'legacy-bin'
    legacy.mkdir()
    (legacy / 'XrayR').write_text('#!/bin/bash\n# XrayR legacy manager\n')
    if not (legacy / 'xrayr').exists():  # macOS may use a case-insensitive volume
        (legacy / 'xrayr').symlink_to(legacy / 'XrayR')
    old_unit = '[Service]\nWorkingDirectory=/usr/local/XrayR/\nExecStart=/usr/local/XrayR/XrayR -config /etc/XrayR/config.yml\nRestartSec=10\n'
    unit.write_text(old_unit)
    (root / 'config/cert').mkdir()
    (root / 'config/cert/example.key').write_text('fictional-test-key\n')
    (root / 'config/route.json').write_text('{"rules": []}\n')
    active_harness = harness.replace('if [[ "$1" == is-active ]]; then return 1; fi;', 'if [[ "$1" == is-active ]]; then return 0; fi;')
    upgraded = subprocess.run(['bash', '-c', active_harness, 'test', str(script), str(root)], capture_output=True, text=True)
    assert upgraded.returncode == 0, upgraded.stderr
    assert unit.read_text() == old_unit.replace(' -config ', ' --config ')
    assert config.read_text() == 'existing: keep\n'
    assert (root / 'config/cert/example.key').read_text() == 'fictional-test-key\n'
    assert (root / 'config/route.json').read_text() == '{"rules": []}\n'
    assert (legacy / 'XrayR').resolve() == (root / 'bin/manager.sh').resolve()
    assert (legacy / 'xrayr').resolve() == (root / 'bin/manager.sh').resolve()
    backups = list((root / 'bin').glob('upgrade-backup.*'))
    assert any((p / 'XrayR.service').read_text() == old_unit for p in backups if (p / 'XrayR.service').exists())
    assert all(p.stat().st_mode & 0o077 == 0 for p in backups)
    # A delayed exit after systemctl restart succeeds must also roll back.
    unit.write_text(old_unit)
    (root / 'bin/XrayR').write_text('previous program\n')
    delayed = harness.replace('if [[ "$1" == is-active ]]; then return 1; fi;', 'if [[ "$1" == is-active ]]; then [[ ! -e "$fixture/started" ]]; return; fi; if [[ "$1" == restart ]]; then touch "$fixture/started"; fi;')
    failed = subprocess.run(['bash', '-c', delayed, 'test', str(script), str(root)], capture_output=True, text=True)
    assert failed.returncode != 0
    assert (root / 'bin/XrayR').read_text() == 'previous program\n'
    assert unit.read_text() == old_unit
    rollback_harness = harness.replace('if [[ "$1" == is-active ]]; then return 1; fi;', 'if [[ "$1" == is-active ]]; then return 0; fi; if [[ "$1" == restart && ! -e "$fixture/restart-failed" ]]; then touch "$fixture/restart-failed"; return 1; fi;')
    rollback = subprocess.run(['bash', '-c', rollback_harness, 'test', str(script), str(root)], capture_output=True, text=True)
    assert rollback.returncode != 0
    assert (root / 'bin/XrayR').read_text() == 'previous program\n'
    (root / 'SHA256SUMS').write_text('0' * 64 + '  ' + archive.name + '\n')
    before = (root / 'bin/XrayR').read_bytes()
    rejected = run()
    assert rejected.returncode != 0
    assert (root / 'bin/XrayR').read_bytes() == before
    one_line = subprocess.run(['bash', '-c', 'bash <(cat "$1") --help', 'test', str(script)], capture_output=True, text=True)
    assert one_line.returncode == 0 and '--version 1.0.0' in one_line.stdout
print('Installer checks passed: one-line invocation, legacy upgrade, local backup, config/cert preservation, service migration, rollback and checksum rejection.')
