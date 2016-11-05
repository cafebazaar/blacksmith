package datasource

import "testing"

func TestValidateVariable(t *testing.T) {
	tests := []struct {
		key        string
		value      string
		forMachine bool
		err        bool
	}{
		{"123", "123", false, false},
		{"123", "", false, false},

		{"", "123", false, true},
		{"_", "123", false, true},

		{"123", "123", true, false},
		{"#123", "123", true, true},

		// CoreosVersion
		{SpecialKeyCoreosVersion, "1010.4.0", true, false},
		{SpecialKeyCoreosVersion, "", true, true},

		// NetworkConfiguration
		{SpecialKeyNetworkConfiguration,
			`{"netmask":"255.255.255.0"}`, true, false},
		// TODO: This should be invalid becuase the 23th bit of 5.6.7.0 is 1
		{SpecialKeyNetworkConfiguration,
			`{"netmask":"255.255.255.0", "router": "172.19.1.1", "classlessRouteOption": [{"router": "172.19.1.2", "size":23, "destination": "5.6.7.0"}]}`, true, false},

		{SpecialKeyNetworkConfiguration, "", true, true},
		{SpecialKeyNetworkConfiguration,
			`{"netmask":"invalid"}`, true, true},
	}

	for i, tt := range tests {
		got := validateVariable(tt.key, tt.value, tt.forMachine)
		if tt.err && got == nil {
			t.Errorf("#%d: expected error, got nil", i)
		} else if !tt.err && got != nil {
			t.Errorf("#%d: expected no error, got %q", i, got)
		}
	}
}
