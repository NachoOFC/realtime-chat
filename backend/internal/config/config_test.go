package config

import "testing"

func TestHTTPAddressEnvironmentPrecedence(t *testing.T) {
	for _, test := range []struct {
		name, address, port, want string
	}{
		{"local default", "", "", ":8080"},
		{"hosting port", "", "10000", ":10000"},
		{"explicit address", "127.0.0.1:8088", "10000", "127.0.0.1:8088"},
		{"blank address uses hosting port", "  ", " 10000 ", ":10000"},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("HTTP_ADDR", test.address)
			t.Setenv("PORT", test.port)
			if got := Load().HTTPAddr; got != test.want {
				t.Fatalf("HTTPAddr = %q, want %q", got, test.want)
			}
		})
	}
}
