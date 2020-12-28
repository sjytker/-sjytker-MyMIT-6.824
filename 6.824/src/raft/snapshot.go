package raft

import "time"

type InstallSnapshotArgs struct {
	Term int
	LeaderId int
	LastIncludeIndex int
	LastIncludeTerm int
	Data []byte
	Done bool
}

type InstallSnapshotReply struct {
	Term int
}



func (rf *Raft) SaveStateAndSnapshot(index int, snapshot []byte) {

	rf.mu.Lock()
	defer rf.mu.Unlock()
	if index < rf.lastSnapshotIndex {
		return
	}

	lastLog := rf.log[rf.getRealLogIndex(index)]
	rf.log = rf.log[rf.getRealLogIndex(index):]
	rf.lastSnapshotIndex = index
	rf.lastSnapshotTerm = lastLog.Term
	persist := rf.getPersistData()
	rf.persister.SaveStateAndSnapshot(persist, snapshot)
	DPrintf("server %v save SS finish, lastSSIdx = %v, lastSSTerm = %v, log : %v, %v\n", rf.me, rf.lastSnapshotIndex, rf.lastSnapshotTerm, len(rf.log), rf.log)
}


func (rf *Raft) InstallSnapshotToPeers(index int) {



	rf.Lock("InstallSnapshotToPeers")
	args := InstallSnapshotArgs{
		Term:             rf.currentTerm,
		LeaderId:         rf.me,
		LastIncludeIndex: rf.lastSnapshotIndex,
		LastIncludeTerm:  rf.lastSnapshotTerm,
		Data:             rf.persister.ReadSnapshot(),
		Done:             false,
	}
	rf.Unlock("InstallSnapshotToPeers")
	timer := time.NewTimer(RPCTimeout)
	defer timer.Stop()
	for {
		DPrintf("server %v installing snapshot to %v\n", rf.me, index)
		res := make(chan bool)
		reply := InstallSnapshotReply{}
		go func(i int) {
			if ok := rf.peers[i].Call("Raft.InstallSnapshot", &args, &reply); !ok {
				DPrintf("install snapshot RPC to %v fail\n", i)
			} else {
				res <- true
			}
		}(index)

		select {
		case <- res:
			DPrintf("install snapshot RPC to %v finish\n", index)
			rf.nextIndex[index] = rf.lastSnapshotIndex + 1
			rf.matchIndex[index] = rf.lastSnapshotIndex
		case <- timer.C:
			DPrintf("install snapshot RPC to %v RPC timeout\n", index)
			timer.Stop()
			timer.Reset(RPCTimeout)
		case <- rf.stopCh:
			return
		}

		rf.Lock("installSnapshot2\n")

		if reply.Term > rf.currentTerm {
			rf.currentTerm = reply.Term
			rf.changeRole(FOLLOWER)
			rf.resetElectionTimer()
			rf.Unlock("installSnapshot2\n")
			return
		}
		rf.Unlock("installSnapshot2\n")
		return
	}
}


func (rf *Raft) InstallSnapshot(args *InstallSnapshotArgs, reply *InstallSnapshotReply) {

	rf.Lock("InstallSnapshot")
	defer rf.Unlock("InstallSnapshot")
	reply.Term = rf.currentTerm
	DPrintf("server %v receive an installsnapshot, lastIncludeIndex = %v, lastIncludeTerm = %v, lastsnapshotIndex = %v, lastsnapshortTerm = %v\n", rf.me, args.LastIncludeIndex, args.LastIncludeTerm, rf.lastSnapshotIndex, rf.lastSnapshotTerm)

	if args.Term < rf.currentTerm {
		return
	}

	if args.Term > rf.currentTerm {
		rf.changeRole(FOLLOWER)
		rf.currentTerm = args.Term
		rf.resetElectionTimer()
		defer rf.persist()
	}

	if rf.lastSnapshotIndex >= args.LastIncludeIndex {
		return
	}

	offset := args.LastIncludeIndex - rf.lastSnapshotIndex

	if offset >= len(rf.log) {
		// some log is missing, past doesn't matter any more
		rf.log = make([]Log, 1)
		rf.log[0].Term = args.LastIncludeTerm
	} else if offset < len(rf.log) {
		rf.log = rf.log[offset:]
	}

	rf.lastSnapshotIndex = args.LastIncludeIndex
	rf.lastSnapshotTerm = args.LastIncludeTerm
	rf.persister.SaveStateAndSnapshot(rf.getPersistData(), args.Data)
	DPrintf("server %v install SS finish, commit = %v, lastApplied = %v\n", rf.me, rf.commitIndex, rf.lastApplied)
}