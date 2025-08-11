// SPDX-License-Identifier: MPL-2.0

package core

import (
	"context"
	"log/slog"
	"strings"

	"github.com/nats-io/nats.go/jetstream"
)

func unscheduleWorkflow(app *App, workflowID string) {
	if existingTimer, hasTimer := app.WorkflowTimers[workflowID]; hasTimer {
		existingTimer.Stop()
		delete(app.WorkflowTimers, workflowID)
	}
}

func (app *App) HandleJob(msg jetstream.Msg) {
	workflowID := strings.TrimPrefix(msg.Subject(), app.JobsSubjectPrefix)
	app.Logger.Info("Handling job", slog.String("workflow", workflowID))
	ctx := ContextWithActor(context.Background(), &Actor{Type: ActorJob})
	workflow, err := GetWorkflow(app, ctx, workflowID)
	if err != nil {
		app.Logger.Error("Error getting workflow", slog.String("workflow", workflowID), slog.Any("error", err))
		if err := msg.Nak(); err != nil {
			app.Logger.Error("Error nack message", slog.Any("error", err))
		}
		return
	}
	_, err = RunWorkflow(app, ctx, workflow.Content, workflowID)
	if err != nil {
		app.Logger.Error("Error running workflow", slog.String("workflow", workflowID), slog.Any("error", err))
		if err := msg.Nak(); err != nil {
			app.Logger.Error("Error nack message", slog.Any("error", err))
		}
		return
	}
	err = msg.Ack()
	if err != nil {
		app.Logger.Error("Error acking message", slog.Any("error", err))
		return
	}
	app.Logger.Info("Workflow run completed", slog.String("workflowID", workflowID))
}
