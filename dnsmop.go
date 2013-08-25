package main

import (
	"./third_party/github.com/miekg/dns"
	"flag"
	"fmt"
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
	conf   *dns.ClientConfig
}

type DNSError struct {
	Rcode int
}

var workerPool *WorkerPool

func newWorkerPool(workers int) (*WorkerPool, error) {
	wp := &WorkerPool{}
	wp.Input = make(chan WorkerInput, 100)
	wp.Output = make(chan WorkerOutput, 100)
	for i := 0; i < workers; i++ {
		wp.wg.Add(1)
		go wp.worker()
	}

	var err error
	wp.conf, err = dns.ClientConfigFromFile("/etc/resolv.conf");
	return wp, err
}

func (wp *WorkerPool) loadServers(fn string) error {
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
	wp.conf = c
	return nil
}

func (wp *WorkerPool) Shutdown() {
	close(wp.Input)
	wp.wg.Wait()
	close(wp.Output)
}

func (wp *WorkerPool) worker() {
	c := new(dns.Client)

	for in := range wp.Input {
		m := new(dns.Msg)
		m.SetQuestion(in.Name, in.Type)
		m.MsgHdr.RecursionDesired = true

		var r *dns.Msg
		for {
			srv := wp.conf.Servers[rand.Intn(len(wp.conf.Servers))]
			var err error
			r, err = c.Exchange(m, srv+":"+wp.conf.Port)

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


func usage(err bool) {
	var w io.Writer
	var ret int
	if err  {
		w = os.Stderr
		ret = 1
	} else {
		w = os.Stdout
		ret = 0
	}
	
	fmt.Fprintln(w, "Usage: dnsmop COMMAND")
	fmt.Fprintln(w, "")
	fmt.Fprintln(w, "Commands:")
	fmt.Fprintln(w, "help                  print this help")
	fmt.Fprintln(w, "zone example.com      scan a zone for subdomains using a word list")
	fmt.Fprintln(w, "subnet 1.2.3.4/24     map a subnet with reverse DNS queries")
	fmt.Fprintln(w, "wildcard example.com  test a domain for a wildcard record")
	fmt.Fprintln(w, "")
	
	os.Exit(ret)
}

func exit(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}

func main() {
	if len(os.Args) < 2 {
		usage(true)
	}

	cmd := os.Args[1]
	fs := flag.NewFlagSet(cmd, flag.ExitOnError)
	var srvFn string
	fs.StringVar(&srvFn, "srv-file", "", "File with one DNS server per line")

	var wordsFn string
	switch cmd {
	case "help":
		usage(false)
	case "zone":
		fs.StringVar(&wordsFn, "words-file", "/usr/share/dict/words", "Word list file")
	case "subnet":
	case "wildcard":
	default:
		usage(true)
	}

	fs.Parse(os.Args[2:])
	var err error
	workerPool, err = newWorkerPool(10)
	if err != nil {
		exit(err)
	}
	
	if len(srvFn) > 0 {
		if err := workerPool.loadServers(srvFn); err != nil {
			exit(err)
		}
	}

	a := fs.Arg(0)
	if len(a) == 0 {
		usage(true)
	}

	switch cmd {
	case "zone":
		zoneCmd(a, wordsFn)
	case "subnet":
		subnetCmd(a)
	case "wildcard":
		wildcardCmd(a)
	}
}
