package notifications

import "testing"

func TestSafeDeepLinkKeepsNavigationLocal(t *testing.T) {
	tests := map[string]string{
		"/hr-leave-request/HRL-1": "/hr-leave-request/HRL-1",
		"https://example.com":     "/",
		"//example.com/path":      "/",
		"":                        "/",
	}
	for value, want := range tests {
		if got := SafeDeepLink(value); got != want {
			t.Errorf("SafeDeepLink(%q) = %q, want %q", value, got, want)
		}
	}
}
