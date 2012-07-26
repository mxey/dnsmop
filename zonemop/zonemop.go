package main

import (
	"github.com/miekg/dns"
	"flag"
	".."
	"io/ioutil"
	"fmt"
	"strings"
)

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
		for _, w := range(words) {
			rname := w + "." + dom + "."
			if len(w) > 0 {
				wp.Input <- dnsmop.WorkerInput{Name: rname, Type: dns.TypeA}
			}
		}
		wp.Shutdown()
	}()
	
	for out, ok := <- wp.Output; ok; out, ok = <- wp.Output {
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
