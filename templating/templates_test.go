package templating

import (
	"net"
	"testing"
	"text/template"

	"github.com/cafebazaar/blacksmith/datasource"
)

func TestExecuteTemplate(t *testing.T) {
	cases := []struct {
		inputTemplteRoot *template.Template
		templateName     string
		err              bool
		expected         string
	}{
	// TODO
	}

	mac1, _ := net.ParseMAC("FF:FF:FF:FF:00:0F")

	ds, err := datasource.ForTest(nil)
	if err != nil {
		t.Error("error in getting a EtcdDatasource instance for our test:", err)
		return
	}

	for i, tt := range cases {
		got, err := executeTemplate(
			tt.inputTemplteRoot, tt.templateName,
			ds, ds.GetEtcdMachine(mac1))

		if tt.err && err == nil {
			t.Errorf("#%d: expected error, got nil", i)
			continue
		} else if !tt.err && err != nil {
			t.Errorf("#%d: expected no error, err=%q", i, err)
			continue
		}

		if tt.expected != got {
			t.Errorf("#%d: expected %q, got %q", i, tt.expected, got)
		}
	}
}
