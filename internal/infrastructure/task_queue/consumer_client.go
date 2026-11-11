package taskqueue

import (
	"context"

	"github.com/hibiken/asynq"
	"github.com/rs/zerolog"
	"github.com/sharathm18/myapp/internal/config"
)

type ConsumerClient struct {
	client *asynq.Server
	mux    *asynq.ServeMux
	logger *zerolog.Logger
	config *config.Config
}

// it strictly pulls and executes payloads from taskqueue
func NewConsumerClient(cfg *config.Config, logger *zerolog.Logger) (*ConsumerClient, error) {
	redisOpt := asynq.RedisClientOpt{
		Addr:     cfg.RedisAsynq.Address,
		Password: cfg.RedisAsynq.Password,
		DB:       cfg.RedisAsynq.Database,
	}

	workerSrv := asynq.NewServer(
		redisOpt,
		asynq.Config{
			Concurrency: cfg.Worker.Concurrency, // number of concurrent goroutines workers/ consumer across all tasks
			Queues: map[string]int{
				"critical": 6, // 60% of worker time (e.g., OTPs)
				"default":  3, // 30% of worker time (e.g., welcome emails)
				"low":      1, // 10% of worker time (e.g., heavy PDF reports)
			},
			// by default, The global error handler executes on every single failure, not just after the maximum retries are reached
			ErrorHandler: asynq.ErrorHandlerFunc(func(ctx context.Context, task *asynq.Task, err error) {
				meta := ExtractTaskMetadata(ctx, task)

				// only trigger the alert if this was the final attempt before moving to DLQ/Archive
				// (retries starts at 0, so if attempt count matches max retries, it's dead)
				if meta.Retries+1 >= meta.MaxRetries {
					// call your notification function passing the meta object and the error
					// SendSlackAlert(meta, err)
				}
			}),
		},
	)

	return &ConsumerClient{
		config: cfg,
		client: workerSrv,
		mux:    asynq.NewServeMux(),
		logger: logger,
	}, nil
}

// register maps a task name (e.g., "email:welcome") to its specific handler logic.
func (w *ConsumerClient) Register(pattern string, handler asynq.HandlerFunc) {
	w.mux.HandleFunc(pattern, handler)
}

func (w *ConsumerClient) Start() error {
	return w.client.Start(w.mux)
}

// stop gracefully shuts down the worker and logs its own success locally
func (w *ConsumerClient) Shutdown() {
	w.client.Shutdown()
	w.logger.Info().Msg("task consumer stopped successfully")
}

// TaskMetadata groups core execution data for logging and alerting
type TaskMetadata struct {
	TaskID     string
	TaskType   string
	Queue      string
	Retries    int
	MaxRetries int
}

func ExtractTaskMetadata(ctx context.Context, task *asynq.Task) TaskMetadata {
	taskID, _ := asynq.GetTaskID(ctx)   // task_a8f9c2...
	taskType := task.Type()             // email:welcome
	queue, _ := asynq.GetQueueName(ctx) // critical, default, low
	retries, _ := asynq.GetRetryCount(ctx)
	maxRetries, _ := asynq.GetMaxRetry(ctx)

	return TaskMetadata{
		TaskID:     taskID,
		TaskType:   taskType,
		Queue:      queue,
		Retries:    retries,
		MaxRetries: maxRetries,
	}
}

// Mock function to handle external notifications
// func SendSlackAlert(meta TaskMetadata, err error) {}

// Note
// When a task fails its final attempt (e.g., the 3rd try), Asynq stops processing it and moves it into the Archived state (commonly known as a Dead Letter Queue or DLQ).
// The task is not deleted. It sits safely in Redis so you do not lose the data. You can then use the open-source asynqmon Web UI or the Asynq CLI tool to manually inspect the error, fix your email provider, and click "Retry" to push the task back into the active queue.
