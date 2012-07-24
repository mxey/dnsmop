package dnsmop

import "sync"

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
