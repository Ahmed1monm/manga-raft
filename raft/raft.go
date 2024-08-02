package raft

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
	term    int
	success bool
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
