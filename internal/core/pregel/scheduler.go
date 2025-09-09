package pregel

import (
    "context"
    "runtime"
    "sync"
    "sync/atomic"
)
import imetrics "github.com/flowgraph/flowgraph/internal/infrastructure/metrics"

// WorkStealingScheduler implements work stealing for load balancing
// PRINCIPLES:
// - Dynamic work distribution
// - Load balancing across workers
// - Efficient parallel execution

type WorkStealingScheduler struct {
    queues     []chan VertexTask
    numWorkers int
    counter    int64
    ctx        context.Context
    cancel     context.CancelFunc
    wg         sync.WaitGroup
}

type VertexTask struct {
	VertexID string
	Program  VertexProgram
	State    map[string]interface{}
	Messages []*Message
	Result   chan VertexResult
}

type VertexResult struct {
	VertexID string
	State    map[string]interface{}
	Messages []*Message
	Halt     bool
	Error    error
}

func NewWorkStealingScheduler(numWorkers int, queueCapacity int) *WorkStealingScheduler {
    // Guard against invalid values; default to number of CPUs for best parallelism.
    if numWorkers <= 0 {
        numWorkers = runtime.NumCPU()
        if numWorkers < 1 {
            numWorkers = 1
        }
    }
    if queueCapacity <= 0 {
        queueCapacity = 100
    }

    queues := make([]chan VertexTask, numWorkers)
    for i := range queues {
        queues[i] = make(chan VertexTask, queueCapacity)
    }

	ctx, cancel := context.WithCancel(context.Background())

    ws := &WorkStealingScheduler{
        queues:     queues,
        numWorkers: numWorkers,
        ctx:        ctx,
        cancel:     cancel,
    }
    imetrics.SetSchedulerWorkers(numWorkers)
    return ws
}

func (ws *WorkStealingScheduler) Schedule(task VertexTask) {
    // Round-robin assignment to worker queues
    worker := atomic.AddInt64(&ws.counter, 1) % int64(ws.numWorkers)
    ws.queues[worker] <- task
    imetrics.AddSchedulerQueued(1)
}

func (ws *WorkStealingScheduler) StartWorkers() {
	for i := 0; i < ws.numWorkers; i++ {
		ws.wg.Add(1)
		go ws.worker(i)
	}
}

func (ws *WorkStealingScheduler) worker(id int) {
	defer ws.wg.Done()

	for {
		select {
		case <-ws.ctx.Done():
			return
		case task := <-ws.queues[id]:
			ws.executeTask(task)
		default:
			// Try to steal work from other queues
			if !ws.stealWork(id) {
				// No work available, check context and continue
				select {
				case <-ws.ctx.Done():
					return
				default:
					continue
				}
			}
		}
	}
}

func (ws *WorkStealingScheduler) stealWork(workerID int) bool {
	for i := 0; i < ws.numWorkers; i++ {
		if i == workerID {
			continue
		}
		select {
		case task := <-ws.queues[i]:
			ws.executeTask(task)
			return true
		default:
		}
	}
	return false
}

func (ws *WorkStealingScheduler) executeTask(task VertexTask) {
	state, messages, halt, err := task.Program.Compute(task.VertexID, task.State, task.Messages)
	result := VertexResult{
		VertexID: task.VertexID,
		State:    state,
		Messages: messages,
		Halt:     halt,
		Error:    err,
	}
	task.Result <- result
}

func (ws *WorkStealingScheduler) Stop() {
    ws.cancel()  // Signal workers to stop
    ws.wg.Wait() // Wait for all workers to finish
    for _, queue := range ws.queues {
        close(queue)
    }
}

// SchedulerStats reports scheduler-level metrics.
type SchedulerStats struct {
    NumWorkers   int
    QueueLengths []int
    TotalQueued  int
}

// Stats returns a snapshot of scheduler metrics.
func (ws *WorkStealingScheduler) Stats() SchedulerStats {
    lengths := make([]int, len(ws.queues))
    total := 0
    for i, q := range ws.queues {
        l := len(q)
        lengths[i] = l
        total += l
    }
    return SchedulerStats{
        NumWorkers:   ws.numWorkers,
        QueueLengths: lengths,
        TotalQueued:  total,
    }
}
