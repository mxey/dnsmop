package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"github.com/miekg/dns"
	"flag"
	".."
)

type SubnetIterator struct {
	Current net.IP
	Net net.IPNet
}

func NewSubnetIterator(s string) (SubnetIterator, error) {
	sni := SubnetIterator{}
	_, ipnet, err := net.ParseCIDR(s)
	if err != nil {
		return sni, err
	}
	sni.Net = *ipnet

	sni.Current = ipnet.IP.Mask(ipnet.Mask)
	if err != nil {
		return sni, err
	}
	
	return sni, nil
}

func (sni *SubnetIterator) Next() bool {
	var intIP int32
	buf := bytes.NewBuffer(sni.Current)
	if err := binary.Read(buf, binary.BigEndian, &intIP); err != nil {
		panic("Internal error converting IP to integer")
	}
	intIP += 1

	buf = new(bytes.Buffer)
	if err := binary.Write(buf, binary.BigEndian, intIP); err != nil {
		panic("Internal error converting integer to IP")
	}
	ip := buf.Bytes()
	
	if !sni.Net.Contains(ip) {
		return false
	}

	sni.Current = ip
	return true
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
	
	sn := flag.Arg(0)
	if sn == "" {
		fmt.Println("Usage: reversemop 1.2.3.4/24")
		return
	}			
	sni, err := NewSubnetIterator(sn)
	if err != nil {
		fmt.Println(err)
		return
	}

	wp := dnsmop.NewWorkerPool(10)
	
	go func() { 
		for {
			rname, _ :=  dns.ReverseAddr(sni.Current.String())
			wp.Input <- dnsmop.WorkerInput{Name: rname, Type: dns.TypePTR}
			
			if !sni.Next() {
				break
			}
		}
		wp.Shutdown()
	}()
	
	for out, ok := <- wp.Output; ok; out, ok = <- wp.Output {
		if (out.Error != nil) {
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
