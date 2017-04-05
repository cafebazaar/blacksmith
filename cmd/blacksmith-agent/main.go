package main

import (
	"bytes"
	"context"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path"
	"strings"
	"time"

	"github.com/cafebazaar/blacksmith/agent"
	etcd "github.com/coreos/etcd/client"
	"github.com/pkg/errors"

	"github.com/Sirupsen/logrus"
)

func main() {
	log.Println("agent-starting...")
	opts := parseFlags()

	logrus.SetLevel(logrus.DebugLevel)

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

	logrus.Debug("Heartbeat starting")
	go agent.StartHeartbeat(ctx, opts.HeartbeatServer, opts.HardwareAddr.String(), "localhost", opts.TLSCert, opts.TLSKey, opts.TLSCa)

	logrus.Debug("Workspace updater starting")
	go agent.WatchCommand(ctx,
		etcd.NewKeysAPI(etcdClient),
		path.Join(opts.ClusterName, "machines", opts.HardwareAddr.String(), "agent", "command"),
		agent.WatchOptions{
			UpdateCallback: func() {
				if ok := execCmd("/usr/bin/coreos-cloudinit", "-validate", "-from-url", opts.CloudconfigURL); ok {
					execCmd("/usr/bin/coreos-cloudinit", "-from-url", opts.CloudconfigURL)
				}
			},
			RebootCallback: func() {
				if ok := execCmd("/usr/bin/locksmithctl", "reboot"); ok {
					os.Exit(0)
				}
			},
		},
	)

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	sig := <-quit
	logrus.WithFields(logrus.Fields{
		"signal": sig,
	}).Info("received signal")
	cancel()
}

func execCmd(name string, args ...string) (ok bool) {
	t0 := time.Now()
	ctx, _ := context.WithTimeout(context.Background(), 2*time.Minute)
	cmd := exec.CommandContext(ctx, name, args...)
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
