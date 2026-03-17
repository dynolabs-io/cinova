package workflow

import (
	"github.com/rs/zerolog/log"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"

	"github.com/foundrylab-app/cinova/backend/internal/chat"
)

// Start creates a Temporal client + worker, registers the chat workflow and
// activities, and starts the worker polling loop.
//
// Returns a shutdown func (stop the worker + close the client) and an error.
// If temporalAddress is empty, returns a no-op shutdown and nil error so the
// API starts normally without Temporal.
func Start(temporalAddress string, chatSvc *chat.Service) (func(), client.Client, error) {
	if temporalAddress == "" {
		log.Warn().Msg("temporal: address not configured — workflow engine disabled")
		return func() {}, nil, nil
	}

	c, err := client.Dial(client.Options{
		HostPort:  temporalAddress,
		Namespace: "default",
	})
	if err != nil {
		return nil, nil, err
	}

	w := worker.New(c, TaskQueue, worker.Options{
		MaxConcurrentActivityExecutionSize:     10,
		MaxConcurrentWorkflowTaskExecutionSize: 10,
	})

	w.RegisterWorkflow(ChatWorkflow)
	w.RegisterActivity(chatSvc)

	if err := w.Start(); err != nil {
		c.Close()
		return nil, nil, err
	}

	log.Info().Str("address", temporalAddress).Str("task_queue", TaskQueue).Msg("temporal worker started")

	shutdown := func() {
		w.Stop()
		c.Close()
		log.Info().Msg("temporal worker stopped")
	}
	return shutdown, c, nil
}
