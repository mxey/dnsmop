package main

import (
	"github.com/miekg/dns"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
)

func createRequests(rname string) {
	workerPool.Input <- WorkerInput{Name: rname, Type: dns.TypeA}
	workerPool.Input <- WorkerInput{Name: rname, Type: dns.TypeAAAA}
	workerPool.Input <- WorkerInput{Name: rname, Type: dns.TypeMX}
	workerPool.Input <- WorkerInput{Name: rname, Type: dns.TypeTXT}
}

func loadWords(fn string) ([]string, error) {
	b, err := ioutil.ReadFile(fn)
	if err != nil {
		return nil, err
	}
	return strings.Split(string(b), "\n"), nil
}

func zoneCmd(dom, fnWords string) {
	words, err := loadWords(fnWords)
	if err != nil {
		exit(err)
	}

	go func() {
		createRequests(dom + ".")

		for _, w := range words {
			rname := w + "." + dom + "."
			if len(w) == 0 {
				continue
			}

			createRequests(rname)

			for i := 0; i < 20; i++ {
				createRequests(w + strconv.Itoa(i) + "." + dom + ".")
			}

		}
		workerPool.Shutdown()
	}()

	for out, ok := <-workerPool.Output; ok; out, ok = <-workerPool.Output {
		if out.Error == nil {
			for _, a := range out.Answer {
				fmt.Println(a)
			}
		} else {
			if err, ok := out.Error.(*DNSError); !ok || err.Rcode != dns.RcodeNameError {
				fmt.Println(out.Name, out.Error)
			}
		}
	}
}
