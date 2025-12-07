package iterm2

import (
	"fmt"

	"github.com/trzsz/iterm2/api"
)

// Tab represents an iTerm2 Tab
type Tab struct {
	app *App
	wid string
	tid string
}

func newTab(app *App, wid, tid string) *Tab {
	return &Tab{app, wid, tid}
}

// GetApp returns the iTerm2 application instance that owns this tab
func (t *Tab) GetApp() *App {
	return t.app
}

// GetWindowID returns the unique identifier of the window containing this tab
func (t *Tab) GetWindowID() string {
	return t.wid
}

// GetWindow returns the window containing this tab
func (t *Tab) GetWindow() *Window {
	return newWindow(t.app, t.wid)
}

// GetTabID returns the unique identifier for this tab
func (t *Tab) GetTabID() string {
	return t.tid
}

// SetTitle changes the tab’s title
func (t *Tab) SetTitle(s string) error {
	invocation := fmt.Sprintf(`iterm2.set_title(title: "%s")`, s)
	resp, err := t.app.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_InvokeFunctionRequest{
			InvokeFunctionRequest: &api.InvokeFunctionRequest{
				Invocation: &invocation,
				Context: &api.InvokeFunctionRequest_Method_{
					Method: &api.InvokeFunctionRequest_Method{
						Receiver: &t.tid,
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

// ListSessions retrieves all sessions in this tab
func (t *Tab) ListSessions() ([]*Session, error) {
	var sessions []*Session
	_, err := findSessionByMatch(t.app, func(wid, tid, sid string) bool {
		if wid == t.wid && tid == t.tid {
			sessions = append(sessions, newSession(t.app, wid, tid, sid))
		}
		return false
	})
	if err != nil {
		return nil, err
	}
	if len(sessions) == 0 {
		return nil, fmt.Errorf("no sessions found in tab: %v", t.tid)
	}
	return sessions, nil
}
