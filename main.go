package main

import (
	"fmt"
	raft "github.com/Ahmed1monm/manga-raft/raft"
)

func main() {
	r := raft.NewRaft()
	fmt.Print(r)
	r.Start()
}
