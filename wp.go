package dnsmop

import (
	"sync"
	"github.com/miekg/dns"
	"math/rand"
)

type WorkerInput struct {
	Name string
	Type uint16
}

type WorkerOutput struct {
	Name string
	Error error
	Answer []dns.RR
}

type WorkerPool struct {
	Input chan WorkerInput
	Output chan WorkerOutput
	wg sync.WaitGroup
}

type DNSError struct {
	Rcode int
}

func NewWorkerPool (workers int) *WorkerPool {
	wp := &WorkerPool{}
	wp.Input = make(chan WorkerInput, 100)
	wp.Output = make(chan WorkerOutput, 100)
	for i := 0; i < workers; i++ {
		wp.wg.Add(1)
		go wp.worker()
	}

	return wp
}

func (wp *WorkerPool) Shutdown() {
	close(wp.Input)
	wp.wg.Wait()
	close(wp.Output)
}

func (wp *WorkerPool) worker() {
	c := new(dns.Client)

	for in, ok := <- wp.Input; ok; in, ok = <- wp.Input {
		m := new(dns.Msg)		
		m.SetQuestion(in.Name, in.Type)
		m.MsgHdr.RecursionDesired = true
		
		var r *dns.Msg
		for {
			srv := dnsConf.Servers[rand.Intn(len(dnsConf.Servers))]
			var err error
			r, err = c.Exchange(m, srv + ":" + dnsConf.Port)
		
			if err == nil {
				break
			}
		}
		
		if r.Rcode == dns.RcodeSuccess {
			wp.Output <- WorkerOutput{Name: in.Name, Answer: r.Answer}
		} else {
			wp.Output <- WorkerOutput{Name: in.Name, Error: &DNSError{Rcode: r.Rcode}}
		}
	}
	wp.wg.Done()
}

func (err *DNSError) Error() string {
	return dns.Rcode_str[err.Rcode]
}
