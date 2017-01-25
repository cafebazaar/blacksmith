package main

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"runtime/trace"
	"strings"
	"time"

	"github.com/cafebazaar/blacksmith/agent"
	etcd "github.com/coreos/etcd/client"
	"github.com/pkg/errors"

	"github.com/Sirupsen/logrus"
)

func main() {
	opts := parseFlags()

	if opts.Debug {
		logrus.SetLevel(logrus.DebugLevel)
	} else {
		logrus.SetLevel(logrus.InfoLevel)
	}

	etcdClient, err := etcd.New(etcd.Config{
		Transport:               etcd.DefaultTransport,
		Endpoints:               opts.EtcdEndPoints,
		HeaderTimeoutPerRequest: time.Second,
	})

	if err != nil {
		logrus.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	heartbeatURL, err := url.Parse(fmt.Sprintf("%s/api/agents/%s/heartbeat",
		opts.Server, opts.HardwareAddr.String()))
	if err != nil {
		logrus.Fatal(err)
	}

	if opts.Tracing {
		logrus.Info("Tracing stopping")
		trace.Start(os.Stdout)
	}

	logrus.Debug("Heartbeat starting")
	go agent.StartHeartbeat(ctx, heartbeatURL)

	logrus.Debug("Workspace updater starting")
	go agent.WatchCommand(ctx,
		etcd.NewKeysAPI(etcdClient),
		path.Join(opts.ClusterName, "machines", opts.HardwareAddr.String(), "agent", "command"),
		agent.WatchOptions{
			UpdateCallback: func() {
				u := fmt.Sprintf("%s/t/cc/%s", opts.Server, opts.HardwareAddr.String())
				execCmd("/usr/bin/coreos-cloudinit", "-validate", "-from-url", u)
			},
			RebootCallback: func() {
				if ok := execCmd("/usr/bin/locksmithctl", "reboot"); ok {
					os.Exit(0)
				}
			},
		},
	)

	waitForInterrupt(func() {
		cancel()
		if opts.Tracing {
			logrus.Info("Tracing stopping")
			trace.Stop()
		}
		os.Exit(0)
	})
}

func waitForInterrupt(callback func()) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			logrus.WithFields(logrus.Fields{
				"signal": sig,
			}).Info("received signal")
			callback()
		}
	}()
}

func execCmd(name string, args ...string) (ok bool) {
	t0 := time.Now()
	cmd := exec.Command(name, args...)
	fullCmd := strings.Join(cmd.Args, " ")
	bufOut, bufErr := new(bytes.Buffer), new(bytes.Buffer)
	cmd.Stdout, cmd.Stderr = bufOut, bufErr

	logrus.WithFields(logrus.Fields{
		"command": fullCmd,
	}).Info("executing command")

	if err := cmd.Run(); err != nil {
		logrus.WithFields(logrus.Fields{
			"command": fullCmd,
			"stdout":  strings.TrimSpace(bufOut.String()),
			"stderr":  strings.TrimSpace(bufErr.String()),
		}).Error(errors.Wrap(err, "command failed"))
		return false
	}

	logrus.WithFields(logrus.Fields{
		"command":  fullCmd,
		"duration": time.Since(t0),
	}).Info("command completed")
	return true
}
