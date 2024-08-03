package raft

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"strconv"
	"time"

	pb "github.com/Ahmed1monm/manga-raft/network/raft"
	"github.com/Ahmed1monm/manga-raft/rpc/rpc_client"
)

const (
	FOLLOWER  = 0
	CANDIDATE = 1
	LEADER    = 2
)

type Config struct {
	clusterSize       int
	port              int
	heartbeatInterval int
	electionTimeout   int
	peers             []string // List of peer addresses
	nodeId            int
}

type LogEntry struct {
	Term    int
	Command interface{}
}

type State struct {
	currentTerm int
	votedFor    int
	log         []LogEntry

	// Volatile state on all servers
	commitedIndex int
	lastApplied   int

	// Volatile state on leaders
	nextIndex  []int
	matchIndex []int
}

type Raft struct {
	state         State
	nodeType      int
	votesReceived int
	config        Config
}

type RequestVoteArgs struct {
	Term         int
	CandidateId  int
	LastLogIndex int
	LastLogTerm  int
}

type RequestVoteReply struct {
	Term        int
	VoteGranted bool
}

type AppendEntriesArgs struct {
	Term         int
	LeaderId     int
	PrevLogIndex int
	PrevLogTerm  int
	Entries      []LogEntry
	LeaderCommit int
}

type AppendEntriesReply struct {
	Term            int
	Success         bool
	LastCommitIndex int
}

func NewRaft() *Raft {
	raft := new(Raft)
	raft.state = State{
		currentTerm:   0,
		votedFor:      -1,
		log:           make([]LogEntry, 0),
		commitedIndex: 0,
		lastApplied:   0,
		nextIndex:     make([]int, 0),
		matchIndex:    make([]int, 0),
	}
	raft.nodeType = FOLLOWER
	raft.votesReceived = 0

	port, err := strconv.Atoi(os.Getenv("PORT"))
	if err != nil {
		fmt.Println("Error in getting port from env, defaulting to 50051")
		port = 50051
	}

	nodeId, err := strconv.Atoi(os.Getenv("NODE_ID"))
	if err != nil {
		fmt.Println("Error in getting node id from env, defaulting to 0")
		nodeId = 0
	}

	min := 50
	max := 100
	heartbeatInterval := rand.Intn(max-min+1) + min

	raft.config = Config{
		clusterSize:       3,
		port:              port,
		heartbeatInterval: heartbeatInterval,
		electionTimeout:   300,
		peers:             []string{"node1:50051", "node2:50052", "node3:50053"},
		nodeId:            nodeId,
	}

	return raft
}

func (r *Raft) RequestVote(args RequestVoteArgs) (error, RequestVoteReply) {
	reply := RequestVoteReply{}
	reply.Term = r.state.currentTerm
	reply.VoteGranted = false

	if args.Term < r.state.currentTerm {
		return fmt.Errorf("RequestVote: Request term is less than current term %d and %d", args.Term, r.state.currentTerm), reply
	}

	if args.Term > r.state.currentTerm {
		r.state.currentTerm = args.Term
		r.state.votedFor = -1
	}

	if r.state.votedFor == -1 || r.state.votedFor == args.CandidateId {
		reply.VoteGranted = true
		r.state.votedFor = args.CandidateId
	}

	return nil, reply
}

func (r *Raft) AppendEntries(args AppendEntriesArgs) (error, AppendEntriesReply) {
	reply := AppendEntriesReply{}

	reply.Term = r.state.currentTerm
	reply.Success = false

	if args.Term > r.state.currentTerm {
		r.state.currentTerm = args.Term
		r.state.votedFor = -1
		r.SwitchState("follower")
	}

	if args.Term < r.state.currentTerm {
		return fmt.Errorf("AppendEntries: Request term is less than current term"), reply
	}

	if r.state.commitedIndex != args.PrevLogIndex {
		return fmt.Errorf("AppendEntries: Previous log index does not match, %d and %d", r.state.commitedIndex, args.PrevLogIndex), reply
	}

	r.state.log = append(r.state.log, args.Entries...)
	r.state.commitedIndex = min(args.LeaderCommit, max(len(r.state.log)-1, 0))
	reply.Success = true

	return nil, reply
}

func (r *Raft) StartElection() error {
	fmt.Println("Starting election")
	r.state.currentTerm++
	r.state.votedFor = r.config.nodeId
	r.votesReceived = 1

	lastTermIndex := 0
	if len(r.state.log) > 0 {
		lastTermIndex = r.state.log[len(r.state.log)-1].Term
	}

	args := RequestVoteArgs{
		Term:         r.state.currentTerm,
		CandidateId:  r.config.nodeId,
		LastLogIndex: max(len(r.state.log)-1, 0),
		LastLogTerm:  lastTermIndex,
	}

	for i, addr := range r.config.peers {
		if i == r.config.nodeId {
			continue
		}

		err := rpc_client.GetClient(addr, func(ctx context.Context, c pb.RaftClient) error {
			reply, err := c.RequestVote(ctx, &pb.RequestVoteArgs{
				Term:         int32(args.Term),
				CandidateId:  int32(args.CandidateId),
				LastLogIndex: int32(args.LastLogIndex),
				LastLogTerm:  int32(args.LastLogTerm),
			})

			if err != nil {
				return err
			}

			if reply.VoteGranted {
				r.votesReceived++
			}
			return nil
		})

		if err != nil {
			return fmt.Errorf("Error in RequestVote: %s", err.Error())
		}
	}

	if r.votesReceived > r.config.clusterSize/2 {
		r.nodeType = LEADER
	}
	return nil
}

func (r *Raft) StartHeartbeat() error {
	fmt.Println("Starting heartbeat")
	lastTermIndex := 0
	if len(r.state.log) > 0 {
		lastTermIndex = r.state.log[len(r.state.log)-1].Term
	}
	args := AppendEntriesArgs{
		Term:         r.state.currentTerm,
		LeaderId:     r.config.nodeId,
		PrevLogIndex: max(len(r.state.log)-1, 0),
		PrevLogTerm:  lastTermIndex,
		Entries:      make([]LogEntry, 0),
		LeaderCommit: r.state.commitedIndex,
	}

	for i, addr := range r.config.peers {
		if i == r.config.nodeId {
			continue
		}

		err := rpc_client.GetClient(addr, func(ctx context.Context, c pb.RaftClient) error {
			_, err := c.AppendEntries(ctx, &pb.AppendEntriesArgs{
				Term:         int32(args.Term),
				LeaderId:     int32(args.LeaderId),
				PrevLogIndex: int32(args.PrevLogIndex),
				PrevLogTerm:  int32(args.PrevLogTerm),
				Entries:      make([]*pb.LogEntry, 0),
				LeaderCommit: int32(args.LeaderCommit),
			})
			if err != nil {
				return err
			}
			return nil
		})

		if err != nil {
			return fmt.Errorf("error in AppendEntries: %s", err.Error())
		}

	}
	return nil
}

func (r *Raft) SwitchState(state string) {
	fmt.Println("Switching state to", state)
	switch state {
	case "follower":
		r.nodeType = FOLLOWER
		break
	case "candidate":
		r.nodeType = CANDIDATE
		break
	case "leader":
		r.nodeType = LEADER
		break
	default:
		r.nodeType = FOLLOWER
	}
}

func (r *Raft) Start() {
	fmt.Println("Starting Raft node")
	for {
		switch r.nodeType {
		case FOLLOWER:
			// Start election timer
			// If election timer times out, start election
			time.Sleep(time.Duration(r.config.electionTimeout) * time.Millisecond)
			r.SwitchState("candidate")
		case CANDIDATE:
			// Start election
			err := r.StartElection()
			if err != nil {
				fmt.Println(err.Error())
			}
		case LEADER:
			for {
				time.Sleep(time.Duration(r.config.heartbeatInterval) * time.Millisecond)
				err := r.StartHeartbeat()
				if err != nil {
					fmt.Println(err.Error())
				}
			}
		default:
			r.SwitchState("follower")
		}
	}
}
