package dnsmop

import (
	"github.com/miekg/dns"
	"io/ioutil"
	"strings"
	"errors"
    "math/rand"
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

func Query(rname string, rtype uint16) ([]dns.RR, error) {
	m := new(dns.Msg)
	c := new(dns.Client)

	m.SetQuestion(rname, dns.TypePTR)
	m.MsgHdr.RecursionDesired = true

	var r *dns.Msg
	for {
		srv := dnsConf.Servers[rand.Intn(len(dnsConf.Servers))]
		var err error
		r, err = c.Exchange(m, srv + ":" + dnsConf.Port)

		if err == nil {
			break
		}
	}

	if r.Rcode != dns.RcodeSuccess {
		return nil, errors.New(dns.Rcode_str[r.Rcode])
	}

	return r.Answer, nil
}