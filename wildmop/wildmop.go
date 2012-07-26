package main

import (
	".."
	"flag"
	"fmt"
	"github.com/miekg/dns"
	"math/rand"
	"net"
	"os"
)

func randalpha(l int) string {
	alphabet := "ABCDEFJKLMNOPQRSTUVWXYZ"
	buf := make([]byte, l)
	for i := 0; i < l; i++ {
		buf[i] = alphabet[rand.Intn(len(alphabet))]
	}
	return string(buf)
}

func noWildcard(reason string) {
	fmt.Println("no wildcard record" + "(" + reason + ")")
	os.Exit(1)
}

func main() {
	var fn string
	flag.StringVar(&fn, "srv-file", "", "File with one DNS server per line")
	flag.Parse()

	var err error
	if len(fn) > 0 {
		err = dnsmop.LoadConfigFromServersFile(fn)
	} else {
		err = dnsmop.LoadConfigFromSystem()
	}
	if err != nil {
		fmt.Println(err)
		return
	}

	dom := flag.Arg(0)
	if dom == "" {
		fmt.Println("Usage: wildmop DOMAIN")
		return
	}

	wp := dnsmop.NewWorkerPool(10)
	numQueries := 100

	go func() {
		for i := 0; i < numQueries; i++ {
			rname := randalpha(5) + "." + dom + "."
			wp.Input <- dnsmop.WorkerInput{Name: rname, Type: dns.TypeA}
		}
		wp.Shutdown()
	}()

	var prev net.IP

	for out, ok := <-wp.Output; ok; out, ok = <-wp.Output {
		if out.Error == nil {
			for _, a := range out.Answer {
				if rr, ok := a.(*dns.RR_A); ok {
					switch {
					case prev == nil:
						prev = rr.A
					case !rr.A.Equal(prev):
						noWildcard("different addresses:" + prev.String() + " " + rr.A.String())
					}
				}
			}
		} else {
			if err, ok := out.Error.(*dnsmop.DNSError); ok && err.Rcode == dns.RcodeNameError {
				noWildcard("NXDOMAIN: " + out.Name)
			} else {
				fmt.Println(out.Name, out.Error)
			}
		}
	}

	fmt.Println("wildcard")
}
