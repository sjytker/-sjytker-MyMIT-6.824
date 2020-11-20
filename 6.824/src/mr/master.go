package mr

import (
	"log"
	"sync"
	"time"
)
import "net"
import "os"
import "net/rpc"
import "net/http"

type Master struct {
	// Your definitions here.
	nTask, nReduce int
	allTask []TaskState
	files []string
	seq int
	mutex sync.Mutex
	freeTaskQueue []Task
	runningWorker map[int]string     // k : workerId, v : task
	phase string    // map, reduce, barrier, fin
}


type IdleType struct {
	isIdle bool
	taskType string
}


func NewMaster(files []string, nReduce int) *Master {
	master := Master{}
	nTask := len(files)
	master.nTask = nTask
	master.nReduce = nReduce
	master.files = files
	max := nTask
	master.phase = "map"
	master.allTask = make([]TaskState, nTask, nTask)
	if nReduce > max {
		max = nReduce
	}
	master.freeTaskQueue = make([]Task, 0, max)
	for i := 0; i < nTask; i ++ {
	//	master.freeTaskQueue = append(master.freeTaskQueue, i)
		master.allTask[i] = TaskState{IDLE, -1, time.Now()}
	}
	return &master
}


// Your code here -- RPC handlers for the worker to call.

//
// an example RPC handler.
//
// the RPC argument and reply types are defined in rpc.go.
// finish all the map phase, before entering the reduce phase
func (m *Master) allocateTask(args *TaskArgs, reply *TaskReply) error {

	//for i := 0; i < m.nTask; i ++ {
	//	if m.mapState[i] == IDLE {
	//		reply.taskType = "map"
	//		reply.fileName = m.files[i]
	//		reply.nReduce = m.nReduce
	//		reply.mapTaskNum = i
	//		m.runningWorker[args.workerId] = i
	//		m.taskStateList[i] = TaskState{IN_PROGRESS, args.workerId, time.Now()}
	//		break
	//	} else if m.mapState[i] == IN_PROGRESS {
	//		reply.taskType = "waiting map"
	//	}
	//}
	//if reply.taskType != "waiting map" {
	//	for i := 0; i < m.nReduce; i ++ {
	//		if m.reduceState[i] == IDLE {
	//			reply.taskType = "reduce"
	//			reply.reduceTask = i
	//			break
	//		} else if m.reduceState[i] == IN_PROGRESS {
	//			reply.taskType = "waiting reduce"
	//		}
	//	}
	//}


	return nil
}


//
// start a thread that listens for RPCs from worker.go
//
func (m *Master) server() {
	rpc.Register(m)
	rpc.HandleHTTP()
	//l, e := net.Listen("tcp", ":1234")
	sockname := masterSock()
	os.Remove(sockname)
	l, e := net.Listen("unix", sockname)
	if e != nil {
		log.Fatal("listen error:", e)
	}
	go http.Serve(l, nil)
}

//
// main/mrmaster.go calls Done() periodically to find out
// if the entire job has finished.
//
func (m *Master) Done() bool {
	if m.phase == "map" {
		return false
	}
	for i := 0; i < m.nReduce; i ++ {
		if m.allTask[i].status != COMPLETED {
			return false
		}
	}
	return true
}


func (m *Master) ReqTask(args *TaskArgs, reply *TaskReply) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	if m.phase == "barrier" {
		reply.taskReplyState = BARRIER
	} else if m.phase == "fin" {
		reply.taskReplyState = FIN
	} else {
		reply.Task = &m.freeTaskQueue[0]
		m.freeTaskQueue = m.freeTaskQueue[1:]
	}
	return nil
}


func (m *Master) WorkerRegister(args *RegisterArgs, reply *RegisterReply) error{
	m.mutex.Lock()
	defer m.mutex.Unlock()
	reply.workerId = m.seq
	m.seq ++
	return nil
}


func (master *Master) ReportMapTask(args *reportArgs, reply *reportReply) error{
	task := args.Task
	phase := task.phase
	if phase == "map" {
		master.allTask[task.imap].status = COMPLETED
	} else {
		master.allTask[task.iReduce].status = COMPLETED
	}
	go master.schedule()
	return nil
}



func (master *Master) getTask(i int) *Task{
	task := Task{}
	if master.phase == "map" {
		task = Task{
			done : false,
			phase : "map",
			fileName: master.files[i],
			imap : i,
			iReduce: -1,
			nMap: master.nTask,
			nReduce: master.nReduce,
		}
	} else {
		task = Task{
			done : false,
			phase : "reduce",
			fileName: "",
			imap : -1,
			iReduce: i,
			nMap: master.nTask,
			nReduce: master.nReduce,
		}
	}
	return &task
}


func (master *Master) schedule() {
	master.mutex.Lock()
	defer master.mutex.Unlock()
	finishMap := true
	finishReduce := true
	barrier := true
	if master.phase == "map" {
		for i, t := range master.allTask {
			switch t.status {
			case IDLE:
				finishMap = false
				barrier = false
				master.freeTaskQueue = append(master.freeTaskQueue, *master.getTask(i))
				master.allTask[i].status = IN_QUEUE
			case IN_QUEUE:
				finishMap = false
			case IN_PROGRESS:
				finishMap = false
				if time.Now().Sub(t.startTime).Seconds() > 10 {
					master.freeTaskQueue = append(master.freeTaskQueue, *master.getTask(i))
					master.allTask[i].status = IN_QUEUE
				}
			}
		}
	} else {
		for i, t := range master.allTask {
			switch t.status {
			case IDLE:
				finishReduce = false
				barrier = false
				master.freeTaskQueue = append(master.freeTaskQueue, *master.getTask(i))
				master.allTask[i].status = IN_QUEUE
			case IN_QUEUE:
				finishReduce = false
			case IN_PROGRESS:
				finishReduce = false
				if time.Now().Sub(t.startTime).Seconds() > 10 {
					master.freeTaskQueue = append(master.freeTaskQueue, *master.getTask(i))
					master.allTask[i].status = IN_QUEUE
				}
			}
		}
	}
	if finishMap && master.phase == "map" {
		// init reduce phase
		master.phase = "reduce"
		for i := 0; i < master.nReduce; i ++ {
			master.allTask[i] = TaskState{
				status:    IDLE,
				workerId:  -1,
				startTime: time.Now(),
			}
		}
	}
	if finishReduce {
		master.phase = "fin"
	} else if barrier {
		master.phase = "barrier"
	}
}


func (m *Master) TickSchedule() {
	if !m.Done() {
		go m.schedule()
		time.Sleep(time.Second)
	}
}

//
// create a Master.
// main/mrmaster.go calls this function.
// nReduce is the number of reduce tasks to use.
//
func MakeMaster(files []string, nReduce int) *Master {

	m := NewMaster(files, nReduce)
	m.server()
	go m.TickSchedule()
	return m
}

