package main

import (
	"encoding/base64"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"strings"

	"github.com/Sirupsen/logrus"
)

type options struct {
	EtcdEndPoints   []string
	Master          string
	ClusterName     string
	HardwareAddr    net.HardwareAddr
	HeartbeatServer string
	Debug           bool
	Tracing         bool
	CloudconfigURL  string
	FileServer      string
	TLSCert         string
	TLSKey          string
	TLSCa           string
}

var (
	// version is set in compile time
	version = "was not built properly"
	// commit is set in compile time
	commit = "was not built properly"
	// buildTime is set in compile time
	buildTime = "was not built properly"
)

func parseFlags() options {
	var (
		etcdEndpointsFlag   = flag.String("etcd", "", "Etcd endpoints")
		clusterNameFlag     = flag.String("cluster-name", "blacksmith", "The name of this cluster. Will be used as etcd path prefixes.")
		macAddrFlag         = flag.String("mac", "", "mac address")
		heartbeatServerFlag = flag.String("heartbeat-server", "http://localhost:8000", "HTTP Server to send heartbeats and messages to")
		cloudconfigURLFlag  = flag.String("cloudconfig-url", "", "cloudconfig url")
		fileServerFlag      = flag.String("file-server", "", "file-server base url")
		tlsCertFlag         = flag.String("tls-cert", "", "base64 encoded tls-cert")
		tlsKeyFlag          = flag.String("tls-key", "", "base64 encoded tls-key")
		tlsCaFlag           = flag.String("tls-ca", "", "base64 encoded tls-ca")
		versionFlag         = flag.Bool("version", false, "Print version info and exit")
		debugFlag           = flag.Bool("debug", false, "Log more things")
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

	tlsCert, err := base64.StdEncoding.DecodeString(*tlsCertFlag)
	if err != nil {
		log.Fatal(err)
	}
	tlsKey, err := base64.StdEncoding.DecodeString(*tlsKeyFlag)
	if err != nil {
		log.Fatal(err)
	}
	tlsCa, err := base64.StdEncoding.DecodeString(*tlsCaFlag)
	if err != nil {
		log.Fatal(err)
	}

	if *fileServerFlag == "" {
		log.Fatal(errors.New("file-server flag must be set"))
	}

	return options{
		EtcdEndPoints:   strings.Split(*etcdEndpointsFlag, ","),
		Master:          "master",
		ClusterName:     *clusterNameFlag,
		HardwareAddr:    macAddr,
		HeartbeatServer: *heartbeatServerFlag,
		Debug:           *debugFlag,
		CloudconfigURL:  *cloudconfigURLFlag,
		FileServer:      *fileServerFlag,
		TLSCert:         string(tlsCert),
		TLSKey:          string(tlsKey),
		TLSCa:           string(tlsCa),
	}
}
