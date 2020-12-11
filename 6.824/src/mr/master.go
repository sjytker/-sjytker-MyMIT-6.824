package mr

import (
	"fmt"
	"log"
	"strings"
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
	master.phase = "map"
	master.allTask = make([]TaskState, nTask, nTask)
	master.freeTaskQueue = make([]Task, 0, nTask)
	for i := 0; i < nTask; i ++ {
	//	master.freeTaskQueue = append(master.freeTaskQueue, i)
		master.allTask[i] = TaskState{IDLE, -1, time.Now()}
	}
	return &master
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
//	l, e := net.Listen("tcp", "localhost:7000")
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
	//if m.phase != "reduce_barrier" {
	//	return false
	//}
	//for i := 0; i < m.nReduce; i ++ {
	//	if m.allTask[i].Status != COMPLETED {
	//		return false
	//	}
	//}
	return m.phase == "fin"
}


func (m *Master) ReqTask(args *TaskArgs, reply *TaskReply) error {
	//m.mutex.Lock()
	//defer m.mutex.Unlock()
	fmt.Println("in master ReqTask,  phase : " , m.phase, ", m.done() : ", m.Done())
	fmt.Println(m.freeTaskQueue)
	if m.phase == "map_barrier" || m.phase == "reduce_barrier" {
		reply.TaskReplyState = BARRIER

	} else if m.phase == "fin" {
		reply.TaskReplyState = FIN
	} else {
		m.mutex.Lock()
		defer m.mutex.Unlock()
		if m.phase == "map" || m.phase == "reduce" {
			if len(m.freeTaskQueue) == 0 {
				fmt.Println("no task in queue")
			} else {
				fmt.Println("getting from queue : ", m.freeTaskQueue[0])
				task := &m.freeTaskQueue[0]
				m.freeTaskQueue = m.freeTaskQueue[1:]
				if m.phase == "map" {
					m.allTask[task.Imap].Status = IN_PROGRESS
				} else {
					m.allTask[task.IReduce].Status = IN_PROGRESS
				}
				reply.Task = task
				reply.TaskReplyState = GOT
			}
		}
	}

//	go m.schedule()
	return nil
}


func (m *Master) WorkerRegister(args *RegisterArgs, reply *RegisterReply) error{
	fmt.Printf("*********in master WorkerRegister, current worker: %v\n", m.seq)
	m.mutex.Lock()
	defer m.mutex.Unlock()
	reply.WorkerId = m.seq
	m.seq ++
	return nil
}


func (m *Master) ReportTask(args *ReportArgs, reply *ReportReply) error{
	m.mutex.Lock()
	defer m.mutex.Unlock()
	task := args.Task
	phase := task.Phase
//	fmt.Println("in ReportTask : ", task)
	if find := strings.Contains(phase, "map"); find {
		m.allTask[task.Imap].Status = COMPLETED
	}
	if find := strings.Contains(phase, "reduce"); find {
		m.allTask[task.IReduce].Status = COMPLETED
	}
//	fmt.Println("in ReportTask, imap, ireduce : ", args.Imap,  args.IReduce)
//	if args.Imap != -1 {
//		master.allTask[task.Imap].Status = COMPLETED
//	}
//	if args.IReduce != -1 {
//		master.allTask[task.IReduce].Status = COMPLETED
//	}
	go m.schedule()
	return nil
}



func (master *Master) getTask(i int) *Task{
	task := Task{}
	if find := strings.Contains(master.phase, "map"); find {
		task = Task{
			Done : false,
			Phase : "map",
			FileName: master.files[i],
			Imap : i,
			IReduce: -1,
			NMap: master.nTask,
			NReduce: master.nReduce,
		}
	} else {
		task = Task{
			Done : false,
			Phase : "reduce",
			FileName: "",
			Imap : -1,
			IReduce: i,
			NMap: master.nTask,
			NReduce: master.nReduce,
		}
	}
	return &task
}


func (master *Master) schedule() {
	master.mutex.Lock()
//	defer master.mutex.Unlock()
//	finishMap := true
//	finishReduce := true
	barrier := true
	hasUnlocked := false
	if master.phase == "map" {
	//	finishReduce = false
		for i, t := range master.allTask {
			switch t.Status {
			case IDLE:
			//	finishMap = false
				barrier = false
				master.freeTaskQueue = append(master.freeTaskQueue, *master.getTask(i))
				master.allTask[i].Status = IN_QUEUE
			case IN_QUEUE:
		//		finishMap = false
				barrier = false
			}
		}
		if barrier {
			master.phase = "map_barrier"
		}
	//	fmt.Println(master.freeTaskQueue)
	} else if master.phase == "map_barrier" {
		allComplete := true
	//	finishReduce = false
		fmt.Println("in map_barrier, freetaskqueue len : ", len(master.freeTaskQueue))
		for i, t := range master.allTask {
			fmt.Printf("%v\t", t.Status)
			switch t.Status {
			case IN_PROGRESS:
		//		finishMap = false
				if time.Now().Sub(t.StartTime).Seconds() > 10 {
					master.freeTaskQueue = append(master.freeTaskQueue, *master.getTask(i))
					master.allTask[i].Status = IN_QUEUE
					master.phase = "map"
				}
			}
			if t.Status != COMPLETED {
				allComplete = false
			}
		}
		fmt.Println("")

		if allComplete {
			master.phase = "reduce"
			master.freeTaskQueue = make([]Task, 0, master.nReduce)
			master.allTask = make([]TaskState, master.nReduce, master.nReduce)
			fmt.Println("initing reduce task")
			for i := 0; i < master.nReduce; i ++ {
				fmt.Println(i)
				master.allTask[i] = TaskState{
					Status:    IDLE,
					WorkerId:  -1,
					StartTime: time.Now(),
				}
			}
			fmt.Println("initing complete")
			hasUnlocked = true
			master.mutex.Unlock()
			master.schedule()
		}
	} else if master.phase == "reduce" {
		fmt.Println("**********in master schedule, reduce phase********")
		for i, t := range master.allTask {
			switch t.Status {
			case IDLE:
		//		finishReduce = false
				barrier = false
			//	fmt.Printf("getting task %v in queue\n", i)
				master.freeTaskQueue = append(master.freeTaskQueue, *master.getTask(i))
				master.allTask[i].Status = IN_QUEUE
			case IN_QUEUE:
		//		finishReduce = false
				barrier = false
			case IN_PROGRESS:
		//		finishReduce = false
				if time.Now().Sub(t.StartTime).Seconds() > 10 {
					master.freeTaskQueue = append(master.freeTaskQueue, *master.getTask(i))
					master.allTask[i].Status = IN_QUEUE
				}
			//case COMPLETED:
			//	cnt ++
			}
		}
		if barrier {
			master.phase = "reduce_barrier"
		}
	} else if master.phase == "reduce_barrier" {
		allComplete := true
		for i, t := range master.allTask {
			switch t.Status {
			case IN_PROGRESS:
			//	finishMap = false
				if time.Now().Sub(t.StartTime).Seconds() > 10 {
					master.freeTaskQueue = append(master.freeTaskQueue, *master.getTask(i))
					master.allTask[i].Status = IN_QUEUE
					master.phase = "reduce"
				}
			}
			if t.Status != COMPLETED {
				allComplete = false
			}
		}
		if allComplete {
			master.phase = "fin"
		}
	}

	if !hasUnlocked {
		master.mutex.Unlock()
	}
	//fmt.Println("current freeTaskQueue")
	//fmt.Println("finishMap : ", finishMap)
	//fmt.Println("finishReduce : ", finishReduce)
//	fmt.Println("barrier : ", barrier)
//	for _, t := range master.freeTaskQueue {
//	//	fmt.Printf("%v\t", t)
//		fmt.Println(t)
//	}
	fmt.Println("")
}


func (m *Master) TickSchedule() {
	for !m.Done() {
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

