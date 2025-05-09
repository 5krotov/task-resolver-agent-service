package agent_service

import (
	"context"

	pb "github.com/5krotov/task-resolver-pkg/grpc-api/v1"
	"github.com/5krotov/task-resolver-pkg/rest-api/v1/api"
	"github.com/5krotov/task-resolver-pkg/rest-api/v1/entity"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AgentServiceServer struct {
	pb.UnimplementedAgentServiceServer
	service *AgentService
}

func NewAgentServiceServer(service *AgentService) *AgentServiceServer {
	return &AgentServiceServer{service: service}
}

func (s *AgentServiceServer) CreateTask(ctx context.Context, request *pb.CreateTaskRequest) (*pb.CreateTaskResponse, error) {
	apiCreateTaskRequest := api.CreateTaskRequest{
		Name:       request.Name,
		Difficulty: int(request.Difficulty),
	}
	apiCreateTaskResponse, err := s.service.CreateTask(ctx, apiCreateTaskRequest)
	if err != nil {
		return nil, err
	}
	return &pb.CreateTaskResponse{
		Task: mapEntityTaskTopPbTask(apiCreateTaskResponse),
	}, nil
}

func mapEntityTaskTopPbTask(task *entity.Task) *pb.Task {
	statusHistory := []*pb.Status{}
	for _, s := range task.StatusHistory {
		statusHistory = append(statusHistory, &pb.Status{
			Status:    pb.StatusValue(s.Status),
			Timestamp: timestamppb.New(s.Timestamp),
		})
	}
	return &pb.Task{
		Id:            task.Id,
		Name:          task.Name,
		Difficulty:    pb.Difficulty(task.Difficulty),
		StatusHistory: statusHistory,
	}
}

func mapPbTaskToEntityTask(task *pb.Task) *entity.Task {
	statusHistory := []entity.Status{}
	for _, s := range task.GetStatusHistory() {
		statusHistory = append(statusHistory, entity.Status{
			Status:    int(s.GetStatus()),
			Timestamp: s.GetTimestamp().AsTime(),
		})
	}
	return &entity.Task{
		Id:            task.GetId(),
		Name:          task.GetName(),
		Difficulty:    int(task.GetDifficulty()),
		StatusHistory: statusHistory,
	}
}

func mapEntityDoTaskRequestToPbDoTaskRequest(task *pb.Task) *entity.Task {
	statusHistory := []entity.Status{}
	for _, s := range task.GetStatusHistory() {
		statusHistory = append(statusHistory, entity.Status{
			Status:    int(s.GetStatus()),
			Timestamp: s.GetTimestamp().AsTime(),
		})
	}
	return &entity.Task{
		Id:            task.GetId(),
		Name:          task.GetName(),
		Difficulty:    int(task.GetDifficulty()),
		StatusHistory: statusHistory,
	}
}
