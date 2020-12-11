package mr

//
// RPC definitions.
//
// remember to capitalize all names.
//

import "os"
import "strconv"


// required for a task
type TaskArgs struct {
	WorkerId int
}

type TaskReply struct {
	TaskReplyState int
	*Task
}

// task was finished, report master
type ReportArgs struct {
	*Task
}

type ReportReply struct {

}

// worker process start, register it to master
type RegisterArgs struct {
}

type RegisterReply struct {
	WorkerId int
}

type TestArgs struct {}
type TestReply struct {}



// Cook up a unique-ish UNIX-domain socket name
// in /var/tmp, for the master.
// Can't use the current directory since
// Athena AFS doesn't support UNIX-domain sockets.
func masterSock() string {
	s := "/var/tmp/824-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}


func workerSock() string {
	s := "/var/tmp/824-mr-"
	s += strconv.Itoa(os.Getuid())
	return s
}
