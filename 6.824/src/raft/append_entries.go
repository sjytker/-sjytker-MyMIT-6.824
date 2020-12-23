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
	RPCTimer := time.NewTimer(RPCTimeout)
	defer RPCTimer.Stop()
	// rf.resetElectionTimer()
	for !rf.killed() {

		rf.Lock("lock1 in AEtoPeer")
		if (rf.state != LEADER) {
			rf.resetAETimer(index)
			rf.Unlock("lock1 in AEtoPeer")   // lock1
			return
		}

		rf.resetAETimer(index)
		rf.resetElectionTimer()

		prevLogTerm, prevLogIndex, AELog := rf.getAEInfo(index)
		DPrintf("leader %v log len: %v, log : %v, lastApplied : %v, commitIndex : %v\n", rf.me, len(rf.log), rf.log, rf.lastApplied, rf.commitIndex)
		DPrintf("leader %v currentTerm : %v \n", rf.me, rf.currentTerm)
		DPrintf("leader %v prevLogTerm = %v, prevLogIndex = %v, nextIndex[%v] = %v\n", rf.me, prevLogTerm, prevLogIndex, index, rf.nextIndex[index])

		args := AppendEntriesArgs{
			Term:         rf.currentTerm,
			LeaderId:       rf.me,
			PrevLogIndex: prevLogIndex,
			PrevLogTerm:  prevLogTerm,
			Entries:      AELog,
			LeaderCommit: rf.commitIndex,
		}

		// If rejected:
		// first check NextIndex == -1, means out of order
		// Second, check XLen, if XLen != -1, then use it
		// Third, check XTerm, if XTerm != -1, use it
		// Fourth, leader doesn't have XTerm, use XIndex
		reply := AppendEntriesReply{}
		resCh := make(chan bool)
		rf.Unlock("lock1 in AEtoPeer")
		RPCTimer.Stop()
		RPCTimer.Reset(RPCTimeout)
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
			// stop AE immediately if RPC timeout
			DPrintf("AE timeout, current leader = %v, sending to %v \n", rf.me, index)
			continue
		}
//--------------------------------------------------------------------------------------
		rf.Lock("lock2 in AEtoPeer")
//--------------------------------------------------------------------------------------
		// if AE to another leader, and its term is larger than mine
		if reply.Term > rf.currentTerm {
			rf.changeRole(FOLLOWER)
			rf.resetElectionTimer()
			rf.currentTerm = reply.Term
			rf.persist()
			rf.Unlock("lock2 in AEtoPeer")
			return
		}

		if rf.state != LEADER || rf.currentTerm != args.Term {
			rf.Unlock("lock2 in AEtoPeer")
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
				DPrintf("leader %v's commit index : %v, apply index : %v\n", rf.me, rf.commitIndex, rf.lastApplied)
			}
			rf.Unlock("lock2 in AEtoPeer")
			return
		}

		// AE fail due to log inconsistency
		// or network delay, term has passed
		// due with it according to paper's figure 2
		// optimize the roll back algorithm, instead of -1 per time
		if reply.Success == false {
			if reply.NextIndex == -1 {
				DPrintf("AE out of order due to network delay\n")
				rf.Unlock("lock2 in AEtoPeer")
				return
			}
			XTerm, XIndex, XLen := reply.Term, reply.XIndex, reply.XLen
			DPrintf("In AEtoPeer: XIndex = %v, XTerm = %v, XLen = %v \n", reply.XIndex, reply.XTerm, reply.XLen)

			var NextIndex int
		//	_, lastLogIndex := rf.lastLogTermIndex()
			if XLen != -1 {
				NextIndex = XLen
			} else if XTerm != -1 {
				if idx := rf.containsXTerm(XTerm); idx != -1 {
					NextIndex = idx + rf.lastSnapshotIndex
				} else {
					NextIndex = XIndex
				}
			}

			rf.nextIndex[index] = NextIndex
			rf.matchIndex[index] = NextIndex - 1
			DPrintf("leader %v -> server %v, nextIndex = %v\n", rf.me, index, NextIndex)
			// follower snapshot is left behind, leader send snapshot to bring it up-to-date
			if NextIndex <= rf.lastSnapshotIndex {
				rf.Unlock("lock2 in AEtoPeer")
				rf.InstallSnapshotToPeers(index)    // use go routine?

				return
			}

		}

		rf.Unlock("lock2 in AEtoPeer") // lock2
		// loop until AE returns true
	}
}

func (rf *Raft) AppendEntries(args *AppendEntriesArgs, reply *AppendEntriesReply) {
	rf.Lock("lock in AE")
	defer rf.Unlock("lock in AE")

	DPrintf("server %v receiving AE from %v\n", rf.me, args.LeaderId)

	reply.XTerm, reply.XIndex, reply.XLen = -1, -1, -1

	if args.Term < rf.currentTerm {
		reply.Success = false
		reply.Term = rf.currentTerm
		return
	}

	rf.resetElectionTimer()
	rf.currentTerm = args.Term
	rf.changeRole(FOLLOWER)
	defer rf.persist()

	// heartbeat, no entries
	if len(args.Entries) == 0 {
		//	reply.Success = true
		DPrintf("server %v receive a heartbeat\n", rf.me)
	}

	_, lastLogIndex := rf.lastLogTermIndex()

	if lastLogIndex < args.PrevLogIndex {
		DPrintf("AE fail, args.prevLogIndex >= current log len + lastssIndex")
		// reply.NextIndex = lastLogIndex + 1
		reply.XLen = lastLogIndex + 1
		reply.Success = false
		return
	}

	DPrintf("server %v log len = %v, prevlogIndex = %v, lastSSIndex = %v, lastSSTerm = %v \n", rf.me, len(rf.log), args.PrevLogIndex, rf.lastSnapshotIndex, rf.lastSnapshotTerm)
	// not enough log

	// means this server used to left behind, but now reconnect
	if args.PrevLogIndex < rf.lastSnapshotIndex {
	//	panic("args.PrevLogIndex < rf.lastSnapshotIndex\n")
		reply.NextIndex = rf.lastSnapshotIndex + 1
		reply.Success = false
		return
	}

	entry := rf.log[rf.getRealLogIndex(args.PrevLogIndex)]
	if args.PrevLogIndex == rf.lastSnapshotIndex {  // already back to bottom
		DPrintf("server %v AE already back to bottom, args = %v\n", rf.me, args)
		if rf.CheckIfAEOutOfOrder(args) {
			reply.Success = false
			reply.NextIndex = -1
			return
		} else {
			rf.log = append(rf.log[:1], args.Entries...)
			reply.Success = true
		}
	} else if args.PrevLogTerm == entry.Term {

		if rf.CheckIfAEOutOfOrder(args) {
			// AE packge delayed in network, it is order
			DPrintf("leader %v AE is out of date, server %v discard it\n", args.LeaderId, rf.me)
			reply.Success = false
			reply.NextIndex = -1
		} else {

			DPrintf("AE match success, leader: %v, follower:%v\n", args.LeaderId, rf.me)
			reply.Success = true
			rf.log = rf.log[:rf.getRealLogIndex(args.PrevLogIndex)+1] // trim right
			rf.log = append(rf.log, args.Entries...)
		}
	} else { // args.PrevLogTerm != entry.Term

			reply.Success = false
			// optimize AE rollback
			reply.XTerm = entry.Term
			reply.XIndex = rf.lastSnapshotIndex + 1
			DPrintf("start finding, XIndex = %v, XTerm = %v, XLen = %v \n", reply.XIndex, reply.XTerm, reply.XLen)
			for reply.XIndex - rf.lastSnapshotIndex < len(rf.log) && rf.log[rf.getRealLogIndex(reply.XIndex)].Term != entry.Term {
				reply.XIndex ++
			}
			DPrintf("AE match fail, leader %v term = %v , follower %v term = %v \n", args.LeaderId, args.Term, rf.me, rf.currentTerm)
			DPrintf("entry.term = %v, prevLogTerm = %v, prevLogIndex = %v\n", entry.Term, args.PrevLogTerm, args.PrevLogIndex)
			DPrintf("XIndex = %v, XTerm = %v, XLen = %v \n", reply.XIndex, reply.XTerm, reply.XLen)
	}


	if reply.Success == true && args.LeaderCommit > rf.commitIndex {
		_, lastIndex := rf.lastLogTermIndex()
		rf.commitIndex = min(args.LeaderCommit, lastIndex)
		DPrintf("server %v applying in AE\n", rf.me)
		rf.notifyApplyCh <- struct{}{}
	}


	DPrintf("follower %v's commitIndex : %v, applyIndex : %v\n", rf.me, rf.commitIndex, rf.lastApplied)
	DPrintf("follower %v's log : %v, %v\n", rf.me, len(rf.log), rf.log)
}


func (rf *Raft) LeaderCommit() {
	hasCommit := false
	for i := rf.commitIndex + 1; i <= len(rf.log) + rf.lastSnapshotIndex; i ++ {
		cnt := 0
		for _,t := range rf.matchIndex {
			if t >= i{
				cnt ++
				if cnt > len(rf.peers) / 2 {
					rf.commitIndex = i
					DPrintf("leader %v commit index:%v", rf.me, i)
					break
				}
			}
		}
	}
	if hasCommit {
		DPrintf("server %v applying in commit\n", rf.me)
		rf.notifyApplyCh <- struct{}{}
	}
}