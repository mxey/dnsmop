package main

import (
	"github.com/miekg/dns"
	"fmt"
	"net"
)

func incrementIP(a net.IP) net.IP {
	for i := 15; i >= 0; i-- {
		b := a[i]

		if b < 255 {
			a[i] = b + 1

			for ii := i + 1; ii <= 15; ii++ {
				a[ii] = 0
			}

			break
		}
	}
	return a
}

func subnetCmd(sn string) {
	_, ipnet, err := net.ParseCIDR(sn)
	if err != nil {
		exit(err)
	}
	ip := ipnet.IP.Mask(ipnet.Mask).To16()

	go func() {
		for ; ipnet.Contains(ip); ip = incrementIP(ip) {
			rname, _ := dns.ReverseAddr(ip.String())
			workerPool.Input <- WorkerInput{Name: rname, Type: dns.TypePTR}
		}
		workerPool.Shutdown()
	}()

	for out, ok := <-workerPool.Output; ok; out, ok = <-workerPool.Output {
		if out.Error != nil {
			fmt.Println(out.Name, out.Error)
		} else {
			for _, a := range out.Answer {
				if ptr, ok := a.(*dns.PTR); ok {
					fmt.Println(out.Name, ptr.Ptr)
				}
			}
		}
	}
}
