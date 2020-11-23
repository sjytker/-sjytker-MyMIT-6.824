package mr

import "time"


// in the MR paper, M = 20000, R = 5000, workers = 2000
// In my simulation, I use 4 core 8 thread cpu. And there are 8 txt input
// So I let workers = 4, M = 8, R = 10
//const (
//	NUM_WORKERS = 4
//	NUM_MAP = 8
//	NUM_REDUCE = 10
//)

type TaskState struct {
	Status int
	WorkerId int
	StartTime time.Time
}

type Task struct {
	Done bool   // all task done?
	Phase string
	FileName string  // map file name
	Imap int
	IReduce int   // iReduce
	NMap int
	NReduce int		// reduce split num
}

