package raft

import (
	"fmt"
	"time"
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
	term    int
	command interface{}
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
	term         int
	candidateId  int
	lastLogIndex int
	lastLogTerm  int
}

type RequestVoteReply struct {
	term        int
	voteGranted bool
}

type AppendEntriesArgs struct {
	term         int
	leaderId     int
	prevLogIndex int
	prevLogTerm  int
	entries      []LogEntry
	leaderCommit int
}

type AppendEntriesReply struct {
	term            int
	success         bool
	lastCommitIndex int
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

	return raft
}

func (r *Raft) RequestVote(args RequestVoteArgs) (error, RequestVoteReply) {
	reply := RequestVoteReply{}
	reply.term = r.state.currentTerm
	reply.voteGranted = false

	if args.term < r.state.currentTerm {
		return fmt.Errorf("RequestVote: Request term is less than current term"), reply
	}

	if args.term > r.state.currentTerm {
		r.state.currentTerm = args.term
		r.state.votedFor = -1
	}

	if r.state.votedFor == -1 || r.state.votedFor == args.candidateId {
		reply.voteGranted = true
		r.state.votedFor = args.candidateId
	}

	return nil, reply
}

func (r *Raft) AppendEntries(args AppendEntriesArgs) (error, AppendEntriesReply) {
	reply := AppendEntriesReply{}

	reply.term = r.state.currentTerm
	reply.success = false

	if args.term > r.state.currentTerm {
		r.state.currentTerm = args.term
		r.state.votedFor = -1
		r.SwitchState("follower")
	}

	if args.term < r.state.currentTerm {
		return fmt.Errorf("AppendEntries: Request term is less than current term"), reply
	}

	if r.state.commitedIndex != args.prevLogIndex {
		return fmt.Errorf("AppendEntries: Previous log index does not match"), reply
	}

	r.state.log = append(r.state.log, args.entries...)
	r.state.commitedIndex = min(args.leaderCommit, len(r.state.log)-1)
	reply.success = true

	return nil, reply
}

func (r *Raft) StartElection() error {
	fmt.Println("Starting election")
	r.state.currentTerm++
	r.state.votedFor = r.config.nodeId
	r.votesReceived = 1

	lastTermIndex := 0
	if len(r.state.log) > 0 {
		lastTermIndex = r.state.log[len(r.state.log)-1].term
	}

	args := RequestVoteArgs{
		term:         r.state.currentTerm,
		candidateId:  r.config.nodeId,
		lastLogIndex: len(r.state.log) - 1,
		lastLogTerm:  lastTermIndex,
	}

	for i, _ := range r.config.peers {
		if i == r.config.nodeId {
			continue
		}

		err, reply := r.RequestVote(args)
		if err != nil {
			return fmt.Errorf("Error in RequestVote: %s", err.Error())
		}
		if reply.voteGranted {
			r.votesReceived++
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
		lastTermIndex = r.state.log[len(r.state.log)-1].term
	}
	args := AppendEntriesArgs{
		term:         r.state.currentTerm,
		leaderId:     r.config.nodeId,
		prevLogIndex: len(r.state.log) - 1,
		prevLogTerm:  lastTermIndex,
		entries:      make([]LogEntry, 0),
		leaderCommit: r.state.commitedIndex,
	}

	for i, _ := range r.config.peers {
		if i == r.config.nodeId {
			continue
		}

		err, _ := r.AppendEntries(args)
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
