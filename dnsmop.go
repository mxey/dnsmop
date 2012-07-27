package main

import (
	"flag"
	"fmt"
	"github.com/miekg/dns"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"
	"sync"
)

type WorkerInput struct {
	Name string
	Type uint16
}

type WorkerOutput struct {
	Name   string
	Error  error
	Answer []dns.RR
}

type WorkerPool struct {
	Input  chan WorkerInput
	Output chan WorkerOutput
	wg     sync.WaitGroup
}

type DNSError struct {
	Rcode int
}

var dnsConf *dns.ClientConfig
var workerPool *WorkerPool

func newWorkerPool(workers int) *WorkerPool {
	wp := &WorkerPool{}
	wp.Input = make(chan WorkerInput, 100)
	wp.Output = make(chan WorkerOutput, 100)
	for i := 0; i < workers; i++ {
		wp.wg.Add(1)
		go wp.worker()
	}

	return wp
}

func (wp *WorkerPool) Shutdown() {
	close(wp.Input)
	wp.wg.Wait()
	close(wp.Output)
}

func (wp *WorkerPool) worker() {
	c := new(dns.Client)

	for in, ok := <-wp.Input; ok; in, ok = <-wp.Input {
		m := new(dns.Msg)
		m.SetQuestion(in.Name, in.Type)
		m.MsgHdr.RecursionDesired = true

		var r *dns.Msg
		for {
			srv := dnsConf.Servers[rand.Intn(len(dnsConf.Servers))]
			var err error
			r, err = c.Exchange(m, srv+":"+dnsConf.Port)

			if err == nil {
				break
			}
		}

		if r.Rcode == dns.RcodeSuccess {
			wp.Output <- WorkerOutput{Name: in.Name, Answer: r.Answer}
		} else {
			wp.Output <- WorkerOutput{Name: in.Name, Error: &DNSError{Rcode: r.Rcode}}
		}
	}
	wp.wg.Done()
}

func (err *DNSError) Error() string {
	return dns.Rcode_str[err.Rcode]
}

func loadConfigFromServersFile(fn string) error {
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

func loadConfigFromSystem() error {
	c, err := dns.ClientConfigFromFile("/etc/resolv.conf")
	if err == nil {
		dnsConf = c
	}
	return err
}

func usage(w io.Writer) {
	fmt.Fprintln(w, "Usage: dnsmop COMMAND")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "help       print this help")
	fmt.Fprintln(w, "zone       scan a zone for subdomains using a word list")
	fmt.Fprintln(w, "subnet     map a subnet with reverse DNS queries")
	fmt.Fprintln(w, "wildcard   test a domain for a wildcard record")
	fmt.Fprintln(w, "")
}

var flags *flag.FlagSet

func main() {
	if len(os.Args) < 2 {
		usage(os.Stderr)
		os.Exit(1)
	}

	cmd := os.Args[1]
	flags = flag.NewFlagSet(cmd, flag.ExitOnError)
	var srvFn string
	flags.StringVar(&srvFn, "srv-file", "", "File with one DNS server per line")

	var wordsFn string
	switch cmd {
	case "help":
		usage(os.Stdout)
		os.Exit(0)
	case "zone":
		flags.StringVar(&wordsFn, "words-file", "/usr/share/dict/words", "Word list file")
	case "subnet":
	case "wildcard":
	default:
		usage(os.Stderr)
		os.Exit(1)
	}

	flags.Parse(os.Args[2:])

	var err error
	if len(srvFn) > 0 {
		err = loadConfigFromServersFile(srvFn)
	} else {
		err = loadConfigFromSystem()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	workerPool = newWorkerPool(10)

	switch cmd {
	case "zone":
		d := flags.Arg(0)
		zoneCmd(d, wordsFn)
	case "subnet":
		sn := flags.Arg(0)
		subnetCmd(sn)
	case "wildcard":
		d := flags.Arg(0)
		wildcardCmd(d)
	}
}
