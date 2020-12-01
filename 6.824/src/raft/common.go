package raft

import (
	"fmt"
	"math/rand"
	"time"
)


const (
	electionTimeOut = 300 * time.Millisecond
	beatPeriod = 150 * time.Millisecond
	RPCTimeout = 100 * time.Millisecond
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
		fmt.Println(rf.matchIndex[i], rf.nextIndex[i])
	}
	fmt.Println("getting server : ", index)
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