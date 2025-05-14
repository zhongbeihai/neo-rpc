package main

import (
	"encoding/json"
	"fmt"
	"log"
	"neo-rpc/internal/codec"
	"neo-rpc/internal/server"
	"net"
	"time"
)

func startServer(addr chan string) {
	// pick a free port
	lis, err := net.Listen("tcp", ":0")
	if err != nil{
		log.Fatal("network error", err)
	}
	log.Println("start rpc server on ", lis.Addr())
	addr <- lis.Addr().String()
	server.AcceptConn(lis)
}

func main(){
	addr := make(chan string)
	go startServer(addr)

	conn, _ := net.Dial("tcp", <-addr)
	defer func ()  {
		_ = conn.Close()
	}()

	time.Sleep(time.Second)

	_ = json.NewEncoder(conn).Encode(server.DefaultOption)
	cc := codec.NewGobCodec(conn)
	for i := range 5{
		h := &codec.Header{
			ServiceMethod: "Neo.Sum",
			Seq: uint64(i),
		}
		_ = cc.Write(h, fmt.Sprintf("neorpc req %d", h.Seq))
		_ = cc.ReadHeader(h)
		var reply string
		_ = cc.ReadBody(&reply)
		log.Println("reply:", reply)
	}
}