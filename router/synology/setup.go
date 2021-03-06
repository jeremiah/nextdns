package synology

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/nextdns/nextdns/config"
	"github.com/nextdns/nextdns/router/internal"
)

type Router struct {
	DNSMasqPath     string
	ListenPort      string
	ClientReporting bool

	disabled bool
}

func New() (*Router, bool) {
	if b, err := exec.Command("uname", "-u").Output(); err != nil ||
		!strings.HasPrefix(string(b), "synology") {
		return nil, false
	}
	return &Router{
		DNSMasqPath: "/etc/dhcpd/dhcpd-vendor.conf",
		ListenPort:  "5342",
	}, true
}

func (r *Router) Configure(c *config.Config) error {
	if b, err := ioutil.ReadFile("/etc/dhcpd/dhcpd.info"); err != nil || !bytes.HasPrefix(b, []byte(`enable="yes"`)) {
		// DHCP is disabled, listen on 53 directly
		c.Listen = ":53"
		r.disabled = true
		return nil
	}
	c.Listen = "127.0.0.1:" + r.ListenPort
	r.ClientReporting = c.ReportClientInfo
	return nil
}

func (r *Router) Setup() error {
	if r.disabled {
		return nil
	}
	if err := internal.WriteTemplate(r.DNSMasqPath, tmpl, r, 0644); err != nil {
		return err
	}
	return restartDNSMasq()
}

func (r *Router) Restore() error {
	if r.disabled {
		return nil
	}
	_ = os.Remove(r.DNSMasqPath)
	if err := restartDNSMasq(); err != nil {
		return fmt.Errorf("service restart_dnsmasq: %v", err)
	}
	return nil
}

func restartDNSMasq() error {
	// Restart dnsmasq.
	if err := exec.Command("/etc/rc.network", "nat-restart-dhcp").Run(); err != nil {
		return fmt.Errorf("/etc/rc.network nat-restart-dhcp: %v", err)
	}
	return nil
}

var tmpl = `# Configuration generated by NextDNS
no-resolv
server=127.0.0.1#{{.ListenPort}}
{{- if .ClientReporting}}
add-mac
add-subnet=32,128
{{- end}}
`
