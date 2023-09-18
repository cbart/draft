package main

import (
	"fmt"
	"math/rand"
	"time"
)

func main() {
	fmt.Println("Hello World!")
}

func electionInterval() time.Duration {
	return time.Duration(150+rand.Intn(150)) * time.Millisecond
}

type AppendEntryRequest struct {
	Term         int64
	LeaderID     int64
	PrevLogIndex int64
	PrevLogTerm  int64
	Entries      []LogEntry
	LeaderCommit int64
	Result       chan<- AppendEntryResult
}

type AppendEntryResult struct {
	Term    int64
	Success bool
}

type server struct {
	CurrentTerm int64
	VotedFor    *int64
	Log         []LogEntry
	CommitIndex int64
	LastApplied int64
	SelfID      int64
	Clients     map[int64]Client
	election    *time.Timer
	appendEntry <-chan AppendEntryRequest
}

type Client interface {
	RequestVote(RequestVoteRequest) RequestVoteResult
}

type RequestVoteRequest struct {
	Term         int64
	CandidateID  int64
	LastLogIndex int64
	LastLogTerm  int64
}
type RequestVoteResult struct {
	Term        int64
	VoteGranted bool
}

type LogEntry struct {
	Index int64
	Term  int64
	Value interface{}
}

type follower struct {
	server
}

type Node interface {
	Next() Node
}

func (f *follower) Next() Node {
	f.election = time.NewTimer(electionInterval())
	for {
		select {
		case r := <-f.appendEntry:
			if r.Term < f.CurrentTerm {
				// Request has stale term, ignore
				r.Result <- AppendEntryResult{
					Term:    f.CurrentTerm,
					Success: false,
				}
				continue
			}
			if !f.containsEntry(r.PrevLogIndex, r.PrevLogTerm) {
				// PrevLogIndex/Term not found in logs, ignore
				r.Result <- AppendEntryResult{
					Term:    f.CurrentTerm,
					Success: false,
				}
				continue
			}
			f.updateEntries(r.Entries)
			if r.LeaderCommit > f.CommitIndex {
				f.CommitIndex = r.LeaderCommit
				if lastNewEntryIdx := r.Entries[len(r.Entries)-1].Index; f.CommitIndex > lastNewEntryIdx {
					f.CommitIndex = lastNewEntryIdx
				}
			}
			f.stopElectionTimer()
			f.election.Reset(electionInterval())
		case <-f.election.C:
			f.stopElectionTimer()
			return &candidate{f.server}
		}
	}
}

func (f *follower) containsEntry(index int64, term int64) bool {
	if index < 0 || int64(len(f.Log)) <= index {
		return false
	}
	return f.Log[index].Term == term
}

func (f *follower) updateEntries(entries []LogEntry) {
	for i, e := range entries {
		if f.Log[e.Index].Term != e.Term {
			f.Log = append(f.Log[:e.Index], entries[i:]...)
			break
		}
	}
}

func (s *server) stopElectionTimer() {
	if s.election == nil {
		s.election = time.NewTimer(electionInterval())
		return
	}
	t := s.election
	if !t.Stop() {
		<-t.C
	}
}

type candidate struct {
	server
}

func (c *candidate) Next() Node {
	c.CurrentTerm++
	self := int64(0)
	c.VotedFor = &self
	c.stopElectionTimer()
	// Request votes from other servers
	results := make(chan RequestVoteResult, len(c.Clients))
	for id, client := range c.Clients {
		go func(id int64, client Client) {
			results <- client.RequestVote(RequestVoteRequest{})
		}(id, client)
	}
	var grantedVotes int64
	for {
		select {
		case r := <-results:
			if r.Term > c.CurrentTerm {
				// Candidate has received messages from next term, stop
				return &follower{server: c.server}
			}
			if r.VoteGranted {
				grantedVotes++
			}
			if grantedVotes > int64(len(c.Clients)/2) {
				// Candidate received votes from majority, become leader
				return nil // Not implemented yet &leader{server: c.server}
			}
		case r := <-c.appendEntry:
			if r.Term > c.CurrentTerm {
				return &follower{server: c.server}
			}
		case <-c.election.C:
			return c
		}
	}
}

// Raft:
// 1. Node starts in Follower state.
// 2. If node does not receive heartbeat from Leader within election timeout, it becomes candidate.
//
// On RequestVote:
// 1. If receiving node has not voted in this term, it votes
// 2. Reset own election timeout.
//
// Candidate - start of new election term
// 1. Node broadcasts RequestVote RPCs to all other servers.
// 2. If node receives votes from majority of servers, it becomes Leader.
//
// Leader
// 1. Leader periodically sends heartbeat to all other servers to maintain leadership.
// 2. If response to heartbeat has higher term, leader steps down.
//
// 1. On a write request, leader replicates log entry to majority of other servers before responding to client.
// 2. When majority of nodes have replicated the entry, leader responds to the client and sends commit to followers.
//
// Election timeout - randomized between 150-300ms
