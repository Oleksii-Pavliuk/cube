package worker

import (
	"fmt"
	"log"
	"time"

	"github.com/golang-collections/collections/queue"

	"cube/stats"
	"cube/store"
	"cube/task"
)

type Worker struct {
    Name      string
    Queue     queue.Queue
    Db    		store.Store
    TaskCount int
		Stats	  *stats.Stats
}

func (w *Worker ) CollectStats() {
	for {
		log.Printf("Collecting stats")
		w.Stats = stats.GetStats()
		w.Stats.TaskCount = w.TaskCount
		time.Sleep(15 * time.Second)
	}
}

func New(name string, taskDbType string) *Worker {
	w := Worker{
			Name:  name,
			Queue: *queue.New(),
	}

	var s store.Store
	var err error
	switch taskDbType {
	case "memory":
		s = store.NewInMemoryTaskStore()
	case "persistent":
		filename := fmt.Sprintf("%s_tasks.db", name)
		s, err = store.NewTaskStore(filename, 0600, "tasks")
	}

	if err != nil {
		log.Fatalf("unable to create task store: %v", err)
	}
	w.Db = s
	return &w
}


func (w *Worker) AddTask(t task.Task){
	w.Queue.Enqueue(t)
}

func (w *Worker) GetTasks() []*task.Task{
	taskList, err := w.Db.List()
	if err != nil {
			log.Printf("error getting list of tasks: %v\n", err)
			return nil
	}

	return taskList.([]*task.Task)
}

func (w *Worker) runTask() task.DockerResult{
	t := w.Queue.Dequeue()

	if t == nil {
		log.Println("No tasks in the queue")
		return task.DockerResult{Error: nil}
	}

	taskQueued := t.(task.Task)
	taskResult,err := w.Db.Get(taskQueued.ID.String());
	if err != nil {
		log.Printf("Task with id %s not found\n",taskQueued.ID)
		return task.DockerResult{Error: nil}
	}
	taskPersisted := taskResult.(*task.Task)
	if taskPersisted == nil {
		taskPersisted = &taskQueued
		w.Db.Put(taskQueued.ID.String(),&taskQueued)
	}

	var result task.DockerResult

	if task.ValidStateTransition(taskPersisted.State,taskQueued.State) {
		switch taskQueued.State {
		case task.Scheduled:
			result = w.StartTask(taskQueued)
		case task.Completed:
			result = w.StopTask(taskQueued)
		}
	} else {
		result.Error = fmt.Errorf("Invalid transition from %v to %v", taskPersisted.State, taskQueued.State)
	}
	return result
}

func (w *Worker) RunTasks() {
	for {
		if w.Queue.Len() != 0 {
			result := w.runTask()
			if result.Error != nil {
				log.Printf("Error running task: %v\n",result.Error)
			}
		} else {
		}
		time.Sleep(10 * time.Second)
	}
}


func (w *Worker) StartTask(t task.Task) task.DockerResult{
	t.StartTime = time.Now().UTC()
	config := task.NewConfig(&t)
	docker := task.NewDocker(config)

	result := docker.Run()
	if result.Error != nil {
		log.Printf("Error running task %v: %v\n", t.ID, result.Error)
		t.State = task.Failed
		w.Db.Put(t.ID.String(),&t)
		return result
	}

	t.ContainerID = result.ContainerId
	t.State = task.Running
	w.Db.Put(t.ID.String(),&t)

	return result
}

func (w *Worker) StopTask(t task.Task) task.DockerResult{
	config := task.NewConfig(&t)
	docker := task.NewDocker(config)

	result := docker.Stop(t.ContainerID,true)

	if result.Error != nil {
		log.Printf("Error stopping container %v: %v\n", t.ContainerID ,result.Error)
	}

	t.FinishTime = time.Now().UTC()
	t.State = task.Completed
	w.Db.Put(t.ID.String(),&t)

	log.Printf("Stopped and removed container $v for task %v\n", t.ContainerID,t.ID)
	return result
}


func (w *Worker) InspectTask(t task.Task) task.DockerInspectResponse {
	config := task.NewConfig(&t)
	d := task.NewDocker(config)
	return d.Inspect(t.ContainerID)
}

func (w *Worker) UpdateTasks() {
	for {
			w.updateTasks()
			time.Sleep(15 * time.Second)
	}
}

func (w *Worker) updateTasks() {
	for id, t := range w.GetTasks() {
		if t.State == task.Running {
				resp := w.InspectTask(*t)
				if resp.Error != nil {
					fmt.Printf("ERROR: %v\n", resp.Error)
				}

				if resp.Container == nil {
					log.Printf("No container for running task %s\n", id)
					t.State = task.Failed
					w.Db.Put(t.ID.String(), t)
				}

				if resp.Container.State.Status == "exited" {
					log.Printf("Container for task %s in non-running state %s",id, resp.Container.State.Status)
					t.State = task.Failed
					w.Db.Put(t.ID.String(), t)
				}
				t.HostPorts = resp.Container.NetworkSettings.NetworkSettingsBase.Ports
				w.Db.Put(t.ID.String(), t)
		}
}
}

