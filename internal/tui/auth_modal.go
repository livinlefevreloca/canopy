package tui

import (
	"github.com/gdamore/tcell/v2"
	awsAuth "github.com/livinlefevreloca/canopy/internal/aws/auth"
	"github.com/livinlefevreloca/canopy/internal/ipc"
	"github.com/rivo/tview"
)

type AuthModal struct {
	ui          tview.Primitive
	name        string // Name of the modal, used for identification
	handle      *AppHandle
	currentPage string // Track the current page in the modal
	pages       map[string]Renderable
}

func NewAuthModal(handle *AppHandle) *AuthModal {
	pages := tview.NewPages()
	pagesMap := make(map[string]Renderable)

	changeProfile := NewChangeProfileView(handle)
	pagesMap[changeProfile.GetName()] = changeProfile

	newAccessKey := NewSetAccessKeysView(handle)
	pagesMap[newAccessKey.GetName()] = newAccessKey

	pages.AddPage(changeProfile.GetName(), changeProfile.ui, true, true)
	pages.AddPage(newAccessKey.GetName(), newAccessKey.ui, true, false)

	am := &AuthModal{
		ui:          makeModal(pages), // Adjust width and height as needed
		name:        ipc.COMPONENT_AUTH_MODAL,
		handle:      handle,
		currentPage: ipc.COMPONENT_CHANGE_PROFILE, // Default to the Change Profile page
		pages:       pagesMap,
	}

	pages.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyTAB:
			am.cycleTab(pages)
			return nil // Prevent further processing of the TAB key
		}
		return event
	})

	return am
}

func (am *AuthModal) Render(events *ipc.Event) tview.Primitive {
	// Return the UI component for this modal
	return am.ui
}

func (am *AuthModal) SetFocus(p tview.Primitive) {
	// Set focus to the given primitive
	am.handle.SetFocus(p)
}

func (am *AuthModal) GetName() string {
	return "AuthModal"
}

func (am *AuthModal) cycleTab(pages *tview.Pages) {
	var newPage, oldPage string
	switch am.currentPage {
	case ipc.COMPONENT_CHANGE_PROFILE:
		oldPage = ipc.COMPONENT_CHANGE_PROFILE
		newPage = ipc.COMPONENT_SET_ACCESS_KEYS
	case ipc.COMPONENT_SET_ACCESS_KEYS:
		oldPage = ipc.COMPONENT_SET_ACCESS_KEYS
		newPage = ipc.COMPONENT_CHANGE_PROFILE
	}
	am.setPage(newPage)
	pages.ShowPage(newPage)
	pages.HidePage(oldPage)
}

func (am *AuthModal) setPage(page string) {
	// Set the current page in the modal based on the selected option
	if _, exists := am.pages[page]; exists {
		am.currentPage = page
	}
}

type ChangeProfileView struct {
	ui              *tview.Pages
	name            string
	handle          *AppHandle
	selectedProfile string
}

func NewChangeProfileView(handle *AppHandle) *ChangeProfileView {
	pages := tview.NewPages()
	view := ChangeProfileView{
		ui:              nil,
		name:            ipc.COMPONENT_CHANGE_PROFILE,
		handle:          handle,
		selectedProfile: "",
	}

	// inputs page
	button := tview.NewButton("Switch Profile").SetSelectedFunc(func() {
		if view.selectedProfile != "" {
			view.ui.ShowPage("changing")
			view.handle.SendTrigger(view.name, ipc.ACTION_CHANGE_PROFILE, ipc.ChangeProfileData{
				Profile: view.selectedProfile,
			})
			view.selectedProfile = "" // Reset selected profile after switching
		}
	})

	profileList := tview.NewList()
	for _, profile := range awsAuth.GetAvailableProfiles() {
		profileList.AddItem(profile, "", 0, func() {
			view.selectedProfile = profile
			view.handle.SetFocus(button)
		})
	}
	profileList.ShowSecondaryText(false)

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(
			tview.NewTextView().
				SetTextAlign(tview.AlignCenter).
				SetText("Select a Profile to Switch To"),
			2, 1, false).
		AddItem(profileList, 0, 1, true).
		AddItem(tview.NewBox(), 1, 1, false). // Spacer
		AddItem(button, 3, 1, false)

	flex.SetBorder(true)
	flex.SetBorderPadding(2, 2, 2, 2)
	flex.SetTitle(" [yellow]Change Profile[white] ═════ Set Access Keys ")

	// switching page
	switching := tview.NewTextView().
		SetText("Switching Profile...").
		SetTextAlign(tview.AlignCenter)

	switching.SetBorder(true)
	switching.SetBorderPadding(2, 2, 2, 2)
	switching.SetTitle(" [yellow]Change Profile[white] ═════ Set Access Keys ")

	// success page
	success := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tview.NewTextView().
							SetText("Access Keys Set Successfully!").
							SetTextAlign(tview.AlignCenter), 0, 1, false).
		AddItem(tview.NewBox(), 1, 1, false). // Spacer
		AddItem(tview.NewButton("Close").SetSelectedFunc(func() {
			pages.HidePage("success")
			pages.ShowPage("inputs")
			view.handle.PassEvent(ipc.Event{
				Component: ipc.COMPONENT_TUI,
				Action:    ipc.ACTION_CLOSE_AUTH_MODAL,
				Data:      nil,
			})
		}), 3, 1, true)

	success.SetBorder(true)
	success.SetBorderPadding(2, 2, 2, 2)
	success.SetTitle(" [yellow]Change Profile[white] ═════ Set Access Keys ")

	pages.AddPage("input", flex, true, true)
	pages.AddPage("switching", switching, true, false)
	pages.AddPage("success", success, true, false)

	pages.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyUp:
			if button.HasFocus() {
				// If the button has focus, move focus back to the profile list
				view.handle.SetFocus(profileList)
			}
		}
		return event
	})

	view.handle.SetSubscription(view.name, &view)
	view.ui = pages

	return &view
}

func (view *ChangeProfileView) Render(event *ipc.Event) tview.Primitive {
	view.ui.HidePage("switching")
	view.ui.ShowPage("success")

	return view.ui
}

func (view *ChangeProfileView) GetName() string {
	return view.name
}

type SetAccessKeysView struct {
	ui      *tview.Pages
	name    string
	handle  *AppHandle
	setting bool
}

func NewSetAccessKeysView(handle *AppHandle) *SetAccessKeysView {
	view := SetAccessKeysView{
		ui:      nil,
		name:    ipc.COMPONENT_SET_ACCESS_KEYS,
		handle:  handle,
		setting: false,
	}
	pages := tview.NewPages()

	// Inputs page
	accessKeyIDInput := tview.NewInputField().
		SetLabel("Access Key ID: ").
		SetFieldWidth(30).
		SetFieldTextColor(tcell.ColorBlack).
		SetFieldBackgroundColor(tcell.ColorWhite) // Set initial background color

	secretAccessKeyInput := tview.NewInputField().
		SetLabel("Secret Access Key: ").
		SetFieldWidth(30).
		SetMaskCharacter('*').
		SetFieldTextColor(tcell.ColorBlack)

	button := tview.NewButton("Set Access Keys").SetSelectedFunc(func() {
		accessKeyID := accessKeyIDInput.GetText()
		secretAccessKey := secretAccessKeyInput.GetText()

		if accessKeyID != "" && secretAccessKey != "" {
			view.setting = true
			view.ui.ShowPage("setting")
			view.handle.SendTrigger(view.name, "reauthWithNewAccessKeys", &ipc.AWSAccessKeysData{
				AccessKeyID:     accessKeyID,
				SecretAccessKey: secretAccessKey,
			})
		}
	})

	flex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(
			tview.NewTextView().
				SetTextAlign(tview.AlignCenter).
				SetText("Set New AWS Access Keys"),
							2, 1, false).
		AddItem(tview.NewBox(), 3, 1, false). // Spacer
		AddItem(accessKeyIDInput, 1, 1, true).
		AddItem(tview.NewBox(), 1, 1, false). // Spacer
		AddItem(secretAccessKeyInput, 1, 1, false).
		AddItem(tview.NewBox(), 3, 1, false). // Spacer
		AddItem(button, 3, 1, false)

	flex.SetBorder(true)
	flex.SetBorderPadding(2, 2, 2, 2)
	flex.SetTitle(" [white]Change Profile ═════ [yellow]Set Access Keys ")

	// Setting page
	setting := tview.NewTextView().
		SetText("Switching Profile...").
		SetTextAlign(tview.AlignCenter)
	setting.SetBorder(true)
	setting.SetBorderPadding(2, 2, 2, 2)
	setting.SetTitle(" [white]Change Profile ═════ [yellow]Set Access Keys ")

	// Success page
	success := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(tview.NewTextView().
							SetText("Access Keys Set Successfully!").
							SetTextAlign(tview.AlignCenter), 0, 1, false).
		AddItem(tview.NewBox(), 1, 1, false). // Spacer
		AddItem(tview.NewButton("Close").SetSelectedFunc(func() {
			pages.HidePage("success")
			pages.ShowPage("inputs")
			view.handle.PassEvent(ipc.Event{
				Component: ipc.COMPONENT_TUI,
				Action:    ipc.ACTION_CLOSE_AUTH_MODAL,
				Data:      nil,
			})
		}), 3, 1, true)
	success.SetBorder(true)
	success.SetBorderPadding(2, 2, 2, 2)
	success.SetTitle(" [white]Change Profile ═════ [yellow]Set Access Keys ")

	pages.AddPage("inputs", flex, true, true)
	pages.AddPage("setting", setting, true, false)
	pages.AddPage("success", success, true, false)

	pages.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyDown:
			if accessKeyIDInput.HasFocus() {
				// If the Access Key ID input has focus, move focus to the Secret Access Key input
				view.handle.SetFocus(secretAccessKeyInput)
				accessKeyIDInput.SetFieldBackgroundColor(tcell.ColorLightGray) // Reset background color
				secretAccessKeyInput.SetFieldBackgroundColor(tcell.ColorWhite) // Reset background color
			} else if secretAccessKeyInput.HasFocus() {
				// If the Secret Access Key input has focus, move focus to the button
				view.handle.SetFocus(button)
				secretAccessKeyInput.SetFieldBackgroundColor(tcell.ColorLightGray) // Reset background color
			}
		case tcell.KeyUp:
			if button.HasFocus() {
				// If the button has focus, move focus back to the Secret Access Key input
				view.handle.SetFocus(secretAccessKeyInput)
				secretAccessKeyInput.SetFieldBackgroundColor(tcell.ColorWhite) // Reset background color
			} else if secretAccessKeyInput.HasFocus() {
				// If the Secret Access Key input has focus, move focus back to the Access Key ID input
				view.handle.SetFocus(accessKeyIDInput)
				secretAccessKeyInput.SetFieldBackgroundColor(tcell.ColorLightGray) // Reset background color
				accessKeyIDInput.SetFieldBackgroundColor(tcell.ColorWhite)         // Reset background color
			}
		}
		return event
	})

	view.handle.SetSubscription(view.name, &view)
	view.ui = pages

	return &view
}

func (view *SetAccessKeysView) Render(event *ipc.Event) tview.Primitive {
	if view.setting {
		view.setting = false
		view.ui.HidePage("setting")
		view.ui.ShowPage("success")
	}

	return view.ui
}

func (view *SetAccessKeysView) GetName() string {
	return view.name
}

func makeModal(p tview.Primitive) tview.Primitive {
	return tview.NewFlex().
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexRow).
			AddItem(nil, 0, 1, false).
			AddItem(p, 20, 1, true).                  // height of the modal content
			AddItem(nil, 0, 1, false), 100, 1, true). // width of the modal content
		AddItem(nil, 0, 1, false)
}
