package kvraft

import (
	"../labgob"
	"../labrpc"
	"log"
	"../raft"
	"sync"
	"sync/atomic"
<<<<<<< HEAD
	"time"
)

const Debug = 1
=======
)

const Debug = 0
>>>>>>> edacab21560e1960d239d963a1287729ab342ea2

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug > 0 {
		log.Printf(format, a...)
	}
	return
}


<<<<<<< HEAD

=======
type Op struct {
	// Your definitions here.
	// Field names must start with capital letters,
	// otherwise RPC will break.
}
>>>>>>> edacab21560e1960d239d963a1287729ab342ea2

type KVServer struct {
	mu      sync.Mutex
	me      int
	rf      *raft.Raft
	applyCh chan raft.ApplyMsg
	dead    int32 // set by Kill()
<<<<<<< HEAD
	stopCh chan struct{}
	data map[string]string
	vis map[int64]map[int64]bool

	findLeaderTimer *time.Timer
=======
>>>>>>> edacab21560e1960d239d963a1287729ab342ea2

	maxraftstate int // snapshot if log grows this big

	// Your definitions here.
}


func (kv *KVServer) Get(args *GetArgs, reply *GetReply) {
<<<<<<< HEAD
	kv.mu.Lock()
	defer kv.mu.Unlock()

	ApplyTimer := time.NewTimer(ApplyTimeout)

	DPrintf("kvleader %v receive Get : %v\n", kv.me, args.Key)
	DPrintf("kvleader %v data : %v\n", kv.me, kv.data)
	if _, ok := kv.vis[args.ClientId][args.MsgId]; ok {
		DPrintf("kvserver %v receive Get again, just return, key = %v\n", kv.me, args.Key)
		reply.Err = OK
		return
	}
	serverOp := Op{
		Command: "Get",
		Key:     args.Key,
		Value:   "",
		RequestId: nrand(),
	}
	_, _, isLeader := kv.rf.Start(serverOp)
	reply.IsLeader = isLeader

	for isLeader {
		DPrintf("kvleader %v has start get cmd, waits for applyCh\n", kv.me)
		select {
		case <- kv.applyCh:
			DPrintf("kvleader %v apply get cmd finish\n", kv.me)
			if v, ok := kv.data[args.Key]; ok {
				reply.Value = v
				reply.Err = OK
			} else {
				reply.Value = ""
				reply.Err = ErrNoKey
			}
			DPrintf("kvleader %v get finish, data = %v\n", kv.me, kv.data)
			return
		case <- ApplyTimer.C:
			DPrintf("ApplyTimeout in kvleader %v get\n", kv.me)
			time.Sleep(50 * time.Millisecond)
			_, isLeader = kv.rf.GetState()
		case <- kv.stopCh:
			return
		}
	}
	reply.Value = ""
	reply.Err = ErrWrongLeader
}

func (kv *KVServer) PutAppend(args *PutAppendArgs, reply *PutAppendReply) {

	ApplyTimer := time.NewTimer(ApplyTimeout)
	if _, ok := kv.vis[args.ClientId][args.MsgId]; ok {
		DPrintf("kvleader %v receive putAppend again, just return, args : %v\n", kv.me, args)
		reply.Err = OK
		return
	}

	DPrintf("kvleader %v receive putAppend, key = %v, value = %v\n", kv.me, args.Key, args.Value)
	//if _, ok := kv.vis[args.ClientId]; !ok {
		kv.vis[args.ClientId] = make(map[int64]bool)
		kv.vis[args.ClientId][args.MsgId] = true
	//}
	// when replicate & apply to all servers, you need all vars : cmd, k, v
	serverOp := Op{
		Command: args.Op,
		Key:     args.Key,
		Value:   args.Value,
		RequestId: nrand(),
	}
	_, _, isLeader := kv.rf.Start(serverOp)

	// leader's apply
	ForEnd:
	for isLeader {
		DPrintf("kvleader %v has start putAppend cmd, waits for applyCh\n", kv.me)
		select {
		case msg :=<- kv.applyCh:
			DPrintf("kvleader %v apply finish\n", kv.me)
			op := msg.Command.(Op)
			if op.RequestId != serverOp.RequestId {
				DPrintf("leader has change, client should update its leader and send again, args: %v\n", args)
				break ForEnd
			}
			//kv.mu.Lock()
			//defer kv.mu.Unlock()
			if args.Op == "Put" {
				kv.data[args.Key] = args.Value
				DPrintf("kvleader %v op == Put, finish, op : %v: \n", kv.me, serverOp)
			} else {
				if v, ok := kv.data[args.Key]; ok {
					newS := v + args.Value
					kv.data[args.Key] = newS
					DPrintf("kvleader %v op == Append, newS = %v, op : %v\n", kv.me, newS, serverOp)
				} else {
					kv.data[args.Key] = args.Value
					DPrintf("kvleader %v op == Append, but key is nil, looks like put, op : %v\n", kv.me, serverOp)
				}
			}
			reply.Err = OK
			DPrintf("kvleader %v start cmd finish, op : %v, data: %v\n", kv.me, serverOp, kv.data)
			return
		case <- ApplyTimer.C:
			DPrintf("ApplyTimeout in kvleader %v putAppend, check if is leader, op : %v\n", kv.me, serverOp)
		//	time.Sleep(50 * time.Millisecond)    maybe deadlock
			_, isLeader = kv.rf.GetState()   // will get rf's lock
		case <- kv.stopCh:
			return
		}
	}
	reply.Err = ErrWrongLeader
	DPrintf("kvserver %v is not a leader now, exit putAppend\n", kv.me)

}


func (kv *KVServer) WaitApplyCh() {

}


//func (kv *KVServer) FollowerApply() {
//
//	_, isLeader := kv.rf.GetState()
//	if isLeader {
//		return
//	}
//	ApplyTimer := time.NewTimer(ApplyTimeout)
//	serverOp := Op{}
//	msg := raft.ApplyMsg{}
//	DPrintf("kvfollower %v waits for apply\n", kv.me)
//
//	select {
//	case msg =<- kv.applyCh:
//		serverOp = msg.Command.(Op)
//		DPrintf("kvfollower %v receive applyCh, op: %v\n", kv.me, serverOp)
//		kv.mu.Lock()
//		defer kv.mu.Unlock()
//		DPrintf("kvfollower %v aquire lock in apply\n", kv.me)
//		if serverOp.Command == "Put" {
//			kv.data[serverOp.Key] = serverOp.Value
//		//	DPrintf("follower %v op == Put\n", kv.me)
//		} else {
//			if v, ok := kv.data[serverOp.Key]; ok {
//				newS := v + serverOp.Value
//				kv.data[serverOp.Key] = newS
//			//	DPrintf("follower %v op == Append, newS = %v\n", kv.me, newS)
//			} else {
//				kv.data[serverOp.Key] = serverOp.Value
//			//	DPrintf("follower %v op == Append, but key is nil, looks like put\n", kv.me)
//			}
//		}
//	case <- ApplyTimer.C:
//		DPrintf("kvfollower %v ApplyTimeout\n", kv.me)
//	case <- kv.stopCh:
//		return
//	}
//	DPrintf("kvfollower %v apply finish\n", kv.me)
//}



=======
	// Your code here.
}

func (kv *KVServer) PutAppend(args *PutAppendArgs, reply *PutAppendReply) {
	// Your code here.
}

>>>>>>> edacab21560e1960d239d963a1287729ab342ea2
//
// the tester calls Kill() when a KVServer instance won't
// be needed again. for your convenience, we supply
// code to set rf.dead (without needing a lock),
// and a killed() method to test rf.dead in
// long-running loops. you can also add your own
// code to Kill(). you're not required to do anything
// about this, but it may be convenient (for example)
// to suppress debug output from a Kill()ed instance.
//
func (kv *KVServer) Kill() {
	atomic.StoreInt32(&kv.dead, 1)
	kv.rf.Kill()
<<<<<<< HEAD
	close(kv.stopCh)
=======
>>>>>>> edacab21560e1960d239d963a1287729ab342ea2
	// Your code here, if desired.
}

func (kv *KVServer) killed() bool {
	z := atomic.LoadInt32(&kv.dead)
	return z == 1
}

<<<<<<< HEAD


//GetState from rpc
func (kv *KVServer) GetState(args *GetStateArgs, reply *GetStateReply)  {

	// Your code here (2A).
	//kv.mu.Lock()
	//defer kv.mu.Unlock()
	DPrintf("kvserver %v receive getstate, lockseq : %v,  %v\n", kv.me, len(kv.rf.LockSeq), kv.rf.LockSeq)
	reply.Term, reply.IsLeader = kv.rf.GetState()
	DPrintf("kvserver %v getstate finish\n", kv.me)
	reply.LeaderId = kv.me
}

func (kv *KVServer) Periodic() {
	go kv.WaitApplyCh()
}
=======
>>>>>>> edacab21560e1960d239d963a1287729ab342ea2
//
// servers[] contains the ports of the set of
// servers that will cooperate via Raft to
// form the fault-tolerant key/value service.
// me is the index of the current server in servers[].
// the k/v server should store snapshots through the underlying Raft
// implementation, which should call persister.SaveStateAndSnapshot() to
// atomically save the Raft state along with the snapshot.
// the k/v server should snapshot when Raft's saved state exceeds maxraftstate bytes,
// in order to allow Raft to garbage-collect its log. if maxraftstate is -1,
// you don't need to snapshot.
// StartKVServer() must return quickly, so it should start goroutines
// for any long-running work.
//
func StartKVServer(servers []*labrpc.ClientEnd, me int, persister *raft.Persister, maxraftstate int) *KVServer {
	// call labgob.Register on structures you want
	// Go's RPC library to marshall/unmarshall.
	labgob.Register(Op{})

<<<<<<< HEAD
	kv := KVServer{
		mu:              sync.Mutex{},
		me:              me,
		rf:              nil,
		applyCh:         make(chan raft.ApplyMsg),
		dead:            0,
		stopCh:          make(chan struct{}),
		data:            make(map[string]string),
		findLeaderTimer: time.NewTimer(FindLeaderTimeout),
		maxraftstate:    maxraftstate,
		vis: 			 make(map[int64]map[int64]bool),
	}

	kv.rf = raft.Make(servers, me, persister, kv.applyCh)
	DPrintf("there are %v servers, me = %v\n", len(servers), me)
	go kv.Periodic()
	return &kv
=======
	kv := new(KVServer)
	kv.me = me
	kv.maxraftstate = maxraftstate

	// You may need initialization code here.

	kv.applyCh = make(chan raft.ApplyMsg)
	kv.rf = raft.Make(servers, me, persister, kv.applyCh)

	// You may need initialization code here.

	return kv
>>>>>>> edacab21560e1960d239d963a1287729ab342ea2
}
