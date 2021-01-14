package duppy

import (
	"fmt"
)

type Runner interface {
	// if context is needed, you can include the context
	Run()
}

type Job interface {
	Runner
}

type Worker struct {
	Pool chan chan Job
	Task chan Job
	off  chan struct{}
}

func NewWorker(pool chan chan Job) Worker {
	return Worker{
		Pool: pool,
		Task: make(chan Job),
		off:  make(chan struct{}),
	}
}

func (w *Worker) On() {
	go func() {
		for {
			w.Pool <- w.Task

			select {
			case job := <-w.Task:
				w.ExecuteJob(job)
			case <-w.off:
				return
			}
		}
	}()
}

//todo:: check would this func cause memory leaking
func (w *Worker) Off() {
	go func() {
		w.off <- struct{}{}
	}()
}

func (w *Worker) ExecuteJob(j Job) {
	//平滑处理Panic，防止单个worker崩溃导致全局崩溃
	defer func() {
		if p := recover(); p != nil {
			fmt.Printf("job running error :%v\n", p)
		}
	}()
	j.Run()
}
