package iterm2

import (
	"fmt"
	"strings"

	"github.com/trzsz/iterm2/api"
)

// SplitPaneOptions configures how a pane is split to create a new session
type SplitPaneOptions struct {
	// Vertical specifies the split orientation
	// If True, the divider is vertical, else horizontal
	Vertical bool
}

// Session represents an iTerm2 Session which is a pane
type Session struct {
	app *App
	wid string
	tid string
	sid string
}

func newSession(app *App, wid, tid, sid string) *Session {
	return &Session{app, wid, tid, sid}
}

// GetApp returns the iTerm2 application instance that owns this session
func (s *Session) GetApp() *App {
	return s.app
}

// GetWindowID returns the unique identifier of the window containing this session
func (s *Session) GetWindowID() string {
	return s.wid
}

// GetWindow returns the window containing this session
func (s *Session) GetWindow() *Window {
	return newWindow(s.app, s.wid)
}

// GetTabID returns the unique identifier of the tab containing this session
func (s *Session) GetTabID() string {
	return s.tid
}

// GetTab returns the tab containing this session
func (s *Session) GetTab() *Tab {
	return newTab(s.app, s.wid, s.tid)
}

// GetSessionID returns the unique identifier for this session
func (s *Session) GetSessionID() string {
	return s.sid
}

// Inject injects data as though it were program output
func (s *Session) Inject(data []byte) error {
	resp, err := s.app.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_InjectRequest{
			InjectRequest: &api.InjectRequest{
				SessionId: []string{s.sid},
				Data:      data,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("call inject_request failed: %w", err)
	}

	iResp := resp.GetInjectResponse()
	if iResp == nil {
		return fmt.Errorf("inject_response is nil")
	}

	status := iResp.GetStatus()
	if len(status) != 1 {
		return fmt.Errorf("inject_response status count is not one: %v", status)
	}
	if status[0] != api.InjectResponse_OK {
		return fmt.Errorf("inject_response status is not ok: %v", status[0])
	}
	return nil
}

// SendText sends text as though the user had typed it
func (s *Session) SendText(text string) error {
	resp, err := s.app.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_SendTextRequest{
			SendTextRequest: &api.SendTextRequest{
				Session: &s.sid,
				Text:    &text,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("call send_text_request failed: %w", err)
	}

	stResp := resp.GetSendTextResponse()
	if stResp == nil {
		return fmt.Errorf("send_text_response is nil")
	}
	if stResp.GetStatus() != api.SendTextResponse_OK {
		return fmt.Errorf("send_text_response status is not ok: %v", stResp.GetStatus())
	}
	return nil
}

// Activate makes the session the active session in its tab
// selectTab: whether the tab this session is in should be selected
// orderWindowFront: whether the window this session is in should be brought to the front and given keyboard focus
func (s *Session) Activate(selectTab, orderWindowFront bool) error {
	resp, err := s.app.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_ActivateRequest{
			ActivateRequest: &api.ActivateRequest{
				Identifier: &api.ActivateRequest_SessionId{
					SessionId: s.sid,
				},
				SelectTab:        &selectTab,
				OrderWindowFront: &orderWindowFront,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("call activate_request failed: %w", err)
	}

	aResp := resp.GetActivateResponse()
	if aResp == nil {
		return fmt.Errorf("activate_response is nil")
	}
	if aResp.GetStatus() != api.ActivateResponse_OK {
		return fmt.Errorf("activate_response status is not ok: %v", aResp.GetStatus())
	}
	return nil
}

// SplitPane splits the pane, creating a new session
func (s *Session) SplitPane(opts SplitPaneOptions) (*Session, error) {
	direction := api.SplitPaneRequest_HORIZONTAL.Enum()
	if opts.Vertical {
		direction = api.SplitPaneRequest_VERTICAL.Enum()
	}

	resp, err := s.app.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_SplitPaneRequest{
			SplitPaneRequest: &api.SplitPaneRequest{
				Session:        &s.sid,
				SplitDirection: direction,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("call split_pane_request failed: %w", err)
	}

	spResp := resp.GetSplitPaneResponse()
	if spResp == nil {
		return nil, fmt.Errorf("split_pane_response is nil")
	}
	if spResp.GetStatus() != api.SplitPaneResponse_OK {
		return nil, fmt.Errorf("split_pane_response status is not ok: %v", spResp.GetStatus())
	}

	sid := spResp.GetSessionId()
	if len(sid) != 1 {
		return nil, fmt.Errorf("split_pane_response session_id count is not one: %v", sid)
	}
	return newSession(s.app, s.wid, s.tid, sid[0]), nil
}

// GetVariable fetches a session variable
func (s *Session) GetVariable(names ...string) ([]string, error) {
	resp, err := s.app.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_VariableRequest{
			VariableRequest: &api.VariableRequest{
				Scope: &api.VariableRequest_SessionId{
					SessionId: s.sid,
				},
				Get: names,
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("call variable_request failed: %w", err)
	}

	vResp := resp.GetVariableResponse()
	if vResp == nil {
		return nil, fmt.Errorf("variable_response is nil")
	}
	if vResp.GetStatus() != api.VariableResponse_OK {
		return nil, fmt.Errorf("variable_response status is not ok: %v", vResp.GetStatus())
	}

	values := vResp.GetValues()
	if len(values) != len(names) {
		return nil, fmt.Errorf("variable_response values count is not %d: %v", len(names), values)
	}

	return values, nil
}

// IsTmuxIntegrationSession reports whether this session is attached to a tmux session
func (s *Session) IsTmuxIntegrationSession() (bool, error) {
	resp, err := s.app.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_TmuxRequest{
			TmuxRequest: &api.TmuxRequest{
				Payload: &api.TmuxRequest_ListConnections_{
					ListConnections: &api.TmuxRequest_ListConnections{},
				},
			},
		},
	})
	if err != nil {
		return false, fmt.Errorf("call tmux_list_connections failed: %w", err)
	}

	tResp := resp.GetTmuxResponse()
	if tResp == nil {
		return false, fmt.Errorf("tmux_response is nil")
	}
	if tResp.GetStatus() != api.TmuxResponse_OK {
		return false, fmt.Errorf("tmux_response status is not ok: %v", tResp.GetStatus())
	}

	payload := tResp.Payload
	if payload == nil {
		return false, nil
	}

	lc, ok := payload.(*api.TmuxResponse_ListConnections_)
	if !ok {
		return false, fmt.Errorf("tmux_response payload is not list_connections: %T", tResp.Payload)
	}

	for _, c := range lc.ListConnections.GetConnections() {
		if c.GetOwningSessionId() == s.sid {
			return true, nil
		}
	}
	return false, nil
}

// RunTmuxCommand invokes a tmux command and return its result
func (s *Session) RunTmuxCommand(command string, timeout float64) (string, error) {
	invocation := "iterm2.run_tmux_command(command: \"" + strings.ReplaceAll(strings.ReplaceAll(command, "\\", "\\\\"), "\"", "\\\"") + "\")"
	resp, err := s.app.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_InvokeFunctionRequest{
			InvokeFunctionRequest: &api.InvokeFunctionRequest{
				Invocation: &invocation,
				Context: &api.InvokeFunctionRequest_Method_{
					Method: &api.InvokeFunctionRequest_Method{
						Receiver: &s.sid,
					},
				},
				Timeout: &timeout,
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("call invoke_function_request failed: %w", err)
	}

	ifResp := resp.GetInvokeFunctionResponse()
	if ifResp == nil {
		return "", fmt.Errorf("invoke_function_response is nil")
	}
	if success := ifResp.GetSuccess(); success != nil {
		return success.GetJsonResult(), nil
	}
	if err := ifResp.GetError(); err != nil {
		return "", fmt.Errorf("invoke_function_response error: %v", err.GetErrorReason())
	}
	return "", fmt.Errorf("unknown invoke_function_response: %+v", ifResp)
}
