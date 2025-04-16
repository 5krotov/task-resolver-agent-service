package app

import (
	"agent-service/internal/config"
	"agent-service/internal/http"
	"agent-service/internal/service/agent_service"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/IBM/sarama"
)

type App struct {
}

func NewApp() *App {
	return &App{}
}

func (*App) Run(cfg config.Config) {
	server := http.NewServer(cfg.Agent.HTTP)
	config := sarama.NewConfig()
	config.Version = sarama.V2_8_1_0

	config.Producer.Return.Successes = true
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Retry.Max = 5

	producer, err := sarama.NewSyncProducer([]string{cfg.Kafka.Addr}, config)
	if err != nil {
		log.Fatalf("failed to create Kafka sync producer: %w", err)
	}
	service := agent_service.NewAgentService(cfg, producer)
	handler := agent_service.NewAgentHandler(service)
	handler.Register(server.Mux)

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		server.Run()
	}()
	defer func() {
		server.Stop()
	}()

	<-stop
}
