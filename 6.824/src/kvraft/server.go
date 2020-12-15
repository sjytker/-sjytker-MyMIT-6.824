package kvraft

import (
	"../labgob"
	"../labrpc"
	"log"
	"../raft"
	"sync"
	"sync/atomic"
	"time"
)

const Debug = 1

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug > 0 {
		log.Printf(format, a...)
	}
	return
}




type KVServer struct {
	mu      sync.Mutex
	me      int
	rf      *raft.Raft
	applyCh chan raft.ApplyMsg

	dead    int32 // set by Kill()
	stopCh chan struct{}
	data map[string]string
	findLeaderTimer *time.Timer
	maxraftstate int // snapshot if log grows this big
	lastApplied map[int64]int64
	notifyData map[int64]chan NotifyMsg
	// Your definitions here.
}


func (kv *KVServer) Get(args *GetArgs, reply *GetReply) {

	DPrintf("kvleader %v receive Get : %v\n", kv.me, args.Key)
	DPrintf("kvleader %v data : %v\n", kv.me, kv.data)

	_, isLeader := kv.rf.GetState()
	if !isLeader {
		reply.Err = ErrWrongLeader
		return
	}

	serverOp := Op{
		Method: "Get",
		Key:     args.Key,
		ClientId:	args.ClientId,
		MsgId:	args.MsgId,
		RequestId:	nrand(),
	}

	// situation : this msg is late, another same one is applied
//	repeat := kv.CheckApplied(args.MsgId, args.ClientId)

	//if !repeat {
		res := kv.waitForRaft(serverOp)
		reply.Err = res.Err
		reply.Value = res.Value
//	}
}

func (kv *KVServer) PutAppend(args *PutAppendArgs, reply *PutAppendReply) {

	DPrintf("kvleader %v receive Get : %v\n", kv.me, args.Key)
	DPrintf("kvleader %v data : %v\n", kv.me, kv.data)

	serverOp := Op{
		Method: args.Op,
		Key:     args.Key,
		Value:   args.Value,
		ClientId:	args.ClientId,
		MsgId:	args.MsgId,
		RequestId:	nrand(),
	}

	// situation : this msg is late, another same one is applied
//	repeat := kv.CheckApplied(args.MsgId, args.ClientId)

	// situation : double msg arrive, one is waiting to be applied.
	//_, waitting := kv.notifyData[args.MsgId]
	// after fully consideration, this situation must sent msg again
	reply.Err = kv.waitForRaft(serverOp).Err
}



func (kv *KVServer) waitForRaft(op Op) (res NotifyMsg) {

	waitTimer := time.NewTimer(WaitTimeout)
	defer waitTimer.Stop()
	_, _, isLeader := kv.rf.Start(op)

	if !isLeader {
		DPrintf("kvleader %v start() fail\n", kv.me)
		res.Err = ErrWrongLeader
		return
	}

	kv.mu.Lock()
	ch := make(chan NotifyMsg)
	kv.notifyData[op.RequestId] = ch
	kv.mu.Unlock()

	DPrintf("kvleader %v has start cmd, waits for applyCh\n", kv.me)
	select {
	case res =<- ch:
		DPrintf("kvleader %v apply finish, op = %v\n", kv.me, op)
		kv.removeCh(op.RequestId)
		return
	case <- waitTimer.C:
		DPrintf("kvleader %v waitTimeout, return to client, op : %v\n", kv.me, op)
		res.Err = ErrTimeOut
		kv.removeCh(op.RequestId)
		return
	//case <- kv.stopCh:
	//	return
	}
}


func (kv *KVServer) WaitApplyCh() {

	for {
		DPrintf("kvserver %v waiting for applyCh\n", kv.me)
		select {
		case msg := <- kv.applyCh:
			if !msg.CommandValid {
				//todo
				log.Fatal("________msgInvalid_______\n")
				continue
			}


			op := msg.Command.(Op)
			repeat := kv.CheckApplied(op.MsgId, op.ClientId)

			DPrintf("kvserver %v receive from applyCh, op = %v, repeat = %v\n", kv.me, op, repeat)
			kv.mu.Lock()
			if !repeat && op.Method == "Put" {
				kv.data[op.Key] = op.Value
				DPrintf("kvserver %v applying method = Put, finish, op : %v: \n", kv.me, op)
				kv.lastApplied[op.ClientId] = op.MsgId
			} else if !repeat && op.Method == "Append" {
				v:= kv.data[op.Key]
				newS := v + op.Value
				kv.data[op.Key] = newS
				kv.lastApplied[op.ClientId] = op.MsgId
				DPrintf("kvserver %v applying method = Append, newS = %v, op : %v\n", kv.me, newS, op)
			} else if op.Method == "Get"{
				DPrintf("kvserver %v applying method = Get, do nothing but notify\n", kv.me)
			}

			if ch, ok := kv.notifyData[op.RequestId]; ok {
				ch <- NotifyMsg{
					Err:   OK,
					Value: kv.data[op.Key],
				}
			}
			kv.mu.Unlock()
			DPrintf("kvserver %v notify finish, op = %v\n", kv.me, op)
		case <- kv.stopCh:
			return
		}

	}
}




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
	close(kv.stopCh)
	// Your code here, if desired.
}

func (kv *KVServer) killed() bool {
	z := atomic.LoadInt32(&kv.dead)
	return z == 1
}


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

func (kv *KVServer) CheckApplied(msgId int64, clientId int64) bool {
	if oldMsgId, ok := kv.lastApplied[clientId]; ok {
		return msgId == oldMsgId
	}
	return false
}

func (kv *KVServer) removeCh(id int64) {
	kv.mu.Lock()
	delete(kv.notifyData, id)
	kv.mu.Unlock()
}


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
		notifyData: 	 map[int64]chan NotifyMsg{},
		lastApplied: 	 map[int64]int64{},
	}

	kv.rf = raft.Make(servers, me, persister, kv.applyCh)
	DPrintf("there are %v servers, me = %v\n", len(servers), me)
	go kv.Periodic()
	return &kv
}
