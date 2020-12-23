package shardmaster

import (
	"log"
	"time"
	"../labgob"
)

//
// Master shard server: assigns shards to replication groups.
//
// RPC interface:
// Join(servers) -- add a set of groups (gid -> server-list mapping).
// Leave(gids) -- delete a set of groups.
// Move(shard, gid) -- hand off one shard from current owner to gid.
// Query(num) -> fetch Config # num, or latest config if num==-1.
//
// A Config (configuration) describes a set of replica groups, and the
// replica group responsible for each shard. Configs are numbered. Config
// #0 is the initial configuration, with no groups and all shards
// assigned to group 0 (the invalid group).
//
// You will need to add fields to the RPC argument structs.
//

const Debug = 1
func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug > 0 {
		log.Printf(format, a...)
	}
	return
}


// The number of shards.
const NShards = 10

// A configuration -- an assignment of shards to groups.
// Please don't change this.
type Config struct {
	Num    int              // config number
	Shards [NShards]int     // shard -> gid
	Groups map[int][]string // gid -> servers[]
}


const (
	OK = "OK"
	ErrNoKey       = "ErrNoKey"
	ErrWrongLeader = "ErrWrongLeader"
	ErrTimeOut 	   = "ErrTimeOut"
	WaitTimeout = 500 * time.Millisecond
)

type Err string

type JoinArgs struct {
	Servers map[int][]string // new GID -> servers mappings
	ClientId int64
	MsgId int64
}

type JoinReply struct {
	WrongLeader bool
	Err         Err
}

type LeaveArgs struct {
	GIDs []int
	ClientId int64
	MsgId int64
}

type LeaveReply struct {
	WrongLeader bool
	Err         Err
}

type MoveArgs struct {
	Shard int
	GID   int
	ClientId int64
	MsgId int64
}

type MoveReply struct {
	WrongLeader bool
	Err         Err
}

type QueryArgs struct {
	Num int // desired config number
	ClientId int64
	MsgId int64
}

type QueryReply struct {
	WrongLeader bool
	Err         Err
	Config      Config
}


func (c Config) copy() Config {
	res := Config{
		Num:    c.Num,
		Shards: c.Shards,
		Groups: make(map[int][]string),
	}
	for k, v := range c.Groups {
		res.Groups[k] = v
	}
	return res
}


func init() {
	labgob.Register(map[int][]string{})
	labgob.Register(Op{})
	labgob.Register(JoinArgs{})
	labgob.Register(LeaveArgs{})
	labgob.Register(MoveArgs{})
	labgob.Register(QueryArgs{})

	labgob.Register(JoinReply{})
	labgob.Register(LeaveReply{})
	labgob.Register(MoveReply{})
	labgob.Register(QueryReply{})
}



func (sm *ShardMaster) Lock(m string) {
	// DPrintf("server %v requesting lock %v, lockseq : %v \n", sm.me, m, sm.LockSeq)
	sm.mu.Lock()
	sm.LockSeq = append(sm.LockSeq, m)
	//if len(rf.LockSeq) > 1 {
	//	log.Fatal("deadLock!  current LockSeq : ", rf.LockSeq, len(rf.LockSeq))
	//}

	//	DPrintf("server %v got lock ---> %v \n", rf.me, m)
	//DPrintf("server %v Lock seq : %v \n", rf.me, rf.lockSeq)
}


func (rf *ShardMaster) Unlock(m string) {

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
