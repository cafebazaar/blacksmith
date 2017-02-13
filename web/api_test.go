package web

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cafebazaar/blacksmith/datasource"
)

func TestMachineVariablesAPI(t *testing.T) {
	mac1, _ := net.ParseMAC("00:11:22:33:44:55")

	ds, err := datasource.ForTest(nil)
	if err != nil {
		t.Error("error in getting a EtcdDataSource instance for our test:", err)
		return
	}

	if err := ds.WhileMaster(); err != nil {
		t.Error("failed to register as the master instance:", err)
		return
	}
	defer func() {
		if err := ds.Shutdown(); err != nil {
			t.Error("failed to shutdown:", err)
		}
	}()

	r := &webServer{ds: ds}
	h := r.Handler()

	mi := ds.GetMachineInterface(mac1)
	_, err = mi.Machine(true, nil)
	if err != nil {
		t.Error("error while creating machine:", err)
		return
	}

	_ = mi.DeleteVariable("test")

	////////////////////////////////
	// MachineVariables
	req, err := http.NewRequest("GET", fmt.Sprintf(
		"http://test.com/api/machines/%s/variables/test", mac1), nil)
	if err != nil {
		t.Error("error while NewRequest:", err)
		return
	}

	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Error("unexpected status code while getting variables [1]:", w.Code)
		return
	}

	variables := make(map[string]string)
	err = json.Unmarshal(w.Body.Bytes(), &variables)
	if err != nil {
		t.Error("error while Unmarshal:", err, ", Body:", w.Body.String())
		return
	}

	if testVal, isIn := variables["test"]; isIn {
		t.Error("test shouldn't be among the variables, but has value=", testVal)
		return
	}

	////////////////////////////////
	// SetMachineVariable
	newVal := "192.168.1.1/24"

	req, err = http.NewRequest("PUT", fmt.Sprintf(
		"http://test.com/api/machines/%s/variables/test?value=%s", mac1, newVal), nil)
	if err != nil {
		t.Error("error while NewRequest:", err)
		return
	}

	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Error("unexpected status code while putting variable test:", w.Code)
		return
	}

	////////////////////////////////
	// MachineVariables
	req, err = http.NewRequest("GET", fmt.Sprintf(
		"http://test.com/api/machines/%s/variables/test", mac1), nil)
	if err != nil {
		t.Error("error while NewRequest:", err)
		return
	}

	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Error("unexpected status code while getting variables [2]:", w.Code)
		return
	}

	variables = make(map[string]string)
	err = json.Unmarshal(w.Body.Bytes(), &variables)
	if err != nil {
		t.Error("error while Unmarshal:", err, ", Body:", w.Body.String())
		return
	}

	testVal, isIn := variables["test"]
	if testVal != newVal {
		t.Errorf("expecting %q for test but got %q. isIn=%v", newVal, testVal, isIn)
		return
	}
}
