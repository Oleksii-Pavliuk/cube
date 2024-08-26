package worker

import (
	"cube/task"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

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

	a.Worker.AddTask(taskEvent.Task)
	log.Printf("Added task %v\n", taskEvent.Task.ID)
	res.WriteHeader(201)
	json.NewEncoder(res).Encode(taskEvent.Task)
}

func (a *Api) GetTasksHandler(res http.ResponseWriter,req *http.Request) {
	res.Header().Set("Content-Type","application/json");
	res.WriteHeader(200)
	json.NewEncoder(res).Encode(a.Worker.GetTasks())
}

func (a *Api)StopTaskHandler(res http.ResponseWriter,req *http.Request) {
	taskId := chi.URLParam(req,"taskId")
	if taskId == "" {
		log.Printf("No taskID passed in request.\n")
		res.WriteHeader(400)
	}

	tID,_ := uuid.Parse(taskId)
	taskToStop, ok := a.Worker.Db[tID];
	if !ok {
		log.Printf("No task with ID %v found\n",tID)
		res.WriteHeader(404)
	}

	taskCopy := *taskToStop
	taskCopy.State = task.Completed

	a.Worker.AddTask(taskCopy)

	log.Printf("Added task %v to stop container %v\n",taskToStop.ID,taskToStop.ContainerID);
	res.WriteHeader(204)

}
