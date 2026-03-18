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
// Returns a shutdown func (stop the worker + close the client) and a nil error.
// Temporal startup failures are non-fatal: the API continues without workflow
// routing and falls back to direct execution.
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
		log.Warn().Err(err).Str("address", temporalAddress).Msg("temporal: dial failed — falling back to direct execution")
		return func() {}, nil, nil
	}

	w := worker.New(c, TaskQueue, worker.Options{
		MaxConcurrentActivityExecutionSize:     10,
		MaxConcurrentWorkflowTaskExecutionSize: 10,
	})

	w.RegisterWorkflow(ChatWorkflow)
	// Register only the three Temporal activity methods — registering chatSvc as a whole
	// would also pick up helpers like FetchCandidates that don't return (T, error).
	w.RegisterActivity(chatSvc.ExtractIntentActivity)
	w.RegisterActivity(chatSvc.FetchCandidatesActivity)
	w.RegisterActivity(chatSvc.WriteRecsActivity)

	if err := w.Start(); err != nil {
		c.Close()
		log.Warn().Err(err).Str("address", temporalAddress).Msg("temporal: worker start failed — falling back to direct execution")
		return func() {}, nil, nil
	}

	log.Info().Str("address", temporalAddress).Str("task_queue", TaskQueue).Msg("temporal worker started")

	shutdown := func() {
		w.Stop()
		c.Close()
		log.Info().Msg("temporal worker stopped")
	}
	return shutdown, c, nil
}
