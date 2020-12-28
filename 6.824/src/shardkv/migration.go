package shardkv

import "time"
import "../shardmaster"

func (kv *ShardKV) migrationToPeer(shard int, newConfig *shardmaster.Config) {
	newGid := newConfig.Shards[shard]
	DPrintf("gid %v shardkv %v migrating shard %v to newGid = %v, groups = %v\n", kv.gid, kv.me, shard, newGid, newConfig.Groups)
	for {
		kv.Lock("migrationToPeer")
		args := MigrationArgs{
			Shard: shard,
			Data:  kv.data[shard],
			NewGid:	newGid,
			ClientId: int64(kv.me),
			MsgId: nrand(),
		}
		kv.Unlock("migrationToPeer")

		if servers, ok := newConfig.Groups[newGid]; ok {
			// try each server for the shard.
			for si := 0; si < len(servers); si++ {
				srv := kv.make_end(servers[si])
				var reply MigrationReply
				if ok := srv.Call("ShardKV.Migration", &args, &reply); ok {
					if reply.Err == OK {
						kv.removeShardData(args.Shard)   // todo : follower should remove data too
						return
					} else if reply.Err == ErrWrongLeader {
						continue
					} else {   // wrong group, waits for update
						break
					}
				} else {
					DPrintf("gid %v shardkv %v Migration RPC fail\n", kv.gid, kv.me)
				}
			}
		}

		//	DPrintf("shardKV %v can not find newGid %v in migration\n", kv.me, newGid)
	//	panic("migration err")
		time.Sleep(100 * time.Millisecond)   // waits for new configuration
	}
}


func (kv *ShardKV) Migration(args *MigrationArgs, reply *MigrationReply) {
	if args.NewGid != kv.gid {
		reply.Err = ErrWrongGroup
		return
	}

	_, isLeader := kv.rf.GetState()
	if !isLeader {
		reply.Err = ErrWrongLeader
		return
	}

	DPrintf("gid %v shardkv %v receive Migration, args = %v\n", kv.gid, kv.me, args)
	serverOp := Op{
		Method: "Migration",
		Shard:         args.Shard,
		ClientId:      int64(kv.me),
		MsgId:	args.MsgId,
		RequestId:	nrand(),
		MigrationData: args.Data,
	}

	res := kv.waitForRaft(serverOp)
	reply.Err = res.Err
}


func (kv *ShardKV) removeShardData(shard int) {
	kv.Lock("LockInRemoveShardData")
	delete(kv.data, shard)
	kv.Unlock("LockInRemoveShardData")
}