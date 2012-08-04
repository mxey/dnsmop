package main

import (
	"./dns"
	"fmt"
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

func wildcardCmd(dom string) {
	numQueries := 100

	go func() {
		for i := 0; i < numQueries; i++ {
			rname := randalpha(5) + "." + dom + "."
			workerPool.Input <- WorkerInput{Name: rname, Type: dns.TypeA}
		}
		workerPool.Shutdown()
	}()

	var prev net.IP

	for out, ok := <-workerPool.Output; ok; out, ok = <-workerPool.Output {
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
			if err, ok := out.Error.(*DNSError); ok && err.Rcode == dns.RcodeNameError {
				noWildcard("NXDOMAIN: " + out.Name)
			} else {
				fmt.Println(out.Name, out.Error)
			}
		}
	}

	fmt.Println("wildcard")
}
