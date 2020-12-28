package shardkv

import (
	"log"
	"time"
)

//
// Sharded key/value server.
// Lots of replica groups, each running op-at-a-time paxos.
// Shardmaster decides which group serves each shard.
// Shardmaster may change shard assignment from time to time.
//
// You will have to modify these definitions.
//

const Debug = 1
func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug > 0 {
		log.Printf(format, a...)
	}
	return
}

const NShard = 10

const (
	STABLE	   = iota
	TRANSITION
)

const (
	OK             = "OK"
	ErrNoKey       = "ErrNoKey"
	ErrWrongGroup  = "ErrWrongGroup"
	ErrWrongLeader = "ErrWrongLeader"
	ErrTimeOut 	   = "ErrTimeOut"
	PullTimeout		= 100 * time.Millisecond
	WaitTimeout = 500 * time.Millisecond
	FreezeTime = 1000 * time.Millisecond
)

func (kv *ShardKV) Lock(m string) {
	kv.mu.Lock()
	DPrintf("server %v requesting lock %v, lockseq : %v \n", kv.me, m, kv.LockSeq)
	kv.LockSeq = append(kv.LockSeq, m)
}


func (kv *ShardKV) Unlock(m string) {

	kv.LockSeq = kv.LockSeq[1:]
	kv.mu.Unlock()
}



type NotifyMsg struct {
	Err Err
	Value string
}

type Err string

// Put or Append
type PutAppendArgs struct {
	Key      string
	Value    string
	Op       string // "Put" or "Append"
	MsgId    int64
	ClientId int64
	Gid      int
	Shard    int
}

type PutAppendReply struct {
	Err Err
}

type GetArgs struct {
	Key      string
	MsgId    int64
	ClientId int64
	Gid      int
	Shard    int
}

type GetReply struct {
	Err   Err
	Value string
}

type MigrationArgs struct {
	Shard  int
	Data   map[string]string
	NewGid int
	MsgId    int64
	ClientId int64
}

type MigrationReply struct {
	Err
}