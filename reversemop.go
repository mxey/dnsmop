package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"os"
	"sync"
)

var wg sync.WaitGroup

func lookup(in_chan chan net.IP, goroutine_id int) {
	for ip := <-in_chan; ip != nil; ip = <-in_chan {
		addrs, err := net.LookupAddr(ip.String())
		if err != nil {
			// fmt.Println(err)
		} else {
			fmt.Println(ip, addrs[0])
		}
	}
	wg.Done()
}

func ipToInt(addr net.IP) (int32, error) {
	var ip_int int32
	buf := bytes.NewBuffer(addr)
	err := binary.Read(buf, binary.BigEndian, &ip_int)
	return ip_int, err
}

func intToIp(ip_int int32) (net.IP, error) {
	buf := new(bytes.Buffer)
	err := binary.Write(buf, binary.BigEndian, ip_int)
	return buf.Bytes(), err
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: reversemop 1.2.3.4/24")
		return
	}
	
	_, ipnet, err := net.ParseCIDR(os.Args[1])
	if err != nil {
		fmt.Println(err)
		return
	}

	ip := ipnet.IP.Mask(ipnet.Mask)
	ip_int, err := ipToInt(ip)
	if err != nil {
		fmt.Println(err)
		return
	}

	in_chan := make(chan net.IP, 100)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go lookup(in_chan, i)
	}

	for {
		ip_int += 1
		ip, err = intToIp(ip_int)
		if err != nil {
			fmt.Println(err)
			continue
		}

		if !ipnet.Contains(ip) {
			break
		}

		in_chan <- ip
	}
	close(in_chan)
	wg.Wait()
}
