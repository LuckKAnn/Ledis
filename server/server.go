package server

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"
)

// Config config
type Config struct {
	Address       string        `yaml:"address"`
	maxConnection uint32        `yaml:"maxConnection"`
	Timeout       time.Duration `yaml:"timeout"`
}

// Handler handler interface
type Handler interface {
	Handler(ctx context.Context, conn net.Conn)
	Close() error
}

func ListenAndCloseWhenSignal(conf *Config, handler Handler) {
	closeChan := make(chan struct{})
	sigChan := make(chan os.Signal)

	signal.Notify(sigChan, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT)

	// send close signal when get os signal
	go func() {
		switch <-sigChan {
		case syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM, syscall.SIGINT:
			closeChan <- struct{}{}
		}
	}()

	listen, err := net.Listen("tcp", conf.Address)
	if err != nil {
		log.Fatal(fmt.Sprintf("starting listen fail : %v", err))
		return
	}
	log.Println(fmt.Sprintf("bind: %s, start listening...", conf.Address))

	ListenAndServe(listen, handler, closeChan)
}

func ListenAndServe(listener net.Listener, handler Handler, closeChan <-chan struct{}) {

	go func() {
		// when close signal
		<-closeChan
		log.Println("close channel execute")
		_ = listener.Close()
		_ = handler.Close()
	}()

	defer func() {
		// close when unExpected close
		// why , may close again
		log.Println("defer close execute")
		_ = listener.Close()
		_ = handler.Close()
	}()
	ctx := context.Background()
	var closeWaitGroup sync.WaitGroup
	for true {
		conn, err := listener.Accept()
		if err != nil {
			//log.Fatal(fmt.Sprintf("accpet error: %v", err))
			break
		}
		closeWaitGroup.Add(1)
		go func() {
			defer func() {
				closeWaitGroup.Done()
			}()
			go handler.Handler(ctx, conn)
		}()
	}
	// how could this execute
	closeWaitGroup.Wait()
}
