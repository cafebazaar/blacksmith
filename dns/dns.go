package dns

import (
	"net"
	"strconv"
	"time"

	"golang.org/x/net/context"

	"github.com/cafebazaar/blacksmith/datasource"
	"github.com/miekg/coredns/middleware/proxy"
	"github.com/miekg/dns"

	log "github.com/Sirupsen/logrus"
)

type dnsServer struct {
	ds         *datasource.EtcdDatasource
	roundRobin int
}

func (dnsServ *dnsServer) clusterDNS(w dns.ResponseWriter, r *dns.Msg) {

	var (
		rr []dns.RR
	)

	rr = make([]dns.RR, 0)
	m := new(dns.Msg)
	m.SetReply(r)

	log.Info("DNS request: " + m.Question[0].Name)

	masterMachines := make([]*datasource.EtcdMachine, 0)
	machines, err := dnsServ.ds.GetEtcdMachines()
	if err != nil {
		log.Error("can't retrieve machines interfaces")
	}
	var masterMachinesNum int
	masterMachinesNum = 0
	for _, machine := range machines {
		isServer, err := machine.GetVariable("blacksmith_server")
		if err != nil {
			log.Error("can't find out if machine is master")
		}
		if isServer == "true" {
			masterMachinesNum = masterMachinesNum + 1
			masterMachines = append(masterMachines, machine)
		}
	}

	machineIP := net.ParseIP("127.0.0.1")
	if masterMachinesNum != 0 {
		for i, machine := range machines {
			machineInstance, err := machine.Machine(false, nil)
			if err != nil {
				log.Error("can't retrieve machine interfaces")
			}
			hostName, err := machine.GetVariable("hostname")
			if err != nil {
				log.Info("there is no hostname variable for machine")
				hostName = machine.Hostname()
			}

			domainName := hostName + "." + dnsServ.ds.ClusterName() + "."
			// Should not be hardcoded
			masterDomainName := "master" + "." + dnsServ.ds.ClusterName() + "."
			if r.Question[0].Name == domainName {
				log.Info("returning IP: " + machineInstance.IP.String())
				machineIP = machineInstance.IP
				rr = append(rr, &dns.A{
					Hdr: dns.RR_Header{Name: r.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 3600},
					A:   machineIP,
				})
			} else if r.Question[0].Name == masterDomainName {
				dnsServ.roundRobin = dnsServ.roundRobin % 0xFFFF
				index := (i + dnsServ.roundRobin) % masterMachinesNum
				masterMachineInterface := masterMachines[index]
				masterMachine, _ := masterMachineInterface.Machine(false, nil)
				machineIP = masterMachine.IP
				log.Info("returning IP: " + masterMachine.IP.String())
				rr = append(rr, &dns.A{
					Hdr: dns.RR_Header{Name: r.Question[0].Name, Rrtype: dns.TypeA, Class: dns.ClassINET, Ttl: 3600},
					A:   machineIP,
				})

			}
		}
	} else {
		log.Info("master machines nummber: " + strconv.Itoa(masterMachinesNum))
	}
	if err != nil {
		log.Error("can't retrieve master machine")
	}

	switch r.Question[0].Qtype {
	default:
		fallthrough
	case dns.TypeAAAA, dns.TypeA:
		for _, r := range rr {
			m.Answer = append(m.Answer, r)
		}
	}

	w.WriteMsg(m)
}

func (dnsServ *dnsServer) generalDNS(w dns.ResponseWriter, r *dns.Msg) {

	pp := proxy.New([]string{"8.8.4.4:53", "8.8.8.8:53"})

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pp.ServeDNS(ctx, w, r)

}

func ServeDNS(tcpServerIP net.TCPAddr, udpServerIP net.UDPAddr, ds *datasource.EtcdDatasource) error {

	dnsServer := &dnsServer{ds: ds}
	dnsServer.roundRobin = 0

	dns.HandleFunc(ds.ClusterName()+".", dnsServer.clusterDNS)
	dns.HandleFunc(".", dnsServer.generalDNS)

	go func() {
		serverUDP := &dns.Server{Addr: udpServerIP.IP.String() + ":" + strconv.Itoa(udpServerIP.Port), Net: "udp", TsigSecret: nil}
		if err := serverUDP.ListenAndServe(); err != nil {
			log.Error("Failed to setup the "+"udp server: %s\n", err.Error())
		}
	}()

	go func() {
		serverTCP := &dns.Server{Addr: tcpServerIP.IP.String() + ":" + strconv.Itoa(tcpServerIP.Port), Net: "tcp", TsigSecret: nil}
		if err := serverTCP.ListenAndServe(); err != nil {
			log.Error("Failed to setup the "+"tcp server: %s\n", err.Error())
		}
	}()
	return nil
}
