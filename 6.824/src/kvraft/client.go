package kvraft

import (
	"../labrpc"
	"sync"
	"sync/atomic"
	"time"
)
import "crypto/rand"
import "math/big"


type Clerk struct {
	servers []*labrpc.ClientEnd
	// You will have to modify this struct.
	dead   int32
	leaderId int
	clientId int64
	leaderMe int
	mu     sync.Mutex
	stopCh chan struct{}

	findLeaderTimer *time.Timer
	Term            int
}

func nrand() int64 {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := rand.Int(rand.Reader, max)
	x := bigx.Int64()
	return x
}

func MakeClerk(servers []*labrpc.ClientEnd) *Clerk {
	ck := Clerk{
		servers:         servers,
		dead:            0,
		leaderId:          -1,
		clientId:  		 nrand(),
		stopCh:          make(chan struct{}),
		mu:      		 sync.Mutex{},
		findLeaderTimer: time.NewTimer(FindLeaderTimeout),
		Term:            0,
	}
	// You'll have to add code here.
	go ck.periodic()
	return &ck
}


func(ck *Clerk) genMsgId() int64 {
	return nrand()
}


func (ck *Clerk) Kill() {
	atomic.StoreInt32(&ck.dead, 1)
	ck.Kill()
	close(ck.stopCh)
}

func (kv *Clerk) killed() bool {
	z := atomic.LoadInt32(&kv.dead)
	return z == 1
}
//
// fetch the current value for a key.
// returns "" if the key does not exist.
// keeps trying forever in the face of all other errors.
//
// you can send an RPC with code like this:
// ok := ck.servers[i].Call("KVServer.Get", &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.
//
func (ck *Clerk) Get(key string) string {

	DPrintf("client %v receive a get command\n", ck.clientId)

	args := GetArgs{
		Key:      key,
		MsgId:    nrand(),
		ClientId: ck.clientId,
	}

	for !ck.killed() {
		reply := GetReply{}
		leaderId := ck.leaderId
		DPrintf("client %v think current leader is : %v, leaderMe %v: \n", ck.clientId, leaderId, ck.leaderMe)
		if leaderId == -1 {
			time.Sleep(100 * time.Millisecond)
		} else if ok := ck.servers[leaderId].Call("KVServer.Get", &args, &reply); !ok {
			DPrintf("client rpc fail in KVServer.Get, retry\n")
			time.Sleep(50 * time.Millisecond)
			continue
		}

		switch reply.Err {
		case OK:
			DPrintf("client %v get success\n", ck.clientId)
			return reply.Value
		case ErrWrongLeader:
			DPrintf("client %v get sent to deposed leader\n", ck.clientId)
			ck.findLeaderNow()
			for ck.leaderId == -1 {
				time.Sleep(50 * time.Millisecond)
			}
		case ErrNoKey:
			DPrintf("client %v get got nokey err\n", ck.clientId)
			return ""
		case ErrTimeOut:
			DPrintf("client %v get ErrTimeOut, args: %v\n", ck.clientId, args)
			ck.findLeaderNow()
			for ck.leaderId == -1 {
				time.Sleep(50 * time.Millisecond)
			}
		default:
			DPrintf("client %v get rpc fail\n", ck.clientId)
		}
	}

	return ""
}

//
// shared by Put and Append.
//
// you can send an RPC with code like this:
// ok := ck.servers[i].Call("KVServer.PutAppend", &args, &reply)
//
// the types of args and reply (including whether they are pointers)
// must match the declared types of the RPC handler function's
// arguments. and reply must be passed as a pointer.
//
func (ck *Clerk) PutAppend(key string, value string, op string) {
	// You will have to modify this function.

	if op == "Put" {
		DPrintf("client %v receive a Put command\n", ck.clientId)
	} else {
		DPrintf("client %v receive a Append command\n", ck.clientId)
	}

	args := PutAppendArgs{
		Key:   key,
		Value: value,
		Op:    op,
		MsgId:    ck.genMsgId(),
		ClientId: ck.clientId,
	}

	for !ck.killed() {
		reply := PutAppendReply{}
		leaderId := ck.leaderId
		DPrintf("client %v think current leader is : %v, leaderMe %v: \n", ck.clientId, leaderId, ck.leaderMe)
		if leaderId == -1 {
			time.Sleep(100 * time.Millisecond)
		} else if ok := ck.servers[leaderId].Call("KVServer.PutAppend", &args, &reply); !ok {
			DPrintf("clerk rpc fail in KVServer.PutAppend, args : %v\n", args)
		}

		switch reply.Err {
		case OK:
			DPrintf("client %v putAppend success, args: %v\n", ck.clientId, args)
			return
		case ErrWrongLeader:
			DPrintf("client %v putAppend sent to deposed leader %v, args: %v\n", ck.clientId, leaderId, args)
			ck.findLeaderNow()
			for ck.leaderId == -1 {
				time.Sleep(50 * time.Millisecond)
			}
		case ErrNoKey:
			DPrintf("client %v putAppend got nokey err, args: %v\n", ck.clientId, args)
			return
		case ErrTimeOut:
			DPrintf("client %v putAppend ErrTimeOut, args: %v\n", ck.clientId, args)
			ck.findLeaderNow()
			for ck.leaderId == -1 {
				time.Sleep(50 * time.Millisecond)
			}
		default:
			DPrintf("client %v putAppend rpc fail\n", ck.clientId)
		}
	}
}

func (ck *Clerk) Put(key string, value string) {
	ck.PutAppend(key, value, "Put")
}
func (ck *Clerk) Append(key string, value string) {
	ck.PutAppend(key, value, "Append")
}

func (ck *Clerk) periodic() {

	// find leader from all kvservers
	go func() {
		for !ck.killed() {
			select {
			case <- ck.findLeaderTimer.C:
				ck.FindLeader()
			case <- ck.stopCh:
				return
			}
		}
	}()
}

func (ck *Clerk) FindLeader() {

	args := GetStateArgs{}
	defer ck.resetFLTimer()

	ck.leaderId = -1
	ck.leaderMe = -1
	wg := sync.WaitGroup{}
	wg.Add(len(ck.servers))

	DPrintf("client %v start finding leader\n", ck.clientId)
	for i, _ := range ck.servers {
		go func(i int) {
			reply := GetStateReply{}
			if ok := ck.servers[i].Call("KVServer.GetState", &args, &reply); ok {
				// should use term, incase the RPC is out of date
				if reply.IsLeader && reply.Term >= ck.Term{
					ck.mu.Lock()
					ck.leaderId = i
					ck.leaderMe = reply.LeaderId
					ck.Term = reply.Term
					DPrintf("current leader :  i == %v, me == %v, term == %v\n", i, reply.LeaderId, ck.Term)
					ck.mu.Unlock()
				} else {
				//	DPrintf("kvserver %v is not leader", i)
				}
			} else {
				DPrintf("client rpc fail in FindLeader")
			}
			wg.Done()
		}(i)
	}
	wg.Wait()
	DPrintf("client %v find leader finish, leaderId = %v, leaderMe = %v\n", ck.clientId, ck.leaderId, ck.leaderMe)
}

func (ck *Clerk) resetFLTimer() {
	DPrintf("FL timer was reset\n")
	ck.findLeaderTimer.Stop()
	ck.findLeaderTimer.Reset(FindLeaderTimeout)
}

func (ck *Clerk) findLeaderNow() {
	ck.findLeaderTimer.Stop()
	ck.findLeaderTimer.Reset(0)
}