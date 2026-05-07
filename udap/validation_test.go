package udap

import "testing"

func TestValidateParameterAcceptsKnownValid(t *testing.T) {
	if err := ValidateParameter("lan_ip_mode", "1"); err != nil {
		t.Fatalf("expected nil error for lan_ip_mode=1, got %v", err)
	}
}

func TestValidateParameterRejectsBadIP(t *testing.T) {
	if err := ValidateParameter("lan_network_address", "not.an.ip"); err == nil {
		t.Fatalf("expected error for invalid IP, got nil")
	}
}

func TestValidateParameterAcceptsUnknownParameter(t *testing.T) {
	// Matches the existing internal behavior: unknown params are not
	// rejected by validation; rejection happens at the CLI boundary.
	if err := ValidateParameter("not_a_real_param", "x"); err != nil {
		t.Fatalf("expected nil error for unknown param, got %v", err)
	}
}
