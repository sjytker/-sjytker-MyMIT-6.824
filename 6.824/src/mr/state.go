package mr

// task state
const (
	IDLE = 0
	IN_QUEUE = 1
	IN_PROGRESS = 2
	COMPLETED = 3
	ERR = 4
)

// worker request state
const (
	GOT = 0
	BARRIER = 1
	FIN = 2
	MAP = 3
	REDUCE = 4
)
