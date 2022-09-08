package kvraft

import (
	"log"
	"sync"
	"sync/atomic"
	"time"

	"6.824/labgob"
	"6.824/labrpc"
	"6.824/raft"
)

const Debug = false

func DPrintf(format string, a ...interface{}) (n int, err error) {
	if Debug {
		log.Printf(format, a...)
	}
	return
}

type Op struct {
	// Your definitions here.
	// Field names must start with capital letters,
	// otherwise RPC will break.
	OpType       string
	Key          string
	Value        string
	Client       int64
	SerialNumber int
}

type KVServer struct {
	mu      sync.Mutex
	me      int
	rf      *raft.Raft
	applyCh chan raft.ApplyMsg
	dead    int32 // set by Kill()

	maxraftstate int // snapshot if log grows this big

	// Your definitions here.
	keyValues          map[string]string
	appliedlogs        map[int]interface{}
	requiredlogs       map[int]int
	clientSerialNumber map[int64]int
}

func (kv *KVServer) Get(args *GetArgs, reply *GetReply) {
	// Your code here.
	ok, val := kv.hasExecuted(args.Client, args.SerialNumber, args.Key)
	if ok {
		reply.Value = val
		return
	}

	op := Op{"Get", args.Key, "", args.Client, args.SerialNumber}
	if !kv.startOp(op) {
		reply.Err = "wrong leader"
		return
	}

	reply.Value = kv.GetVal(args.Key)
}

func (kv *KVServer) GetVal(key string) string {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	val, ok := kv.keyValues[key]
	if !ok {
		return ""
	}

	return val
}

func (kv *KVServer) PutAppend(args *PutAppendArgs, reply *PutAppendReply) {
	// Your code here.
	ok, _ := kv.hasExecuted(args.Client, args.SerialNumber, args.Key)
	if ok {
		return
	}

	op := Op{args.Op, args.Key, args.Value, args.Client, args.SerialNumber}
	if !kv.startOp(op) {
		// todo whether to return leaderId if possible
		reply.Err = "wrong leader"
		return
	}

}

func (kv *KVServer) hasExecuted(client int64, serialNumber int, key string) (bool, string) {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	executed := false
	var val string
	if kv.clientSerialNumber[client] >= serialNumber {
		executed = true
		val = kv.keyValues[key]
	}

	return executed, val
}

func (kv *KVServer) startOp(op Op) bool {
	// timer := time.NewTimer(2 * time.Second)
	// defer timer.Stop()
	index, _, isLeader := kv.rf.Start(op)
	if !isLeader {
		return false
	}

	kv.addWaitForIndex(index)

	for {
		if kv.checkIndex(index, op) {
			return true
		}

		if !kv.isLeader() {
			break
		}

		time.Sleep(2 * time.Millisecond)
	}

	return false
}

func (kv *KVServer) addWaitForIndex(index int) {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	kv.requiredlogs[index] = 1
}

func (kv *KVServer) isLeader() bool {
	_, isLeader := kv.rf.GetState()
	return isLeader
}

func (kv *KVServer) checkIndex(index int, command interface{}) bool {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	appliedCommand, ok := kv.appliedlogs[index]
	if ok {
		delete(kv.appliedlogs, index)
		delete(kv.requiredlogs, index)
	}

	return ok && appliedCommand == command
}

func (kv *KVServer) readFromApplyCh() {
	for !kv.killed() {
		for msg := range kv.applyCh {
			if msg.CommandValid {
				kv.dealWithCommand(msg.CommandIndex, msg.Command)
			} else if msg.SnapshotValid {
				kv.dealWithSnapShot()
			} else {
				// todo log leaderId
			}
		}
	}
}

func (kv *KVServer) dealWithCommand(commandIndex int, command interface{}) {
	kv.mu.Lock()
	defer kv.mu.Unlock()

	// todo record newest serial number

	_, ok := kv.requiredlogs[commandIndex]
	if ok {
		kv.appliedlogs[commandIndex] = command
	}

	// persist putandappend result
	op, ok := command.(Op)
	if ok {
		serialNumber, ok := kv.clientSerialNumber[op.Client]
		if ok && serialNumber >= op.SerialNumber {
			return
		}

		if op.OpType == "Put" {
			kv.PutValNL(op.Key, op.Value)
		} else if op.OpType == "Append" {
			kv.AppendValNL(op.Key, op.Value)
		}

		kv.clientSerialNumber[op.Client] = op.SerialNumber
	}
}

func (kv *KVServer) PutValNL(key string, val string) {
	kv.keyValues[key] = val
}

func (kv *KVServer) AppendValNL(key string, val string) {
	_, ok := kv.keyValues[key]
	if !ok {
		kv.PutValNL(key, val)
		return
	}

	kv.keyValues[key] += val
}

func (kv *KVServer) dealWithSnapShot() {

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
	// Your code here, if desired.
}

func (kv *KVServer) killed() bool {
	z := atomic.LoadInt32(&kv.dead)
	return z == 1
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

	kv := new(KVServer)
	kv.me = me
	kv.maxraftstate = maxraftstate

	// You may need initialization code here.

	kv.applyCh = make(chan raft.ApplyMsg)
	kv.rf = raft.Make(servers, me, persister, kv.applyCh)

	// You may need initialization code here.
	kv.keyValues = make(map[string]string)
	kv.appliedlogs = make(map[int]interface{})
	kv.requiredlogs = make(map[int]int)
	kv.clientSerialNumber = make(map[(int64)]int)

	go kv.readFromApplyCh()

	return kv
}
