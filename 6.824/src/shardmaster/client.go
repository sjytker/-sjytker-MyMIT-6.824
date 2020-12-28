package shardmaster

//
// Shardmaster clerk.
//

import "../labrpc"
import "time"
import "crypto/rand"
import "math/big"

type Clerk struct {
	servers []*labrpc.ClientEnd
	clientId int64
}

func nrand() int64 {
	max := big.NewInt(int64(1) << 62)
	bigx, _ := rand.Int(rand.Reader, max)
	x := bigx.Int64()
	return x
}

func MakeClerk(servers []*labrpc.ClientEnd) *Clerk {
	ck := new(Clerk)
	ck.servers = servers
	ck.clientId = nrand()

	return ck
}

func (ck *Clerk) Query(num int) *Config {
	args := &QueryArgs{
		Num:      num,
		ClientId: ck.clientId,
		MsgId:    nrand(),
	}

	for {
		// try each known server.
		for _, srv := range ck.servers {
			var reply QueryReply
			ok := srv.Call("ShardMaster.Query", args, &reply)
			if ok && reply.WrongLeader == false {
				return &reply.Config
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (ck *Clerk) Join(servers map[int][]string) {
	DPrintf("client %v receive join, servers = %v\n", ck.clientId, servers)
	args := &JoinArgs{
		Servers:  servers,
		ClientId: ck.clientId,
		MsgId:    nrand(),
	}

	for {
		// try each known server.
		for _, srv := range ck.servers {
			var reply JoinReply
			ok := srv.Call("ShardMaster.Join", args, &reply)
			if ok && reply.WrongLeader == false {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (ck *Clerk) Leave(gids []int) {
	args := &LeaveArgs{
		GIDs:     gids,
		ClientId: ck.clientId,
		MsgId:    nrand(),
	}

	for {
		// try each known server.
		for _, srv := range ck.servers {
			var reply LeaveReply
			ok := srv.Call("ShardMaster.Leave", args, &reply)
			if ok && reply.WrongLeader == false {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (ck *Clerk) Move(shard int, gid int) {
	args := &MoveArgs{
		Shard:    shard,
		GID:      gid,
		ClientId: ck.clientId,
		MsgId:    nrand(),
	}

	for {
		// try each known server.
		for _, srv := range ck.servers {
			var reply MoveReply
			ok := srv.Call("ShardMaster.Move", args, &reply)
			if ok && reply.WrongLeader == false {
				return
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (ck *Clerk) Stable() bool {
	args := struct {}{}

	for _, srv := range ck.servers {
		var reply StableReply
		if ok := srv.Call("ShardMaster.Stable", args, &reply); ok {
			if reply.Err == OK {
				return true
			}
		}
	}
	return false
}


func (ck *Clerk) FindGid(shard int) int {
	args := findGidArgs{shard: shard}

	for {
		for _, srv := range ck.servers {
			var reply findGidReply
			if ok := srv.Call("ShardMaster.findGid", args, &reply); ok {
				if reply.Err == OK {
					return reply.gid
				}
			}
		}
		time.Sleep(10 * time.Millisecond)
	}
	return -1
}
