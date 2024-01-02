package server

import (
	"bufio"
	"context"
	"github.com/hdt3213/godis/lib/sync/wait"
	"io"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"
)

type Client struct {
	conn    net.Conn
	waiting wait.Wait
}

// implement of Handler
type EchoHandler struct {
	activeConnMap sync.Map
	closingFlag   atomic.Bool
}

func MakeEchoHandler() *EchoHandler {
	return &EchoHandler{}
}
func (client *Client) Close() error {
	// wait data transfer ending
	client.waiting.WaitWithTimeout(10 * time.Second)
	client.conn.Close()
	return nil
}

func (handler *EchoHandler) Handler(ctx context.Context, conn net.Conn) {
	closeFlag := handler.closingFlag.Load()
	if closeFlag {
		// has already close
		// why close here
		conn.Close()
		return
	}
	// pointer
	client := &Client{conn: conn}

	handler.activeConnMap.Store(client, struct{}{})

	reader := bufio.NewReader(conn)
	for {
		// block until get an \n
		readString, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				log.Println("connection close")
				// end here
				handler.activeConnMap.Delete(client)
			} else {
				log.Println(err)
			}
			return
		}
		// prevent to close
		client.waiting.Add(1)

		content := []byte(readString)
		conn.Write(content)
		// free prevent
		client.waiting.Done()
	}
}

func (handler *EchoHandler) Close() error {
	// handler close
	handler.closingFlag.Store(true)
	// traverse active conn map
	handler.activeConnMap.Range(func(key interface{}, val interface{}) bool {
		client := key.(*Client)
		client.Close()
		return true
	})
	return nil
}
