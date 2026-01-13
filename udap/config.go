package udap

import (
	"context"
	"fmt"
	"maps"
	"net"
	"time"
)

// GetDeviceConfig retrieves configuration from a device
func (c *Client) GetDeviceConfig(device *Device, params []string) (map[string]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return c.GetDeviceConfigWithContext(ctx, device, params)
}

// GetDeviceConfigWithContext retrieves configuration from a device with context
func (c *Client) GetDeviceConfigWithContext(ctx context.Context, device *Device, params []string) (map[string]string, error) {
	config := make(map[string]string)

	// Create GetData packet
	packet := c.CreateGetDataPacket(device, params)

	// Send to device
	var destAddr *net.UDPAddr
	var err error

	if device.IP == "0.0.0.0" {
		// Device in bootstrap mode - use broadcast
		destAddr, err = net.ResolveUDPAddr("udp", fmt.Sprintf("255.255.255.255:%d", Port))
	} else {
		destAddr, err = net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", device.IP, Port))
	}

	if err != nil {
		return nil, fmt.Errorf("failed to resolve address: %w", err)
	}

	// Use packet capture helper for GetData responses
	captureConfig := PacketCaptureConfig{
		Purpose:    "GetData responses",
		Timeout:    5 * time.Second,
		SourceIP:   LocalIP,
		SourcePort: 17784,
	}

	// Start packet capture in background
	captureCtx, captureCancel := context.WithCancel(ctx)
	defer captureCancel()

	type captureResponse struct {
		result *PacketCaptureResult
		err    error
	}
	captureChan := make(chan captureResponse, 1)
	go func() {
		result, err := c.capturePacketWithContext(captureCtx, captureConfig)
		captureChan <- captureResponse{result: result, err: err}
	}()

	// Give capture a moment to start
	time.Sleep(100 * time.Millisecond)

	// NOW send the GetData packet
	_, err = c.conn.WriteToUDP(packet, destAddr)
	if err != nil {
		captureCancel()
		return nil, fmt.Errorf("failed to send GetData packet: %w", err)
	}

	c.logger.Info("Sent GetData request", "device_mac", device.MAC)

	// Wait for response
	select {
	case <-ctx.Done():
		captureCancel()
		return nil, fmt.Errorf("context cancelled while waiting for GetData response: %w", ctx.Err())
	case resp := <-captureChan:
		if resp.err != nil {
			return nil, fmt.Errorf("packet capture failed: %w", resp.err)
		}
		if resp.result == nil || len(resp.result.Payload) == 0 {
			c.logger.Warn("No response received from device")
			return nil, fmt.Errorf("no GetData response received from device %s", device.MAC)
		}

		result := resp.result
		c.logger.Info("Captured GetData response", "bytes", len(result.Payload))

		// Debug: log raw packet
		limit := min(len(result.Payload), 50)
		hexData := fmt.Sprintf("%x", result.Payload[:limit])
		if len(result.Payload) > 50 {
			hexData += "..."
		}
		c.logger.Debug("Raw response hex", "data", hexData)

		// Parse as UDAP packet
		respPacket, data, err := ParsePacket(result.Payload)
		if err != nil {
			return nil, fmt.Errorf("failed to parse captured response: %w", err)
		}

		// Check if it's a DataResp or Error
		switch respPacket.UCPMethod {
		case MethodDataResp:
			// Parse TLV data
			tlvs := DecodeTLV(data)
			var currentParam string

			for _, tlv := range tlvs {
				switch tlv.Type {
				case TLVTypeParameterName: // Parameter name
					currentParam = string(tlv.Value)
				case TLVTypeParameterValue: // Parameter value
					if currentParam != "" {
						config[currentParam] = string(tlv.Value)
						currentParam = ""
					}
				}
			}
		case MethodError:
			return nil, fmt.Errorf("device %s returned error response", device.MAC)
		}

		return config, nil
	}
}

// GetAllDeviceConfig retrieves all known parameters from a device
func (c *Client) GetAllDeviceConfig(device *Device) error {
	c.logger.Info("Reading all device parameters", "device_mac", device.MAC)

	// Read all known parameters
	config, err := c.GetDeviceConfig(device, KnownParameters)
	if err != nil {
		return fmt.Errorf("failed to read device parameters: %w", err)
	}

	// Store parameters in device
	if device.Parameters == nil {
		device.Parameters = make(map[string]string)
	}

	maps.Copy(device.Parameters, config)

	c.logger.Info("Read parameters from device", "param_count", len(config), "device_mac", device.MAC)
	return nil
}

// SetDeviceConfig sets configuration parameters on a device
func (c *Client) SetDeviceConfig(device *Device, config map[string]string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return c.SetDeviceConfigWithContext(ctx, device, config)
}

// SetDeviceConfigWithContext sets configuration parameters on a device with context
func (c *Client) SetDeviceConfigWithContext(ctx context.Context, device *Device, config map[string]string) error {
	// First ensure we have all current device parameters
	if len(device.Parameters) == 0 {
		c.logger.Info("Device parameters not loaded, reading current configuration")
		err := c.GetAllDeviceConfig(device)
		if err != nil {
			c.logger.Warn("Could not read current parameters", "error", err)
			c.logger.Info("Proceeding with just the new parameters")
			if device.Parameters == nil {
				device.Parameters = make(map[string]string)
			}
		}
	}

	// Merge new configuration with existing parameters
	allParams := make(map[string]string)
	maps.Copy(allParams, device.Parameters)
	for param, value := range config {
		allParams[param] = value
		// Also update the device's stored parameters
		device.Parameters[param] = value
	}

	c.logger.Info("Sending complete configuration", "total_params", len(allParams))
	for param, value := range config {
		c.logger.Info("Parameter changed", "param", param, "value", value)
	}

	// Create SetData packet with ALL parameters
	packet := c.CreateSetDataPacket(device, allParams)

	// Send to device
	var destAddr *net.UDPAddr
	var err error

	if device.IP == "0.0.0.0" {
		// Device in bootstrap mode - use broadcast
		destAddr, err = net.ResolveUDPAddr("udp", fmt.Sprintf("255.255.255.255:%d", Port))
	} else {
		destAddr, err = net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", device.IP, Port))
	}

	if err != nil {
		return fmt.Errorf("failed to resolve address: %w", err)
	}

	// Use packet capture helper for SetData responses
	captureConfig := PacketCaptureConfig{
		Purpose:    "SetData responses",
		Timeout:    5 * time.Second,
		SourceIP:   "0.0.0.0",
		SourcePort: 17784,
	}

	// Start packet capture in background
	captureCtx, captureCancel := context.WithCancel(ctx)
	defer captureCancel()

	type captureResponse struct {
		result *PacketCaptureResult
		err    error
	}
	captureChan := make(chan captureResponse, 1)
	go func() {
		result, err := c.capturePacketWithContext(captureCtx, captureConfig)
		captureChan <- captureResponse{result: result, err: err}
	}()

	// Give capture a moment to start
	time.Sleep(100 * time.Millisecond)

	// NOW send the SetData packet
	_, err = c.conn.WriteToUDP(packet, destAddr)
	if err != nil {
		captureCancel()
		return fmt.Errorf("failed to send SetData packet: %w", err)
	}

	c.logger.Info("Sent SetData request", "device_mac", device.MAC)

	// Wait for response
	select {
	case <-ctx.Done():
		captureCancel()
		return fmt.Errorf("context cancelled while waiting for SetData response: %w", ctx.Err())
	case resp := <-captureChan:
		if resp.err != nil {
			return fmt.Errorf("packet capture failed: %w", resp.err)
		}
		if resp.result == nil || len(resp.result.Payload) == 0 {
			c.logger.Warn("No response received from device - assuming configuration was applied")
			return nil
		}

		c.logger.Info("Captured broadcast response", "bytes", len(resp.result.Payload))

		// Debug: log raw packet
		limit := min(len(resp.result.Payload), 50)
		hexData := fmt.Sprintf("%x", resp.result.Payload[:limit])
		if len(resp.result.Payload) > 50 {
			hexData += "..."
		}
		c.logger.Debug("Raw response hex", "data", hexData)

		// Parse as UDAP packet
		respPacket, data, err := ParsePacket(resp.result.Payload)
		if err != nil {
			c.logger.Error("Failed to parse captured response", "error", err)
			return fmt.Errorf("failed to parse response: %w", err)
		}

		c.logger.Debug("Response packet details", "udap_type", fmt.Sprintf("0x%04x", respPacket.UDAPType), "ucp_method", fmt.Sprintf("0x%04x", respPacket.UCPMethod))

		// Process the response
		switch respPacket.UCPMethod {
		case MethodDataResp, MethodSetData, MethodGetData, MethodSetDataAck:
			c.logger.Info("Device acknowledged configuration change", "method", fmt.Sprintf("0x%04x", respPacket.UCPMethod))
			if len(data) > 0 {
				tlvs := DecodeTLV(data)
				for _, tlv := range tlvs {
					c.logger.Debug("Response TLV data", "type", tlv.Type, "length", tlv.Length)
				}
			}
			return nil
		case MethodError:
			if len(data) > 0 {
				tlvs := DecodeTLV(data)
				for _, tlv := range tlvs {
					if tlv.Type == TLVTypeErrorMessage {
						return fmt.Errorf("device error: %s", string(tlv.Value))
					}
				}
			}
			return fmt.Errorf("device %s returned error response", device.MAC)
		default:
			return fmt.Errorf("unexpected response method: 0x%04x", respPacket.UCPMethod)
		}
	}
}

// ResetDevice sends a reset command to restart the device
func (c *Client) ResetDevice(device *Device) error {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	return c.ResetDeviceWithContext(ctx, device)
}

// ResetDeviceWithContext sends a reset command to restart the device with context
func (c *Client) ResetDeviceWithContext(ctx context.Context, device *Device) error {
	// Create Reset packet
	packet := c.CreateResetPacket(device)

	// Send to device
	var destAddr *net.UDPAddr
	var err error

	if device.IP == "0.0.0.0" {
		// Device in bootstrap mode - use broadcast
		destAddr, err = net.ResolveUDPAddr("udp", fmt.Sprintf("255.255.255.255:%d", Port))
	} else {
		destAddr, err = net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", device.IP, Port))
	}

	if err != nil {
		return fmt.Errorf("failed to resolve address: %w", err)
	}

	// Use packet capture helper for Reset responses
	captureConfig := PacketCaptureConfig{
		Purpose:    "Reset responses",
		Timeout:    3 * time.Second,
		SourceIP:   "0.0.0.0",
		SourcePort: 17784,
	}

	// Start packet capture in background
	captureCtx, captureCancel := context.WithCancel(ctx)
	defer captureCancel()

	type captureResponse struct {
		result *PacketCaptureResult
		err    error
	}
	captureChan := make(chan captureResponse, 1)
	go func() {
		result, err := c.capturePacketWithContext(captureCtx, captureConfig)
		captureChan <- captureResponse{result: result, err: err}
	}()

	// Give capture a moment to start
	time.Sleep(100 * time.Millisecond)

	// Send reset packet
	c.logger.Info("Sending reset packet", "bytes", len(packet), "dest_addr", destAddr.String())
	limit := min(len(packet), 30)
	hexData := fmt.Sprintf("%x", packet[:limit])
	if len(packet) > 30 {
		hexData += "..."
	}
	c.logger.Debug("Reset packet hex", "data", hexData)

	_, err = c.conn.WriteToUDP(packet, destAddr)
	if err != nil {
		captureCancel()
		return fmt.Errorf("failed to send Reset packet: %w", err)
	}

	c.logger.Info("Sent Reset command", "device_mac", device.MAC)

	// Wait for response with shorter timeout (device may reset immediately)
	select {
	case <-ctx.Done():
		captureCancel()
		c.logger.Info("No response from device - it may have reset immediately")
		return nil
	case resp := <-captureChan:
		if resp.err != nil {
			c.logger.Warn("Packet capture failed", "error", resp.err)
			// Don't fail the reset operation due to capture error
			return nil
		}
		if resp.result == nil || len(resp.result.Payload) == 0 {
			c.logger.Info("No response from device - it may have reset immediately")
			return nil
		}

		c.logger.Info("Received reset acknowledgment", "bytes", len(resp.result.Payload))

		// Debug: print raw packet
		limit := min(len(resp.result.Payload), 30)
		hexData := fmt.Sprintf("%x", resp.result.Payload[:limit])
		if len(resp.result.Payload) > 30 {
			hexData += "..."
		}
		c.logger.Debug("Raw response hex", "data", hexData)

		// Parse response
		respPacket, _, err := ParsePacket(resp.result.Payload)
		if err != nil {
			c.logger.Warn("Could not parse response", "error", err)
		} else {
			c.logger.Debug("Response packet details", "udap_type", fmt.Sprintf("0x%04x", respPacket.UDAPType), "ucp_method", fmt.Sprintf("0x%04x", respPacket.UCPMethod))
			switch respPacket.UCPMethod {
			case MethodGetData:
				// Based on net-udap capture, device responds with GetData (0x0001) to reset
				c.logger.Info("Device acknowledged reset command - restarting")
			case MethodSetData:
				c.logger.Info("Device acknowledged reset command")
			}
		}
	}

	c.logger.Info("Device is resetting...")
	return nil
}

// SaveDeviceConfig sends save_data using the exact net-udap packet format
// Now uses the corrected packet structure that matches net-udap byte-for-byte
func (c *Client) SaveDeviceConfig(device *Device) error {
	ctx, cancel := context.WithTimeout(context.Background(), 7*time.Second)
	defer cancel()
	return c.SaveDeviceConfigWithContext(ctx, device)
}

// SaveDeviceConfigWithContext sends save_data with context
func (c *Client) SaveDeviceConfigWithContext(ctx context.Context, device *Device) error {
	c.logger.Info("Saving device configuration to persistent storage")

	// First ensure we have all current device parameters
	if len(device.Parameters) == 0 {
		c.logger.Info("Device parameters not loaded, reading current configuration")
		err := c.GetAllDeviceConfig(device)
		if err != nil {
			return fmt.Errorf("failed to read device parameters for save: %w", err)
		}
	}

	c.logger.Info("Sending save request", "total_params", len(device.Parameters))

	// Single save operation using the corrected net-udap packet format
	err := c.saveDeviceConfigWithAllParamsCtx(ctx, device, device.Parameters)
	if err != nil {
		return fmt.Errorf("save operation failed: %w", err)
	}

	c.logger.Info("✓ Configuration saved to persistent storage")
	return nil
}

// saveDeviceConfigWithAllParamsCtx sends save data with context
func (c *Client) saveDeviceConfigWithAllParamsCtx(ctx context.Context, device *Device, allParams map[string]string) error {
	c.logger.Info("Sending save_data packet", "param_count", len(allParams), "method", "0x0006")

	// Create SaveData packet with method 0x0006 (like net-udap)
	packet := c.CreateSaveDataPacket(device, allParams)

	c.logger.Info("Created save packet", "bytes", len(packet))
	limit := min(len(packet), 50)
	hexData := fmt.Sprintf("%x", packet[:limit])
	c.logger.Debug("Save packet hex (first 50 bytes)", "data", hexData)

	// Send to device using broadcast
	destAddr, err := net.ResolveUDPAddr("udp", fmt.Sprintf("%s:%d", BroadcastIP, Port))
	if err != nil {
		return fmt.Errorf("failed to resolve broadcast address: %w", err)
	}

	// Use packet capture helper for SaveData responses
	captureConfig := PacketCaptureConfig{
		Purpose:    "SaveData responses",
		Timeout:    5 * time.Second,
		SourceIP:   "0.0.0.0",
		SourcePort: 17784,
	}

	// Start packet capture in background
	captureCtx, captureCancel := context.WithCancel(ctx)
	defer captureCancel()

	type captureResponse struct {
		result *PacketCaptureResult
		err    error
	}
	captureChan := make(chan captureResponse, 1)
	go func() {
		result, err := c.capturePacketWithContext(captureCtx, captureConfig)
		captureChan <- captureResponse{result: result, err: err}
	}()

	// Give capture a moment to start
	time.Sleep(100 * time.Millisecond)

	// NOW send the SaveData packet
	_, err = c.conn.WriteToUDP(packet, destAddr)
	if err != nil {
		captureCancel()
		return fmt.Errorf("failed to send SaveData packet: %w", err)
	}

	c.logger.Info("Sent SaveData request to broadcast", "method", "0x0001", "note", "corrected from packet capture")

	// Wait for response with timeout
	select {
	case <-ctx.Done():
		captureCancel()
		c.logger.Warn("Timeout waiting for response - save may have succeeded")
		return nil
	case resp := <-captureChan:
		if resp.err != nil {
			return fmt.Errorf("packet capture failed: %w", resp.err)
		}
		if resp.result == nil || len(resp.result.Payload) == 0 {
			c.logger.Warn("No response received from device - save may have succeeded")
			return nil
		}

		c.logger.Info("Captured SaveData response", "bytes", len(resp.result.Payload))

		// Debug: log raw packet
		limit := min(len(resp.result.Payload), 50)
		hexData := fmt.Sprintf("%x", resp.result.Payload[:limit])
		if len(resp.result.Payload) > 50 {
			hexData += "..."
		}
		c.logger.Debug("Raw response hex", "data", hexData)

		// Parse as UDAP packet
		respPacket, data, err := ParsePacket(resp.result.Payload)
		if err != nil {
			c.logger.Error("Failed to parse captured response", "error", err)
			return fmt.Errorf("invalid response from device %s: %w", device.MAC, err)
		}

		c.logger.Debug("Response packet details", "udap_type", fmt.Sprintf("0x%04x", respPacket.UDAPType), "ucp_method", fmt.Sprintf("0x%04x", respPacket.UCPMethod))

		// Check for successful save response (device may respond with GetData 0x0001 or SetDataAck 0x0008)
		switch respPacket.UCPMethod {
		case MethodDataResp, MethodSetData, MethodGetData, MethodSetDataAck:
			c.logger.Info("Device acknowledged save operation", "method", fmt.Sprintf("0x%04x", respPacket.UCPMethod))
			c.logger.Debug("Response method details", "method", fmt.Sprintf("0x%04x", respPacket.UCPMethod), "note", "save_data success")

			// For save_data operations, check if we get the expected response
			if len(data) > 0 {
				tlvs := DecodeTLV(data)
				c.logger.Debug("Response TLV data", "param_count", len(tlvs))
			}
			return nil // Successful save confirmed
		case MethodError:
			if len(data) > 0 {
				tlvs := DecodeTLV(data)
				for _, tlv := range tlvs {
					if tlv.Type == TLVTypeErrorMessage {
						return fmt.Errorf("device error: %s", string(tlv.Value))
					}
				}
			}
			return fmt.Errorf("device %s returned error response", device.MAC)
		default:
			return fmt.Errorf("unexpected response method: 0x%04x", respPacket.UCPMethod)
		}
	}
}
