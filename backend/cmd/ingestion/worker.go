package main

import (
	"context"
	"os"
	"os/exec"
	"time"

	"github.com/rs/zerolog/log"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

const ingestionTaskQueue = "cinova-ingestion"

// IngestionInput defines the parameters for a scheduled ingestion run.
type IngestionInput struct {
	Mode          string
	MediaType     string
	Country       string
	MinVotes      string
	MinPopularity string
}

// IngestionWorkflow is a thin orchestrator that delegates to RunIngestionActivity.
// The activity heartbeats every 2 minutes so Temporal knows it is still alive
// during the multi-hour full-ingestion run.
func IngestionWorkflow(ctx workflow.Context, input IngestionInput) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("IngestionWorkflow started", "mode", input.Mode, "media_type", input.MediaType)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 50 * time.Hour, // full mode can take ~48 h
		HeartbeatTimeout:    10 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 1, // ingestion is not idempotent; no automatic retry
		},
	}

	return workflow.ExecuteActivity(
		workflow.WithActivityOptions(ctx, ao),
		RunIngestionActivity,
		input,
	).Get(ctx, nil)
}

// RunIngestionActivity re-executes this binary as a subprocess in the specified
// mode. Using a subprocess avoids duplicating ingestion logic and lets the worker
// heartbeat independently while the long-running process runs.
func RunIngestionActivity(ctx context.Context, input IngestionInput) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	args := []string{"--mode", input.Mode}
	if input.MediaType != "" {
		args = append(args, "--media-type", input.MediaType)
	}
	if input.Country != "" {
		args = append(args, "--country", input.Country)
	}
	if input.MinVotes != "" {
		args = append(args, "--min-votes", input.MinVotes)
	}
	if input.MinPopularity != "" {
		args = append(args, "--min-popularity", input.MinPopularity)
	}

	cmd := exec.Command(exe, args...) //nolint:gosec
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	log.Info().Str("mode", input.Mode).Strs("args", args).Msg("ingestion: starting subprocess")

	if err := cmd.Start(); err != nil {
		return err
	}

	done := make(chan error, 1)
	go func() { done <- cmd.Wait() }()

	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case err := <-done:
			if err != nil {
				log.Error().Err(err).Str("mode", input.Mode).Msg("ingestion subprocess failed")
			} else {
				log.Info().Str("mode", input.Mode).Msg("ingestion subprocess complete")
			}
			return err
		case <-ticker.C:
			activity.RecordHeartbeat(ctx, "running")
		case <-ctx.Done():
			_ = cmd.Process.Kill()
			return ctx.Err()
		}
	}
}

// runWorker starts the Temporal worker for ingestion workflows and upserts
// the three Temporal Schedules that replace the K8s CronJobs.
func runWorker() {
	temporalAddress := os.Getenv("TEMPORAL_ADDRESS")
	if temporalAddress == "" {
		temporalAddress = "temporal-server.temporal.svc.cluster.local:7233"
	}

	c, err := client.Dial(client.Options{
		HostPort:  temporalAddress,
		Namespace: "default",
	})
	if err != nil {
		log.Fatal().Err(err).Str("address", temporalAddress).Msg("temporal: dial failed")
	}
	defer c.Close()

	w := worker.New(c, ingestionTaskQueue, worker.Options{
		MaxConcurrentActivityExecutionSize:     1, // one ingestion job at a time
		MaxConcurrentWorkflowTaskExecutionSize: 5,
	})
	w.RegisterWorkflow(IngestionWorkflow)
	w.RegisterActivity(RunIngestionActivity)

	ensureSchedules(context.Background(), c)

	log.Info().
		Str("address", temporalAddress).
		Str("task_queue", ingestionTaskQueue).
		Msg("ingestion temporal worker started")

	if err := w.Run(worker.InterruptCh()); err != nil {
		log.Fatal().Err(err).Msg("ingestion temporal worker stopped with error")
	}
}

type ingestionScheduleSpec struct {
	id      string
	cron    string
	input   IngestionInput
	overlap client.ScheduleOverlapPolicy
}

// ensureSchedules creates the three ingestion Temporal Schedules on first start.
// Subsequent starts skip creation if the schedule already exists.
func ensureSchedules(ctx context.Context, c client.Client) {
	schedules := []ingestionScheduleSpec{
		{
			id:   "cinova-ingestion-delta",
			cron: "0 2 * * *", // daily 02:00 UTC
			input: IngestionInput{
				Mode:      "delta",
				MediaType: "all",
				Country:   "US",
				MinVotes:  "50",
			},
			overlap: client.ScheduleOverlapPolicySkip,
		},
		{
			id:   "cinova-ingestion-enrich",
			cron: "0 6 * * *", // daily 06:00 UTC
			input: IngestionInput{
				Mode:      "enrich-only",
				MediaType: "all",
			},
			overlap: client.ScheduleOverlapPolicySkip,
		},
		{
			id:   "cinova-ingestion-full",
			cron: "0 3 * * 0", // Sunday 03:00 UTC
			input: IngestionInput{
				Mode:          "full",
				MediaType:     "all",
				Country:       "US",
				MinVotes:      "100",
				MinPopularity: "1",
			},
			overlap: client.ScheduleOverlapPolicySkip,
		},
	}

	for _, s := range schedules {
		_, err := c.ScheduleClient().Create(ctx, client.ScheduleOptions{
			ID: s.id,
			Spec: client.ScheduleSpec{
				CronExpressions: []string{s.cron},
			},
			Action: &client.ScheduleWorkflowAction{
				Workflow:  IngestionWorkflow,
				TaskQueue: ingestionTaskQueue,
				Args:      []interface{}{s.input},
			},
			Overlap: s.overlap,
		})
		if err != nil {
			if st, ok := status.FromError(err); ok && st.Code() == codes.AlreadyExists {
				log.Info().Str("id", s.id).Msg("temporal schedule already exists — skipping")
			} else {
				log.Warn().Err(err).Str("id", s.id).Msg("failed to create temporal schedule")
			}
		} else {
			log.Info().Str("id", s.id).Str("cron", s.cron).Msg("temporal schedule created")
		}
	}
}
