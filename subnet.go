package main

import (
	"fmt"
	"github.com/miekg/dns"
	"net"
)

type SubnetIterator struct {
	Current net.IP
	Net     net.IPNet
}

func NewSubnetIterator(s string) (SubnetIterator, error) {
	sni := SubnetIterator{}
	_, ipnet, err := net.ParseCIDR(s)
	if err != nil {
		return sni, err
	}
	sni.Net = *ipnet

	sni.Current = ipnet.IP.Mask(ipnet.Mask).To16()
	if err != nil {
		return sni, err
	}

	return sni, nil
}

func (sni *SubnetIterator) Next() bool {
	a := sni.Current
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

	if !sni.Net.Contains(a) {
		return false
	}

	sni.Current = a
	return true
}

func subnetCmd(sn string) {
	sni, err := NewSubnetIterator(sn)
	if err != nil {
		fmt.Println(err)
		return
	}

	go func() {
		for {
			rname, _ := dns.ReverseAddr(sni.Current.String())
			workerPool.Input <- WorkerInput{Name: rname, Type: dns.TypePTR}

			if !sni.Next() {
				break
			}
		}
		workerPool.Shutdown()
	}()

	for out, ok := <-workerPool.Output; ok; out, ok = <-workerPool.Output {
		if out.Error != nil {
			fmt.Println(out.Name, out.Error)
		} else {
			for _, a := range out.Answer {
				if ptr, ok := a.(*dns.RR_PTR); ok {
					fmt.Println(out.Name, ptr.Ptr)
				}
			}
		}
	}
}
