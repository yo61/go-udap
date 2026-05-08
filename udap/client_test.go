package udap

import (
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	client, err := newClientWithPort(0, NewNoOpLogger())
	if err != nil {
		t.Fatalf("Failed to create new client: %v", err)
	}
	defer client.Close()

	if client == nil {
		t.Fatal("Client should not be nil")
	}

	if client.conn == nil {
		t.Error("Client connection should not be nil")
	}

	if client.devices == nil {
		t.Error("Client devices map should not be nil")
	}

	if len(client.devices) != 0 {
		t.Error("Client devices map should be empty initially")
	}
}

func TestNewClientWithLogger(t *testing.T) {
	logger := &TestLogger{logs: make([]LogEntry, 0)}
	client, err := newClientWithPort(0, logger)
	if err != nil {
		t.Fatalf("Failed to create new client with logger: %v", err)
	}
	defer client.Close()

	if client == nil {
		t.Fatal("Client should not be nil")
	}

	// Verify client was created with a logger (can't directly compare interface to concrete type)
	if client.logger == nil {
		t.Error("Client logger should not be nil")
	}
}

func TestClientClose(t *testing.T) {
	client, err := newClientWithPort(0, NewNoOpLogger())
	if err != nil {
		t.Fatalf("Failed to create new client: %v", err)
	}

	// Close should not panic or error
	client.Close()

	// Second close should also not panic
	client.Close()
}

func TestClientDeviceManagement(t *testing.T) {
	client, err := newClientWithPort(0, NewNoOpLogger())
	if err != nil {
		t.Fatalf("Failed to create new client: %v", err)
	}
	defer client.Close()

	// Test adding a device
	device := &Device{
		MAC:      "00:04:20:12:34:56",
		IP:       "192.168.1.100",
		Name:     "Test Device",
		Model:    "Squeezebox",
		Firmware: "7.8.0",
		UUID:     "12345678-1234-1234-1234-123456789abc",
		LastSeen: time.Now(),
		Parameters: map[string]string{
			"hostname": "testbox",
		},
	}

	client.devices[device.MAC] = device

	// Test that device was added
	if len(client.devices) != 1 {
		t.Errorf("Expected 1 device, got %d", len(client.devices))
	}

	retrievedDevice, exists := client.devices[device.MAC]
	if !exists {
		t.Error("Device should exist in client devices map")
	}

	if retrievedDevice.MAC != device.MAC {
		t.Errorf("Expected MAC %s, got %s", device.MAC, retrievedDevice.MAC)
	}

	// Test GetDevices
	devicesMap := client.GetDevices()
	if len(devicesMap) != 1 {
		t.Errorf("Expected 1 device from GetDevices, got %d", len(devicesMap))
	}

	if devicesMap[device.MAC] == nil {
		t.Error("Expected device to be in GetDevices map")
	}

	// Test GetDevice
	foundDevice := client.GetDevice(device.MAC)
	if foundDevice == nil {
		t.Error("GetDevice should return the device")
	} else if foundDevice.MAC != device.MAC {
		t.Errorf("Expected MAC %s from GetDevice, got %s", device.MAC, foundDevice.MAC)
	}

	// Test GetDevice with non-existent MAC
	notFoundDevice := client.GetDevice("aa:bb:cc:dd:ee:ff")
	if notFoundDevice != nil {
		t.Error("GetDevice should return nil for non-existent device")
	}
}

func TestPacketCaptureConfig(t *testing.T) {
	config := PacketCaptureConfig{
		Purpose:    "test capture",
		Timeout:    5 * time.Second,
		SourceIP:   "192.168.1.100",
		SourcePort: 17784,
	}

	if config.Timeout != 5*time.Second {
		t.Errorf("Expected timeout 5s, got %v", config.Timeout)
	}

	if config.Purpose != "test capture" {
		t.Errorf("Expected purpose 'test capture', got %s", config.Purpose)
	}

	if config.SourceIP != "192.168.1.100" {
		t.Errorf("Expected SourceIP 192.168.1.100, got %s", config.SourceIP)
	}

	if config.SourcePort != 17784 {
		t.Errorf("Expected source port 17784, got %d", config.SourcePort)
	}
}

func TestPacketCaptureResult(t *testing.T) {
	result := PacketCaptureResult{
		Payload: []byte{0x00, 0x01, 0x02, 0x03},
		SrcIP:   "192.168.1.100",
		SrcPort: 17784,
	}

	if len(result.Payload) != 4 {
		t.Errorf("Expected payload length 4, got %d", len(result.Payload))
	}

	if result.SrcIP != "192.168.1.100" {
		t.Errorf("Expected SrcIP 192.168.1.100, got %s", result.SrcIP)
	}

	if result.SrcPort != 17784 {
		t.Errorf("Expected SrcPort 17784, got %d", result.SrcPort)
	}
}

func TestGetActiveNetworkInterface(t *testing.T) {
	// Test getting active network interface
	iface, err := getActiveNetworkInterface()

	// Should return some interface name or an error
	// We can't test for specific values since it depends on the system
	if err != nil {
		t.Logf("No active network interface found: %v", err)
	} else {
		t.Logf("Active network interface: %s", iface)
	}
}

func TestClientValidation(t *testing.T) {
	client, err := newClientWithPort(0, NewNoOpLogger())
	if err != nil {
		t.Fatalf("Failed to create new client: %v", err)
	}
	defer client.Close()

	if err := client.Validate(); err != nil {
		t.Errorf("Validate on a fresh client should pass, got: %v", err)
	}
}

func TestGetLocalIPsFromClient(t *testing.T) {
	// Test getting local IPs
	localIPs := getLocalIPs()

	// Should return at least localhost
	if localIPs == nil {
		t.Error("getLocalIPs should return a map")
	}

	t.Logf("Local IPs found: %d", len(localIPs))
}

// TestLogger implements Logger for testing
type TestLogger struct {
	logs []LogEntry
}

type LogEntry struct {
	Level   string
	Message string
	Fields  map[string]any
}

func (l *TestLogger) Info(msg string, fields ...any) {
	entry := LogEntry{
		Level:   "INFO",
		Message: msg,
		Fields:  make(map[string]any),
	}

	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			if key, ok := fields[i].(string); ok {
				entry.Fields[key] = fields[i+1]
			}
		}
	}

	l.logs = append(l.logs, entry)
}

func (l *TestLogger) Error(msg string, fields ...any) {
	entry := LogEntry{
		Level:   "ERROR",
		Message: msg,
		Fields:  make(map[string]any),
	}

	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			if key, ok := fields[i].(string); ok {
				entry.Fields[key] = fields[i+1]
			}
		}
	}

	l.logs = append(l.logs, entry)
}

func (l *TestLogger) Debug(msg string, fields ...any) {
	entry := LogEntry{
		Level:   "DEBUG",
		Message: msg,
		Fields:  make(map[string]any),
	}

	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			if key, ok := fields[i].(string); ok {
				entry.Fields[key] = fields[i+1]
			}
		}
	}

	l.logs = append(l.logs, entry)
}

func (l *TestLogger) Warn(msg string, fields ...any) {
	entry := LogEntry{
		Level:   "WARN",
		Message: msg,
		Fields:  make(map[string]any),
	}

	for i := 0; i < len(fields); i += 2 {
		if i+1 < len(fields) {
			if key, ok := fields[i].(string); ok {
				entry.Fields[key] = fields[i+1]
			}
		}
	}

	l.logs = append(l.logs, entry)
}

func (l *TestLogger) SetLevel(level LogLevel) {
	// No-op for test logger
}

func (l *TestLogger) GetLogs() []LogEntry {
	return l.logs
}

func (l *TestLogger) ClearLogs() {
	l.logs = make([]LogEntry, 0)
}

func TestClientWithTestLogger(t *testing.T) {
	logger := &TestLogger{logs: make([]LogEntry, 0)}
	client, err := newClientWithPort(0, logger)
	if err != nil {
		t.Fatalf("Failed to create client with test logger: %v", err)
	}
	defer client.Close()

	// Discard log entries emitted during socket setup so we only assert on
	// the entries this test triggers below.
	logger.ClearLogs()

	// Trigger some logging
	client.logger.Info("Test message", "key", "value")
	client.logger.Error("Test error", "error", "test error")

	logs := logger.GetLogs()
	if len(logs) != 2 {
		t.Errorf("Expected 2 log entries, got %d", len(logs))
	}

	if logs[0].Level != "INFO" {
		t.Errorf("Expected first log level INFO, got %s", logs[0].Level)
	}

	if logs[1].Level != "ERROR" {
		t.Errorf("Expected second log level ERROR, got %s", logs[1].Level)
	}
}
