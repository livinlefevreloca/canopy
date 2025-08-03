package tui

import (
	"fmt"

	"github.com/livinlefevreloca/canopy/internal/ipc"
	"github.com/rivo/tview"
)

type ErrorModal struct {
	ui      tview.Primitive
	name    string // Name of the modal, used for identification
	handle  *AppHandle
	message string
}

func NewErrorModal(handle *AppHandle) *ErrorModal {
	errorModal := &ErrorModal{
		ui:      nil,
		name:    ipc.COMPONENT_ERROR_MODAL,
		handle:  handle,
		message: "",
	}

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tview.NewTextView().SetTextAlign(tview.AlignCenter).SetText("An error occurred!"), 0, 1, false).
		AddItem(tview.NewTextView().SetTextAlign(tview.AlignCenter).SetText(""), 0, 1, false).
		AddItem(tview.NewButton("Ok").SetSelectedFunc(func() {
			errorModal.handle.PassEvent(ipc.Event{
				Component: ipc.COMPONENT_TUI,
				Action:    ipc.ACTION_CLOSE_ERROR_MODAL,
				Data:      nil,
			})

		}), 1, 1, false)

	errorModal.ui = flex
	return errorModal
}

func (em *ErrorModal) Render(events *ipc.Event) tview.Primitive {
	errData := events.Data.(ipc.ErrorData)
	em.ui.(*tview.TextView).SetText(fmt.Sprintf("Error: %s", errData.Message))
	return em.ui
}

func (em *ErrorModal) GetName() string {
	return em.name
}
