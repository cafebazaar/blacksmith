package agent_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"golang.org/x/net/context"

	"github.com/Sirupsen/logrus"
	"github.com/cafebazaar/blacksmith/agent"
)

func init() {
	logrus.SetLevel(logrus.WarnLevel)
}

func TestHeartbeat(t *testing.T) {
	heartbeatCounter := 0
	defer func() {
		if got, want := heartbeatCounter, 1; got != want {
			t.Errorf("heartbeat sent %d requests, wanted %d request", got, want)
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		heartbeatCounter++
		w.Write([]byte("ok"))
		cancel()
	}))
	defer server.Close()

	u, err := url.Parse(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	agent.StartHeartbeat(ctx, u)
}
