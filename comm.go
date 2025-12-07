package iterm2

import (
	"fmt"

	"github.com/trzsz/iterm2/api"
)

func findSessionByMatch(app *App, matchFn func(wid, tid, sid string) bool) (*Session, error) {
	resp, err := app.c.Call(&api.ClientOriginatedMessage{
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

	for _, win := range lsResp.GetWindows() {
		for _, tab := range win.GetTabs() {
			root := tab.GetRoot()
			if root == nil {
				continue
			}
			for _, link := range root.GetLinks() {
				if session := findSessionInNodeLink(link, func(sid string) bool {
					return matchFn(win.GetWindowId(), tab.GetTabId(), sid)
				}); session != nil {
					return newSession(app, win.GetWindowId(), tab.GetTabId(), session.GetUniqueIdentifier()), nil
				}
			}
		}
	}

	return nil, nil
}

func findSessionInNodeLink(link *api.SplitTreeNode_SplitTreeLink, matchFn func(string) bool) *api.SessionSummary {
	child := link.GetChild()
	if child == nil {
		return nil
	}

	switch child := child.(type) {
	case *api.SplitTreeNode_SplitTreeLink_Session:
		if child.Session != nil {
			id := child.Session.GetUniqueIdentifier()
			if matchFn(id) {
				return child.Session
			}
		}
	case *api.SplitTreeNode_SplitTreeLink_Node:
		for _, childLink := range child.Node.GetLinks() {
			if session := findSessionInNodeLink(childLink, matchFn); session != nil {
				return session
			}
		}
	}

	return nil
}
