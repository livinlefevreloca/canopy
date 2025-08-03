package ipc

import (
	"log/slog"
	"sync"
)

type Event struct {
	Component string      // The component to send the update to
	Action    string      // The action to be performed by the component
	Data      interface{} // Data to be sent back to the component
}

type Trigger struct {
	Event
	Responder chan []Event // A channel to send the event back to the triggerer
}

func NewTrigger(event Event) Trigger {
	return Trigger{
		Event:     event,                 // Initialize the Event part of the Trigger
		Responder: make(chan []Event, 1), // Buffered channel for event
	}
}

type TriggerHandler struct {
	tx         *chan Trigger
	responders []chan []Event    // A FIFO Queue for event channels using a channel
	eventLock  sync.Mutex        // Mutex to protect access to the queues
	events     map[string]*Event // A map of queues for each component
	hasEvents  bool              // Flag to indicate if any event was received
}

func NewTriggerHandler(tx *chan Trigger) *TriggerHandler {
	return &TriggerHandler{
		tx:         tx,
		responders: make([]chan []Event, 0),
		eventLock:  sync.Mutex{},
		events:     make(map[string]*Event),
		hasEvents:  false,
	}
}

func (r *TriggerHandler) MakeTrigger(event Event) {
	trigger := NewTrigger(event)
	responder := make(chan []Event, 1)
	trigger.Responder = responder
	*r.tx <- trigger
	r.responders = append(r.responders, responder)
}

// A function that can be used by one component to pass an event to another component.
func (r *TriggerHandler) PassEvent(event Event) {
	r.routeEvent(event) // Route the event to the appropriate queue
}

func (r *TriggerHandler) routeEvent(event Event) {
	r.eventLock.Lock() // Lock the mutex to protect access to event slots
	defer r.eventLock.Unlock()
	// Override any existing event for the component. If it hasnt been taken yet we want to skip it anyway.
	r.events[event.Component] = &event
	r.hasEvents = true // Set the flag to true indicating a event was received
}

func (r *TriggerHandler) RecieveEvents() {
	slog.Debug("Checking for events from responders", "responders", len(r.responders))
	remainingResponders := make([]chan []Event, 0)
	for _, responder := range r.responders {
		select {
		case events := <-responder: // Wait for a event from the responder channel
			for _, event := range events {
				slog.Debug("Received event", "component", event.Component, "action", event.Action)
				r.routeEvent(event) // Route the event to the appropriate queue
			}
		default:
			// No event available, continue to the next responder and keep it in the queue
			remainingResponders = append(remainingResponders, responder)
		}
	}
	r.responders = remainingResponders // Update the responders queue with the remaining responders
}

func (r *TriggerHandler) GetEvents() (bool, map[string]*Event) {
	r.eventLock.Lock()         // Lock the mutex to protect access to the queues
	defer r.eventLock.Unlock() // Ensure the mutex is unlocked after accessing the queues
	if r.hasEvents {
		r.hasEvents = false                // Reset the flag indicating no events are left
		events := r.events                 // take a copy of the events map
		r.events = make(map[string]*Event) // Clear the events map after retrieving
		return true, events                // Return true indicating events are available and the map of events
	}
	return false, nil // Return nil if no events are available
}
