package backend

import (
	"log/slog"
	"strings"

	awsAuth "github.com/livinlefevreloca/canopy/internal/aws/auth"
	awsSso "github.com/livinlefevreloca/canopy/internal/aws/sso"
	"github.com/livinlefevreloca/canopy/internal/ipc"
)

type Server struct {
	tx         *chan ipc.Trigger  // Channel for outgoing triggers
	config     *awsAuth.AWSConfig // AWS configuration
	ssoExpired bool               // Flag to indicate if SSO session is expired
}

func NewServer(tx *chan ipc.Trigger, profile string, region string) *Server {
	config, err := awsAuth.GetAwsConfigFromProfileConfig(profile, region)
	ssoExpired := false
	if err != nil {
		if strings.Contains(err.Error(), "SSO session is expired or invalid") {
			ssoExpired = true
		}
		slog.Error("Failed to get AWS configuration", "error", err)
		config = nil
	}
	return &Server{
		tx:         tx,
		config:     config,
		ssoExpired: ssoExpired,
	}
}

func (s *Server) Run() {
	// Here we would typically start the server, listen for incoming requests,
	// and handle them accordingly. For now, we'll just simulate a simple run.
	slog.Info("Server is starting")
	end := false
	for {
		select {
		case trigger := <-*s.tx:
			// Handle the trigger
			end = s.handleTrigger(trigger)
		}

		if end {
			slog.Info("Server is shutting down")
			break
		}
	}
}

func (s *Server) handleTrigger(trigger ipc.Trigger) bool {
	// Process the trigger based on its type
	switch trigger.Component {
	case ipc.COMPONENT_HEADER:
		s.handleHeaderTrigger(trigger)
	case ipc.COMPONENT_CHANGE_PROFILE:
		s.handleSwitchProfileView(trigger)
	case ipc.COMPONENT_REFRESH_SSO:
		s.handleRefreshSSO(trigger)
	case ipc.COMPONENT_QUIT:
		slog.Info("Received quit trigger, shutting down server")
		events := make([]ipc.Event, 0)
		events = append(events, ipc.Event{
			Component: ipc.COMPONENT_QUIT,
			Action:    ipc.ACTION_END,
			Data:      nil,
		})
		trigger.Responder <- events
		return true
	}
	return false
}

func (s *Server) handleHeaderTrigger(trigger ipc.Trigger) {
	events := make([]ipc.Event, 0)
	switch trigger.Action {
	case ipc.ACTION_GET_AUTH_DATA:
		if s.ssoExpired {
			events = append(events, ipc.Event{
				Component: ipc.COMPONENT_TUI,
				Action:    ipc.ACTION_SHOW_REAUTHENTICATE_SSO_MODAL,
				Data:      nil,
			})
			events = append(events, ipc.Event{
				Component: ipc.COMPONENT_REFRESH_SSO,
				Action:    ipc.ACTION_MUST_REAUTHENTICATE_SSO,
				Data:      nil,
			})
			trigger.Responder <- events
			slog.Info("SSO session expired, prompting reauthentication")
			return
		}
		events = append(events, ipc.Event{
			Component: ipc.COMPONENT_HEADER,
			Action:    ipc.ACTION_GET_AUTH_DATA,
			Data:      s.config.AWSConfigData,
		})
		trigger.Responder <- events
	}
}

func (s *Server) handleSwitchProfileView(trigger ipc.Trigger) {
	switch trigger.Action {
	case ipc.ACTION_CHANGE_PROFILE:
		profileData, ok := trigger.Data.(ipc.ChangeProfileData)
		if !ok {
			panic("Expected ChangeProfileData")
		}
		s.refreshAwsConfig(profileData.Profile, s.config.Region, &trigger.Responder)
		slog.Info("Switched AWS profile", "profile", profileData.Profile)

		events := make([]ipc.Event, 0)
		events = append(events, ipc.Event{
			Component: ipc.COMPONENT_HEADER,
			Action:    ipc.ACTION_GET_AUTH_DATA,
			Data:      s.config.AWSConfigData,
		})
		events = append(events, ipc.Event{
			Component: ipc.COMPONENT_CHANGE_PROFILE,
			Action:    ipc.ACTION_CHANGE_PROFILE,
			Data:      nil,
		})
		trigger.Responder <- events
	}
}

func (s *Server) handleRefreshSSO(trigger ipc.Trigger) {
	switch trigger.Action {
	case ipc.ACTION_REAUTHENTICATE_SSO:
		refreshData, ok := trigger.Data.(ipc.ReauthenticateSSOData)
		if !ok {
			panic("Expected ReauthenticateSSOData")
		}
		err := awsSso.ExecAwsSSOLogin(refreshData.Profile)
		if err != nil {
			slog.Error("Failed to reauthenticate SSO session", "error", err)
			triggerErrorMessage("Failed to reauthenticate SSO session: "+err.Error(), &trigger.Responder)
			return
		}

		region := ""
		if s.config != nil {
			region = s.config.Region
		}

		s.refreshAwsConfig(refreshData.Profile, region, &trigger.Responder)
		slog.Info("Reauthenticated SSO session")
		s.ssoExpired = false

		events := make([]ipc.Event, 0)
		events = append(events, ipc.Event{
			Component: ipc.COMPONENT_HEADER,
			Action:    ipc.ACTION_GET_AUTH_DATA,
			Data:      s.config.AWSConfigData,
		})
		events = append(events, ipc.Event{
			Component: ipc.COMPONENT_REFRESH_SSO,
			Action:    ipc.ACTION_FINISH_REAUTHENTICATE_SSO,
			Data:      nil,
		})
		trigger.Responder <- events
	}
}

func triggerErrorMessage(errorMessage string, responder *chan []ipc.Event) {
	events := make([]ipc.Event, 0)
	events = append(events, ipc.Event{
		Component: ipc.COMPONENT_TUI,
		Action:    ipc.ACTION_SHOW_ERROR_MODAL,
		Data:      nil,
	})
	events = append(events, ipc.Event{
		Component: ipc.COMPONENT_ERROR_MODAL,
		Action:    ipc.ACTION_SHOW_ERROR_MESSAGE,
		Data: ipc.ErrorData{
			Message: errorMessage,
		},
	})

	*responder <- events
}

func (s *Server) refreshAwsConfig(profile string, region string, responder *chan []ipc.Event) {
	cfg, err := awsAuth.GetAwsConfigFromProfileConfig(profile, region)
	if err != nil {
		slog.Error("Failed to get AWS configuration for new profile", "error", err)
		triggerErrorMessage("Failed to refresh AWS configuration: "+err.Error(), responder)
		return
	}
	s.config = cfg
}
