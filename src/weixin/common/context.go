package common

import (
	"context"
	"os"
	"os/signal"
	"sync"
	"syscall"
)

const ContextKeyName = "data"

type Values struct {
	m map[string]interface{}
}

func (v Values) Get(key string) interface{} {
	return v.m[key]
}

type ServerContext struct {
	ctx    context.Context
	cancel context.CancelFunc
	wait   chan bool
	signal chan os.Signal
	wg     *sync.WaitGroup
	sync.Mutex
}

func NewServerContext() (sc *ServerContext) {
	v := Values{
		m: make(map[string]interface{}),
	}

	parentContext := context.WithValue(context.Background(), ContextKeyName, v)
	ctx, cancel := context.WithCancel(parentContext)

	sc = &ServerContext{
		ctx:    ctx,
		cancel: cancel,
		wait:   make(chan bool),
		signal: make(chan os.Signal),
		wg:     &sync.WaitGroup{},
	}

	signal.Notify(sc.signal, syscall.SIGINT, syscall.SIGTERM)
	return
}

func (sc *ServerContext) Context() context.Context {
	return sc.ctx
}

func (sc *ServerContext) Interrupt() <-chan os.Signal {
	return sc.signal
}

func (sc *ServerContext) Cancel() {
	sc.cancel()
}

func (sc *ServerContext) Quit() <-chan struct{} {
	return sc.ctx.Done()
}

func (sc *ServerContext) Add() {
	sc.wg.Add(1)
}

func (sc *ServerContext) Done() {
	sc.wg.Done()
}

func (sc *ServerContext) Wait() {
	sc.wg.Wait()
}

func (sc *ServerContext) Set(key string, value interface{}) {
	sc.Lock()
	defer sc.Unlock()

	sc.ctx.Value(ContextKeyName).(Values).m[key] = value
}

func (sc *ServerContext) Get(key string) interface{} {
	sc.Lock()
	defer sc.Unlock()

	if v, ok := sc.ctx.Value(ContextKeyName).(Values).m[key]; ok {
		return v
	}

	return nil
}
