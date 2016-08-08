package datasource

import "testing"

func TestValidateVariable(t *testing.T) {
	tests := []struct {
		key   string
		value string
		err   bool
	}{
		{"123", "123", false},
		{"123", "", false},

		{"", "123", true},
		{"_", "123", true},

		// CoreosVersion
		{SpecialKeyCoreosVersion, "1010.4.0", false},
		{SpecialKeyCoreosVersion, "", true},

		// NetworkConfiguration
		{SpecialKeyNetworkConfiguration,
			`{"netmask":"255.255.255.0"}`, false},
		{SpecialKeyNetworkConfiguration,
			`{"netmask":"255.255.255.0", "router": "172.19.1.1", "classlessRouteOption": [{"router": "172.19.1.2", "size":23, "destination": "5.6.7.0"}]}`, false},

		{SpecialKeyNetworkConfiguration, "", true},
		{SpecialKeyNetworkConfiguration,
			`{"netmask":"invalid"}`, true},
	}

	for i, tt := range tests {
		got := validateVariable(tt.key, tt.value)
		if tt.err && got == nil {
			t.Errorf("#%d: expected error, got nil", i)
		} else if !tt.err && got != nil {
			t.Errorf("#%d: expected no error, got %q", i, got)
		}
	}
}
