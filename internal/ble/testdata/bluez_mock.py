import dbus
import dbus.service
from dbus.mainloop.glib import DBusGMainLoop
from gi.repository import GLib
import json
import os
import subprocess
import sys

BLUEZ = "org.bluez"
MANAGER = "org.freedesktop.DBus.ObjectManager"
PROPS = "org.freedesktop.DBus.Properties"
ADAPTER_IFACE = "org.bluez.Adapter1"
DEVICE_IFACE = "org.bluez.Device1"
ADAPTER = "/org/bluez/hci0"
DEVICE = ADAPTER + "/dev_00_11_22_33_44_55"
UUID = "0000fd3d-0000-1000-8000-00805f9b34fb"

DBusGMainLoop(set_as_default=True)
bus = dbus.SessionBus()
name = dbus.service.BusName(BLUEZ, bus=bus)
loop = GLib.MainLoop()

class Manager(dbus.service.Object):
    @dbus.service.method(MANAGER, out_signature="a{oa{sa{sv}}}")
    def GetManagedObjects(self):
        return {
            ADAPTER: {ADAPTER_IFACE: {"Powered": dbus.Boolean(True)}},
            DEVICE: {DEVICE_IFACE: {
                "Address": "00:11:22:33:44:55", "Adapter": dbus.ObjectPath(ADAPTER),
                "RSSI": dbus.Int16(-99),
                "ServiceData": dbus.Dictionary({UUID: dbus.ByteArray(b"stale")}, signature="sv"),
            }},
        }

class Device(dbus.service.Object):
    @dbus.service.signal(PROPS, signature="sa{sv}as")
    def PropertiesChanged(self, interface, values, invalidated):
        pass

class Adapter(dbus.service.Object):
    started = 0
    stopped = 0
    fail = False

    @dbus.service.method(ADAPTER_IFACE, in_signature="a{sv}")
    def SetDiscoveryFilter(self, filters):
        assert dict(filters) == {"Transport": "le", "DuplicateData": True}

    @dbus.service.method(ADAPTER_IFACE)
    def StartDiscovery(self):
        self.started += 1
        GLib.idle_add(self.publish)

    @dbus.service.method(ADAPTER_IFACE)
    def StopDiscovery(self):
        self.stopped += 1

    @dbus.service.signal(PROPS, signature="sa{sv}as")
    def PropertiesChanged(self, interface, values, invalidated):
        pass

    def publish(self):
        if self.fail:
            self.PropertiesChanged(ADAPTER_IFACE, {"Powered": dbus.Boolean(False)}, [])
            return False
        device.PropertiesChanged(DEVICE_IFACE, {"ManufacturerData": dbus.Dictionary({dbus.UInt16(0x0969): dbus.ByteArray(b"\x01\x02")}, signature="qv")}, [])
        for _ in range(2):
            device.PropertiesChanged(DEVICE_IFACE, {"ServiceData": dbus.Dictionary({UUID: dbus.ByteArray(b"T\x01")}, signature="sv"), "RSSI": dbus.Int16(-52)}, [])
        return False

manager = Manager(name, "/")
adapter = Adapter(name, ADAPTER)
device = Device(name, DEVICE)
env = dict(os.environ, DBUS_SYSTEM_BUS_ADDRESS=os.environ["DBUS_SESSION_BUS_ADDRESS"])

def run_probe(*args, cancel=False):
    process = subprocess.Popen([sys.argv[1], "--duration", "250ms", *args], stdout=subprocess.PIPE, stderr=subprocess.PIPE, env=env, text=True)
    if cancel:
        expected_starts = adapter.started + 1

        def terminate_after_start():
            if process.poll() is not None:
                return False
            if adapter.started < expected_starts:
                return True
            process.terminate()
            return False

        GLib.timeout_add(10, terminate_after_start)

    def poll():
        if process.poll() is None:
            return True
        loop.quit()
        return False
    GLib.timeout_add(10, poll)
    loop.run()
    return process.returncode, *process.communicate(timeout=2)

code, stdout, stderr = run_probe("--mac", "00:11:22:33:44:55")
assert code == 0, (code, stderr)
rows = [json.loads(line) for line in stdout.splitlines()]
assert len(rows) == 3, rows
assert "rssi" not in rows[0] and "serviceData" not in rows[0], rows[0]
assert rows[0]["switchbotFields"] == [{"adType": 255, "id": "0969", "payloadHex": "0102"}], rows[0]
for row in rows[1:]:
    assert row["rssiChanged"] and row["rssi"] == -52 and "manufacturerData" not in row, row
    assert row["switchbotFields"] == [{"adType": 22, "id": "fd3d", "payloadHex": "5401"}], row
assert (adapter.started, adapter.stopped) == (1, 1)

adapter.fail = True
code, stdout, stderr = run_probe()
assert code == 1 and "powered off" in stderr, (code, stdout, stderr)
assert (adapter.started, adapter.stopped) == (2, 2)
code, stdout, stderr = run_probe("--adapter", "hci9")
assert code == 1 and "not found" in stderr and not stdout, (code, stdout, stderr)
adapter.fail = False
code, stdout, stderr = run_probe("--duration", "3s", cancel=True)
assert code == 130 and "context canceled" in stderr, (code, stdout, stderr)
assert (adapter.started, adapter.stopped) == (3, 3)
print("D-Bus integration passed: LE filter, start/stop, ordered duplicate notifications, stale snapshots, power-off, missing adapter and cancellation")
