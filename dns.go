package dnsmop

import (
	"github.com/miekg/dns"
	"io/ioutil"
	"strings"
)

var dnsConf *dns.ClientConfig

func LoadConfigFromServersFile(fn string) error {
	c := new(dns.ClientConfig)
	c.Search = make([]string, 0)
	c.Port = "53"
	c.Ndots = 1
	c.Timeout = 5
	c.Attempts = 2

	b, err := ioutil.ReadFile(fn)
	if err != nil {
		return err
	}

	c.Servers = strings.Split(string(b), "\n")
	dnsConf = c
	return nil
}

func LoadConfigFromSystem() error {
	c, err := dns.ClientConfigFromFile("/etc/resolv.conf")
	if err == nil {
		dnsConf = c
	}
	return err
}
