package workpool

import (
	"sync"
	"tool-attendance/log"
)

type Worker interface {
	Task() error
}

type Pool struct {
	work chan Worker
	open bool
	wg   sync.WaitGroup
	mu   sync.RWMutex
}

func New(num int) *Pool {
	p := Pool{
		work: make(chan Worker),
		open: true,
	}
	p.wg.Add(num)
	for i := 0; i < num; i++ {
		go func() {
			for w := range p.work {
				defer func() {
					if err := recover(); err != nil {
						log.Log.Error("task error:", err)
					}
				}()
				if err := w.Task(); err != nil {
					log.Log.Error("task error:", err)
				}
			}
			p.wg.Done()
		}()
	}

	return &p
}

func (p *Pool) SubmitWork(w Worker) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.open {
		p.work <- w
	}

}

func (p *Pool) Shutdown() {
	p.mu.Lock()
	p.open = false
	p.mu.Unlock()
	close(p.work)
	p.wg.Wait()
}
