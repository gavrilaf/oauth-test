package main

import (
	"context"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/gavrilaf/oauth-test/pkg/httpx"
)

func run(ctx context.Context, id int, timeout time.Duration, provider httpx.TokenProvider) {
	doer := httpx.MakeAuthDoer(http.DefaultClient, provider)

	for {
		req, err := http.NewRequest("GET", "http://localhost:7575/do", nil)
		if err != nil {
			panic(err.Error())
		}

		resp, err := doer.Do(req)
		if err != nil {
			fmt.Printf("%d failed\n", id)
		} else {
			//fmt.Printf("%d success\n", id)
			resp.Body.Close()
		}

		select {
		case <- ctx.Done():
			fmt.Printf("worker %d done\n", id)
			return
		case <- time.After(timeout):
			break
		}
	}
}

const workers = 5

func main() {
	provider := httpx.MakeTokenProvider("http://127.0.0.1:7575/auth")

	ctx, cancelFn := context.WithCancel(context.Background())

	wg := sync.WaitGroup{}

	rand.Seed(time.Now().UnixNano())

	for i := 1; i <= workers; i++ {
		wg.Add(1)
		timeout := time.Duration(rand.Intn(500)) * time.Millisecond + 100 * time.Millisecond
		go func(i int) {
			run(ctx, i, timeout, provider)
			wg.Done()
		}(i)
	}

	go func() {
		c := make(chan os.Signal)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c

		cancelFn()
	}()

	wg.Wait()
	fmt.Println("all workers done")
}
