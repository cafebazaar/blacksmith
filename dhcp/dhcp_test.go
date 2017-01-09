package dhcp

import (
	"bytes"
	"net"
	"testing"

	"github.com/cafebazaar/blacksmith/datasource"
)

func TestDnsAddressesForDHCP(t *testing.T) {
	cases := []struct {
		input    []datasource.InstanceInfo
		expected []byte
	}{
		{[]datasource.InstanceInfo{}, []byte{}},
		{
			[]datasource.InstanceInfo{
				{IP: net.IPv4(1, 2, 3, 4)},
				{IP: net.IPv4(1, 2, 3, 5)},
			},
			[]byte{1, 2, 3, 4, 1, 2, 3, 5},
		},
	}

	for i, tt := range cases {
		got := dnsAddressesForDHCP(&tt.input)
		if res := bytes.Compare(tt.expected, got); res != 0 {
			t.Errorf(
				"#%d: expected same []byes, but Compare(%q, %q)=%d",
				i, tt.expected, got, res)
		}
	}
}
