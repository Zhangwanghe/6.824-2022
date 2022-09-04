package raft

import "fmt"

type Entry struct {
	Term    int
	Command interface{}
}

func (e *Entry) String() string {
	return fmt.Sprintf("term = %d, command = %v", e.Term, e.Command)
}

type Log struct {
	Logs       []Entry
	StartIndex int
}

func makeEmptyLog() Log {
	// log start from 1
	return Log{make([]Entry, 1), 0}
}

func appendLogNL(log *Log, term int, command interface{}) {
	log.Logs = append(log.Logs, Entry{term, command})
}

func getLastLogIndexNL(log *Log) int {
	return len(log.Logs) - 1
}

func getLastLogTermNL(log *Log) int {
	return log.Logs[len(log.Logs)-1].Term
}

func getPrevLogAndNewEntriesNL(log *Log, index int) (int, int, []Entry) {
	entries := make([]Entry, len(log.Logs)-index)
	copy(entries, log.Logs[index:])

	if index <= 1 {
		return index - 1, -1, entries
	} else {
		return index - 1, log.Logs[index-1].Term, entries
	}
}

func hasPrevLogNL(log *Log, index int, term int) bool {
	if index <= 0 {
		return true
	}

	return len(log.Logs) > index && log.Logs[index].Term == term
}

func getLogInfoBeforeConflictingNL(log *Log, index int) (int, int) {
	if len(log.Logs) <= index {
		index = len(log.Logs) - 1
	}

	conflictingTerm := log.Logs[index].Term
	var conflictingIndex int

	for i := index; i >= 0; i-- {
		if conflictingTerm != log.Logs[i].Term {
			conflictingIndex = i
			break
		}
	}

	return conflictingTerm, conflictingIndex
}

func appendAndRemoveConflictinLogFromIndexNL(log *Log, lastLogIndex int, entries []Entry) {
	if len(entries) == 0 {
		// heartbeat
		return
	}

	i := lastLogIndex + 1
	for ; i < Min(len(log.Logs), lastLogIndex+1+len(entries)); i++ {
		if log.Logs[i].Term != entries[i-lastLogIndex-1].Term {
			break
		}
	}

	if i-lastLogIndex-1 >= len(entries) {
		return
	}

	log.Logs = append(log.Logs[:i], entries[i-lastLogIndex-1:]...)
}

func getCommitLogNL(log *Log, prevCommit int, newCommit int) []ApplyMsg {
	ret := make([]ApplyMsg, newCommit-prevCommit)
	for i := prevCommit + 1; i <= newCommit; i++ {
		ret[i-prevCommit-1].Command = log.Logs[i].Command
		ret[i-prevCommit-1].CommandIndex = i
		ret[i-prevCommit-1].CommandValid = true
	}
	return ret
}

func getLastLogIndexForTermNL(log *Log, term int) int {
	var left = 0
	var right = len(log.Logs) - 1

	for left+1 < right {
		var mid = (left + right) / 2
		if log.Logs[mid].Term > term {
			right = mid
		} else {
			left = mid
		}
	}

	return left
}

func getTermForGivenIndexNL(log *Log, index int) int {
	return log.Logs[index].Term
}
