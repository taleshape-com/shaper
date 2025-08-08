// SPDX-License-Identifier: MPL-2.0

package core

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"strings"

	"github.com/nats-io/nats.go/jetstream"
)

// We use something like event sourcing for all internal state.
// All database changes go through NATS first.
// This allows to replay changes across multiple instances of Shaper.
// It also allows restoring the system from partial state.
// Also, it leaves an audit trail and change log that we will make use of later on.
// The function here handles the messages from NATS.
// All handler functions are immutable. You can apply them multiple times and the end result looks the same. This gives us end-to-end consistency.
func (app *App) HandleState(msg jetstream.Msg) {
	event := strings.TrimPrefix(msg.Subject(), app.StateSubjectPrefix)
	data := msg.Data()
	handler := func(app *App, data []byte) bool {
		app.Logger.Error("Unknown state message subject", slog.String("event", event))
		return false
	}
	switch event {
	case "create_dashboard":
		handler = HandleCreateDashboard
	case "update_dashboard_content":
		handler = HandleUpdateDashboardContent
	case "update_dashboard_name":
		handler = HandleUpdateDashboardName
	case "update_dashboard_visibility":
		handler = HandleUpdateDashboardVisibility
	case "delete_dashboard":
		handler = HandleDeleteDashboard
	case "create_workflow":
		handler = HandleCreateWorkflow
	case "update_workflow_content":
		handler = HandleUpdateWorkflowContent
	case "update_workflow_name":
		handler = HandleUpdateWorkflowName
	case "delete_workflow":
		handler = HandleDeleteWorkflow
	case "create_api_key":
		handler = HandleCreateAPIKey
	case "delete_api_key":
		handler = HandleDeleteAPIKey
	case "create_user":
		handler = HandleCreateUser
	case "create_session":
		handler = HandleCreateSession
	case "delete_session":
		handler = HandleDeleteSession
	case "delete_user":
		handler = HandleDeleteUser
	case "create_invite":
		handler = HandleCreateInvite
	case "claim_invite":
		handler = HandleClaimInvite
	case "delete_invite":
		handler = HandleDeleteInvite
	}
	app.Logger.Info("Handling shaper state change", slog.String("event", event))
	ok := handler(app, data)
	if ok {
		err := msg.Ack()
		if err != nil {
			app.Logger.Error("Error acking message", slog.Any("error", err))
		}
	}
}

// All changes to the internal state go through this function.
// SubmitState writes changes to NATS and waits until they have been processed successfully by the stream consumer.
// This is to make sure you can read your own writes.
func (app *App) SubmitState(ctx context.Context, action string, data any) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return err
	}
	// We listen on the ACK subject for the consumer to know when the message has been processed
	// We need to subscribe before publishing the message to avoid missing the ACK
	sub, err := app.NATSConn.SubscribeSync("$JS.ACK." + app.StateStreamName + "." + app.StateConsumerName + ".>")
	if err != nil {
		return err
	}
	ack, err := app.JetStream.Publish(ctx, app.StateSubjectPrefix+action, payload)
	if err != nil {
		return err
	}
	ackSeq := strconv.FormatUint(ack.Sequence, 10)
	// Wait for the ACK
	// If context is cancelled, we return an error
	for {
		msg, err := sub.NextMsgWithContext(ctx)
		if err != nil {
			return err
		}
		// The sequence number is the part of the subject after the container of how many deliveries have been made
		// We trust the shape of the subject to be correct and panic otherwise
		seq := strings.Split(strings.TrimPrefix(msg.Subject, "$JS.ACK."+app.StateStreamName+"."+app.StateConsumerName+"."), ".")[1]
		if seq == ackSeq {
			return nil
		}
	}
}
