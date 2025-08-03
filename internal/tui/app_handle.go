package tui

import (
	"log/slog"

	"github.com/livinlefevreloca/canopy/internal/ipc"
	"github.com/rivo/tview"
)

type AppHandle struct {
	*tview.Application
	triggerHandler *ipc.TriggerHandler
	subscriptions  map[string]Renderable
}

func NewAppHandle(triggerHandler *ipc.TriggerHandler, app *tview.Application) *AppHandle {
	return &AppHandle{
		Application:    app,
		triggerHandler: triggerHandler,
		subscriptions:  make(map[string]Renderable),
	}
}

func (a *AppHandle) SetSubscription(component string, sub Renderable) {
	a.subscriptions[component] = sub
}

func (a *AppHandle) SendTrigger(component string, action string, data interface{}) {
	event := ipc.Event{
		Component: component,
		Action:    action,
		Data:      data,
	}
	a.triggerHandler.MakeTrigger(event)
}

func (a *AppHandle) PassEvent(response ipc.Event) {
	a.triggerHandler.PassEvent(response)
}

func (a *AppHandle) RunEventHandler() error {
	slog.Info("Starting event handler for TUI application")
	for {
		a.triggerHandler.RecieveEvents()
		if hasEvents, events := a.triggerHandler.GetEvents(); hasEvents { // Only update the UI if there are new events
			slog.Debug("Received new events, processing them")
			_, ok := events[ipc.COMPONENT_QUIT]
			if ok {
				slog.Info("Received quit signal, exiting event handler")
				return nil
			}
			// If we have responses for other components, we update the UI
			a.QueueUpdateDraw(func() {
				for component, sub := range a.subscriptions {
					event, ok := events[component] // Get the event for the component
					if ok {
						slog.Debug("Processing event for component", slog.String("component", component))
						sub.Render(event)
					}
				}
			})
		}
	}
}
