package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOriginGuard(t *testing.T) {
	for _, test := range []struct {
		method, origin, site string
		want                 int
	}{
		{"POST", "https://frontend.test", "cross-site", 204},
		{"POST", "https://evil.test", "cross-site", 403},
		{"POST", "null", "cross-site", 403},
		{"POST", "", "cross-site", 403},
		{"POST", "", "", 204},
		{"GET", "https://evil.test", "cross-site", 204},
	} {
		req := httptest.NewRequest(test.method, "https://backend.test/api/auth/logout", nil)
		req.Header.Set("Origin", test.origin)
		req.Header.Set("Sec-Fetch-Site", test.site)
		out := httptest.NewRecorder()
		OriginGuard("https://frontend.test")(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(204) })).ServeHTTP(out, req)
		if out.Code != test.want {
			t.Errorf("%+v got %d", test, out.Code)
		}
	}
}
