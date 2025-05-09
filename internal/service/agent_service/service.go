package agent_service

import (
	"agent-service/internal/config"
	"context"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"time"

	pb "github.com/5krotov/task-resolver-pkg/grpc-api/v1"
	"github.com/5krotov/task-resolver-pkg/rest-api/v1/api"
	"github.com/5krotov/task-resolver-pkg/rest-api/v1/entity"
	"github.com/IBM/sarama"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/protobuf/types/known/timestamppb"
)

type AgentService struct {
	config             config.Config
	kafka              sarama.SyncProducer
	logger             *zap.SugaredLogger
	DataProviderConn   grpc.ClientConnInterface
	DataProviderClient pb.DataProviderServiceClient
}

func NewAgentService(config config.Config, kafka sarama.SyncProducer, dataProvider config.DataProviderConfig) *AgentService {
	logger, _ := zap.NewProduction()

	var dataProviderConn grpc.ClientConnInterface
	if dataProvider.UseTLS {
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM([]byte(dataProvider.CaCert)) {
			logger.Fatal("failed to add CA certificate for dataProvider service")
		}

		creds := credentials.NewClientTLSFromCert(caCertPool, dataProvider.GrpcServerName)
		var err error
		dataProviderConn, err = grpc.NewClient(dataProvider.Addr, grpc.WithTransportCredentials(creds))
		if err != nil {
			logger.Fatal(fmt.Sprintf("failed to connect to %v, error: %v", dataProvider.Addr, err))
		}
	} else {
		var err error
		dataProviderConn, err = grpc.NewClient(dataProvider.Addr)
		if err != nil {
			logger.Fatal(fmt.Sprintf("failed to connect to %v, error: %v", dataProvider.Addr, err))
		}
	}

	dataProviderServiceClient := pb.NewDataProviderServiceClient(dataProviderConn)

	svc := &AgentService{
		config:             config,
		kafka:              kafka,
		logger:             logger.Sugar(),
		DataProviderConn:   dataProviderConn,
		DataProviderClient: dataProviderServiceClient,
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
		err := h.svc.SaveNewStatus(context.Background(), status)
		if err != nil {
			h.logger.Warn("Failed to save update status", zap.Error(err))
		}
		session.MarkMessage(msg, "")
	}
	return nil
}

func (as *AgentService) CreateTask(ctx context.Context, apiTask api.CreateTaskRequest) (*entity.Task, error) {
	as.logger.Info("Creating task", zap.Any("apiTask", apiTask))

	resp, err := as.DataProviderClient.CreateTask(ctx, &pb.CreateTaskRequest{
		Name:       apiTask.Name,
		Difficulty: pb.Difficulty(apiTask.Difficulty),
	})
	if err != nil {
		as.logger.Error("Failed to create task on data provider", zap.Error(err))
		return nil, fmt.Errorf("Failed to create task on data provider: %w", err)
	}
	task := mapPbTaskToEntityTask(resp.GetTask())

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
	return task, nil
}

func (as *AgentService) ConsumeStatus(ctx context.Context) error {
	as.logger.Info("Starting Kafka consumer for status updates")

	config := sarama.NewConfig()
	config.Version = sarama.V2_8_1_0
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
			as.logger.Errorf(err.Error())
			time.Sleep(time.Second)
		}
	}
}

func (as *AgentService) SaveNewStatus(ctx context.Context, status api.UpdateStatusRequest) error {
	as.logger.Info("Saving new status update", zap.Any("statusUpdate", status))

	_, err := as.DataProviderClient.UpdateStatus(ctx, &pb.UpdateStatusRequest{
		Id: status.Id,
		Status: &pb.Status{
			Status:    pb.StatusValue(status.Status.Status),
			Timestamp: timestamppb.New(status.Status.Timestamp),
		},
	})

	if err != nil {
		as.logger.Warn("Unable to save updated status",
			zap.Int64("statusId", status.Id), zap.String("statusUpdateTimestamp",
				status.Status.Timestamp.String()))
		return err
	}

	as.logger.Info("Status update saved successfully for task",
		zap.Int64("statusId", status.Id), zap.String("statusUpdateTimestamp",
			status.Status.Timestamp.String()))
	return nil
}
