package tui

import (
	"fmt"
	"log/slog"

	"github.com/gdamore/tcell/v2"
	"github.com/livinlefevreloca/canopy/internal/ipc"
	"github.com/rivo/tview"
)

const ASCIIArt = `
.................................................................................
.................................................................................
.................................................................................
....######+......#######.........................................................
.......+####...#####.............................................................
..........###..##................................................................
...########.#.##.########........................................................
.####-..-#########+...+####......................................................
............#.####............######..-##.####.....#####+...##.#####..###....###.
..........-#....####.........##...###.+###..###..###...###..###+..###.-##...+##..
..........##......###..........-#####.+##....##-.##+....###.##+....###.###..##-..
.........##........##.......###...###.+##....##-.###....###.###....###..######...
.........##.........#.......###..-###.+##....##+.####.-###..####.-###....####....
.........###.................####+###.+##....##+...#####....##+#####.....###-....
.........###..........######................................##+..........###.....
..........####.........##.##................................##+........####......
...........+###########...#......................................................
...............###-..............................................................
.................................................................................
.................................................................................
.................................................................................
.................................................................................
.................................................................................
.................................................................................
`

// Interface for renderable components in the TUI.
// A renderable component `Usually` consists of a tview.Primitive
// and some state to manage. Each renderable component is then
// registered with the EventHandler which is run by the AppHandle.
// The EventHandler runs in a separate goroutine and listens for
// responses to ui triggers which it routes to the proper component.
// The handler than calls the Render method of the component in
// QueueUpdateDraw which state on the component and then redraws
// the screen
type Renderable interface {
	// Render takes a Event from the EventHandler.
	// It also returns the primitive from the Renderable
	// though this if for some special cases and is not
	// used by the event handler.
	Render(*ipc.Event) tview.Primitive
	GetName() string // GetName returns the name of the component, used for identification
}

// The Tui struct represents the main TUI application.
type Tui struct {
	handle      *AppHandle
	name        string                // Name of the TUI application
	ui          *tview.Pages          // The main layout of the TUI application
	currentPage string                // Track the current page in the TUI
	pages       map[string]Renderable // Map of pages in the TUI
}

// Create a TUI instance and initialize it with the given trigger handler.
// Create the main layout for the app and setup up toplevel keybindings.
func NewTui(reqhandler *ipc.TriggerHandler) *Tui {
	app := tview.NewApplication()
	handle := NewAppHandle(reqhandler, app)
	// Run the event handler in a separate goroutine
	go handle.RunEventHandler()
	errorModal := NewErrorModal(handle)
	authModal := NewAuthModal(handle)
	helpModal := NewHelpModal(handle)
	ssoModal := NewSSOReauthenticationModal(handle)
	// Initialize the Tui instance with the AppHandle and modals
	pages := make(map[string]Renderable)
	pages[errorModal.GetName()] = errorModal
	pages[authModal.GetName()] = authModal
	pages[helpModal.GetName()] = helpModal
	pages[ssoModal.GetName()] = ssoModal

	tui := &Tui{
		handle:      handle,
		ui:          nil,
		name:        ipc.COMPONENT_TUI,
		currentPage: "",
		pages:       pages,
	}
	tui.handle.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyCtrlA:
			tui.toggleComponent(authModal.GetName())
			tui.handle.SetRoot(tui.ui, true)
		case tcell.KeyCtrlH:
			tui.toggleComponent(helpModal.GetName())
			tui.handle.SetRoot(tui.ui, true)
		case tcell.KeyCtrlS:
			tui.toggleComponent(ipc.COMPONENT_REFRESH_SSO)
			tui.handle.SetRoot(tui.ui, true)
		case tcell.KeyCtrlC:
			tui.handle.SendTrigger(ipc.COMPONENT_QUIT, ipc.ACTION_END, nil)
		default:
		}
		return event
	})

	configData := ipc.AWSConfigData{}
	header := NewHeader(configData, tui.handle)

	mainText := tview.NewTextView().
		SetText(ASCIIArt).
		SetTextAlign(tview.AlignCenter)

	mainText.SetBorder(true)
	mainText.SetTitle("Canopy TUI")

	mainLayout := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(header.ui, 7, 1, false).
		AddItem(mainText, 0, 1, false)

	mainPages := tview.NewPages()
	mainPages.AddPage("main", mainLayout, true, true)
	mainPages.AddPage(authModal.GetName(), authModal.ui, true, false)
	mainPages.AddPage(helpModal.GetName(), helpModal.ui, true, false)
	mainPages.AddPage(errorModal.GetName(), errorModal.ui, true, false)
	mainPages.AddPage(ssoModal.GetName(), ssoModal.ui, true, false)

	tui.handle.SetSubscription(tui.GetName(), tui)

	tui.ui = mainPages
	tui.handle.SetRoot(mainPages, true)

	return tui
}

func (t *Tui) Render(response *ipc.Event) tview.Primitive {
	slog.Debug("Tui Render: Received event", "event", response)
	switch response.Action {
	case ipc.ACTION_SHOW_ERROR_MODAL:
		t.ShowComponent(ipc.COMPONENT_ERROR_MODAL)
	case ipc.ACTION_CLOSE_ERROR_MODAL:
		t.HideComponent(ipc.COMPONENT_ERROR_MODAL)
	case ipc.ACTION_SHOW_REAUTHENTICATE_SSO_MODAL:
		t.ShowComponent(ipc.COMPONENT_REFRESH_SSO)
	case ipc.ACTION_CLOSE_REAUTHENTICATE_SSO_MODAL:
		t.HideComponent(ipc.COMPONENT_REFRESH_SSO)
	case ipc.ACTION_CLOSE_AUTH_MODAL:
		t.HideComponent(ipc.COMPONENT_AUTH_MODAL)
	}

	return t.ui
}

// Start the event handler and the TUI application.
func (t *Tui) Run() error {
	// Start the TUI application
	if err := t.handle.Run(); err != nil {
		return err
	}

	return nil
}

func (t *Tui) toggleComponent(componentName string) {
	component, _ := t.pages[componentName]
	otherComponents := make([]string, len(t.pages)-1)
	for name, _ := range t.pages {
		if name != componentName {
			otherComponents = append(otherComponents, name)
		}
	}
	if component == nil {
		panic("Tui toggleComponent: Component " + componentName + " not found")
	}
	if componentName != t.currentPage {
		t.currentPage = componentName
		t.ui.ShowPage(componentName)
		for _, comp := range otherComponents {
			t.ui.HidePage(comp)
		}
	} else {
		t.currentPage = ""
		t.ui.HidePage(componentName)
	}
}

func (t *Tui) ShowComponent(componentName string) {
	_, exists := t.pages[componentName]
	if !exists {
		panic(fmt.Sprintf("Tui ShowComponent: Component %s not found", componentName))
	}
	if t.currentPage != componentName {
		t.currentPage = componentName
		t.ui.ShowPage(componentName)
	}
	return
}

func (t *Tui) HideComponent(componentName string) {
	slog.Debug("Tui HideComponent: Hiding component", "component", componentName)
	_, exists := t.pages[componentName]
	if !exists {
		panic(fmt.Sprintf("Tui HideComponent: Component %s not found", componentName))
	}
	if t.currentPage == componentName {
		t.currentPage = "main"
		t.ui.HidePage(componentName)
	}
}

func (t *Tui) GetName() string {
	return t.name
}

type Header struct {
	handle *AppHandle
	name   string
	ui     tview.Primitive
	ipc.AWSConfigData
}

func NewHeader(configData ipc.AWSConfigData, handle *AppHandle) *Header {
	header := Header{
		ui:            nil,
		name:          ipc.COMPONENT_HEADER,
		handle:        handle,
		AWSConfigData: configData,
	}

	text := "[yellow]AWS Profile: [white]" + header.Profile + "\n" +
		"[yellow]AWS SSO Role Name: [white]" + header.SSORoleName + "\n" +
		"[yellow]AWS Account Id: [white]" + header.AccountId + "\n" +
		"[yellow]AWS Assumed Role: [white]" + header.AssumeRoleARN + "\n" +
		"[yellow]AWS Access Key ID: [white]" + header.AccessKeyID + "\n" +
		"[yellow]AWS Credentials Source: [white]" + header.CredentialsSource + "\n" +
		"[yellow]AWS Region: [white]" + header.Region

	ui := tview.NewTextView().
		SetDynamicColors(true).
		SetTextAlign(tview.AlignLeft).
		SetText(text)

	header.ui = ui

	header.handle.SetSubscription(header.GetName(), &header)
	// Trigger the initial AWS config data
	header.TriggerAuth()

	return &header
}

func (h *Header) TriggerAuth() {
	h.handle.SendTrigger(h.name, ipc.ACTION_GET_AUTH_DATA, nil)
}

func (h *Header) Render(event *ipc.Event) tview.Primitive {
	// take the last response and update the header with the latest config data
	slog.Debug("Header Render: Received event", "event", event)

	configData, ok := event.Data.(ipc.AWSConfigData)
	if !ok {
		panic(fmt.Sprintf("Header Render: Expected AWSConfigData, got %x", event.Data))
	}
	h.AWSConfigData = configData

	// Return the UI component for this header
	text := "[yellow]AWS Profile: [white]" + h.Profile + "\n" +
		"[yellow]AWS SSO Role Name: [white]" + h.SSORoleName + "\n" +
		"[yellow]AWS Account Id: [white]" + h.AccountId + "\n" +
		"[yellow]AWS Assumed Role: [white]" + h.AssumeRoleARN + "\n" +
		"[yellow]AWS Access Key ID: [white]" + h.AccessKeyID + "\n" +
		"[yellow]AWS Credentials Source: [white]" + h.CredentialsSource + "\n" +
		"[yellow]AWS Region: [white]" + h.Region

	h.ui.(*tview.TextView).SetText(text)

	return h.ui
}

func (h *Header) GetName() string {
	return h.name
}
