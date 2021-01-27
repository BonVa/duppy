package duppy

import (
	"fmt"
	"runtime"
	"sync"
	"time"
)

// noticed:
//	this package is a study case from fasthttp/workerpool.go, and some code has been modified
//	use the actual fasthttp/workerpool.go instead of my study case
//	comments will be Chinese and English mixed, don't read them
//  在这个学习案例中，真正池化的对象并非worker，而是工作台，任务需要放置到工作台上，才会有"工人"去执行
//  至于工人怎么执行，并不关心
//  管理的对象其实是channel
//  暂时觉得这个设计相对来说比较清晰，但不是特别抽象

//worker接收的任务体结构, you don't have this in the origin
type Job interface {
	Run(params interface{}) error
	// if context is needed, you can include the context
}

//worker处理任务的方法
type WorkerFunction func(job Job) error

//worker的工作台
type workerBench struct {
	lastUsedTime time.Time
	bench        chan Job
}

//闲者入池，忙者入台 rest in the pool, work at the bench
type workerPool struct {
	//deal with the "Job"
	WorkerFunc WorkerFunction
	//最大worker数量
	MaxWorkerLimit int
	LogAllErrors   bool
	//最大worker空闲时长
	WorkerMaxIdleDuration time.Duration
	sync.Mutex
	//已有worker数量
	workersCount int
	//结束标记
	stop bool
	//池中worker数量，也就是空闲的worker数量
	inThePool []*workerBench
	//整个worker池的停止信号，该信号的出现意味着要1，结束池子的goroutine；2，结束所有worker的goroutine（需要等待worker手头的工作完成，平滑结束）
	stopSign chan struct{}
	//工作台复用池，与其一直在销毁和创建工作台（worker），不如用sync.pool复用起来，让空闲的worker可以把工作台让出来
	workerBenchCache sync.Pool
}

//todo::解释
var workerBenchSize = func() int {
	if runtime.GOMAXPROCS(0) == 1 {
		return 0
	}
	return 1
}()

//start the pool
func (wp *workerPool) Start() {
	////如果在开启池的时候，发现停止运行的通道非空，那么说明这个池已经在运行了, 比方说你忘记你在别的代码中已经start了
	if wp.stopSign != nil {
		panic("worker pool is running already, you may wanna restart or stop.")
	}
	wp.stopSign = make(chan struct{})
	stopSign := wp.stopSign
	//定义工作台缓存池的New方法
	wp.workerBenchCache.New = func() interface{} {
		return &workerBench{
			bench: make(chan Job, workerBenchSize),
		}
	}
	//并发地去做这件事：
	go func() {
		var recycle []*workerBench
		for {
			//清空空闲超时的worker
			wp.clean(&recycle)
			//阻塞监听通道
			select {
			//如果接收到退出信息，这个并发的goroutine就退出
			case <-stopSign:
				return
			//没有的话，该goroutine休眠一个空转周期
			default:
				time.Sleep(wp.getWorkerMaxIdleDuration())
			}
		}
	}()
}

func (wp *workerPool) Stop() {
	if wp.stopSign == nil {
		panic("pool is not running")
	}
	//先关闭这个通道，这个时候监听者就会发现这个通道已经被关闭了，那么监听者自己就会退出了
	close(wp.stopSign)
	wp.stopSign = nil

	wp.Lock()
	//第一步停掉空闲的worker
	inPool := wp.inThePool
	for i := range inPool {
		//给这些空闲工人发送空任务，并且把空闲栈的这个slot清空
		inPool[i].bench <- nil
		inPool[i] = nil
	}
	//然后把池子的空闲栈置空
	wp.inThePool = inPool[:0]
	//标记这个池子的关闭状态
	wp.stop = true
	wp.Unlock()
	//注意，这个时候池子的整个生态还没有完全关闭，因为worker待会还要回来检查wp的stop标志
	//第二步就是忽略忙碌的worker，他们会在当前任务执行完毕后去检查
}

func (wp *workerPool) Serve(j Job) error {
	ch := wp.getBench(j)
	if ch == nil {
		return fmt.Errorf("all workers are busy at the moment")
	}
	ch.bench <- j
	return nil
}

//清空空闲超时的worker
func (wp *workerPool) clean(recycle *[]*workerBench) {
	//得到允许工人空闲的最大时长
	maxDuration := wp.getWorkerMaxIdleDuration()
	//得到当前时间+最大空闲时长的时间节点，用来判断是否让worker takes a break
	breaking := time.Now().Add(maxDuration)

	//存在竟态，锁一下
	wp.Lock()
	//池中谁人闲得蛋疼
	inPool := wp.inThePool
	n := len(inPool)
	//待我二分便知汝等闲人谁的蛋实在太疼
	l, r, mid := 0, n-1, 0
	for l <= r {
		mid = l + (r-l)>>1
		if breaking.After(wp.inThePool[mid].lastUsedTime) {
			l = mid + 1
		} else {
			r = mid - 1
		}
	}
	i := r
	//没人太疼
	if i == -1 {
		wp.Unlock()
		return
	}
	//这帮人闲了太久，就把你们的指针放进「回收」列表吧
	*recycle = append((*recycle)[:0], inPool[:i+1]...)
	//把闲人列表裁剪一下，因为有一帮worker马上要消失了
	m := copy(inPool, inPool[i+1:])
	for i = m; i < n; i++ {
		inPool[i] = nil
	}
	wp.inThePool = inPool[:m]
	wp.Unlock()
	//问题就解决了吗？ 没有，那些超时worker依然存在cpu上，那么接下来我们要做的就是对他们进行真正的回收
	//这里你好不好奇，为什么不直接range &recycle，底层指向同一个slice，这个变量有何必要呢？
	//答案，如果不用tmp，代码可读性很差  like : (*recycle)[i].bench <- nil
	tmp := *recycle
	for i := range tmp {
		//工人在台上接收到nil的时候就知道自己可以takes a break了
		tmp[i].bench <- nil
		tmp[i] = nil
	}
}

func (wp *workerPool) getWorkerMaxIdleDuration() time.Duration {
	if wp.WorkerMaxIdleDuration <= 0 {
		return 10 * time.Second
	}
	return wp.WorkerMaxIdleDuration
}

func (wp *workerPool) getBench(j Job) *workerBench {
	var ch *workerBench
	createWorker := false
	wp.Lock()
	inPool := wp.inThePool
	n := len(inPool) - 1
	if n < 0 {
		if wp.workersCount < wp.MaxWorkerLimit {
			createWorker = true
			wp.workersCount++
		}
	} else {
		ch = inPool[n]
		inPool[n] = nil
		wp.inThePool = inPool[:n]
	}
	wp.Unlock()
	if ch == nil {
		if !createWorker {
			return nil
		}
		valueCh := wp.workerBenchCache.Get()
		ch = valueCh.(*workerBench)
		go func() {
			defer func() {
				if p := recover(); p != nil {
					fmt.Printf("job running error :%v\n", p)
				}
			}()
			//here needs a logger
			_ = wp.WorkerFunc(j)
			wp.release(ch)
			//执行到这一步就是说工人忙完了，已经进入了ready栈，它的工作台暂时用不上了
			//于是就把这个工作台缓存起来，下一次有工人被分配到任务了，再去缓存池中取台子
			wp.workerBenchCache.Put(ch)
		}()
	}
	return ch
}

func (wp *workerPool) release(bench *workerBench) bool {
	bench.lastUsedTime = time.Now()
	wp.Lock()
	if wp.stop {
		wp.Unlock()
		return false
	}
	wp.inThePool = append(wp.inThePool, bench)
	wp.Unlock()
	return true
}
