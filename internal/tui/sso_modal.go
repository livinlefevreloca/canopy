package tui

import (
	"log/slog"

	"github.com/gdamore/tcell/v2"
	awsAuth "github.com/livinlefevreloca/canopy/internal/aws/auth"
	"github.com/livinlefevreloca/canopy/internal/ipc"
	"github.com/rivo/tview"
)

type SSOReauthenticationModal struct {
	ui              tview.Primitive // The UI component for the modal
	pages           *tview.Pages    // Pages for the modal
	name            string
	handle          *AppHandle
	setMessage      func(string) // Function to set the message in the UI
	selectedProfile string
}

func NewSSOReauthenticationModal(handle *AppHandle) *SSOReauthenticationModal {
	modal := SSOReauthenticationModal{
		ui:              nil,
		name:            ipc.COMPONENT_REFRESH_SSO,
		handle:          handle,
		setMessage:      nil,
		selectedProfile: "",
	}

	pages := tview.NewPages()

	// inputs page
	button := tview.NewButton("Rerefresh SSO").SetSelectedFunc(func() {
		if modal.selectedProfile != "" {
			modal.handle.SendTrigger(modal.GetName(), ipc.ACTION_REAUTHENTICATE_SSO, ipc.ReauthenticateSSOData{
				Profile: modal.selectedProfile,
			})
			modal.pages.ShowPage("refreshing")
			modal.selectedProfile = "" // Reset selected profile after reauthentication
		}
	})

	profileList := tview.NewList()
	profileList.ShowSecondaryText(false)

	for _, profile := range awsAuth.GetAvailableProfiles() {
		profileList.AddItem(profile, "", 0, func() {
			modal.selectedProfile = profile
			modal.handle.SetFocus(button)
		})
	}

	textView := tview.NewTextView().
		SetTextAlign(tview.AlignCenter).
		SetText("Refresh your AWS SSO Credentials")

	modal.setMessage = func(message string) {
		textView.SetText(message)
	}

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(textView, 2, 1, false).
		AddItem(profileList, 0, 1, true).
		AddItem(tview.NewBox(), 1, 1, false). // Spacer
		AddItem(button, 3, 1, false)
	flex.SetBorder(true)
	flex.SetBorderPadding(2, 2, 2, 2)
	flex.SetTitle("[yellow]SSO Authentication")

	// reauthenticating page
	reauth := tview.NewTextView().
		SetText("Refreshing SSO Credentials...").
		SetTextAlign(tview.AlignCenter)

	reauth.SetBorder(true)
	reauth.SetBorderPadding(2, 2, 2, 2)
	flex.SetTitle("[yellow]SSO Authentication")

	// success page
	success := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tview.NewTextView().
							SetText("AWS SSO Credentials were refreshed successfully!").
							SetTextAlign(tview.AlignCenter), 0, 1, false).
		AddItem(tview.NewBox(), 1, 1, false). // Spacer
		AddItem(tview.NewButton("Close").SetSelectedFunc(func() {
			pages.HidePage("success")
			pages.ShowPage("inputs")
			modal.handle.PassEvent(ipc.Event{
				Component: ipc.COMPONENT_TUI,
				Action:    ipc.ACTION_CLOSE_REAUTHENTICATE_SSO_MODAL,
				Data:      nil,
			})

		}), 3, 1, true)

	success.SetBorder(true)
	success.SetBorderPadding(2, 2, 2, 2)
	flex.SetTitle("[yellow]SSO Authentication")

	pages.AddPage("inputs", flex, true, true)
	pages.AddPage("refreshing", reauth, true, false)
	pages.AddPage("success", success, true, false)

	pages.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyUp:
			if button.HasFocus() {
				// If the button has focus, move focus back to the profile list
				modal.handle.SetFocus(profileList)
			}
		}
		return event
	})

	modal.handle.SetSubscription(modal.GetName(), &modal)
	modal.pages = pages
	modal.ui = makeModal(pages)

	return &modal
}

func (modal *SSOReauthenticationModal) Render(event *ipc.Event) tview.Primitive {
	slog.Debug("SSOReauthenticationModal Render: Received event", "event", event)
	switch event.Action {
	case ipc.ACTION_MUST_REAUTHENTICATE_SSO:
		modal.setMessage("Your SSO Session has expired. Please Reauthenticate to continue.")
	case ipc.ACTION_FINISH_REAUTHENTICATE_SSO:
		// Reset the message in case this was a forced reauthentication
		modal.setMessage("Refresh your AWS SSO Credentials")
		modal.pages.HidePage("refreshing")
		modal.pages.ShowPage("success")
	}

	return modal.ui
}

func (modal *SSOReauthenticationModal) GetName() string {
	return modal.name
}
