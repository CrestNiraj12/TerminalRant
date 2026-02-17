package feed

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) handleFeedLoadingMsg(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case RantsLoadedMsg:
		if msg.ReqSeq != m.feedReqSeq {
			return m, nil
		}
		if msg.QueryKey != m.currentFeedQueryKey() {
			return m, nil
		}
		// Reconciliation: Merge remote results with inflight optimistic items.
		rants := msg.Rants
		if m.feedSource == sourceFollowing {
			rants = filterOutOwnRants(rants)
		}
		newRants := make([]RantItem, len(rants))
		for i, r := range rants {
			newRants[i] = RantItem{Rant: r, Status: StatusNormal}
		}

		// Keep Pending items that aren't reconciled yet.
		var pendingItems []RantItem
		for _, ri := range m.rants {
			if ri.Status == StatusNormal || ri.Status == StatusFailed {
				continue
			}

			// For PendingUpdate/Delete, try to find the item in the new list.
			found := false
			if ri.Status == StatusPendingUpdate || ri.Status == StatusPendingDelete {
				for i, nr := range newRants {
					if nr.Rant.ID == ri.Rant.ID {
						found = true
						if ri.Status == StatusPendingDelete {
							// Successfully deleted on server but still in list?
						} else {
							// For simplicity, server wins once it arrives.
							ri.Status = StatusNormal
							newRants[i] = ri
						}
						break
					}
				}
			} else if ri.Status == StatusPendingCreate {
				// Match by content for creation. Fuzzy match to ignore hashtags/whitespace.
				for _, nr := range newRants {
					if strings.Contains(nr.Rant.Content, ri.Rant.Content) ||
						strings.Contains(ri.Rant.Content, nr.Rant.Content) {
						found = true
						break
					}
				}
			}

			if !found {
				pendingItems = append(pendingItems, ri)
			}
		}

		m.rants = append(pendingItems, newRants...)
		m.normalizeFeedOrder()
		m.loading = false
		m.loadingMore = false
		m.err = nil
		m.pagingNotice = ""
		m.oldestFeedID = m.lastFeedID()

		switch m.feedSource {
		case sourceTrending:
			m.hasMoreFeed = false
			m.oldestFeedID = ""
		case sourceFollowing:
			raw := msg.RawCount
			if raw == 0 {
				raw = len(msg.Rants)
			}
			m.hasMoreFeed = raw == defaultLimit
		default:
			m.hasMoreFeed = len(rants) == defaultLimit
		}

		if m.cursor >= len(m.rants) {
			m.cursor = 0
		}
		if m.feedSource == sourceFollowing {
			m.followingDirty = false
		}
		m.ensureFeedCursorVisible()
		return m, tea.Batch(m.ensureMediaPreviewCmd(), m.fetchRelationshipsForRants(msg.Rants))

	case RantsErrorMsg:
		if msg.ReqSeq != m.feedReqSeq {
			return m, nil
		}
		if msg.QueryKey != m.currentFeedQueryKey() {
			return m, nil
		}
		m.loading = false
		m.loadingMore = false
		m.err = msg.Err
		return m, nil

	case RantsPageLoadedMsg:
		if msg.ReqSeq != m.feedReqSeq {
			return m, nil
		}
		if msg.QueryKey != m.currentFeedQueryKey() {
			return m, nil
		}
		anchorTopID, anchorOffset, anchored := m.captureFeedTopAnchor()
		anchorID := ""
		if len(m.rants) > 0 && m.cursor >= 0 && m.cursor < len(m.rants) {
			anchorID = m.rants[m.cursor].Rant.ID
		}
		m.loadingMore = false
		m.err = nil
		rants := msg.Rants
		if m.feedSource == sourceFollowing {
			rants = filterOutOwnRants(rants)
		}
		if len(rants) == 0 && msg.RawCount == 0 {
			m.hasMoreFeed = false
			if len(m.rants) > 0 {
				m.pagingNotice = "🚀 End of the rantverse reached."
			}
			return m, nil
		}
		existing := make(map[string]struct{}, len(m.rants))
		for _, ri := range m.rants {
			existing[ri.Rant.ID] = struct{}{}
		}
		added := 0
		for _, r := range rants {
			if _, ok := existing[r.ID]; ok {
				continue
			}
			m.rants = append(m.rants, RantItem{Rant: r, Status: StatusNormal})
			added++
		}
		m.oldestFeedID = m.lastFeedID()
		switch m.feedSource {
		case sourceTrending:
			m.hasMoreFeed = false
			m.oldestFeedID = ""
		case sourceFollowing:
			raw := msg.RawCount
			if raw == 0 {
				raw = len(msg.Rants)
			}
		default:
			m.hasMoreFeed = len(rants) == defaultLimit && added > 0
		}

		if added == 0 && len(m.rants) > 0 && m.feedSource != sourceFollowing {
			m.hasMoreFeed = false
			m.pagingNotice = "🚀 End of the rantverse reached."
		} else if m.hasMoreFeed {
			m.pagingNotice = ""
		}
		if anchorID != "" {
			m.setCursorByID(anchorID)
		}
		if anchored {
			m.restoreFeedTopAnchor(anchorTopID, anchorOffset)
		}
		return m, m.fetchRelationshipsForRants(msg.Rants)

	case RantsPageErrorMsg:
		if msg.ReqSeq != m.feedReqSeq {
			return m, nil
		}
		if msg.QueryKey != m.currentFeedQueryKey() {
			return m, nil
		}
		m.loadingMore = false
		m.err = msg.Err
		return m, nil

	}

	return m, nil
}
