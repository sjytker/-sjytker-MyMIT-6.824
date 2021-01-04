package mr

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sort"
	"strconv"
	"time"
)
import "log"
import "hash/fnv"



// Map functions return a slice of KeyValue.
type KeyValue struct {
	Key   string
	Value string
}

// to sort KeyValue by key
type byKey []KeyValue

func (p byKey) Len() int {return len(p)}
func (p byKey) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}
func (p byKey) Less(i, j int) bool {
	return p[i].Key < p[j].Key
}

type Worker struct {
	// Your definitions here.
	//intermediate [][]KeyValue
	mapf func(string, string) []KeyValue
	reducef func(string, []string) string
	workerId int
}


func NewWorker (mapf func(string, string) []KeyValue, reducef func(string, []string) string) *Worker{
	worker := Worker{}
	worker.mapf = mapf
	worker.reducef = reducef
	return &worker
}


//
// use ihash(key) % NReduce to choose the reduce
// task number for each KeyValue emitted by Map.
//
func ihash(key string) int {
	h := fnv.New32a()
	h.Write([]byte(key))
	return int(h.Sum32() & 0x7fffffff)
}



func (worker *Worker) doMapTask(task *Task) error{
	fmt.Printf("worker %v opening file : %v\n", worker.workerId, task.FileName)
	file, err := os.Open(task.FileName)
	if err != nil {
		log.Fatal("cannot open %v", task.FileName)
	}
	defer file.Close()
	content, err := ioutil.ReadAll(file)
	if err != nil {
		log.Fatal("cannot read %v", task.FileName)
	}
	inter := worker.mapf(task.FileName, string(content))
	partitions := make([][]KeyValue, task.NReduce)

	for _, kv := range inter {
		key := kv.Key
		value := kv.Value
		bucket := ihash(key) % task.NReduce
		partitions[bucket] = append(partitions[bucket], KeyValue{key, value})
	}

	for i, par := range partitions {
		interName := "./mr-" + strconv.Itoa(task.Imap) + "-" + strconv.Itoa(i)
	//	fmt.Printf("worker %v, saving interfile :%v \n", worker.workerId, interName)
		file, err = os.Create(interName)
		if err != nil {
			log.Fatal(err)
		}
		defer file.Close()
		encoder := json.NewEncoder(file)
		for _, kv := range par {
			if err = encoder.Encode(&kv); err != nil {
				log.Fatal(err)
			}
		}
	}
	return nil
}


func (worker *Worker) doReduceTask(task *Task) error{
	oName := "mr-out-" + strconv.Itoa(task.IReduce)
	fmt.Println("***reduce file name*** : " + oName)
	ofile, _ := os.Create(oName)
	defer ofile.Close()
	intermediate := make([]KeyValue, 0)
	for i := 0; i < task.NMap; i ++ {
		mName := "mr-" + strconv.Itoa(i) + "-" + strconv.Itoa(task.IReduce)
		file, err := os.Open(mName)
		if err != nil {
			log.Fatal("err in do reduce")
		}
		defer file.Close()
		dec := json.NewDecoder(file)
		for dec.More() {
			var kv KeyValue
			if err := dec.Decode(&kv); err != nil {
				break
			}
			intermediate = append(intermediate, kv)
		}
	}
	// sort key, then reduce
	sort.Sort(byKey(intermediate))
	n := len(intermediate)
	//
	// call Reduce on each distinct key in intermediate[],
	// and print the result to mr-out-0.
	//
	i := 0
	for i < n {
		j := i + 1
		for j < n && intermediate[j].Key == intermediate[i].Key {
			j++
		}
		values := []string{}
		for k := i; k < j; k++ {
			values = append(values, intermediate[k].Value)
		}
		output := worker.reducef(intermediate[i].Key, values)

		// this is the correct format for each line of Reduce output.
		fmt.Fprintf(ofile, "%v %v\n", intermediate[i].Key, output)

		i = j
	}
	return nil
}


func (worker *Worker) ReportTask(task *Task) error {

	args := ReportArgs{task}
	reply := ReportReply{}
	if ok := call("Master.ReportTask", &args, &reply); !ok {
		log.Fatal("error in ReportTask rpc")
	}
	return nil
}


//func (worker *Worker) ReportReduceTask(iReduce int) error {
//
//	args := reportArgs{}
//	reply := reportReply{}
//	args.iReduce = iReduce
//	if ok := call("Master.ReportReduceTask", args, reply); !ok {
//		log.Fatal("error in ReportMapTask rpc")
//	}
//	return nil
//}


func (worker *Worker) run() {

//	cnt := 0
	for {
		args := TaskArgs{worker.workerId}
		reply := TaskReply{}
		if ok := call("Master.ReqTask", &args, &reply); !ok {
			log.Fatal("error when requesting rpc")
		}
		task := reply.Task
		state := reply.TaskReplyState
		fmt.Printf("worker %v got a task : %v\t\t%v\n", worker.workerId, state, task)
		if state == FIN {
			break
		} else if state == BARRIER {
			time.Sleep(2 * time.Second)
			continue
		}
		fmt.Printf("task state : %v, phase : %v\n", state, task.Phase)
		if state == GOT && task.Phase == "map" {
			if err := worker.doMapTask(task); err != nil {
				log.Fatal("err in doMapTask")
			}
			fmt.Println("do map finish ")
			if err := worker.ReportTask(task); err != nil {
				log.Fatal("err in report")
			}
		} else if state == GOT && task.Phase == "reduce" {
			if err := worker.doReduceTask(task); err != nil {
				log.Fatal("err in doReduceTask")
			}
			if err := worker.ReportTask(task); err != nil {
				log.Fatal("err in report")
			}
		} else {
			// barrier
			// map or reduce is busy doing, just wait
			time.Sleep(time.Second)
		}
	}
}

func (worker *Worker) register() error{
	args := RegisterArgs{}
	reply := RegisterReply{}
	if ok := call("Master.WorkerRegister", &args, &reply); !ok {
		log.Fatal("worker register fail")
	}
	worker.workerId = reply.WorkerId
	return nil
}


//
// main/mrworker.go calls this function.
//
func MakeWorker(mapf func(string, string) []KeyValue,
	reducef func(string, []string) string) {

	// Your worker implementation here.
	worker := NewWorker(mapf, reducef)
	if err := worker.register(); err != nil {
		log.Fatal("err in register")
	}
	worker.run()
}




