package iterm2

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/trzsz/iterm2/api"
	"github.com/trzsz/iterm2/client"
)

// App represents an open iTerm2 application
type App interface {
	io.Closer

	CreateWindow() (Window, error)
	ListWindows() ([]Window, error)
	SelectMenuItem(item string) error
	GetCurrentWindowSession() (Window, Session, error)
}

// NewApp establishes a connection
// with iTerm2 and returns an App.
// Name is an optional parameter that
// can be used to register your application
// name with iTerm2 so that it doesn't
// require explicit permissions every
// time you run the plugin.
func NewApp(name string) (App, error) {
	c, err := client.New(name)
	if err != nil {
		return nil, err
	}

	return &app{c: c}, nil
}

type app struct {
	c *client.Client
}

func (a *app) CreateWindow() (Window, error) {
	resp, err := a.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_CreateTabRequest{
			CreateTabRequest: &api.CreateTabRequest{},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("could not create window tab: %w", err)
	}
	ctr := resp.GetCreateTabResponse()
	if ctr.GetStatus() != api.CreateTabResponse_OK {
		return nil, fmt.Errorf("unexpected window tab status: %s", ctr.GetStatus())
	}
	return &window{
		c:       a.c,
		id:      ctr.GetWindowId(),
		session: ctr.GetSessionId(),
	}, nil
}

func (a *app) ListWindows() ([]Window, error) {
	list := []Window{}
	resp, err := a.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_ListSessionsRequest{
			ListSessionsRequest: &api.ListSessionsRequest{},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("could not list sessions: %w", err)
	}
	for _, w := range resp.GetListSessionsResponse().GetWindows() {
		list = append(list, &window{
			c:  a.c,
			id: w.GetWindowId(),
		})
	}
	return list, nil
}

func (a *app) Close() error {
	return a.c.Close()
}

func str(s string) *string {
	return &s
}

func (a *app) SelectMenuItem(item string) error {
	resp, err := a.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_MenuItemRequest{
			MenuItemRequest: &api.MenuItemRequest{
				Identifier: &item,
			},
		},
	})
	if err != nil {
		return fmt.Errorf("error selecting menu item %q: %w", item, err)
	}
	if resp.GetMenuItemResponse().GetStatus() != api.MenuItemResponse_OK {
		return fmt.Errorf("menu item %q returned unexpected status: %q", item, resp.GetMenuItemResponse().GetStatus().String())
	}
	return nil
}

func (a *app) GetCurrentWindowSession() (Window, Session, error) {
	sessionID := os.Getenv("ITERM_SESSION_ID")
	if sessionID == "" {
		return nil, nil, fmt.Errorf("ITERM_SESSION_ID environment variable is not set")
	}

	resp, err := a.c.Call(&api.ClientOriginatedMessage{
		Submessage: &api.ClientOriginatedMessage_ListSessionsRequest{
			ListSessionsRequest: &api.ListSessionsRequest{},
		},
	})
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list sessions: %w", err)
	}

	lsr := resp.GetListSessionsResponse()
	for _, win := range lsr.GetWindows() {
		for _, tab := range win.GetTabs() {
			root := tab.GetRoot()
			if root == nil {
				continue
			}
			for _, link := range root.GetLinks() {
				if sess := getSessionFromLink(link, sessionID); sess != nil {
					w := &window{
						c:  a.c,
						id: win.GetWindowId(),
					}
					s := &session{
						c:  a.c,
						id: sess.GetUniqueIdentifier(),
					}
					return w, s, nil
				}
			}
		}
	}

	return nil, nil, fmt.Errorf("no session found for session ID: %s", sessionID)
}

func getSessionFromLink(link *api.SplitTreeNode_SplitTreeLink, sessionID string) *api.SessionSummary {
	child := link.GetChild()
	if child == nil {
		return nil
	}
	switch child := child.(type) {
	case *api.SplitTreeNode_SplitTreeLink_Session:
		if child.Session != nil {
			id := child.Session.GetUniqueIdentifier()
			if id != "" && strings.Contains(sessionID, id) {
				return child.Session
			}
		}
	case *api.SplitTreeNode_SplitTreeLink_Node:
		for _, childLink := range child.Node.GetLinks() {
			if session := getSessionFromLink(childLink, sessionID); session != nil {
				return session
			}
		}
	}
	return nil
}
