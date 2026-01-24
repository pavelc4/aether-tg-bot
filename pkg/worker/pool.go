package worker

import (
	"log"
	"sync"
)

type Job func() error
type Pool struct {
	maxWorkers int
	jobs       chan Job
	wg         sync.WaitGroup
	stopped    bool
	mu         sync.Mutex
}


func NewPool(maxWorkers int) *Pool {
	p := &Pool{
		maxWorkers: maxWorkers,
		jobs:       make(chan Job, maxWorkers*2),
	}
	p.start()
	return p
}


func (p *Pool) start() {
	for i := 0; i < p.maxWorkers; i++ {
		p.wg.Add(1)
		go p.worker(i)
	}
}


func (p *Pool) worker(id int) {
	defer p.wg.Done()
	for job := range p.jobs {
		if err := job(); err != nil {
			log.Printf("Worker %d: Job failed: %v", id, err)
		}
	}
}

func (p *Pool) Submit(job Job) bool {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.stopped {
		return false
	}

	p.jobs <- job
	return true
}

func (p *Pool) Stop() {
	p.mu.Lock()
	if !p.stopped {
		close(p.jobs)
		p.stopped = true
	}
	p.mu.Unlock()

	p.wg.Wait()
}
