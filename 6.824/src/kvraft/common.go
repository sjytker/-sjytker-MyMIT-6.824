package kvraft

<<<<<<< HEAD
import "time"

=======
>>>>>>> edacab21560e1960d239d963a1287729ab342ea2
const (
	OK             = "OK"
	ErrNoKey       = "ErrNoKey"
	ErrWrongLeader = "ErrWrongLeader"
<<<<<<< HEAD
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


=======
)

>>>>>>> edacab21560e1960d239d963a1287729ab342ea2
type Err string

// Put or Append
type PutAppendArgs struct {
	Key   string
	Value string
	Op    string // "Put" or "Append"
<<<<<<< HEAD
	MsgId int64
	ClientId int64
=======
	// You'll have to add definitions here.
	// Field names must start with capital letters,
	// otherwise RPC will break.
>>>>>>> edacab21560e1960d239d963a1287729ab342ea2
}

type PutAppendReply struct {
	Err Err
}

type GetArgs struct {
<<<<<<< HEAD
	Key   string
	MsgId int64
	ClientId int64
=======
	Key string
	// You'll have to add definitions here.
>>>>>>> edacab21560e1960d239d963a1287729ab342ea2
}

type GetReply struct {
	Err   Err
	Value string
<<<<<<< HEAD
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

=======
}
>>>>>>> edacab21560e1960d239d963a1287729ab342ea2
