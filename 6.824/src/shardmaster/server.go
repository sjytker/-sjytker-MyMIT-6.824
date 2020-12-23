package shardmaster


import (
	"../raft"
	"time"
)
import "../labrpc"
import "sync"
import "../labgob"



type NotifyMsg struct {
	Err Err
	WrongLeader bool
	Config Config
}


type ShardMaster struct {
	mu      sync.Mutex
	me      int
	rf      *raft.Raft
	applyCh chan raft.ApplyMsg
	LockSeq []string

	// Your data here.

	configs []Config // indexed by config num
	lastApplies map[int64]int64
	notifyData map[int64]chan NotifyMsg
}


type Op struct {
	Method string
	ClientId int64
	MsgId int64
	RequestId int64
	Args interface{}
}


func (sm *ShardMaster) Join(args *JoinArgs, reply *JoinReply) {
	_, isLeader := sm.rf.GetState()
	if !isLeader {
		reply.WrongLeader = true
		reply.Err = ErrWrongLeader
		return
	}

	op := Op{
		Method:    "Join",
		ClientId:  args.ClientId,
		MsgId:     args.MsgId,
		RequestId: nrand(),
	//	server:	   args.Servers,
		Args :	   *args,
	}

	res := sm.waitForRaft(op)
	reply.WrongLeader = res.WrongLeader
	reply.Err = res.Err
}

func (sm *ShardMaster) Leave(args *LeaveArgs, reply *LeaveReply) {
	_, isLeader := sm.rf.GetState()
	if !isLeader {
		reply.WrongLeader = true
		reply.Err = ErrWrongLeader
		return
	}

	op := Op{
		Method:    "Leave",
		ClientId:  args.ClientId,
		MsgId:     args.MsgId,
		RequestId: nrand(),
		Args:	   *args,
	}

	res := sm.waitForRaft(op)
	reply.WrongLeader = res.WrongLeader
	reply.Err = res.Err
}

func (sm *ShardMaster) Move(args *MoveArgs, reply *MoveReply) {
	_, isLeader := sm.rf.GetState()
	if !isLeader {
		reply.WrongLeader = true
		reply.Err = ErrWrongLeader
		return
	}

	op := Op{
		Method:    "Move",
		ClientId:  args.ClientId,
		MsgId:     args.MsgId,
		RequestId: nrand(),
		Args:	   *args,
	}

	res := sm.waitForRaft(op)
	reply.WrongLeader = res.WrongLeader
	reply.Err = res.Err
}

func (sm *ShardMaster) Query(args *QueryArgs, reply *QueryReply) {
	_, isLeader := sm.rf.GetState()
	if !isLeader {
		reply.WrongLeader = true
		reply.Err = ErrWrongLeader
		return
	}

	op := Op{
		Method:    "Query",
		ClientId:  args.ClientId,
		MsgId:     args.MsgId,
		RequestId: nrand(),
		Args:	   *args,
	}

	res := sm.waitForRaft(op)
	reply.WrongLeader = res.WrongLeader
	reply.Err = res.Err
	reply.Config = sm.getConfigCopy(args.Num)
}


//
// the tester calls Kill() when a ShardMaster instance won't
// be needed again. you are not required to do anything
// in Kill(), but it might be convenient to (for example)
// turn off debug output from this instance.
//
func (sm *ShardMaster) Kill() {
	sm.rf.Kill()
	// Your code here, if desired.
}

// needed by shardkv tester
func (sm *ShardMaster) Raft() *raft.Raft {
	return sm.rf
}

func (sm *ShardMaster) waitForRaft(op Op) (res NotifyMsg) {

	waitTimer := time.NewTimer(WaitTimeout)
	defer waitTimer.Stop()
	_, _, isLeader := sm.rf.Start(op)
	if (!isLeader) {
		res.Err = ErrWrongLeader
		res.WrongLeader = true
		return
	}
	sm.Lock("lockInWait")
	ch := make(chan NotifyMsg)
	sm.notifyData[op.RequestId] = ch
	sm.Unlock("lockInWait")

	DPrintf("ShardMaster %v has start cmd, waits for applyCh\n", sm.me)
	select {
	case res =<- ch:
		DPrintf("ShardMaster %v apply finish, op = %v\n", sm.me, op)
		sm.removeCh(op.RequestId)
		return
	case <- waitTimer.C:
		DPrintf("ShardMaster %v waitTimeout, return to client, op : %v\n", sm.me, op)
		res.Err = ErrTimeOut
		sm.removeCh(op.RequestId)
		return
	//case <- sm.stopCh:
	//	return
	}
}

func (sm *ShardMaster) apply() {
	for {
		DPrintf("ShardMaster %v waiting for applyCh\n", sm.me)
		select {
		case msg := <- sm.applyCh:
			if !msg.CommandValid {
				//todo
				DPrintf("ShardMaster %v receive an invalid apply msg\n", sm.me)
				continue
			}

			op := msg.Command.(Op)
			repeat := sm.CheckApplied(op.MsgId, op.ClientId)

			DPrintf("ShardMaster %v receive from applyCh, op = %v, repeat = %v\n", sm.me, op, repeat)


			if !repeat {
				switch op.Method{
				case "Join" :
					sm.join(op.Args.(JoinArgs))
				case "Leave" :
					sm.leave(op.Args.(LeaveArgs))
				case "Move" :
					sm.move(op.Args.(MoveArgs))
				case "Query" :
				}
			}

			sm.Lock("LockInApply")
			sm.lastApplies[op.ClientId] = op.MsgId
			if ch, ok := sm.notifyData[op.RequestId]; ok {
				ch <- NotifyMsg{
					Err:   OK,
				}
			}
			sm.Unlock("LockInApply")
			DPrintf("ShardMaster %v notify finish, op = %v\n", sm.me, op)
		//case <- kv.stopCh:
		//	return
		}
	}
}


func (sm *ShardMaster) removeCh(id int64) {
	sm.mu.Lock()
	delete(sm.notifyData, id)
	sm.mu.Unlock()
}


func (sm *ShardMaster) CheckApplied(msgId int64, clientId int64) bool {
	if oldMsgId, ok := sm.lastApplies[clientId]; ok {
		return msgId == oldMsgId
	}
	return false
}


func (sm *ShardMaster) join(args JoinArgs) {
	DPrintf("server %v receive join, args = %v\n", sm.me, args)
	config := sm.getConfigCopy(-1)
	config.Num ++

	for k, v := range args.Servers {
		config.Groups[k] = v
	}

	sm.adjustConfig(&config)
	sm.Lock("lockInJoin")
	sm.configs = append(sm.configs, config)
	sm.Unlock("lockInJoin")
}


func (sm *ShardMaster) leave(args LeaveArgs) {
	config := sm.getConfigCopy(-1)
	config.Num ++
	DPrintf("server %v receive leave, before leaving, groups: %v,  %v\n", sm.me, len(config.Groups), config.Groups)

	sm.Lock("lockInLeave1")
	for _, gid := range args.GIDs {
		delete(config.Groups, gid)
	}
	sm.Unlock("lockInLeave1")

	DPrintf("server %v finish leaving(not adjust), groups: %v,  %v\n", sm.me, len(config.Groups), config.Groups)
	sm.adjustConfig(&config)

	sm.Lock("lockInLeave2")
	sm.configs = append(sm.configs, config)
	sm.Unlock("lockInLeave2")
}


func (sm *ShardMaster) move(args MoveArgs) {
	config := sm.getConfigCopy(-1)
	config.Num ++
	config.Shards[args.Shard] = args.GID

	sm.Lock("lockInMove")
	sm.configs = append(sm.configs, config)
	sm.Unlock("lockInMove")
}


// adjust shard between new and old config group
func (sm *ShardMaster) adjustConfig(config *Config) {
	if len(config.Groups) == 0 {
		config.Shards = [NShards]int{}
	} else if len(config.Groups) == 1 {
		for gid, _ := range config.Groups {
			for i := 0; i < NShards; i ++ {
				config.Shards[i] = gid
			}
		}
	} else {
		lastConfig := sm.getConfigCopy(-1)
		lastLen := len(lastConfig.Groups)
		curLen := len(config.Groups)

		avg := NShards / curLen
		left := NShards % curLen

		lastCnt := make(map[int]int)   // group -> numShard
		// count last group gid num of shard
		for gid, _ := range config.Groups {
			// if last config group exist this gid
			for i := 0; i < NShards; i ++ {
				if lastConfig.Shards[i] == gid {
					lastCnt[gid] ++
				}
			}
		}

		counts := map[int]int{}
		for _, g := range config.Shards {
			counts[g] += 1
		}
		DPrintf("before adjust config, current shard :\n")
		for k, v := range counts {
			DPrintf(" %v   %v\n", k, v)
		}

		// assume gid coverage like this
		// ###x1......|||x2..###......x3|||
		// x2 is common gid area, x1 is new, x3 is old

		if curLen < lastLen {
			// now we have less cluster, num of shard should be increase
			// find out last gid key and num to increase

			DPrintf("shardmaster adjuster find curLen < lastLen, maybe leave has benn called \n")

		//	finish := make(map[int]bool)

			// FIRST, deal with same gid area
			// move x3 to x2
			for gid, _ := range config.Groups {
				if _, ok := lastCnt[gid]; ok {   // if lastGroup exist this gid
					r := avg - lastCnt[gid]
					if r == 0 {
						continue
					}
					if left > 0 {
						r ++
						left --
					}
					if r == 0 {
						continue
					}

					MOVEGID1:
					for oldGid, _ := range lastConfig.Groups {
						if _, ok1 := config.Groups[oldGid]; !ok1 {
							for i := 0; i < NShards; i ++ {
								if lastConfig.Shards[i] == oldGid {
									lastConfig.Shards[i] = 0
									lastCnt[oldGid] --
									config.Shards[i] = gid
									r --
								}
								if r == 0 {
									break MOVEGID1
								}
							}
						}
					} // MOVEGID
				}
			}

			//SECOND, deal with different gid area
			// move x3 to x1
			for gid, _ := range config.Groups {
				if _, ok := lastCnt[gid]; !ok {   // if lastGroup doesn't exist this gid
					r := avg
					if left > 0 {
						r ++
						left --
					}

				MOVEGID2:
					for oldGid, _ := range lastConfig.Groups {
						if _, ok1 := config.Groups[oldGid]; !ok1 {
							for i := 0; i < NShards; i ++ {
								if lastConfig.Shards[i] == oldGid {
									lastConfig.Shards[i] = 0
									lastCnt[oldGid] --
									config.Shards[i] = gid
									r --
								}
								if r == 0 {
									break MOVEGID2
								}
							}
						}
					} // MOVEGID
				}
			}
		} else {
			// now we have more cluster, num of shard should be decrease
			// but don't do like less cluster situation
			// first collect all shards which should be reassign
			// then assign these shards to x1
			DPrintf("shardmaster adjuster find curLen < lastLen, maybe join has benn called \n")

			finish := make(map[int]bool)

			// FIRST, clean up rest
			for lastGid, _ := range lastConfig.Groups {
				if _, ok := config.Groups[lastGid]; ok {   // if current group exist this gid
					r := lastCnt[lastGid] - avg
					if r == 0 {   // last gid num == cur gid num, do nothing
						finish[lastGid] = true
						continue
					}
					if left > 0 {
						r --
						left --
					}
					if r == 0 {   // last gid num == cur gid num + 1, do nothing but adjust "left"
						finish[lastGid] = true
						continue
					}
					// clean up rest
					for i := 0; i < NShards; i ++ {
						if config.Shards[i] == lastGid {
							config.Shards[i] = 0
							r --
						}
						if r == 0 {
							finish[lastGid] = true
							break
						}
					}
				} else {    // if current group doesn't exist this gid
					for i := 0; i < NShards; i ++ {
						if config.Shards[i] == lastGid {
							config.Shards[i] = 0
						}
					}
				}
			}

			//SECOND, reassign shard to new group
			for gid, _ := range config.Groups {
				if finish[gid] {
					continue
				}
				r := avg
				if left > 0 {
					r ++
					left --
				}
				for i := 0; i < NShards; i ++ {
					if config.Shards[i] == 0 {
						config.Shards[i] = gid
						r --
					}
					if r == 0 {
						finish[gid] = true   // for completeness, not useful
						break
					}
				}
			}
		}
	}

	counts := map[int]int{}
	for _, g := range config.Shards {
		counts[g] += 1
	}
	DPrintf("adjust config finish, current shard :\n")
	for k, v := range counts {
		DPrintf(" %v   %v\n", k, v)
	}
}

func (sm *ShardMaster) getConfigCopy(i int) Config{
	sm.Lock("lockIngetConfigCopy")
	defer sm.Unlock("lockIngetConfigCopy")
	if (i < 0 || i >= len(sm.configs)) {
		return sm.configs[len(sm.configs) - 1].copy()
	} else {
		return sm.configs[i].copy()
	}
}

//
// servers[] contains the ports of the set of
// servers that will cooperate via Paxos to
// form the fault-tolerant shardmaster service.
// me is the index of the current server in servers[].
//
func StartServer(servers []*labrpc.ClientEnd, me int, persister *raft.Persister) *ShardMaster {
	sm := ShardMaster{
		mu:          sync.Mutex{},
		me:          me,
		rf:          nil,
		applyCh:     make(chan raft.ApplyMsg, 10),
		configs:     make([]Config, 1),
		lastApplies: make(map[int64]int64),
		notifyData:  make(map[int64]chan NotifyMsg),
		LockSeq :	 make([]string, 0),
	}

	sm.configs[0].Groups = map[int][]string{}
	sm.rf = raft.Make(servers, me, persister, sm.applyCh)

	labgob.Register(Op{})

	DPrintf("register complete\n")
	go sm.apply()
	
	return &sm
}
