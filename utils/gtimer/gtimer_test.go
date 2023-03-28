package gtimer

import (
	"context"
	"fmt"
	"testing"
	"time"
)

func TestSetInterval(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	SetInterval(time.Second*1, ctx, func() {
		fmt.Println("--------1----")
	})
	time.Sleep(time.Second * 5)
	cancel()
	fmt.Println("--------1---- cancel")
	time.Sleep(time.Second * 5)
}

func TestSetInterval2(t *testing.T) {
	ctx, _ := context.WithCancel(context.Background())
	SetInterval(time.Second*1, ctx, func() {
		fmt.Println("--------2----")
	})
	time.Sleep(time.Second * 5)
	StopAll()
	fmt.Println("--------2---- cancel")
	time.Sleep(time.Second * 5)
}
