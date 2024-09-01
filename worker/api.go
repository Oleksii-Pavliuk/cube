package worker

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
)

type Api struct {
	Address string
	Port int
	Worker *Worker
	Router *chi.Mux
}

type ErrResponse struct {
	HTTPStatusCode int
	Message        string
}


func (a *Api) initRouter() {
	a.Router = chi.NewRouter()
	a.Router.Route("/tasks",func(router chi.Router){
		router.Post("/",a.StartTaskHandler)
		router.Get("/",a.GetTasksHandler)
		router.Route("/{taskId}",func(router chi.Router) {
			router.Delete("/", a.StopTaskHandler)
		})
	})
	a.Router.Route("/stats",func(router chi.Router) {
		router.Get("/",a.GetStats)
	})
}

func (a *Api) Start() {
	a.initRouter()
	http.ListenAndServe(fmt.Sprintf("%s:%d",a.Address,a.Port),a.Router)
}
