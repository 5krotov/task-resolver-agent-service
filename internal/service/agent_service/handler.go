package agent_service

import (
	"encoding/json"
	"net/http"

	api "github.com/5krotov/task-resolver-pkg/api/v1"
	"github.com/gorilla/mux"
)

type AgentHandler struct {
	svc *AgentService
}

func NewAgentHandler(svc *AgentService) *AgentHandler {
	return &AgentHandler{svc: svc}
}

func (s *AgentHandler) Register(mux *mux.Router) {
	task := mux.PathPrefix("/api/v1/task").Subrouter()
	task.HandleFunc("", s.CreateTask)
}

func (h *AgentHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Invalid method", http.StatusMethodNotAllowed)
		return
	}

	var task api.CreateTaskRequest
	err := json.NewDecoder(r.Body).Decode(&task)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	taskResp, err := h.svc.CreateTask(task)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(taskResp)
}
