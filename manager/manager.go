package manager

import (
	"bytes"
	"cube/task"
	"cube/worker"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/docker/go-connections/nat"
	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
)

type Manager struct {
    Pending       queue.Queue
    TaskDb        map[uuid.UUID]*task.Task
    EventDb       map[uuid.UUID]*task.TaskEvent
    Workers       []string
    WorkerTaskMap map[string][]uuid.UUID
    TaskWorkerMap map[uuid.UUID]string
		LastWorker    int
}



func New(workers []string) *Manager {
	taskDb := make(map[uuid.UUID]*task.Task)
	eventDb := make(map[uuid.UUID]*task.TaskEvent)
	workerTaskMap := make(map[string][]uuid.UUID)
	taskWorkerMap := make(map[uuid.UUID]string)

	for worker := range workers{
		workerTaskMap[workers[worker]] = []uuid.UUID{}
	}

	return &Manager{
		Pending:       *queue.New(),
    TaskDb:        taskDb,
    EventDb:       eventDb,
    Workers:       workers,
    WorkerTaskMap: workerTaskMap,
    TaskWorkerMap: taskWorkerMap,
	}
}


func (m *Manager) GetTasks() []*task.Task{
	tasks := []*task.Task{}

	for _, t :=range m.TaskDb {
		tasks = append(tasks,t)
	}
	return tasks
}

func (m *Manager) SelectWorker() string{
	var newWorker int
	if m.LastWorker+1 < len(m.Workers){
		m.LastWorker++
		newWorker = m.LastWorker;
	}else{
		newWorker, m.LastWorker = 0,0
	}
	return m.Workers[newWorker]
}

func (m *Manager) AddTask(taskEvent task.TaskEvent){
	m.Pending.Enqueue(taskEvent)
}

func (m *Manager) updateTasks(){
	for _, worker := range m.Workers {
		log.Printf("Checking worker %v for task updates",worker)

		url := fmt.Sprintf("http://%s/tasks",worker);
		res,err := http.Get(url)
		if err != nil {
			log.Printf("Error connecting to %v: %v\n", worker, err)
		}

		decoder := json.NewDecoder(res.Body);
		var tasks []*task.Task
		err = decoder.Decode(&tasks)
		if err != nil {
			log.Printf("Error unmarshaling task: %s\n",err)
		}

		for _,t := range tasks {
			log.Printf("Attempting to update task %v\n", t.ID)

			_, ok := m.TaskDb[t.ID]
			if !ok {
				log.Printf("Task with id %s not found\n",t.ID)
				return
			}

			m.TaskDb[t.ID].State = t.State
			m.TaskDb[t.ID].StartTime = t.StartTime
			m.TaskDb[t.ID].FinishTime = t.FinishTime
			m.TaskDb[t.ID].ContainerID = t.ContainerID
			m.TaskDb[t.ID].HostPorts = t.HostPorts
			m.TaskDb[t.ID].HealthCheck = t.HealthCheck
		}
	}
}


func (m *Manager) UpdateTasks() {
	for {
			log.Println("Checking for task updates from workers")
			m.updateTasks()
			log.Println("Task updates completed")
			log.Println("Sleeping for 15 seconds")
			time.Sleep(15 * time.Second)
	}
}

func (m *Manager) ProcessTasks() {
	for{
		log.Printf("Processing any tasks in the queue")
		m.SendWork()
		log.Printf("Sleeping for 10 seconds")
		time.Sleep(10*time.Second)
	}
}

func (m *Manager) SendWork(){
	if m.Pending.Len() > 0 {
		w := m.SelectWorker()

		event := m.Pending.Dequeue()
		taskEvent := event.(task.TaskEvent)
		t := taskEvent.Task
		log.Printf("Pulled %v off pending queue\n", t)

		m.EventDb[taskEvent.ID] = &taskEvent
		m.WorkerTaskMap[w] = append(m.WorkerTaskMap[w], taskEvent.Task.ID)
		m.TaskWorkerMap[t.ID] = w

		t.State = task.Scheduled
		m.TaskDb[t.ID] = &t

		data, err := json.Marshal(taskEvent)
		if err != nil {
			log.Printf("Unable to marshal task object: %v \n",t)
		}

		url := fmt.Sprintf("http://%s/tasks",w)
		res, err := http.Post(url,"application/json",bytes.NewBuffer(data));
		if err != nil {
			log.Printf("Error connecting to %v: %v \n", w, err)
			m.Pending.Enqueue(taskEvent)
			return
		}

		decoder := json.NewDecoder(res.Body)
		if res.StatusCode != http.StatusCreated {
			e := worker.ErrResponse{}
			err := decoder.Decode(&e)
			if err != nil {
				fmt.Printf("Error decoding response: %s\n", err.Error())
				return
			}
			log.Printf("Response error (%d): %s\n",e.HTTPStatusCode, e.Message);
		}
		t = task.Task{}
		err = decoder.Decode(&t)
		if err != nil {
			fmt.Printf("Error decoding response: %s\n", err.Error())
			return
		}
		log.Printf("%v\n",t)

	} else {
		log.Println("No workers in the queue");
	}
}


func (m *Manager) checkTaskHealth(t task.Task) error {
	log.Printf("Calling health check for task %s: %s\n", t.ID, t.HealthCheck)

	w := m.TaskWorkerMap[t.ID]
	hostPort := getHostPort(t.HostPorts)
	worker := strings.Split(w, ":")
	url := fmt.Sprintf("http://%s:%s%s", worker[0], *hostPort, t.HealthCheck)
	log.Printf("Calling health check for task %s: %s\n", t.ID, url)
	resp, err := http.Get(url)
	if err != nil {
			msg := fmt.Sprintf("Error connecting to health check %s", url)
			log.Println(msg)
			return errors.New(msg)
	}

	if resp.StatusCode != http.StatusOK {
			msg := fmt.Sprintf("Error health check for task %s did not return 200\n", t.ID)
			log.Println(msg)
			return errors.New(msg)
	}

	log.Printf("Task %s health check response: %v\n", t.ID, resp.StatusCode)

	return nil
}

func (m *Manager) DoHealthChecks() {
	for {
			log.Println("Performing task health check")
			m.doHealthChecks()
			log.Println("Task health checks completed")
			log.Println("Sleeping for 60 seconds")
			time.Sleep(60 * time.Second)
	}
}


func (m *Manager) doHealthChecks() {
	m.updateTasks()
	for _, t := range m.GetTasks() {
			if t.State == task.Running && t.RestartCount < 3 {
					err := m.checkTaskHealth(*t)
					if err != nil {
						m.restartTask(t)
					}
			} else if t.State == task.Failed && t.RestartCount < 3 {
					m.restartTask(t)
			}
	}
}

func (m *Manager) restartTask(t *task.Task) {
	w := m.TaskWorkerMap[t.ID]
	t.State = task.Scheduled
	t.RestartCount++
	m.TaskDb[t.ID] = t

	te := task.TaskEvent{
			ID:        uuid.New(),
			State:     task.Running,
			Timestamp: time.Now(),
			Task:      *t,
	}
	data, err := json.Marshal(te)
	if err != nil {
			log.Printf("Unable to marshal task object: %v.", t)
			return
	}

	url := fmt.Sprintf("http://%s/tasks", w)
	resp, err := http.Post(url, "application/json", bytes.NewBuffer(data))
	if err != nil {
			log.Printf("Error connecting to %v: %v", w, err)
			m.Pending.Enqueue(t)
			return
	}

	d := json.NewDecoder(resp.Body)
	if resp.StatusCode != http.StatusCreated {
			e := worker.ErrResponse{}
			err := d.Decode(&e)
			if err != nil {
					fmt.Printf("Error decoding response: %s\n", err.Error())
					return
			}
			log.Printf("Response error (%d): %s", e.HTTPStatusCode, e.Message)
			return
	}

	newTask := task.Task{}
	err = d.Decode(&newTask)
	if err != nil {
			fmt.Printf("Error decoding response: %s\n", err.Error())
			return
	}
	log.Printf("%#v\n", t)
}



func getHostPort(ports nat.PortMap) *string {
	for k, _ := range ports {
			return &ports[k][0].HostPort
	}
	return nil
}
