package app

import (
	"net/http"
	"os"
	"todo/internal/app/store"
)

type APIServer struct {
	addr  string
	store *store.Store
}

func New() *APIServer {
	os.Setenv("TODO_PORT", ":7540")
	addr := os.Getenv("TODO_PORT")

	return &APIServer{
		addr: addr,
	}
}

func (s *APIServer) Start() error {
	if err := s.configureStore(); err != nil {
		return err
	}

	s.configureRouter()

	return http.ListenAndServe(s.addr, nil)
}

func (s *APIServer) configureStore() error {
	st := store.NewStore()
	if err := st.Open(); err != nil {
		return err
	}

	s.store = st

	return nil
}

func (s *APIServer) configureRouter() {
	http.Handle("/", http.FileServer(http.Dir("web")))
	http.HandleFunc("/api/nextdate", s.NextDateHandler())
	http.HandleFunc("/api/task", s.ApiTaskMethods())
	http.HandleFunc("/api/tasks", s.GetTasks())
	http.HandleFunc("/api/task/done", s.DoneTask())
}
