package shardkv

import (
	"../shardmaster"
	"../raft"
)


func (kv *ShardKV) WaitApplyCh() {
	for {
		DPrintf("kvserver %v waiting for applyCh\n", kv.me)
		select {
		case msg := <- kv.applyCh:
			if !msg.CommandValid {
				//todo
				DPrintf("ShardKV %v receive an installSnapshot apply msg\n", kv.me)
				kv.mu.Lock()
				kv.ReadSnapshot(kv.persister.ReadSnapshot())
				kv.mu.Unlock()
				continue
			}

			if op, ok := msg.Command.(Op); ok {
				kv.applyOp(msg, op)
			} else if config, ok := msg.Command.(shardmaster.Config); ok {
				kv.applyConfig(msg, config)
			}
		}
	}
}


func (kv *ShardKV) CheckApplied(msgId int64, clientId int64) bool {
	if oldMsgId, ok := kv.lastApplied[clientId]; ok {
		return msgId == oldMsgId
	}
	return false
}



func (kv *ShardKV) applyOp(msg raft.ApplyMsg, op Op) {

	repeat := kv.CheckApplied(op.MsgId, op.ClientId)

	DPrintf("gid %v ShardKV %v receive from applyCh, op = %v, repeat = %v\n", kv.gid, kv.me, op, repeat)
	kv.mu.Lock()
	if !repeat && op.Method == "Migration" {

		if _, ok := kv.data[op.Shard];!ok {
			kv.data[op.Shard] = make(map[string]string)
		}
		kv.data[op.Shard] = op.MigrationData
		DPrintf("gid %v ShardKV %v applying method = Migration, finish, op = %v, data = %v \n", kv.gid, kv.me, op, kv.data)

	} else if !repeat && op.Method == "Put" {

		DPrintf("gid %v ShardKV %v applying method = Put, finish, op : %v \n", kv.gid, kv.me, op)
		DPrintf("gid %v ShardKV %v kv.data[op.Shard] len = %v, lastApplied = %v \n", kv.gid, kv.me, len(kv.data[op.Shard]), kv.lastApplied)

		if _, ok := kv.data[op.Shard];!ok {
			kv.data[op.Shard] = make(map[string]string)
		}

		if kv.data[op.Shard] == nil {
			kv.data[op.Shard] = make(map[string]string)
		}
		kv.data[op.Shard][op.Key] = op.Value
		kv.lastApplied[op.ClientId] = op.MsgId

	} else if !repeat && op.Method == "Append" {

		if _, ok := kv.data[op.Shard];!ok {
			kv.data[op.Shard] = make(map[string]string)
		}
		v:= kv.data[op.Shard][op.Key]
		newS := v + op.Value
		kv.data[op.Shard][op.Key] = newS
		kv.lastApplied[op.ClientId] = op.MsgId
		DPrintf("gid %v ShardKV %v applying method = Append, newS = %v, op : %v\n", kv.gid, kv.me, newS, op)

	} else if op.Method == "Get"{
		DPrintf("gid %v ShardKV %v applying method = Get, do nothing but notify\n", kv.gid, kv.me)
	}

	kv.saveSnapshot(msg.CommandIndex)
	if ch, ok := kv.notifyData[op.RequestId]; ok {
		ch <- NotifyMsg{
			Err:   OK,
			Value: kv.data[op.Shard][op.Key],
		}
	}
	kv.mu.Unlock()
	if op.Method != "Query" {
		DPrintf("gid %v ShardKV %v notify finish, op : %v, data : %v\n", kv.gid, kv.me, op, kv.data)
	}
}

func (kv *ShardKV) applyConfig(msg raft.ApplyMsg, newConfig shardmaster.Config) {
	kv.Lock("applyConfig")
	defer kv.Unlock("applyConfig")

	DPrintf("gid %v ShardKV %v applying config = %v, lastConfig = %v\n", kv.gid, kv.me, newConfig, kv.config)
	if kv.config != nil && newConfig.Num <= kv.config.Num {
		kv.saveSnapshot(msg.CommandIndex)
		return
	}

	kv.config = &newConfig
}
