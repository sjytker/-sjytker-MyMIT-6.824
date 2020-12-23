package raft

import (
	"math/rand"
	"time"
)


const (
	electionTimeOut = 300 * time.Millisecond
	beatPeriod = 100 * time.Millisecond
	RPCTimeout = 150 * time.Millisecond
	ApplyInterval = 100 * time.Millisecond

)


func max(x int, y int) int{
	if x >= y {
		return x
	}
	return y
}


func min(x int, y int) int {
	if x < y {
		return x
	}
	return y
}



func GetRandElectionTime() time.Duration{
	r := time.Duration(rand.Int63())  % electionTimeOut
	return electionTimeOut + r
}


func (rf *Raft) lastLogTermIndex()(int, int) {
	// DPrintf("getting server %v lastlogTermIndex : %v \n", rf.me, rf.log)
	n := len(rf.log)
	term := rf.log[n - 1].Term
	index := rf.lastSnapshotIndex + len(rf.log) - 1
	return term, index
}


func (rf *Raft) getAEInfo(index int)(prevTerm int, prevIdx int, res []Log) {
	DPrintf("leader %v getting server %v prev ", rf.me, index)
	for i := 0; i < len(rf.peers); i ++ {
		DPrintf("%v %v\n", rf.matchIndex[i], rf.nextIndex[i])
	}

	lastLogTerm, lastLogIndex := rf.lastLogTermIndex()
	if rf.nextIndex[index] <= rf.lastSnapshotIndex {
		//  select any idx > rf.lastSnapshotIndex is ok, since reply will install snapshot
		prevTerm = lastLogTerm
		prevIdx = lastLogIndex
		DPrintf("server %v prevIdx : %v,  lastSnapshotIndex : %v \n", rf.me, prevIdx, rf.lastSnapshotIndex)
		return
	} else {
		prevIdx = rf.nextIndex[index] - 1
		prevTerm = rf.log[prevIdx - rf.lastSnapshotIndex].Term
		res = rf.log[rf.getRealLogIndex(rf.nextIndex[index]) :]
		DPrintf("server %v prevIdx : %v,  lastSnapshotIndex : %v \n", rf.me, prevIdx, rf.lastSnapshotIndex)
	}
	return
}


func (rf *Raft) resetAETimer(index int) {
	rf.appendEntriesTimer[index].Stop()
	rf.appendEntriesTimer[index].Reset(beatPeriod)
}

func (rf *Raft) AENow() {
	for i, _ := range rf.appendEntriesTimer {
		rf.appendEntriesTimer[i].Stop()
		rf.appendEntriesTimer[i].Reset(0)
	}
}


func (rf *Raft) resetElectionTimer() {
	rf.electionTimer.Stop()
	t := GetRandElectionTime()
	rf.electionTimer.Reset(t)
	DPrintf("server %v election time was reset to : %v\n", rf.me, t)
}


func (rf *Raft) Lock(m string) {
	// DPrintf("server %v requesting lock %v, lockseq : %v \n", rf.me, m, rf.LockSeq)

<<<<<<< HEAD
=======

//	DPrintf("server %v requesting lock %v \n", rf.me, m)

>>>>>>> 2694adff741d395474232a36415f966716f74bd9
	rf.mu.Lock()
	rf.LockSeq = append(rf.LockSeq, m)
	//if len(rf.LockSeq) > 1 {
	//	log.Fatal("deadLock!  current LockSeq : ", rf.LockSeq, len(rf.LockSeq))
	//}

//	DPrintf("server %v got lock ---> %v \n", rf.me, m)
	//DPrintf("server %v Lock seq : %v \n", rf.me, rf.lockSeq)
}


func (rf *Raft) Unlock(m string) {

	if m == rf.LockSeq[0] {
		if len(rf.LockSeq) == 1 {
			rf.LockSeq = make([]string, 0)
		} else {
			rf.LockSeq = rf.LockSeq[1:]
		}
	//	DPrintf("server %v releasing %v, LockSeq:%v \n", rf.me, m, rf.LockSeq)
	} else {
	//	DPrintf("server %v unlock error? m = %v, lockseq = %v\n", rf.me, m, rf.LockSeq)
	}
	rf.mu.Unlock()

//	rf.LockSeq = rf.lockSeq[:len(rf.LockSeq) - 1]
//	DPrintf("server %v releasing %v \n", rf.me, m)
}
