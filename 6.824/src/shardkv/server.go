package shardkv


// import "../shardmaster"
import (
	"../labrpc"
	"bytes"
	"os"
	"time"
)
import "../raft"
import "sync"
import "../labgob"
import "../shardmaster"



type Op struct {
	Method string
	Shard  int
	Key    string
	Value  string
	ClientId      int64
	MsgId         int64
	RequestId     int64
	MigrationData map[string]string
}

type ShardKV struct {
	mu           sync.Mutex
	me           int
	rf           *raft.Raft
	applyCh      chan raft.ApplyMsg
	make_end     func(string) *labrpc.ClientEnd
	gid          int
	masters      []*labrpc.ClientEnd
	maxraftstate int // snapshot if log grows this big

	configTimer *time.Timer

	mck            *shardmaster.Clerk
	config         *shardmaster.Config
	persister      *raft.Persister
	data           map[int]map[string]string // shard -> k -> v
	lastApplied    map[int64]int64
	notifyData     map[int64]chan NotifyMsg
	lastApplyIndex int
	lastApplyTerm  int
	shards         []int	// current config shard
	LockSeq        []string
	state          int
}


func (kv *ShardKV) Get(args *GetArgs, reply *GetReply) {

	//if args.Gid != kv.gid {   // might not happen
	//	reply.Err = ErrWrongGroup
	//	return
	//}

	_, isLeader := kv.rf.GetState()
	if !isLeader {
		reply.Err = ErrWrongLeader
		return
	}

	DPrintf("group %v ShardKV %v receive Get : %v\n", kv.gid, kv.me, args)
	DPrintf("group %v ShardKV %v data : %v\n", kv.gid, kv.me, kv.data)

	// waits for STABLE
	for {
		kv.mu.Lock()
		if kv.state == STABLE {
			kv.mu.Unlock()
			break
		}
		kv.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
	}

	if args.Gid != kv.gid {   // might not happen
		reply.Err = ErrWrongGroup
		return
	}

	if !kv.checkAndMigrateShard(args.Shard) {
		reply.Err = ErrWrongGroup
		return
	}

	serverOp := Op{
		Method: "Get",
		Shard: args.Shard,
		Key:     args.Key,
		ClientId:	args.ClientId,
		MsgId:	args.MsgId,
		RequestId:	nrand(),
	}

	res := kv.waitForRaft(serverOp)
	reply.Err = res.Err
	reply.Value = res.Value
}

func (kv *ShardKV) PutAppend(args *PutAppendArgs, reply *PutAppendReply) {


	//if args.Gid != kv.gid {
	//	reply.Err = ErrWrongGroup
	//	return
	//}

	_, isLeader := kv.rf.GetState()
	if !isLeader {
		reply.Err = ErrWrongLeader
		return
	}

	DPrintf("group %v kvleader %v receive put : %v\n", kv.gid, kv.me, args)
	DPrintf("group %v ShardKV %v data : %v\n", kv.gid, kv.me, kv.data)

	// waits for STABLE
	for {
		kv.mu.Lock()
		if kv.state == STABLE {
			kv.mu.Unlock()
			break
		}
		kv.mu.Unlock()
		time.Sleep(10 * time.Millisecond)
	}

	if args.Gid != kv.gid {
		reply.Err = ErrWrongGroup
		return
	}

	if !kv.checkAndMigrateShard(args.Shard) {
		reply.Err = ErrWrongGroup
		return
	}

	serverOp := Op{
		Method: args.Op,
		Shard: args.Shard,
		Key:     args.Key,
		Value:   args.Value,
		ClientId:	args.ClientId,
		MsgId:	args.MsgId,
		RequestId:	nrand(),
	}

	DPrintf("kvleader %v start replicating : %v\n", kv.me, kv.data)
	reply.Err = kv.waitForRaft(serverOp).Err
}

//
// the tester calls Kill() when a ShardKV instance won't
// be needed again. you are not required to do anything
// in Kill(), but it might be convenient to (for example)
// turn off debug output from this instance.
//
func (kv *ShardKV) Kill() {
	kv.rf.Kill()
	// Your code here, if desired.
}

func (kv *ShardKV) findConfigNow() {
	kv.configTimer.Reset(0)
}

func (kv *ShardKV) FindConfig() {

	// time.Sleep(FreezeTime)
	for {
		select {
		case <- kv.configTimer.C:
			_, isLeader := kv.rf.GetState()
			if !isLeader {
				kv.configTimer.Reset(PullTimeout)
				break
			}

			curConfig := kv.mck.Query(-1)
			DPrintf("gid %v ShardKV %v find curConfig = %v\n", kv.gid, kv.me, curConfig)
			// should do migration from this gid to other's
			if  kv.config != nil &&
				len(kv.config.Groups) != 0 &&
				len(curConfig.Groups) != 0 &&
				len(curConfig.Groups) != len(kv.config.Groups) {
		//		DPrintf("gid %v ShardKV %v find group len has change\n", kv.gid, kv.me)

				kv.Lock("updateconfig")
				kv.state = TRANSITION
				kv.Unlock("updateconfig")
				lastConfigShards := make(map[int]bool)     // last config, which shard my gid has?
				for i := 0; i < NShard; i ++ {
					if kv.config.Shards[i] == kv.gid {
						lastConfigShards[i] = true
					}
				}

				DPrintf("gid %v ShardKV %v find group len has change, lastConfigShards : %v\n", kv.gid, kv.me, lastConfigShards)
				for shard, _ := range lastConfigShards {     // migrate the shards which differs in gid
					newGid := curConfig.Shards[shard]
					if newGid != kv.gid {
					//	DPrintf("gid %v ShardKV %v migrate shard %v to newGid %v, shardData = %v\n", kv.gid, kv.me, shard, newGid, kv.data)
						kv.migrationToPeer(shard, curConfig)
					}
				}
				DPrintf("gid %v ShardKV migration to peers finish\n", kv.gid)
				kv.Lock("updateconfig")
				kv.state = STABLE
				kv.Unlock("updateconfig")
			}
			kv.mu.Lock()
			DPrintf("gid %v shardkv %v updating config : %v\n", kv.gid, kv.me, curConfig)
		//	kv.config = curConfig   change implementation, leader replicate config to all
			kv.rf.Start(curConfig.Copy())
			//kv.shards = make([]int, 0)
			//for i := 0; i < NShard; i ++ {
			//	if kv.config.Shards[i] == kv.gid {
			//		kv.shards = append(kv.shards, kv.config.Shards[i])
			//	}
			//}
			kv.mu.Unlock()
			kv.configTimer.Reset(PullTimeout)
		}
	}
}

func (kv *ShardKV) waitForRaft(op Op) (res NotifyMsg) {

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
		res.Err = OK
		kv.removeCh(op.RequestId)
		return
	case <- waitTimer.C:
		DPrintf("kvleader %v waitTimeout, return to client, op : %v\n", kv.me, op)
		res.Err = ErrTimeOut
		kv.removeCh(op.RequestId)
		return
	}
}

func (kv *ShardKV) removeCh(id int64) {
	kv.mu.Lock()
	delete(kv.notifyData, id)
	kv.mu.Unlock()
}

func (kv *ShardKV) checkAndMigrateShard(shard int) bool{
//	kv.findConfigNow()
	time.Sleep(50 * time.Millisecond)
	kv.mu.Lock()
	newGid := kv.config.Shards[shard]
	if newGid == kv.gid {
		kv.mu.Unlock()
		return true
	}
	kv.mu.Unlock()
	DPrintf("gid %v ShardKV %v check shard = %v, newGid = %v\n", kv.gid, kv.me, shard, newGid)
//	kv.migrationToPeer(shard, newGid)
	return false
}


//  todo:  haven't been modified
func (kv *ShardKV) saveSnapshot(index int) {
	if kv.maxraftstate == -1 || kv.persister.RaftStateSize() < kv.maxraftstate {
		return
	}
	DPrintf("server %v saving snapshot, rfsize = %v, kv.maxrfstate = %v\n", kv.me, kv.persister.RaftStateSize(), kv.maxraftstate)
	w := new(bytes.Buffer)
	e := labgob.NewEncoder(w)
	if err := e.Encode(kv.data); err != nil {
		panic(err)
	}
	if err := e.Encode(kv.lastApplied); err != nil {
		panic(err)
	}
	if err := e.Encode(kv.config); err != nil {
		panic(err)
	}
	kvsData := w.Bytes()
	kv.rf.SaveStateAndSnapshot(index, kvsData)
}


func (kv *ShardKV) ReadSnapshot(data []byte) {
	if data == nil { // bootstrap without any state?
		return
	}
	r := bytes.NewBuffer(data)
	d := labgob.NewDecoder(r)
	var lastApplied map[int64]int64
	var kvdata map[int]map[string]string
	var config *shardmaster.Config
	if d.Decode(&kvdata) != nil ||
		d.Decode(&lastApplied) != nil ||
		d.Decode(&config) != nil {
		DPrintf("readSnapshot error\n")
		os.Exit(1)
	} else {
		kv.lastApplied = lastApplied
		kv.data = kvdata
	}
}




//
// servers[] contains the ports of the servers in this group.
//
// me is the index of the current server in servers[].
//
// the k/v server should store snapshots through the underlying Raft
// implementation, which should call persister.SaveStateAndSnapshot() to
// atomically save the Raft state along with the snapshot.
//
// the k/v server should snapshot when Raft's saved state exceeds
// maxraftstate bytes, in order to allow Raft to garbage-collect its
// log. if maxraftstate is -1, you don't need to snapshot.
//
// gid is this group's GID, for interacting with the shardmaster.
//
// pass masters[] to shardmaster.MakeClerk() so you can send
// RPCs to the shardmaster.
//
// make_end(servername) turns a server name from a
// Config.Groups[gid][i] into a labrpc.ClientEnd on which you can
// send RPCs. You'll need this to send RPCs to other groups.
//
// look at client.go for examples of how to use masters[]
// and make_end() to send RPCs to the group owning a specific shard.
//
// StartServer() must return quickly, so it should start goroutines
// for any long-running work.
//
func StartServer(servers []*labrpc.ClientEnd, me int, persister *raft.Persister, maxraftstate int, gid int, masters []*labrpc.ClientEnd, make_end func(string) *labrpc.ClientEnd) *ShardKV {
	// call labgob.Register on structures you want
	// Go's RPC library to marshall/unmarshall.
	labgob.Register(Op{})

	kv := ShardKV{
		mu:           sync.Mutex{},
		me:           me,
		rf:           nil,
		applyCh:      make(chan raft.ApplyMsg),
		make_end:     make_end,
		gid:          gid,
		masters:      masters,
		maxraftstate: maxraftstate,
		configTimer:    time.NewTimer(PullTimeout),
		mck:            shardmaster.MakeClerk(masters),
		config:         nil,
		persister:      persister,
		data:           make(map[int]map[string]string),
		lastApplied:    make(map[int64]int64),
		notifyData:     make(map[int64]chan NotifyMsg),
		lastApplyIndex: 0,
		lastApplyTerm:  0,
		LockSeq: 	  make([]string, 0),
	}
	kv.rf = raft.Make(servers, me, persister, kv.applyCh)
	kv.ReadSnapshot(kv.persister.ReadSnapshot())
	go kv.FindConfig()
	go kv.WaitApplyCh()

	kv.findConfigNow()
	time.Sleep(FreezeTime)
	return &kv
}
