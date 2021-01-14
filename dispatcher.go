package duppy

import "time"

const Buffer = 20

const MinWorker = 10
const MaxWorker = 100

type Dispatcher struct {
	Workers      []*Worker
	Pool         chan chan Job
	WorkerNumber uint8
	Queue        chan Job
	timeout      time.Duration
}

func NewDispatcher(num uint8) *Dispatcher {
	return &Dispatcher{
		Pool:         make(chan chan Job, num),
		WorkerNumber: num,
		Queue:        make(chan Job),
	}
}

func (d *Dispatcher) Run() {
	for i := uint8(1); i <= d.WorkerNumber; i++ {
		w := NewWorker(d.Pool)
		w.On()
		d.Workers = append(d.Workers, &w)
	}
	go d.dispatch()
}

func (d *Dispatcher) dispatch() {
	for {
		select {
		case job := <-d.Queue:
			go func(job Job) {
				whoIsFree := <-d.Pool
				whoIsFree <- job
			}(job)
		}
	}
}

func (d *Dispatcher) SetTimeout(duration time.Duration) {

}

func (d *Dispatcher) ShutDown() {
	for _, w := range d.Workers {
		w.Off()
	}
}
