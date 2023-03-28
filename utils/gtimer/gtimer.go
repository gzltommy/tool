package gtimer

import (
	"context"
	"fmt"
	"os"
	"sync/atomic"
	"time"
)

type JobFunc = func()

var gCtx, gCancel = context.WithCancel(context.Background())

func SetInterval(interval time.Duration, ctx context.Context, job JobFunc) {
	var j = jobIm{
		quit:     0,
		interval: interval,
		ctx:      ctx,
		job:      job,
	}
	j.run()
}

// StopAll TODO:支持同步操作（这里只是触发了停止 ticker 任务，并未等待所有的任务退出，是一个异步操作）
func StopAll() {
	gCancel()
}

type jobIm struct {
	quit     int32
	interval time.Duration
	ctx      context.Context
	job      JobFunc
}

func (j *jobIm) run() {
	go func() {
		for !j.isQuit() {
			j.wrapperJob()
		}
	}()
}

func (j *jobIm) isQuit() bool {
	return atomic.LoadInt32(&j.quit) > 0
}

func (j *jobIm) normalQuit() {
	atomic.StoreInt32(&j.quit, 1)
}

func (j *jobIm) wrapperJob() {
	t := time.NewTicker(j.interval)
	defer func() {
		t.Stop()
		if err := recover(); err != nil {
			fmt.Fprintf(os.Stderr, "gtimer run job panic recover.err:%v", err)
		}
	}()
	for {
		select {
		case <-gCtx.Done():
			j.normalQuit()
			return
		case <-j.ctx.Done():
			j.normalQuit()
			return
		case <-t.C:
			j.job()
		}
	}
}
