package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"sync"
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

type WorkerPool struct {
	inChan chan interface{}
	wg sync.WaitGroup
}

func NewWorkerPool (workers int) *WorkerPool {
	wp := &WorkerPool{}
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
		ip := in.(net.IP)
		addrs, err := net.LookupAddr(ip.String())
		if err != nil {
			// fmt.Println(err)
		} else {
			fmt.Println(ip, addrs[0])
		}
	}
	wp.wg.Done()
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: reversemop 1.2.3.4/24")
		return
	}
	
	wp := NewWorkerPool(10)
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
