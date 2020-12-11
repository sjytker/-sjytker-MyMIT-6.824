package kvraft

import "time"

const (
	OK             = "OK"
	ErrNoKey       = "ErrNoKey"
	ErrWrongLeader = "ErrWrongLeader"
	ErrTimeOut 	   = "ErrTimeOut"
	RPCTimeout = 150 * time.Millisecond
	FindLeaderTimeout = 300 * time.Millisecond
	ApplyTimeout = 300 * time.Millisecond
)

type Op struct {
	Command string
	Key string
	Value string
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

