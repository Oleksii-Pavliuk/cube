package main

import (
	"cube/task"
	"cube/worker"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/golang-collections/collections/queue"
	"github.com/google/uuid"
)


func main() {

	host := os.Getenv("CUBE_HOST")
	port,_ := strconv.Atoi(os.Getenv("CUBE_PORT"))

	fmt.Println("Starting Cube Worker")



	db := make(map[uuid.UUID]*task.Task)
	w := worker.Worker{
			Queue: *queue.New(),
			Db:    db,
	}

	api := worker.Api{Address: host,Port: port, Worker: &w}

	go runTask(&w)
	api.Start()
}


func runTask( w *worker.Worker) {
	for {
		if w.Queue.Len() != 0 {
			result := w.RunTask()
			if result.Error != nil {
				log.Printf("Error running task: %v\n",result.Error)
			}
		} else {
			log.Println("No tasks to process currently")
		}
		log.Println("Sleeping for 10 seconds")
		time.Sleep(10 * time.Second)
	}
}