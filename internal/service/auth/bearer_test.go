package auth

import "testing"

func TestBearerTokenFromHeader(t *testing.T) {
	t.Helper()

	tests := []struct {
		header string
		want   string
	}{
		{"Bearer abc.def.ghi", "abc.def.ghi"},
		{"bearer abc", ""},
		{"Basic abc", ""},
		{"", ""},
	}

	for _, tt := range tests {
		if got := bearerTokenFromHeader(tt.header); got != tt.want {
			t.Fatalf("header %q: got %q want %q", tt.header, got, tt.want)
		}
	}
}

func TestValidateBearerClaims(t *testing.T) {
	t.Helper()

	runtime := &bearerRuntime{
		issuer: "https://sso.example.com/realms/example",
		allowedClients: map[string]struct{}{
			"individual": {},
		},
	}

	futureExp := float64(9999999999)

	err := validateBearerClaims(map[string]any{
		"iss": "https://sso.example.com/realms/example",
		"azp": "individual",
		"exp": futureExp,
	}, runtime)
	if err != nil {
		t.Fatalf("expected valid claims: %v", err)
	}

	err = validateBearerClaims(map[string]any{
		"iss": "https://sso.example.com/realms/example",
		"azp": "individual",
	}, runtime)
	if err == nil {
		t.Fatal("expected error for missing exp")
	}

	err = validateBearerClaims(map[string]any{
		"iss": "https://sso.example.com/realms/example",
		"azp": "individual",
		"exp": float64(1),
	}, runtime)
	if err == nil {
		t.Fatal("expected error for expired token")
	}

	err = validateBearerClaims(map[string]any{
		"iss": "https://sso.example.com/realms/example",
		"azp": "other-client",
		"exp": futureExp,
	}, runtime)
	if err == nil {
		t.Fatal("expected client rejection")
	}
}
