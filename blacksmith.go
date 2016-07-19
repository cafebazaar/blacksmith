package main // import "github.com/cafebazaar/blacksmith"

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/cafebazaar/blacksmith/datasource"
	"github.com/cafebazaar/blacksmith/dhcp"
	"github.com/cafebazaar/blacksmith/logging"
	"github.com/cafebazaar/blacksmith/pxe"
	"github.com/cafebazaar/blacksmith/web"
	etcd "github.com/coreos/etcd/client"
)

//go:generate esc -o pxe/pxelinux_autogen.go -prefix=pxe -pkg pxe -ignore=README.md pxe/pxelinux
//go:generate esc -o web/ui_autogen.go -prefix=web -ignore=bower_components -pkg web web/ui

const (
	workspacePathHelp = `Path to workspace which obey following structure
		/config/bootparams/main
		/config/cloudconfig/main
		/config/ignition/main
		/images/{core-os-version}/coreos_production_pxe_image.cpio.gz
		/images/{core-os-version}/coreos_production_pxe.vmlinuz
		/initial.yaml
`
	debugTag = "MAIN"

	httpListenFlagDefaultTCPAddress = "interface-ip:8000"
)

var (
	versionFlag       = flag.Bool("version", false, "Print version info and exit")
	debugFlag         = flag.Bool("debug", false, "Log more things that aren't directly related to booting a recognized client")
	listenIFFlag      = flag.String("if", "0.0.0.0", "Interface name for DHCP and PXE to listen on")
	httpListenFlag    = flag.String("http-listen", httpListenFlagDefaultTCPAddress, "IP range to listen on for web requests")
	workspacePathFlag = flag.String("workspace", "/workspace", workspacePathHelp)
	etcdFlag          = flag.String("etcd", "", "Etcd endpoints")
	clusterNameFlag   = flag.String("cluster-name", "blacksmith", "The name of this cluster. Will be used as etcd path prefixes.")
	dnsAddressesFlag  = flag.String("dns", "8.8.8.8", "comma separated IPs which will be used as default nameservers for skydns.")

	leaseStartFlag  = flag.String("lease-start", "", "Begining of lease starting IP")
	leaseRangeFlag  = flag.Int("lease-range", 0, "Lease range")
	leaseSubnetFlag = flag.String("lease-subnet", "", "Subnet of specified lease")
	leaseRouterFlag = flag.String("router", "", "Default router that assigned to DHCP clients")

	version   string
	commit    string
	buildTime string
)

func init() {
	// If version, commit, or build time are not set, make that clear.
	if version == "" {
		version = "unknown"
	}
	if commit == "" {
		commit = "unknown"
	}
	if buildTime == "" {
		buildTime = "unknown"
	}
}

func interfaceIP(iface *net.Interface) (net.IP, error) {
	addrs, err := iface.Addrs()
	if err != nil {
		return nil, err
	}
	fs := [](func(net.IP) bool){
		net.IP.IsGlobalUnicast,
		net.IP.IsLinkLocalUnicast,
		net.IP.IsLoopback,
	}
	for _, f := range fs {
		for _, a := range addrs {
			ipaddr, ok := a.(*net.IPNet)
			if !ok {
				continue
			}
			ip := ipaddr.IP.To4()
			if ip == nil {
				continue
			}
			if f(ip) {
				return ip, nil
			}
		}
	}
	return nil, fmt.Errorf("interface %s has no usable unicast addresses", iface.Name)
}

func gracefulShutdown(etcdDataSource datasource.DataSource) {
	err := etcdDataSource.RemoveInstance()
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nError while removing the instance: %s\n", err)
	} else {
		fmt.Fprint(os.Stderr, "\nBlacksmith is gracefully shutdown\n")
	}
	os.Exit(0)
}

func main() {
	var err error
	flag.Parse()

	fmt.Printf("Blacksmith (%s)\n", version)
	fmt.Printf("  Commit:        %s\n", commit)
	fmt.Printf("  Build Time:    %s\n", buildTime)

	if *versionFlag {
		os.Exit(0)
	}

	// etcd config
	if etcdFlag == nil || clusterNameFlag == nil {
		fmt.Fprint(os.Stderr, "\nPlease specify the etcd endpoints\n")
		os.Exit(1)
	}

	// finding interface by interface name
	var dhcpIF *net.Interface
	if *listenIFFlag != "" {
		dhcpIF, err = net.InterfaceByName(*listenIFFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "\nError while trying to get the interface (%s): %s\n", *listenIFFlag, err)
			os.Exit(1)
		}
	} else {
		fmt.Fprint(os.Stderr, "\nPlease specify an interface\n")
		os.Exit(1)
	}

	serverIP, err := interfaceIP(dhcpIF)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nError while trying to get the ip from the interface (%s)\n", dhcpIF)
		os.Exit(1)
	}

	// web api can be configured to listen on a custom address
	webAddr := net.TCPAddr{IP: serverIP, Port: 8000}
	if *httpListenFlag != httpListenFlagDefaultTCPAddress {
		splitAddress := strings.Split(*httpListenFlag, ":")
		if len(splitAddress) > 2 {
			fmt.Printf("Incorrect tcp address provided: %s\n", *httpListenFlag)
			os.Exit(1)
		}
		if len(splitAddress) == 1 {
			splitAddress = append(splitAddress, "8000")
		}

		webAddr.IP = net.ParseIP(splitAddress[0])
		port, err := strconv.ParseInt(splitAddress[1], 10, 64)

		if err != nil {
			fmt.Printf("Incorrect tcp address provided: %s\n", *httpListenFlag)
			os.Exit(1)
		}
		webAddr.Port = int(port)
	}

	// other services are exposed just through the given interface, on hard coded ports
	var httpBooterAddr = net.TCPAddr{IP: serverIP, Port: 70}
	var tftpAddr = net.UDPAddr{IP: serverIP, Port: 69}
	var pxeAddr = net.UDPAddr{IP: serverIP, Port: 4011}
	// 67 -> dhcp

	// dhcp setting
	leaseStart := net.ParseIP(*leaseStartFlag)
	leaseRange := *leaseRangeFlag
	leaseSubnet := net.ParseIP(*leaseSubnetFlag)
	leaseRouter := net.ParseIP(*leaseRouterFlag)

	dnsIPStrings := strings.Split(*dnsAddressesFlag, ",")
	if len(dnsIPStrings) == 0 {
		fmt.Fprint(os.Stderr, "\nPlease specify an DNS server\n")
		os.Exit(1)
	}
	for _, ipString := range dnsIPStrings {
		ip := net.ParseIP(ipString)
		if ip == nil {
			fmt.Fprintf(os.Stderr, "\nInvalid dns ip: %s\n", ipString)
			os.Exit(1)
		}
	}

	if leaseStart == nil {
		fmt.Fprint(os.Stderr, "\nPlease specify the lease start ip\n")
		os.Exit(1)
	}
	if leaseRange <= 1 {
		fmt.Fprint(os.Stderr, "\nLease range should be greater that 1\n")
		os.Exit(1)
	}
	if leaseSubnet == nil {
		fmt.Fprint(os.Stderr, "\nPlease specify the lease subnet\n")
		os.Exit(1)
	}
	if leaseRouter == nil {
		fmt.Fprint(os.Stderr, "\nNo network router is defined.\n")
	}

	fmt.Printf("Interface IP:    %s\n", serverIP.String())
	fmt.Printf("Interface Name:  %s\n", dhcpIF.Name)

	// datasources
	etcdClient, err := etcd.New(etcd.Config{
		Endpoints:               strings.Split(*etcdFlag, ","),
		HeaderTimeoutPerRequest: 5 * time.Second,
	})
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nCouldn't create etcd connection: %s\n", err)
		os.Exit(1)
	}
	kapi := etcd.NewKeysAPI(etcdClient)

	selfInfo := datasource.InstanceInfo{
		IP:               serverIP,
		Nic:              dhcpIF.HardwareAddr,
		WebPort:          webAddr.Port,
		Version:          version,
		Commit:           commit,
		BuildTime:        buildTime,
		ServiceStartTime: time.Now().UTC().Unix(),
	}
	etcdDataSource, err := datasource.NewEtcdDataSource(kapi, etcdClient,
		leaseStart, leaseRange, *clusterNameFlag, *workspacePathFlag,
		dnsIPStrings, selfInfo)
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nCouldn't create runtime configuration: %s\n", err)
		os.Exit(1)
	}

	go func() {
		logging.RecordLogs(log.New(os.Stderr, "", log.LstdFlags), *debugFlag)
	}()

	// serving api
	go func() {
		err := web.ServeWeb(etcdDataSource, webAddr)
		log.Fatalf("\nError while serving api: %s\n", err)
	}()

	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for _ = range c {
			gracefulShutdown(etcdDataSource)
		}
	}()

	// waiting til we're officially the master instance
	for !etcdDataSource.IsMaster() {
		logging.Debug(debugTag, "Not master, waiting to be promoted...")
		time.Sleep(datasource.StandbyMasterUpdateTime)
	}

	logging.Debug(debugTag, "Now we're the master instance. Starting the services...")

	// serving http booter
	go func() {
		err := pxe.ServeHTTPBooter(httpBooterAddr, etcdDataSource, webAddr.Port)
		log.Fatalf("\nError while serving http booter: %s\n", err)
	}()

	// serving tftp
	go func() {
		err := pxe.ServeTFTP(tftpAddr)
		log.Fatalf("\nError while serving tftp: %s\n", err)
	}()

	// pxe protocol
	go func() {
		err := pxe.ServePXE(pxeAddr, serverIP, httpBooterAddr)
		log.Fatalf("\nError while serving pxe: %s\n", err)
	}()

	// serving dhcp
	go func() {
		err := dhcp.ServeDHCP(&dhcp.DHCPSetting{
			IFName:     dhcpIF.Name,
			ServerIP:   serverIP,
			RouterAddr: leaseRouter,
			SubnetMask: leaseSubnet,
		}, etcdDataSource)
		log.Fatalf("\nError while serving dhcp: %s\n", err)
	}()

	for etcdDataSource.IsMaster() {
		time.Sleep(datasource.ActiveMasterUpdateTime)
	}

	logging.Debug(debugTag, "Now we're NOT the master. Terminating. Hoping to be restarted by the service manager.")

	gracefulShutdown(etcdDataSource)
}
