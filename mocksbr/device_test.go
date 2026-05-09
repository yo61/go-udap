package mocksbr

import "testing"

func TestDeviceFactoryDefaults(t *testing.T) {
	d := newDevice(DeviceConfig{MAC: "00:04:20:00:00:01"})
	if d.workingMemory["lan_ip_mode"] != "1" {
		t.Errorf("expected lan_ip_mode=1 (DHCP) by default, got %q", d.workingMemory["lan_ip_mode"])
	}
	if d.nvram["lan_ip_mode"] != "1" {
		t.Errorf("expected nvram lan_ip_mode=1, got %q", d.nvram["lan_ip_mode"])
	}
}

func TestDeviceSetDataMutatesWorkingMemoryOnly(t *testing.T) {
	d := newDevice(DeviceConfig{MAC: "00:04:20:00:00:01"})
	d.applySet(map[string]string{"hostname": "test-host"})
	if d.workingMemory["hostname"] != "test-host" {
		t.Errorf("working memory not updated: got %q", d.workingMemory["hostname"])
	}
	if d.nvram["hostname"] == "test-host" {
		t.Errorf("nvram should not be modified by applySet")
	}
}

func TestDeviceSaveCopiesWorkingToNVRAM(t *testing.T) {
	d := newDevice(DeviceConfig{MAC: "00:04:20:00:00:01"})
	d.applySet(map[string]string{"hostname": "saved-host"})
	d.applySave()
	if d.nvram["hostname"] != "saved-host" {
		t.Errorf("expected nvram hostname=saved-host after save, got %q", d.nvram["hostname"])
	}
}

func TestDeviceResetReloadsNVRAMIntoWorkingMemory(t *testing.T) {
	d := newDevice(DeviceConfig{MAC: "00:04:20:00:00:01"})
	d.applySet(map[string]string{"hostname": "saved-host"})
	d.applySave()
	d.applySet(map[string]string{"hostname": "unsaved-host"})
	d.applyReset()
	if d.workingMemory["hostname"] != "saved-host" {
		t.Errorf("expected hostname to revert to saved-host after reset, got %q", d.workingMemory["hostname"])
	}
}

func TestDeviceStateInitVsConfigured(t *testing.T) {
	d := newDevice(DeviceConfig{MAC: "00:04:20:00:00:01"})
	if got := d.state(); got != "init" {
		t.Errorf("factory device state: got %q, want %q", got, "init")
	}
	d.applySet(map[string]string{"hostname": "configured"})
	if got := d.state(); got != "wait_slimserver" {
		t.Errorf("configured device state: got %q, want %q", got, "wait_slimserver")
	}
}

func TestAutoConfigDeterministicByIndex(t *testing.T) {
	c1 := autoConfig(1)
	c1again := autoConfig(1)
	c2 := autoConfig(2)

	if c1.MAC != "00:04:20:00:00:01" {
		t.Errorf("idx=1 MAC: got %q, want 00:04:20:00:00:01", c1.MAC)
	}
	if c2.MAC != "00:04:20:00:00:02" {
		t.Errorf("idx=2 MAC: got %q, want 00:04:20:00:00:02", c2.MAC)
	}
	if c1.UUID != "mock-sbr-001" {
		t.Errorf("idx=1 UUID: got %q, want mock-sbr-001", c1.UUID)
	}
	if c1.MAC != c1again.MAC || c1.UUID != c1again.UUID {
		t.Errorf("autoConfig(1) returned different values on repeat calls")
	}
}

func TestAutoConfigIndex255(t *testing.T) {
	c := autoConfig(255)
	if c.MAC != "00:04:20:00:00:ff" {
		t.Errorf("idx=255 MAC: got %q, want 00:04:20:00:00:ff", c.MAC)
	}
}
