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

	"github.com/labstack/echo"

	"github.com/gavrilaf/oauth-test/pkg/httpx"
)

type metricsClient struct {
	m    map[string]int
	lock *sync.Mutex
}

func (c *metricsClient) RecordCount(name string) {
	c.lock.Lock()
	c.m[name] += 1
	c.lock.Unlock()
}

var metrics = &metricsClient{
	m:    map[string]int{},
	lock: &sync.Mutex{},
}

func run(ctx context.Context, id int, timeout time.Duration, provider httpx.TokenProvider) {
	doer := httpx.MakeAuthDoer(http.DefaultClient, provider, metrics)

	for {
		req, err := http.NewRequest("GET", "http://localhost:7575/do", nil)
		if err != nil {
			panic(err.Error())
		}

		resp, err := doer.Do(req)
		if err != nil {
			fmt.Printf("%d failed\n", id)
			metrics.RecordCount("requests-failed")
		} else {
			metrics.RecordCount("requests-succeeded")
			resp.Body.Close()
		}

		select {
		case <-ctx.Done():
			fmt.Printf("worker %d done\n", id)
			return
		case <-time.After(timeout):
			break
		}
	}
}

const workers = 5

func main() {
	provider := httpx.MakeTokenProvider("http://127.0.0.1:7575/auth", metrics)

	ctx, cancelFn := context.WithCancel(context.Background())

	wg := sync.WaitGroup{}

	rand.Seed(time.Now().UnixNano())

	for i := 1; i <= workers; i++ {
		wg.Add(1)
		timeout := time.Duration(rand.Intn(500))*time.Millisecond + 500*time.Millisecond
		go func(i int) {
			run(ctx, i, timeout, provider)
			wg.Done()
		}(i)
	}

	e := echo.New()

	go func() {
		c := make(chan os.Signal, 5)
		signal.Notify(c, os.Interrupt, syscall.SIGTERM)
		<-c

		err := e.Shutdown(ctx)
		if err != nil {
			fmt.Printf("echo server shutdown error, %v\n", err)
		} else {
			fmt.Printf("echo server shutdowned\n")
		}

		cancelFn()
	}()

	e.GET("/metrics", func(c echo.Context) error {
		res := map[string]interface{}{
			"workers": workers,
			"metrics": metrics.m,
		}

		return c.JSON(200, res)
	})

	e.Start(":7676")

	wg.Wait()
	fmt.Println("all workers done")
}
