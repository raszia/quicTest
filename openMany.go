package main

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/qtest/client"
)

func openManyConnections() {
	time.Sleep(time.Second * 3)
	ch := make(chan struct{}, 50)
	t := time.NewTicker(20 * time.Second)
	defer t.Stop()

	for {
		select {
		case <-t.C:
			log.Println("openManyConnections done")
			// log.Println("Trigger a full garbage collection cycle")
			return
		default:
			ch <- struct{}{}
			go func() {
				defer func() {
					<-ch
				}()
				b := []byte("hello")
				h := make(http.Header)
				h.Set("k1", "v1")
				req := client.ReqCreate("/",
					http.MethodPost,
					serverAddr).
					SchemeUrlSetHTTPS().
					RetrySet(2).
					AllHeaderSet(h).
					ByteBodySet(b)

				ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
				defer cancel()
				res := req.SendWithContext(ctx)
				if res.ErrGet() != nil {
					return
				}

				_, err := res.ByteBodyGet()
				if err != nil {
					return
				}
				// log.Println(len(x))
			}()
		}
	}

}
