package raft

import "time"


type AppendEntriesArgs struct {
	Term int
	LeaderId int
	PrevLogIndex int
	PrevLogTerm int
	Entries []Log
	LeaderCommit int
}

type AppendEntriesReply struct {
	Success bool
	Term int
	XTerm int
	XIndex int
	XLen int
	NextIndex int
}



// if rf is a leader, append entries to peer "index"
func (rf *Raft) appendEntriesToPeer(index int) {

	for !rf.killed() {

		rf.Lock("lock1 in AEtoPeer")
		if (rf.state != LEADER) {
			rf.Unlock("lock1 in AEtoPeer")   // lock1
			return
		}
	//	n := len(rf.peers)
		rf.resetElectionTimer()
		prevLogTerm, prevLogIndex := rf.PrevLogTermIndex(index)
		DPrintf("leader %v log len: %v, log : %v, lastApplied : %v, commitIndex : %v\n", rf.me, len(rf.log), rf.log, rf.lastApplied, rf.commitIndex)
		DPrintf("leader %v currentTerm : %v \n", rf.me, rf.currentTerm)
		DPrintf("leader %v prevTerm = %v, prevIndex = %v, nextIndex[%v] = %v\n", rf.me, prevLogTerm, prevLogIndex, index, rf.nextIndex[index])
		args := AppendEntriesArgs{
			Term:         rf.currentTerm,
			LeaderId:       rf.me,
			PrevLogIndex: prevLogIndex,
			PrevLogTerm:  prevLogTerm,
			Entries:      rf.log[rf.nextIndex[index] : len(rf.log)],
			LeaderCommit: rf.commitIndex,
		}
		reply := AppendEntriesReply{
			Success:   false,
			Term:      0,
			XTerm:     0,
			XIndex:    0,
			XLen:      0,
			NextIndex: 0,
		}
		resCh := make(chan bool)
		RPCTimer := time.NewTimer(RPCTimeout)
		defer RPCTimer.Stop()

		rf.Unlock("lock1 in AEtoPeer")
		go func(args *AppendEntriesArgs, reply *AppendEntriesReply) {
			ok := rf.sendAppendEntries(index, args, reply)
			resCh <- ok
		}(&args, &reply)

		select {
		case ok := <- resCh:
			if !ok {
				DPrintf("CALL AE fail, current leader = %v, sending to %v \n", rf.me, index)
				continue
			}
		case <- rf.stopCh:
			return
		case <- RPCTimer.C:
			DPrintf("AE timeout, current leader = %v, sending to %v \n", rf.me, index)
			RPCTimer.Stop()
			RPCTimer.Reset(RPCTimeout)
			continue
		}

		rf.Lock("lock2 in AEtoPeer")
		// if AE to another leader, and its term is larger than mine
		if reply.Term > rf.currentTerm {
			rf.state = FOLLOWER
			rf.Unlock("lock2 in AEtoPeer")  // lock2
			return
		}

		if reply.Success == true {
			rf.nextIndex[index] = rf.nextIndex[index] + len(args.Entries)
			rf.matchIndex[index] = rf.nextIndex[index] - 1
			if len(args.Entries) > 0 && args.Entries[len(args.Entries) - 1].Term == rf.currentTerm{
				// if there is any new log and append successfully,
				// leader should commit itself
				// followers will commit themselves during the next AE
				rf.LeaderCommit()
			//	rf.Apply()
				DPrintf("leader %v's commit index : %v, apply index : %v\n", rf.me, rf.commitIndex, rf.lastApplied)
			}
			rf.Unlock("lock2 in AEtoPeer")      // lock2
			return
		}

		// AE fail due to log inconsistency
		// due with it according to paper's figure 2
		// optimize the roll back algorithm, instead of -1 per time
		if reply.Success == false {

			XTerm, XIndex, XLen := reply.Term, reply.XIndex, reply.XLen
			DPrintf("In AEtoPeer: XIndex = %v, XTerm = %v, XLen = %v \n", reply.XIndex, reply.XTerm, reply.XLen)
			//NextIndex := XLen
			var NextIndex int
			if XLen <= args.PrevLogIndex {
				NextIndex = XLen
			}
			if !rf.containsXTerm(XTerm, rf.nextIndex[index]) {
			//	NextIndex = min(NextIndex, XIndex)
				NextIndex = XIndex
			} else {
				leftMost := rf.nextIndex[index] - 1
				start := false
				for leftMost > 0 && (!start || rf.log[leftMost - 1].Term == XTerm) {
					if rf.log[leftMost].Term == XTerm {
						start = true
					}
					leftMost --
				}
				NextIndex = leftMost + 1
			}
			rf.nextIndex[index] = NextIndex
			rf.matchIndex[index] = NextIndex - 1
			DPrintf("leader %v -> server %v, nextIndex = %v\n", rf.me, index, NextIndex)
			//rf.nextIndex[index] --
			//rf.matchIndex[index] = rf.nextIndex[index] - 1
		}

		rf.Unlock("lock2 in AEtoPeer") // lock2
		// loop until AE returns true
	}
}

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.Lock("lock in AE")
	defer rf.Unlock("lock in AE")

	if args.Term < rf.currentTerm {
		reply.Success = false
		reply.Term = rf.currentTerm
		return
	}

	if args.Term > rf.currentTerm {
		// if this rf is a leader, but another leader's term is larger.
		//	reply.Success = false
		rf.currentTerm = args.Term
		rf.state = FOLLOWER
		rf.resetElectionTimer()
		DPrintf("leader %v turning to follower in AE\n", rf.me)
	} else if rf.state == CANDIDATE && args.Term == rf.currentTerm {
		//	if this rf is a candidiate, but there is already a leader in the cluster
		DPrintf("---- receive AE from %v when being a candidate, me = %v--------\n", args.LeaderId, rf.me)
		rf.state = FOLLOWER
		//	reply.Success = false
		rf.resetElectionTimer()
	}

	DPrintf("server %v log len :%v, prevlogIndex : %v \n", rf.me, len(rf.log), args.PrevLogIndex)
	// not enough log
	reply.XLen = len(rf.log)
	if len(rf.log) <= args.PrevLogIndex {
		DPrintf("AE fail, args.prevLogIndex >= current log len")
		reply.NextIndex = len(rf.log)
		reply.Success = false
		return
	}

	// heartbeat, no entries
	if len(args.Entries) == 0{
		reply.Success = true
	}

	entry := rf.log[args.PrevLogIndex]

	if entry.Term != args.PrevLogTerm {
		reply.Success = false
		// optimize AE rollback
		reply.XTerm = entry.Term
		reply.XIndex = args.PrevLogIndex
		for reply.XIndex > 0 && rf.log[reply.XIndex - 1].Term == entry.Term {
			reply.XIndex --
		}
		DPrintf("AE match fail, leader %v term = %v , follower %v term = %v \n", args.LeaderId, args.Term, rf.me, rf.currentTerm)
		DPrintf("entry.term = %v, prevLogTerm = %v, prevLogIndex = %v\n", entry.Term, args.PrevLogTerm, args.PrevLogIndex)
		DPrintf("XIndex = %v, XTerm = %v, XLen = %v \n", reply.XIndex, reply.XTerm, reply.XLen)
	} else {
		DPrintf("AE match success, leader: %v, follower:%v \n", args.LeaderId, rf.me)
		reply.Success = true
		rf.resetElectionTimer()
		rf.log = rf.log[:args.PrevLogIndex + 1]   // trim right
		rf.log = append(rf.log, args.Entries...)
		if args.LeaderCommit > rf.commitIndex {
			_, lastIndex := rf.lastLogTermIndex()
			rf.commitIndex = min(args.LeaderCommit, lastIndex)
		}
	}
	// follower commit
	//if args.LeaderCommit > rf.commitIndex {
	//	_, lastIndex := rf.lastLogTermIndex()
	//	rf.commitIndex = min(args.LeaderCommit, lastIndex)
	//}

	DPrintf("follower %v's commitIndex : %v, applyIndex : %v\n", rf.me, rf.commitIndex, rf.lastApplied)
	DPrintf("follower %v's log : %v\n", rf.me, rf.log)
}


func (rf *Raft) LeaderCommit() {

	for i := rf.commitIndex + 1; i <= len(rf.log); i ++ {
		cnt := 0
		for _,t := range rf.matchIndex {
			if t >= i{
				cnt ++
			}
		}
		if cnt > len(rf.peers) / 2 {
			rf.commitIndex = i
		} else {
			break
		}
	}
}