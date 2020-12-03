package raft

//
// this is an outline of the API that raft must expose to
// the service (or tester). see comments below for
// each of these functions for more details.
//
// rf = Make(...)
//   create a new Raft server.
// rf.Start(command interface{}) (index, term, isleader)
//   start agreement on a new log entry
// rf.GetState() (term, isLeader)
//   ask a Raft for its current term, and whether it thinks it is leader
// ApplyMsg
//   each time a new entry is committed to the log, each Raft peer
//   should send an ApplyMsg to the service (or tester)
//   in the same server.
//

import (
	"sync"
	"time"
)
import "sync/atomic"
import "../labrpc"

import "bytes"
import "../labgob"




//
// as each Raft peer becomes aware that successive log entries are
// committed, the peer should send an ApplyMsg to the service (or
// tester) on the same server, via the applyCh passed to Make(). set
// CommandValid to true to indicate that the ApplyMsg contains a newly
// committed log entry.
//
// in Lab 3 you'll want to send other kinds of messages (e.g.,
// snapshots) on the applyCh; at that point you can add fields to
// ApplyMsg, but set CommandValid to false for these other uses.
//
type ApplyMsg struct {
	CommandValid bool
	Command      interface{}
	CommandIndex int
}

type Log struct {
	Command interface{}
	Term int
//	Index int
}

//
// A Go object implementing a single Raft peer.
//
type Raft struct {
	mu        sync.Mutex          // Lock to protect shared access to this peer's state
	peers     []*labrpc.ClientEnd // RPC end points of all peers
	persister *Persister          // Object to hold this peer's persisted state
	me        int                 // this peer's index into peers[]
	dead      int32               // set by Kill()

	state int
	refreshT time.Time     // time refresh by requestRPC, appendEntries
	currentTerm int
	termEnd time.Time
	voteFor int
	log []Log
	commitIndex int
	lastApplied int
	nextIndex []int  // for each server, index of the next log entry to send to that server
	matchIndex []int // for each server, index of highest log entry known to be replicated on server

	electionTimer *time.Timer
	appendEntriesTimer []*time.Timer
	applyTimer *time.Timer
	stopCh chan struct{}
	applyCh chan ApplyMsg
	lockSeq []string
}

// return currentTerm and whether this server
// believes it is the leader.
func (rf *Raft) GetState() (int, bool) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	// Your code here (2A).
	return rf.currentTerm, rf.state == LEADER
}

//
// save Raft's persistent state to stable storage,
// where it can later be retrieved after a crash and restart.
// see paper's Figure 2 for a description of what should be persistent.
//
func (rf *Raft) persist() {
	// Your code here (2C).
	// Example:
	// w := new(bytes.Buffer)
	// e := labgob.NewEncoder(w)
	// e.Encode(rf.xxx)
	// e.Encode(rf.yyy)
	// data := w.Bytes()
	// rf.persister.SaveRaftState(data)
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	e.Encode(rf.currentTerm)
	e.Encode(rf.voteFor)
	e.Encode(rf.log)
	data := w.Bytes()
	rf.persister.SaveRaftState(data)
}


//
// restore previously persisted state.
//
func (rf *Raft) readPersist(data []byte) {
	if data == nil || len(data) < 1 { // bootstrap without any state?
		return
	}
	// Your code here (2C).
	// Example:
	// r := bytes.NewBuffer(data)
	// d := labgob.NewDecoder(r)
	// var xxx
	// var yyy
	// if d.Decode(&xxx) != nil ||
	//    d.Decode(&yyy) != nil {
	//   error...
	// } else {
	//   rf.xxx = xxx
	//   rf.yyy = yyy
	// }
}




//
// example RequestVote RPC arguments structure.
// field names must start with capital letters!
//
type RequestVoteArgs struct {
	// Your data here (2A, 2B).
	Term int
	CandidateId int
	LastLogIndex int
	LastLogterm int
}

//
// example RequestVote RPC reply structure.
// field names must start with capital letters!
//
type RequestVoteReply struct {
	// Your data here (2A).
	Term int
	VoteGranted bool
}

//
// example RequestVote RPC handler.
//
func (rf *Raft) RequestVote(args *RequestVoteArgs, reply *RequestVoteReply) {
	// Your code here (2A, 2B).
	rf.Lock("lock in RE")
	defer rf.Unlock("lock in RE")
	lastTerm, lastIndex := rf.lastLogTermIndex()
	DPrintf("candidate %v's term = %v, my(%v) term = %v\n", args.CandidateId, args.Term, rf.me, rf.currentTerm)
	DPrintf("args.LastLogterm = %v, args.LastLogIndex = %v, lastTerm = %v, lastIndex = %v\n", args.LastLogterm, args.LastLogIndex, lastTerm, lastIndex)
	reply.VoteGranted = false
	if args.Term < rf.currentTerm {
		//	DPrintf("candidate %v's term is smaller than %v, reject \n", args.CandidateId, rf.me)
		return
	}

	if rf.state == LEADER {
		return
	}
	if rf.voteFor == -1 || rf.voteFor == args.CandidateId {
		DPrintf("my(%v) votefor == -1 or has voted for %v, accept\n",rf.me, args.CandidateId)
		rf.currentTerm = args.Term
		rf.state = FOLLOWER
		reply.VoteGranted = true
		rf.voteFor = args.CandidateId
		rf.resetElectionTimer()
	} else if lastTerm < args.LastLogterm || (lastTerm == args.LastLogterm && lastIndex <= args.LastLogIndex) {
		DPrintf("candidate %v log is newer, accept\n", args.CandidateId)
		reply.VoteGranted = true
		rf.state = FOLLOWER
		rf.voteFor = args.CandidateId
		rf.currentTerm = args.Term
		rf.resetElectionTimer()
	}

 	if args.Term > rf.currentTerm {
		rf.currentTerm = args.Term
		rf.state = FOLLOWER
	//	rf.voteFor = -1     // todo: not for sure if it is necessary
	}

}

//
// example code to send a RequestVote RPC to a server.
// server is the index of the target server in rf.peers[].
// expects RPC arguments in args.
// fills in *reply with RPC reply, so caller should
// pass &reply.
// the types of the args and reply passed to Call() must be
// the same as the types of the arguments declared in the
// handler function (including whether they are pointers).
//
// The labrpc package simulates a lossy network, in which servers
// may be unreachable, and in which requests and replies may be lost.
// Call() sends a request and waits for a reply. If a reply arrives
// within a timeout interval, Call() returns true; otherwise
// Call() returns false. Thus Call() may not return for a while.
// A false return can be caused by a dead server, a live server that
// can't be reached, a lost request, or a lost reply.
//
// Call() is guaranteed to return (perhaps after a delay) *except* if the
// handler function on the server side does not return.  Thus there
// is no need to implement your own timeouts around Call().
//
// look at the comments in ../labrpc/labrpc.go for more details.
//
// if you're having trouble getting RPC to work, check that you've
// capitalized all field names in structs passed over RPC, and
// that the caller passes the address of the reply struct with &, not
// the struct itself.
//
func (rf *Raft) sendRequestVote(server int, args *RequestVoteArgs, reply *RequestVoteReply) bool {
	ok := rf.peers[server].Call("Raft.RequestVote", args, reply)
	return ok
}

func (rf *Raft) sendAppendEntries(server int, args *AppendEntriesArgs, reply *AppendEntriesReply) bool {
	ok := rf.peers[server].Call("Raft.AppendEntries", args, reply)
	return ok
}

func (rf *Raft) SendRequest() int{
	rf.Lock("lock in SendRequest") // lock1
	res := 0
	n := len(rf.peers)
	//
	// note : some server might be down, we can't just use waitGroup to wait for all
	// use a time.Timer to timeout
	//
	wg := sync.WaitGroup{}
	wg.Add(n - 1)
	lastTerm, lastIndex := rf.lastLogTermIndex()
	args := RequestVoteArgs{
		Term:         rf.currentTerm,
		CandidateId:  rf.me,
		LastLogIndex: lastIndex,
		LastLogterm:  lastTerm,
	}
	rf.Unlock("lock in SendRequest")  // lock1

	RPCTimer := time.NewTimer(RPCTimeout)
	defer RPCTimer.Stop()
	resCh := make(chan bool)
	go func () {
		for i, _ := range rf.peers {
			//	sig <- 0
			if i == rf.me {
				continue
			}
			go func(i int) {
				reply := RequestVoteReply{}
				if ok := rf.sendRequestVote(i, &args, &reply); ok {
					if reply.VoteGranted == true {
				//		rf.mu.Lock()  // lock2
						res += 1
				//		rf.mu.Unlock()  // lock2
					}
				}
				wg.Done()
			}(i)
		}
		wg.Wait()
		resCh <- true
	}()

	select {
	case <- RPCTimer.C:
		DPrintf("RPCTimeout in SendRequest, stop waitting for down server \n")
	case <- resCh:
		DPrintf("Receive all server in SendRequest")
	}
	return res
}



//
// the service using Raft (e.g. a k/v server) wants to start
// agreement on the next command to be appended to Raft's log. if this
// server isn't the leader, returns false. otherwise start the
// agreement and return immediately. there is no guarantee that this
// command will ever be committed to the Raft log, since the leader
// may fail or lose an election. even if the Raft instance has been killed,
// this function should return gracefully.
//
// the first return value is the index that the command will appear at
// if it's ever committed. the second return value is the current
// term. the third return value is true if this server believes it is
// the leader.
//
func (rf *Raft) Start(command interface{}) (int, int, bool) {
	rf.mu.Lock()
	defer rf.mu.Unlock()
	term := rf.currentTerm
	isLeader := rf.state == LEADER
	_, lastIndex := rf.lastLogTermIndex()
	index := lastIndex + 1
	if !isLeader {
		// DPrintf("putting command, but candidate %v is not a leader", rf.me)
	} else {
		DPrintf("putting command in leader %v \n", rf.me)
		log := Log{
			Command: command,
			Term: rf.currentTerm,
		//	Index: lastIndex +1,
		}
		rf.log = append(rf.log, log)
		rf.nextIndex[rf.me] = len(rf.log)
		rf.matchIndex[rf.me] = lastIndex + 1
	}
	return index, term, isLeader
}


func (rf *Raft) SendHeartBeat() {

	rf.mu.Lock()
	args := AppendEntriesArgs{
		Term:         rf.currentTerm,
		LeaderId:       rf.me,
		PrevLogIndex: 0,
		PrevLogTerm:  0,
		Entries:      nil,
		LeaderCommit: 0,
	}
	rf.mu.Unlock()
	reply := AppendEntriesReply{}
	for i, _ := range rf.peers {
		if i != rf.me {
			go func(i int) {
				rf.sendAppendEntries(i, &args, &reply)
			}(i)
		}
	}
}



//
// the tester doesn't halt goroutines created by Raft after each test,
// but it does call the Kill() method. your code can use killed() to
// check whether Kill() has been called. the use of atomic avoids the
// need for a lock.
//
// the issue is that long-running goroutines use memory and may chew
// up CPU time, perhaps causing later tests to fail and generating
// confusing debug output. any goroutine with a long-running loop
// should call killed() to check whether it should stop.
//
func (rf *Raft) Kill() {
	atomic.StoreInt32(&rf.dead, 1)
	// Your code here, if desired.
	close(rf.stopCh)
}

func (rf *Raft) killed() bool {
	z := atomic.LoadInt32(&rf.dead)
	return z == 1
}




func (rf * Raft) Election() {
	rf.Lock("lock1 in election")  // lock1
	defer rf.resetElectionTimer()
	if rf.state == LEADER {
//		rf.mu.Unlock()  // lock1
		return
	}
	DPrintf("election timeout, starting election for server %v\n", rf.me)
	rf.currentTerm ++

	rf.state = CANDIDATE
	rf.Unlock("lock1 in election")   // lock1
	count := rf.SendRequest()
	DPrintf("candidate %v request count : %v, len(rf.peers) / 2: %v \n", rf.me, count, len(rf.peers) / 2)

	rf.Lock("lock2 in election") // lock2
	if rf.state == CANDIDATE && count >= len(rf.peers) / 2 {
		// leader comes to power
		// initialize all nextIndex values to the index just after the last one in leader's k
		DPrintf("candidate %v comes to power, current term = %v\n", rf.me, rf.currentTerm)
		rf.voteFor = rf.me
		rf.state = LEADER
		n := len(rf.peers)
		_, lastIndex := rf.lastLogTermIndex()
		match := make([]int, n, n)
		next := make([]int, n, n)
		for i := 0; i < n; i ++ {
			next[i] = lastIndex +1
		}
		for i := 0; i < n; i ++ {
			if i == rf.me {
				match[i] = next[i] - 1;
			} else {
				match[i] = 0
			}
		}
		match[rf.me] = lastIndex
		rf.nextIndex = next
		rf.matchIndex = match
	} else {
		rf.currentTerm --
		rf.state = FOLLOWER
	}
	rf.Unlock("lock2 in election")   // lock2
}


func (rf *Raft) Periodic() {

	// append entries to all
	for i, _ := range rf.peers {
		if i == rf.me {
			continue
		}
		go func (index int) {
			for !rf.killed() {
				select {
				case <- rf.appendEntriesTimer[index].C:
				//	DPrintf("AE timeout, server % v is appending AE for %v\n", rf.me, index)
					rf.appendEntriesToPeer(index)
					rf.resetAETimer(index)
				//	DPrintf("server % v finishe append AE for %v\n", rf.me, index)
				case <- rf.stopCh:
					return
				}
			}
		}(i)
	}


	// start election
	go func () {
		for !rf.killed() {
			select {
			case <- rf.electionTimer.C:
				rf.Election()
			case <-rf.stopCh:
				return
			}
		}
	}()

	// apply log
	go func() {
		for {
			select {
			case <-rf.stopCh:
				return
			case <-rf.applyTimer.C:
				rf.Apply()
			}
		}
	}()
}

func (rf *Raft) Apply() {

	rf.Lock("Lock in apply")
	defer rf.Unlock("Lock in apply")
	defer rf.applyTimer.Reset(ApplyInterval)

	tag := false
	DPrintf("sever %v starts to apply \n", rf.me)
	for i := rf.lastApplied + 1; i <= rf.commitIndex; i ++ {
		tag = true
		msg := ApplyMsg{
			CommandValid: true,
			Command:      rf.log[i].Command,
			CommandIndex: i,
		}
		rf.applyCh <- msg
	}
	if !tag {
		DPrintf("sever %v apply nothing \n", rf.me)
	}
	if rf.commitIndex > rf.lastApplied {
		DPrintf("server %v lastApplied = %v, commitIndex = %v, log = %v \n", rf.me, rf.lastApplied, rf.commitIndex, rf.log)
		rf.lastApplied = rf.commitIndex
	}

}

func (rf *Raft) containsXTerm(XTerm int, rightBound int) bool {
	for i := 0; i < rightBound; i ++ {
		if rf.log[i].Term == XTerm {
			return true
		}
	}
	return false
}


//
// the service or tester wants to create a Raft server. the ports
// of all the Raft servers (including this one) are in peers[]. this
// server's port is peers[me]. all the servers' peers[] arrays
// have the same order. persister is a place for this server to
// save its persistent state, and also initially holds the most
// recent saved state, if any. applyCh is a channel on which the
// tester or service expects Raft to send ApplyMsg messages.
// Make() must return quickly, so it should start goroutines
// for any long-running work.
//
func Make(peers []*labrpc.ClientEnd, me int,
	persister *Persister, applyCh chan ApplyMsg) *Raft {
	n := len(peers)
	rf := &Raft{
		mu:          sync.Mutex{},
		peers:       peers,
		persister:   persister,
		me:          me,
		dead:        0,
		state:       FOLLOWER,
		refreshT:    time.Now(),
		currentTerm: 0,
		termEnd:     time.Time{},
		voteFor:     -1,
		log:         nil,
		commitIndex: 0,
		lastApplied: 0,
		nextIndex:   nil,
		matchIndex:  nil,
		electionTimer: nil,
		appendEntriesTimer: make([]*time.Timer, n),
		applyTimer: time.NewTimer(ApplyInterval),
		stopCh: make(chan struct{}),
		applyCh: applyCh,
		lockSeq: make([]string, 0),
	}
	// Your initialization code here (2A, 2B, 2C).

	// initialize from state persisted before a crash
	rf.readPersist(persister.ReadRaftState())

	t := GetRandElectionTime()
	DPrintf("server %v's random election time : %v\n", rf.me, t)
	rf.electionTimer = time.NewTimer(t)
	log := Log{
		Command: "",
		Term:    0,
	}
	rf.log = append(rf.log, log)
	for i, _ := range rf.appendEntriesTimer {
		rf.appendEntriesTimer[i] = time.NewTimer(beatPeriod)
	}
	//rf.nextIndex = next
	DPrintf("There are %v peers \n", n)
	go rf.Periodic()
	return rf
}
