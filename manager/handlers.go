package manager

import (
	"cube/task"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/go-chi/chi"
	"github.com/google/uuid"
)

func (a *Api) StartTaskHandler(res http.ResponseWriter,req *http.Request) {
	data := json.NewDecoder(req.Body)
	data.DisallowUnknownFields()

	taskEvent := task.TaskEvent{}
	err := data.Decode(&taskEvent)

	if err != nil {
		msg := fmt.Sprintf("Error matching the body: %v\n",err)
		log.Print(msg)
		res.WriteHeader(400)
		e := ErrResponse{
			HTTPStatusCode: 400,
			Message: msg,
		}
		json.NewEncoder(res).Encode(e)
		return
	}

	a.Manager.AddTask(taskEvent)
	log.Printf("Added task %v\n", taskEvent.Task.ID)
	res.WriteHeader(201)
	json.NewEncoder(res).Encode(taskEvent.Task)
}

func (a *Api) GetTasksHandler(res http.ResponseWriter,req *http.Request) {
	res.Header().Set("Content-Type","application/json");
	res.WriteHeader(200)
	json.NewEncoder(res).Encode(a.Manager.GetTasks())
}

func (a *Api)StopTaskHandler(res http.ResponseWriter,req *http.Request) {
	taskId := chi.URLParam(req,"taskId")
	if taskId == "" {
		log.Printf("No taskID passed in request.\n")
		res.WriteHeader(400)
	}

	tID,_ := uuid.Parse(taskId)
	result, ok := a.Manager.TaskDb.Get(tID.String());
	if ok != nil {
		log.Printf("No task with ID %v found\n",tID)
		res.WriteHeader(404)
	}

	taskToStop := result.(*task.Task)
	taskCopy := *taskToStop
	taskCopy.State = task.Completed

	te := task.TaskEvent{
		ID: uuid.New(),
		State: task.Completed,
		Timestamp: time.Now(),
		Task: taskCopy,
	}

	a.Manager.AddTask(te)

	log.Printf("Added task %v to stop container %v\n",taskToStop.ID,taskToStop.ContainerID);
	res.WriteHeader(204)

}

func (a *Api) GetNodesHandler(res http.ResponseWriter, req *http.Request){
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(200)
	json.NewEncoder(res).Encode(a.Manager.WorkerNodes)
}