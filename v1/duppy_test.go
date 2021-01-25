package v1

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"
)

type Hello struct {
	ctx context.Context
	Id  int
	Job
}

type CDNHit struct {
	URL string
	Job
}

func (h *Hello) Run() {
	fmt.Printf("hello, I am %d\n", h.Id)
}

func (c *CDNHit) Run() {
	checkHead(c.URL)
}

/*func TestDuppy(t *testing.T) {
	d := NewDispatcher(10)
	d.Run()

	for i:=1; i<=20;i++ {
		job := new(Hello)
		job.Id = i
		d.Queue <- job
	}

	time.Sleep(1 * time.Second)
	d.ShutDown()
}
*/
func TestDuppy2(t *testing.T) {
	d := NewDispatcher(10)
	d.Run()

	for i := 1; i <= 2000; i++ {
		job := new(CDNHit)
		job.URL = "https://www.baidu.com"
		d.Queue <- job
	}
	time.Sleep(10 * time.Second)
	d.ShutDown()
}

func checkHead(url string)  {
	_, _ = http.Head(url)
	fmt.Printf("|")
}
