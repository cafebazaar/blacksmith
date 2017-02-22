package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/Sirupsen/logrus"
)

// Options represents all the commandline flags
type Options struct {
	EtcdEndPoints []string
	Master        string
	ClusterName   string
	HardwareAddr  net.HardwareAddr
	Server        string
	Debug         bool
	Tracing       bool
}

var (
	// version is set in compile time
	version = "was not built properly"
	// commit is set in compile time
	commit = "was not built properly"
	// buildTime is set in compile time
	buildTime = "was not built properly"
)

func parseFlags() Options {
	var (
		etcdEndpointsFlag = flag.String("etcd", "", "Etcd endpoints")
		clusterNameFlag   = flag.String("cluster-name", "blacksmith", "The name of this cluster. Will be used as etcd path prefixes.")
		macAddrFlag       = flag.String("mac", "", "mac address")
		serverFlag        = flag.String("server", "http://localhost:8000", "HTTP Server to send heartbeats and messages to")
		versionFlag       = flag.Bool("version", false, "Print version info and exit")
		debugFlag         = flag.Bool("debug", false, "Log more things")
	)

	flag.Parse()

	if *versionFlag || *debugFlag {
		fmt.Printf(`blacksmith-agent
  version:   %s
  commit:    %s
  buildTime: %s
`, version, commit, buildTime)
	}

	if *versionFlag {
		os.Exit(0)
	}

	macAddr, err := net.ParseMAC(*macAddrFlag)
	if err != nil {
		logrus.Fatal(err)
	}

	return Options{
		EtcdEndPoints: strings.Split(*etcdEndpointsFlag, ","),
		Master:        "master",
		ClusterName:   *clusterNameFlag,
		HardwareAddr:  macAddr,
		Server:        *serverFlag,
		Debug:         *debugFlag,
	}
}
