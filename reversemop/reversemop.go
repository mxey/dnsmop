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

var wg sync.WaitGroup

func lookup(inChan chan net.IP, goroutineId int) {
	for ip := <-inChan; ip != nil; ip = <-inChan {
		addrs, err := net.LookupAddr(ip.String())
		if err != nil {
			// fmt.Println(err)
		} else {
			fmt.Println(ip, addrs[0])
		}
	}
	wg.Done()
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: reversemop 1.2.3.4/24")
		return
	}
	
	inChan := make(chan net.IP, 100)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go lookup(inChan, i)
	}
	
	sni, err := NewSubnetIterator(os.Args[1])
	if err != nil {
		fmt.Println(err)
		return
	}

	inChan <- sni.Current
	for sni.Next() {
		inChan <- sni.Current
	}
	close(inChan)
	wg.Wait()
}
