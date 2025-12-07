package iterm2

import (
	"fmt"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/trzsz/iterm2/api"
	"github.com/trzsz/iterm2/client"
)

// NewApp establishes a connection to iTerm2 and returns an App instance.
// The optional name parameter registers the application with iTerm2 to avoid repeated permission prompts.
func NewApp(name string) (*App, error) {
	c, err := client.New(name)
	if err != nil {
		return nil, err
	}
	return newApp(c), nil
}

// App represents an open iTerm2 application instance
type App struct {
	c *client.Client
}

func newApp(c *client.Client) *App {
	return &App{c}
}

// Close closes the iTerm2 application connection
func (a *App) Close() error {
	return a.c.Close()
}

// CreateWindow creates a new terminal window in iTerm2
func (a *App) CreateWindow() (*Window, *Session, error) {
	resp, err := a.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_CreateTabRequest{
			CreateTabRequest: &api.CreateTabRequest{},
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

	session := newSession(a, ctResp.GetWindowId(), strconv.Itoa(int(ctResp.GetTabId())), ctResp.GetSessionId())
	return session.GetWindow(), session, nil
}

// ListWindows retrieves all terminal windows in iTerm2
func (a *App) ListWindows() ([]*Window, error) {
	resp, err := a.c.Call(&api.ClientOriginatedMessage{
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
	windows := lsResp.GetWindows()
	list := make([]*Window, 0, len(windows))
	for _, w := range windows {
		list = append(list, newWindow(a, w.GetWindowId()))
	}
	return list, nil
}

// SelectMenuItem selects a menu item
func (a *App) SelectMenuItem(item string) error {
	resp, err := a.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_MenuItemRequest{
			MenuItemRequest: &api.MenuItemRequest{
				Identifier: &item,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("call menu_item_request failed: %w", err)
	}

	miResp := resp.GetMenuItemResponse()
	if miResp == nil {
		return fmt.Errorf("menu_item_response is nil")
	}
	if miResp.GetStatus() != api.MenuItemResponse_OK {
		return fmt.Errorf("menu_item_response status is not ok: %v", miResp.GetStatus())
	}
	return nil
}

// GetCurrentHostSession returns the session that the current process belongs to
func (a *App) GetCurrentHostSession() (*Session, error) {
	sessionId := os.Getenv("ITERM_SESSION_ID")
	if sessionId == "" {
		return nil, fmt.Errorf("ITERM_SESSION_ID is not set")
	}

	session, err := findSessionByMatch(a, func(wid, tid, sid string) bool {
		return sid != "" && strings.Contains(sessionId, sid)
	})
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, fmt.Errorf("host session not found: %v", sessionId)
	}
	return session, nil
}

func (a *App) getFocusInfo() (string, []string, []string, error) {
	focusResp, err := a.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_FocusRequest{
			FocusRequest: &api.FocusRequest{},
		},
	})
	if err != nil {
		return "", nil, nil, fmt.Errorf("call focus_request failed: %w", err)
	}

	fResp := focusResp.GetFocusResponse()
	if fResp == nil {
		return "", nil, nil, fmt.Errorf("focus_response is nil")
	}

	var windows []*api.FocusChangedNotification_Window
	var tabs []string
	var sessions []string

	for _, n := range fResp.GetNotifications() {
		switch ev := n.Event.(type) {
		case *api.FocusChangedNotification_Window_:
			windows = append(windows, ev.Window)
		case *api.FocusChangedNotification_SelectedTab:
			tabs = append(tabs, ev.SelectedTab)
		case *api.FocusChangedNotification_Session:
			sessions = append(sessions, ev.Session)
		}
	}

	if len(windows) == 0 {
		return "", nil, nil, fmt.Errorf("no active window in focus_response")
	}

	if len(windows) > 1 {
		sort.Slice(windows, func(i, j int) bool {
			return windows[i].GetWindowStatus() < windows[j].GetWindowStatus()
		})
	}

	return windows[0].GetWindowId(), tabs, sessions, nil
}

// GetCurrentActiveSession returns the session that currently has user focus
func (a *App) GetCurrentActiveSession() (*Session, error) {
	focusWid, focusTabs, focusSessions, err := a.getFocusInfo()
	if err != nil {
		return nil, err
	}

	session, err := findSessionByMatch(a, func(wid, tid, sid string) bool {
		return wid == focusWid && slices.Contains(focusTabs, tid) && slices.Contains(focusSessions, sid)
	})
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, fmt.Errorf("active session not found")
	}
	return session, nil
}

// GetCurrentTmuxSession returns any available tmux session related to the current process, active preferred
func (a *App) GetCurrentTmuxSession() (*Session, error) {
	focusWid, focusTabs, focusSessions, err := a.getFocusInfo()
	if err != nil {
		return nil, err
	}

	pid := os.Getpid()
	var tmuxSessions []*Session
	session, err := findSessionByMatch(a, func(wid, tid, sid string) bool {
		session := newSession(a, wid, tid, sid)
		values, err := session.GetVariable("jobPid", "tmuxWindowPane")
		if err != nil || len(values) != 2 || values[1] == "null" {
			return false
		}
		if _, err := strconv.ParseUint(values[1], 10, 32); err != nil {
			return false
		}
		if jobPid, err := strconv.ParseUint(values[0], 10, 32); err != nil || pid != int(jobPid) {
			return false
		}
		tmuxSessions = append(tmuxSessions, session)
		return wid == focusWid && slices.Contains(focusTabs, tid) && slices.Contains(focusSessions, sid)
	})
	if err != nil {
		return nil, err
	}

	if session != nil {
		return session, nil
	}
	for _, tmuxSession := range tmuxSessions {
		if slices.Contains(focusSessions, tmuxSession.GetSessionID()) {
			return tmuxSession, nil
		}
	}
	if len(tmuxSessions) > 0 {
		return tmuxSessions[0], nil
	}
	return nil, fmt.Errorf("tmux session not found")
}
