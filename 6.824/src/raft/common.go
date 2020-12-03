package raft

import (
	"math/rand"
	"time"
)


const (
	electionTimeOut = 300 * time.Millisecond
	beatPeriod = 150 * time.Millisecond
	RPCTimeout = 100 * time.Millisecond
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
	n := len(rf.log)
	return rf.log[n - 1].Term, n - 1
}


func (rf *Raft) PrevLogTermIndex(index int)(int, int) {
	for i := 0; i < len(rf.peers); i ++ {
		DPrintf("%v %v\n", rf.matchIndex[i], rf.nextIndex[i])
	}
	DPrintf("leader %v getting server %v prev ", rf.me, index)
	prevIdx := rf.nextIndex[index] - 1
	prevTerm := rf.log[prevIdx].Term
	return prevTerm, prevIdx
}


func (rf *Raft) resetAETimer(index int) {
	rf.appendEntriesTimer[index].Stop()
	rf.appendEntriesTimer[index].Reset(beatPeriod)
}


func (rf *Raft) resetElectionTimer() {
	rf.electionTimer.Stop()
	t := GetRandElectionTime()
	rf.electionTimer.Reset(t)
	DPrintf("server %v election time was reset to : %v\n", rf.me, t)
}


func (rf *Raft) Lock(m string) {
	// dt := time.NewTimer(500 * time.Millisecond)
	rf.mu.Lock()
	//rf.lockSeq = append(rf.lockSeq, m)
	//if len(rf.lockSeq) > 1 {
	//	log.Fatal("deadLock!  current lockSeq : ", rf.lockSeq, len(rf.lockSeq))
	//}

//	DPrintf("server %v requesting %v \n", rf.me, m)
//	DPrintf("server %v Lock seq : %v \n", rf.me, rf.lockSeq)
}


func (rf *Raft) Unlock(m string) {

	rf.mu.Unlock()
//	rf.lockSeq = rf.lockSeq[:len(rf.lockSeq) - 1]
	//DPrintf("server %v releasing %v \n", rf.me, m)
}


func (rf *Raft) PrintLog() {
	DPrintf("leader %v log len: %v, log : %v, lastApplied : %v, commitIndex : %v\n", rf.me, len(rf.log), rf.log, rf.lastApplied, rf.commitIndex)
	//DPrintf("other peers' log: \n")
}