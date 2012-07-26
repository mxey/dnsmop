package main

import (
	".."
	"flag"
	"fmt"
	"github.com/miekg/dns"
	"io/ioutil"
	"strconv"
	"strings"
)

func createRequests(ch chan dnsmop.WorkerInput, rname string) {
	ch <- dnsmop.WorkerInput{Name: rname, Type: dns.TypeA}
	ch <- dnsmop.WorkerInput{Name: rname, Type: dns.TypeAAAA}
	ch <- dnsmop.WorkerInput{Name: rname, Type: dns.TypeMX}
}

func main() {
	var fnConf, fnWords string
	flag.StringVar(&fnConf, "srv-file", "", "File with one DNS server per line")
	flag.StringVar(&fnWords, "words-file", "/usr/share/dict/words", "Word list file")
	flag.Parse()

	var err error
	if len(fnConf) > 0 {
		err = dnsmop.LoadConfigFromServersFile(fnConf)
	} else {
		err = dnsmop.LoadConfigFromSystem()
	}
	if err != nil {
		fmt.Println(err)
		return
	}

	dom := flag.Arg(0)
	if dom == "" {
		fmt.Println("Usage: zonemop DOMAIN")
		return
	}

	b, err := ioutil.ReadFile(fnWords)
	if err != nil {
		fmt.Println(err)
		return
	}

	words := strings.Split(string(b), "\n")

	wp := dnsmop.NewWorkerPool(10)

	go func() {
		createRequests(wp.Input, dom+".")

		for _, w := range words {
			rname := w + "." + dom + "."
			if len(w) == 0 {
				continue
			}

			createRequests(wp.Input, rname)

			for i := 0; i < 20; i++ {
				createRequests(wp.Input, w+strconv.Itoa(i)+"."+dom+".")
			}

		}
		wp.Shutdown()
	}()

	for out, ok := <-wp.Output; ok; out, ok = <-wp.Output {
		if out.Error == nil {
			for _, a := range out.Answer {
				fmt.Println(a)
			}
		} else {
			if err, ok := out.Error.(*dnsmop.DNSError); !ok || err.Rcode != dns.RcodeNameError {
				fmt.Println(out.Name, out.Error)
			}
		}
	}
}
