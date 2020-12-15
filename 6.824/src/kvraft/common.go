package kvraft

import "time"

const (
	OK             = "OK"
	ErrNoKey       = "ErrNoKey"
	ErrWrongLeader = "ErrWrongLeader"
	ErrTimeOut 	   = "ErrTimeOut"
	ChangeLeaderInterval = 20 * time.Millisecond
	FindLeaderTimeout = 300 * time.Millisecond
	WaitTimeout = 500 * time.Millisecond
)

type Op struct {
	Method string
	Key string
	Value string
	ClientId int64
	MsgId int64
	RequestId int64
}


type Err string

// Put or Append
type PutAppendArgs struct {
	Key   string
	Value string
	Op    string // "Put" or "Append"
	MsgId int64
	ClientId int64
}

type PutAppendReply struct {
	Err Err
}

type GetArgs struct {
	Key   string
	MsgId int64
	ClientId int64
}

type GetReply struct {
	Err   Err
	Value string
	Success bool
	IsLeader bool
}


type GetStateArgs struct {

}


type GetStateReply struct {
	Term int
	IsLeader bool
	LeaderId int
}


type NotifyMsg struct {
	Err Err
	Value string
}
