package rpc_client

import (
	"context"
	"log"
	"time"

	pb "github.com/Ahmed1monm/manga-raft/network/raft"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func GetClient(addr string, runnable func(context.Context, pb.RaftClient) error) error {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := pb.NewRaftClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err = runnable(ctx, c)
	if err != nil {
		return err
	}
	return nil
}
