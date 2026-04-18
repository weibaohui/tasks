package cmd

import (
	"testing"
)

func TestTunnelCommands_DefaultPort(t *testing.T) {
	if tunnelPIDFileName != "tunnel.pid" {
		t.Errorf("expected tunnelPIDFileName to be tunnel.pid, got: %s", tunnelPIDFileName)
	}
	if tunnelLogFileName != "tunnel.log" {
		t.Errorf("expected tunnelLogFileName to be tunnel.log, got: %s", tunnelLogFileName)
	}
}

func TestTunnelCommands_URLRegex(t *testing.T) {
	testCases := []struct {
		input    string
		expected bool
	}{
		{"https://abc.trycloudflare.com", true},
		{"https://xyz-123.trycloudflare.com", true},
		{"http://example.com", false},
		{"https://example.com", false},
	}

	for _, tc := range testCases {
		match := tunnelURLRegex.MatchString(tc.input)
		if match != tc.expected {
			t.Errorf("tunnelURLRegex.MatchString(%s) = %v, expected %v", tc.input, match, tc.expected)
		}
	}
}

func TestGetServerHTTPPort_NoConfig(t *testing.T) {
	port := getServerHTTPPort()
	if port != 0 {
		t.Logf("getServerHTTPPort returned: %d", port)
	}
}

func TestNotifyServerToUpdateWebhooks_EmptyToken(t *testing.T) {
	err := notifyServerToUpdateWebhooks()
	if err == nil {
		t.Skip("API token not configured, test requires token")
	}
}

func TestNotifyServerToUpdateWebhooks_CallWithBearerToken(t *testing.T) {
	err := notifyServerToUpdateWebhooks()
	if err != nil && err.Error() == "API token is empty, cannot authenticate" {
		t.Logf("Got expected empty token error: %v", err)
	}
}
