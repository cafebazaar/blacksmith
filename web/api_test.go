package web

import (
	"testing"

	"github.com/cafebazaar/blacksmith/datasource"
)

func TestAPI(t *testing.T) {
	ds := datasource.ForTest(t)

	if ds == nil {
		t.Error("failed to GetDatasource")
	}

}
