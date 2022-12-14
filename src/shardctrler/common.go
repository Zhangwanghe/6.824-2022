package shardctrler

//
// Shard controler: assigns shards to replication groups.
//
// RPC interface:
// Join(servers) -- add a set of groups (gid -> server-list mapping).
// Leave(gids) -- delete a set of groups.
// Move(shard, gid) -- hand off one shard from current owner to gid.
// Query(num) -> fetch Config # num, or latest config if num==-1.
//
// A Config (configuration) describes a set of replica groups, and the
// replica group responsible for each shard. Configs are numbered. Config
// #0 is the initial configuration, with no groups and all shards
// assigned to group 0 (the invalid group).
//
// You will need to add fields to the RPC argument structs.
//

// The number of shards.
const NShards = 10

// A configuration -- an assignment of shards to groups.
// Please don't change this.
type Config struct {
	Num    int              // config number
	Shards [NShards]int     // shard -> gid
	Groups map[int][]string // gid -> servers[]
}

const (
	OK = "OK"
)

type Err string

type JoinArgs struct {
	Servers      map[int][]string // new GID -> servers mappings
	Client       int64
	SerialNumber int
}

type JoinReply struct {
	WrongLeader bool
	Err         Err
}

type LeaveArgs struct {
	GIDs         []int
	Client       int64
	SerialNumber int
}

type LeaveReply struct {
	WrongLeader bool
	Err         Err
}

type MoveArgs struct {
	Shard        int
	GID          int
	Client       int64
	SerialNumber int
}

type MoveReply struct {
	WrongLeader bool
	Err         Err
}

type QueryArgs struct {
	Num          int // desired config number
	Client       int64
	SerialNumber int
}

type QueryReply struct {
	WrongLeader bool
	Err         Err
	Config      Config
}

func makeMap(keys []int, vals []int) map[int]int {
	ret := make(map[int]int)
	for index, key := range keys {
		ret[key] = vals[index]
	}

	return ret
}

func diffMap(a map[int]int, b map[int]int) map[int]int {
	counts := make(map[int]int)

	for k, v := range a {
		val, ok := b[k]
		if ok {
			counts[k] = v - val
		} else {
			counts[k] = v
		}
	}

	for k, v := range b {
		_, ok := a[k]
		if !ok {
			counts[k] = -v
		}
	}

	return counts
}
