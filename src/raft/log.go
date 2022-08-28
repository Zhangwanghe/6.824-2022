package raft

type Entry struct {
	term    int
	command interface{}
}

type Log struct {
	logs       []Entry
	startIndex int
}

func makeEmptyLog() Log {
	return Log{make([]Entry, 1), 0}
}

func getLastLogIndex(log *Log) int {
	return len(log.logs) - 1
}
