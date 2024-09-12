package main

import (
	"cube/manager"
	"cube/task"
	"cube/worker"
	"fmt"
	"os"
	"strconv"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
)


func main() {

	whost := os.Getenv("CUBE_WORKER_HOST")
	wport, _ := strconv.Atoi(os.Getenv("CUBE_WORKER_PORT"))

	mhost := os.Getenv("CUBE_MANAGER_HOST")
	mport, _ := strconv.Atoi(os.Getenv("CUBE_MANAGER_PORT"))


	w := worker.Worker{
		Queue: *queue.New(),
		Db:    make(map[uuid.UUID]*task.Task),
}

	wapi := worker.Api{Address: whost,Port: wport, Worker: &w}
	fmt.Println("Starting Cube Worker")

	go w.RunTasks()
 	// go w.CollectStats()
	go w.UpdateTasks()
	go wapi.Start()

	workers := []string{fmt.Sprintf("%s:%d",whost,wport)}
	m := manager.New(workers)
	mapi := manager.Api{Address: mhost,Port: mport, Manager: m}

	go m.ProcessTasks()
	go m.UpdateTasks()
	go m.DoHealthChecks()

	mapi.Start()

}

