package tui

import (
	"github.com/livinlefevreloca/canopy/internal/ipc"
	"github.com/rivo/tview"
)

type HelpModal struct {
	ui     tview.Primitive
	name   string // Name of the modal, used for identification
	handle *AppHandle
}

func NewHelpModal(handle *AppHandle) *HelpModal {
	helpText := `
	- Press [yellow]'ctrl-h'[white] to show this help.
	- Press [yellow]'ctrl-c'[white] to quit the application.
	- Press [yellow]'ctrl-a'[white] to open the authentication modal.
	- Use arrow keys to navigate through the UI.
	- Press [yellow]'Enter'[white] to select an option.
	`

	textView := tview.NewTextView().
		SetText(helpText).
		SetTextAlign(tview.AlignLeft).
		SetDynamicColors(true)

	textView.SetBorder(true)
	textView.SetTitle("Help")
	textView.SetBorderPadding(2, 2, 2, 2)

	modal := makeModal(textView)

	return &HelpModal{
		ui:     modal,
		name:   ipc.COMPONENT_HELP_MODAL,
		handle: handle,
	}
}

func (h *HelpModal) Render(*ipc.Event) tview.Primitive {
	// Return the UI component for the help modal
	return h.ui
}

func (h *HelpModal) SetFocus(p tview.Primitive) {
	h.handle.SetFocus(p)
}

func (h *HelpModal) GetName() string {
	return h.name
}
