package agent_service

import (
	"agent-service/internal/config"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	api "github.com/5krotov/task-resolver-pkg/api/v1"
	entity "github.com/5krotov/task-resolver-pkg/entity/v1"
	"github.com/IBM/sarama"
	"go.uber.org/zap"
)

type AgentService struct {
	config config.Config
	kafka  sarama.SyncProducer
	logger *zap.SugaredLogger
}

func NewAgentService(config config.Config, kafka sarama.SyncProducer) *AgentService {
	logger, _ := zap.NewProduction()

	svc := &AgentService{
		config: config,
		kafka:  kafka,
		logger: logger.Sugar(),
	}

	go svc.ConsumeStatus(context.Background())
	return svc
}

type ConsumerGroupHandler struct {
	logger *zap.SugaredLogger
	svc    *AgentService
}

func (h *ConsumerGroupHandler) Setup(sarama.ConsumerGroupSession) error   { return nil }
func (h *ConsumerGroupHandler) Cleanup(sarama.ConsumerGroupSession) error { return nil }
func (h *ConsumerGroupHandler) ConsumeClaim(session sarama.ConsumerGroupSession, claim sarama.ConsumerGroupClaim) error {
	for msg := range claim.Messages() {
		var status api.UpdateStatusRequest
		if err := json.Unmarshal(msg.Value, &status); err != nil {
			h.logger.Warn("Failed to unmarshal status update message", zap.Error(err))
			continue
		}
		h.logger.Info("Received status update message", zap.Any("statusUpdate", status))
		go h.svc.SaveNewStatus(status)
		session.MarkMessage(msg, "")
	}
	return nil
}

func (as *AgentService) CreateTask(apiTask api.CreateTaskRequest) (*entity.Task, error) {
	as.logger.Info("Creating task", zap.Any("apiTask", apiTask))

	data, err := json.Marshal(apiTask)
	if err != nil {
		as.logger.Error("Failed to marshal task", zap.Error(err))
		return nil, fmt.Errorf("failed to marshal task: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/task", as.config.DataProvider.Addr)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		as.logger.Error("Failed to create HTTP request", zap.Error(err))
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		as.logger.Error("Failed to send HTTP request", zap.Error(err))
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		as.logger.Warn("Unexpected status code from DataProvider", zap.Int("statusCode", resp.StatusCode))
		return nil, fmt.Errorf("unexpected status code of dataprovider: %d", resp.StatusCode)
	}

	var task entity.Task
	err = json.NewDecoder(resp.Body).Decode(&task)
	if err != nil {
		as.logger.Error("Failed to unmarshal response", zap.Error(err))
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	kafkaTask := api.StartTaskRequest{
		Id:         task.Id,
		Difficulty: task.Difficulty,
	}
	dataKafkaTask, err := json.Marshal(kafkaTask)
	if err != nil {
		as.logger.Error("Failed to marshal Kafka task", zap.Error(err))
		return nil, fmt.Errorf("failed to marshal Kafka task: %w", err)
	}

	msg := &sarama.ProducerMessage{
		Topic: as.config.Kafka.TaskTopic,
		Value: sarama.ByteEncoder(dataKafkaTask),
	}

	as.logger.Info("Sending Kafka message", zap.String("topic", as.config.Kafka.TaskTopic), zap.Any("message", kafkaTask))

	_, _, err = as.kafka.SendMessage(msg)
	if err != nil {
		as.logger.Error("Failed to send Kafka message", zap.Error(err))
		return nil, fmt.Errorf("failed to send Kafka message: %w", err)
	}

	as.logger.Info("Task created successfully", zap.Any("task", task))
	return &task, nil
}

func (as *AgentService) ConsumeStatus(ctx context.Context) error {
	as.logger.Info("Starting Kafka consumer for status updates")

	config := sarama.NewConfig()
	config.Version = sarama.V2_1_0_0
	config.Consumer.Offsets.AutoCommit.Enable = true
	config.Consumer.Group.Rebalance.Strategy = sarama.BalanceStrategyRoundRobin
	config.Consumer.Offsets.Initial = sarama.OffsetOldest

	groupID := as.config.Kafka.Group
	brokers := []string{as.config.Kafka.Addr}
	topic := as.config.Kafka.StatusTopic

	handler := &ConsumerGroupHandler{logger: as.logger, svc: as}
	group, err := sarama.NewConsumerGroup(brokers, groupID, config)
	if err != nil {
		as.logger.Fatal(err)
	}
	defer group.Close()

	for {
		if err := group.Consume(ctx, []string{topic}, handler); err != nil {
			as.logger.Fatalf(err.Error())
		}
	}
}

func (as *AgentService) SaveNewStatus(status api.UpdateStatusRequest) {
	as.logger.Info("Saving new status update", zap.Any("statusUpdate", status))

	data, err := json.Marshal(status)
	if err != nil {
		as.logger.Error("Failed to marshal new status update data", zap.Error(err))
		return
	}

	url := fmt.Sprintf("%s/api/v1/task/%d/status", as.config.DataProvider.Addr, status.Id)
	req, err := http.NewRequest("PUT", url, bytes.NewBuffer(data))
	if err != nil {
		as.logger.Error("Failed to create HTTP request for saving status update", zap.Error(err))
		return
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		as.logger.Error("Failed to send HTTP request for saving status update", zap.Error(err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		as.logger.Warn("Unexpected response code while saving status update",
			zap.Int("statusCode", resp.StatusCode), zap.Int64("taskId", status.Id))
		return
	}

	var task entity.Task
	err = json.NewDecoder(resp.Body).Decode(&task)
	if err != nil {
		as.logger.Error("Failed to unmarshal response while saving status update", zap.Error(err))
		return
	}

	as.logger.Info("Status update saved successfully for task",
		zap.Int64("taskId", task.Id), zap.String("statusUpdateTimestamp",
			status.Status.Timestamp.String()))
}
