package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"sync"
	"github.com/miekg/dns"
	"math/rand"
)

var dnsConf *dns.ClientConfig

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

type WorkerPool struct {
	inChan chan interface{}
	wg sync.WaitGroup
	jobFunction func(interface{})
}

func NewWorkerPool (workers int, jobFunction func(interface{})) *WorkerPool {
	wp := &WorkerPool{jobFunction: jobFunction}
	wp.inChan = make(chan interface{}, 100)
	for i := 0; i < workers; i++ {
		wp.wg.Add(1)
		go wp.worker()
	}
	
	return wp
}

func (wp *WorkerPool) AddJob(job interface{}) {
	wp.inChan <- job
}

func (wp *WorkerPool) Shutdown() {
	close(wp.inChan)
	wp.wg.Wait()
}

func (wp *WorkerPool) worker() {
	for in := <- wp.inChan; in != nil; in = <- wp.inChan {
		wp.jobFunction(in)
	}
	wp.wg.Done()
}

func reverseLookupJob(in interface{}) {
	ip := in.(net.IP)
	m := new(dns.Msg)
	c := new(dns.Client)
	
	rname, _ :=  dns.ReverseAddr(ip.String())
	m.SetQuestion(rname, dns.TypePTR)
	m.MsgHdr.RecursionDesired = true
	
	idx := rand.Intn(len(dnsConf.Servers))
	srv := dnsConf.Servers[idx]
	fmt.Println(idx, srv)
	r, err := c.Exchange(m, srv + ":" + dnsConf.Port)
	if err != nil {
		fmt.Println(err)
		return
	}
	if r.Rcode != dns.RcodeSuccess {
		fmt.Println(ip, "failed: ", dns.Rcode_str[r.Rcode])
		return
	}
	
	for _, a := range r.Answer {
		if ptr, ok := a.(*dns.RR_PTR); ok {
			fmt.Println(ip, ptr.Ptr)
		}
	}
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: reversemop 1.2.3.4/24")
		return
	}
	
	var err error
	dnsConf, err = dns.ClientConfigFromFile("/etc/resolv.conf")
	if err != nil {
		fmt.Println(err)
		return
	}
	
		
	wp := NewWorkerPool(10, reverseLookupJob)
	sni, err := NewSubnetIterator(os.Args[1])
	if err != nil {
		fmt.Println(err)
		return
	}

	wp.AddJob(sni.Current)
	for sni.Next() {
		wp.AddJob(sni.Current)
	}
	
	wp.Shutdown()
}
