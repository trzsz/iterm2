package iterm2

import (
	"fmt"
	"strconv"

	"github.com/trzsz/iterm2/api"
)

// Window represents an iTerm2 Window
type Window struct {
	app *App
	wid string
}

func newWindow(app *App, wid string) *Window {
	return &Window{app, wid}
}

// GetApp returns the iTerm2 application instance that owns this window
func (w *Window) GetApp() *App {
	return w.app
}

// GetWindowID returns the unique identifier for this window
func (w *Window) GetWindowID() string {
	return w.wid
}

// SetTitle changes the window’s title
func (w *Window) SetTitle(s string) error {
	invocation := fmt.Sprintf(`iterm2.set_title(title: "%s")`, s)
	resp, err := w.app.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_InvokeFunctionRequest{
			InvokeFunctionRequest: &api.InvokeFunctionRequest{
				Invocation: &invocation,
				Context: &api.InvokeFunctionRequest_Method_{
					Method: &api.InvokeFunctionRequest_Method{
						Receiver: &w.wid,
					},
				},
			},
		},
	})
	if err != nil {
		return fmt.Errorf("call set_title failed: %w", err)
	}

	ifResp := resp.GetInvokeFunctionResponse()
	if ifResp == nil {
		return fmt.Errorf("invoke_function_response is nil")
	}
	if err := ifResp.GetError(); err != nil {
		return fmt.Errorf("set_title error: %s", err.GetErrorReason())
	}
	return nil
}

// CreateTab creates a new tab in this window
func (w *Window) CreateTab() (*Tab, *Session, error) {
	resp, err := w.app.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_CreateTabRequest{
			CreateTabRequest: &api.CreateTabRequest{
				WindowId: &w.wid,
			},
		},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("call create_tab_request failed: %w", err)
	}

	ctResp := resp.GetCreateTabResponse()
	if ctResp == nil {
		return nil, nil, fmt.Errorf("create_tab_response is nil")
	}
	if ctResp.GetStatus() != api.CreateTabResponse_OK {
		return nil, nil, fmt.Errorf("create_tab_response status is not ok: %v", ctResp.GetStatus())
	}

	session := newSession(w.app, ctResp.GetWindowId(), strconv.Itoa(int(ctResp.GetTabId())), ctResp.GetSessionId())
	return session.GetTab(), session, nil
}

// ListTabs retrieves all tabs in this window
func (w *Window) ListTabs() ([]*Tab, error) {
	resp, err := w.app.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_ListSessionsRequest{
			ListSessionsRequest: &api.ListSessionsRequest{},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("call list_sessions_request failed: %w", err)
	}

	lsResp := resp.GetListSessionsResponse()
	if lsResp == nil {
		return nil, fmt.Errorf("list_sessions_response is nil")
	}
	for _, window := range lsResp.GetWindows() {
		if window.GetWindowId() == w.wid {
			tabs := window.GetTabs()
			list := make([]*Tab, 0, len(tabs))
			for _, t := range tabs {
				list = append(list, newTab(w.app, w.wid, t.GetTabId()))
			}
			return list, nil
		}
	}
	return nil, fmt.Errorf("window not found: %v", w.wid)
}
