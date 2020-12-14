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
	DPrintf("getting server %v lastlogTermIndex : %v \n", rf.me, rf.log)
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


//	DPrintf("server %v requesting lock %v \n", rf.me, m)

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
