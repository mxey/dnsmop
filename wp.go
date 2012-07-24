package dnsmop

import "sync"

type WorkerPool struct {
	Input chan interface{}
	wg sync.WaitGroup
	jobFunction func(interface{})
}

func NewWorkerPool (workers int, jobFunction func(interface{})) *WorkerPool {
	wp := &WorkerPool{jobFunction: jobFunction}
	wp.Input = make(chan interface{}, 100)
	for i := 0; i < workers; i++ {
		wp.wg.Add(1)
		go wp.worker()
	}

	return wp
}

func (wp *WorkerPool) Shutdown() {
	close(wp.Input)
	wp.wg.Wait()
}

func (wp *WorkerPool) worker() {
	for in := <- wp.Input; in != nil; in = <- wp.Input {
		wp.jobFunction(in)
	}
	wp.wg.Done()
}
