package main

import (
	"fmt"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/mediocregopher/radix.v2/redis"
	"github.com/rcrowley/go-metrics"

	"go-jasperlib/jlog"
)

const (
	numThreads = 4
	slave1     = "qa-scl007-009:6381"
	slave2     = "qa-scl007-009:6382"
	slave3     = "qa-scl007-010:6380"
	slave4     = "qa-scl007-010:6381"
)

var doneChan chan bool
var slaveNodes []string
var redisOpts metrics.Timer
var wg sync.WaitGroup
var conns []*redis.Client

func initialize() error {
	doneChan = make(chan bool, numThreads)
	redisOpts = metrics.NewRegisteredTimer("replicateToSlave", metrics.DefaultRegistry)
	slaveNodes = make([]string, numThreads)

	slaveNodes = append(slaveNodes, slave1, slave2, slave3, slave4)

	return nil
}

func stop() {
	for i := 0; i < numThreads; i++ {
		doneChan <- true
	}
}

func querySlave(num int) {
	defer wg.Done()
	redisClient := conns[num]
	if redisClient == nil {
		jlog.Warn("Redis connection is broken, exit")
		return
	}

	for {
		select {
		case <-doneChan:
			jlog.Info("Stop writing...")
			return
		default:
			// continue working
		}

		resp := redisClient.Cmd("GET", "redisReplicationLagTest")
		if resp.Err != nil {
			jlog.Warn(resp.Err.Error())
			continue // continue to try
		}
		if resp.IsType(redis.Nil) {
			continue
		}
		if ans, err := resp.Int(); err == nil && ans == 1 {
			return
		} else if err != nil {
			jlog.Warn(err.Error())
		}
	}
}

func wait() {
	// wait for a signal to shutdown
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		<-signals
		stop()
	}()

	wg.Wait()
}

func main() {
	err := initialize()
	if err != nil {
		jlog.Warn("initialize failed")
		os.Exit(1)
	}
	r, err := redis.Dial("tcp", "qa-scl007-009:6380")
	if err != nil {
		jlog.Warn(err.Error())
		os.Exit(1)
	}
	jlog.Info("program started")
	conns = make([]*redis.Client, 0, numThreads)
	r1, err := redis.Dial("tcp", slave1)
	if err != nil {
		jlog.Warn(err.Error())
		return
	}
	defer r1.Close()
	r2, err := redis.Dial("tcp", slave2)
	if err != nil {
		jlog.Warn(err.Error())
		return
	}
	defer r2.Close()
	r3, err := redis.Dial("tcp", slave3)
	if err != nil {
		jlog.Warn(err.Error())
		return
	}
	defer r3.Close()
	r4, err := redis.Dial("tcp", slave4)
	if err != nil {
		jlog.Warn(err.Error())
		return
	}
	defer r4.Close()
	conns = append(conns, r1, r2, r3, r4)

	err = r.Cmd("SET", "redisReplicationLagTest", 1).Err
	if err != nil {
		jlog.Warn(err.Error())
		return
	}
	wg.Add(numThreads)
	start := time.Now()
	for i := 0; i < numThreads; i++ {
		go querySlave(i)
	}
	wait()
	lag := time.Now().Sub(start)
	jlog.Info(fmt.Sprintf("Replication lag: %v\n", lag))
	err = r.Cmd("DEL", "redisReplicationLagTest").Err
	if err != nil {
		jlog.Warn("Test cleanup faild")
	}
}
