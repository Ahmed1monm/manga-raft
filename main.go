package main

import (
	"fmt"
	"os"

	raft "github.com/Ahmed1monm/manga-raft/raft"
	rpc "github.com/Ahmed1monm/manga-raft/rpc"
)

func main() {
	r := raft.NewRaft()
	fmt.Printf("%+v\n", r)
	server := rpc.NewServer(r)
	go server.Start(os.Getenv("PORT"))
	r.Start()
}
