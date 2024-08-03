package rpc

import (
	"context"
	"fmt"
	"log"
	"net"

	pb "github.com/Ahmed1monm/manga-raft/network/raft"
	r "github.com/Ahmed1monm/manga-raft/raft"
	"google.golang.org/grpc"
)

// Creates an RPC client
type Server struct {
	raft *r.Raft
	pb.UnimplementedRaftServer
}

func NewServer(raft *r.Raft) *Server {
	return &Server{
		raft: raft,
	}
}

func (s *Server) Start(port string) {
	fmt.Println("Starting RPC server")
	// Start the RPC server
	lis, err := net.Listen("tcp", fmt.Sprintf(":%s", port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	grpcServer := grpc.NewServer()
	pb.RegisterRaftServer(grpcServer, s)
	log.Printf("Server listening at %v", lis.Addr())
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func (s *Server) RequestVote(ctx context.Context, args *pb.RequestVoteArgs) (*pb.RequestVoteReply, error) {

	err, requestVoteReply := s.raft.RequestVote(r.RequestVoteArgs{
		Term:         int(args.Term),
		CandidateId:  int(args.CandidateId),
		LastLogIndex: int(args.LastLogIndex),
		LastLogTerm:  int(args.LastLogTerm),
	})

	if err != nil {
		return &pb.RequestVoteReply{}, err
	}

	return &pb.RequestVoteReply{
		Term:        int32(requestVoteReply.Term),
		VoteGranted: requestVoteReply.VoteGranted,
	}, nil
}

func (s *Server) AppendEntries(ctx context.Context, args *pb.AppendEntriesArgs) (*pb.AppendEntriesReply, error) {

	logEntries := make([]r.LogEntry, 0)

	for _, entry := range args.Entries {
		logEntries = append(logEntries, r.LogEntry{
			Term:    int(entry.Term),
			Command: entry.Command,
		})
	}

	err, appendEntriesReply := s.raft.AppendEntries(r.AppendEntriesArgs{
		Term:         int(args.Term),
		LeaderId:     int(args.LeaderId),
		PrevLogIndex: int(args.PrevLogIndex),
		PrevLogTerm:  int(args.PrevLogTerm),
		Entries:      logEntries,
		LeaderCommit: int(args.LeaderCommit),
	})

	if err != nil {
		return &pb.AppendEntriesReply{}, err
	}

	return &pb.AppendEntriesReply{
		Term:    int32(appendEntriesReply.Term),
		Success: appendEntriesReply.Success,
	}, nil
}
