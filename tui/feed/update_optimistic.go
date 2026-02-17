package feed

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/CrestNiraj12/terminalrant/domain"
)

func (m Model) handleOptimisticMsg(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case AddOptimisticRantMsg:
		newItem := RantItem{
			Rant: domain.Rant{
				ID:        fmt.Sprintf("local-%d", time.Now().UnixNano()),
				Content:   msg.Content,
				Author:    "You", // Generic placeholder
				Username:  "you",
				IsOwn:     true,
				CreatedAt: time.Now(),
			},
			Status: StatusPendingCreate,
		}

		m.rants = append([]RantItem{newItem}, m.rants...)
		m.cursor = 0     // Focus the new item
		m.startIndex = 0 // Scroll to top
		m.scrollLine = 0
		return m, nil
	case AddOptimisticReplyMsg:
		reply := domain.Rant{
			ID:          fmt.Sprintf("local-reply-%d", time.Now().UnixNano()),
			Content:     msg.Content,
			Author:      "You",
			Username:    "you",
			IsOwn:       true,
			CreatedAt:   time.Now(),
			InReplyToID: msg.ParentID,
		}
		if !m.showDetail {
			return m, nil
		}
		threadID := m.currentThreadRootID()
		if threadID == "" || !m.belongsToCurrentThread(msg.ParentID) {
			return m, nil
		}
		m.replyAll = append(m.replyAll, reply)
		m.replyVisible = len(m.replyAll)
		m.replies = m.replyAll
		m.hasMoreReplies = false
		if data, ok := m.threadCache[threadID]; ok {
			data.Descendants = append(data.Descendants, reply)
			m.threadCache[threadID] = data
		}
		return m, nil

	case LikeRantMsg:
		m.applyLikeToggle(msg.ID)
		m.toggleLikeInThreadCache(msg.ID)
		return m, nil

	case LikeResultMsg:
		if msg.Err != nil {
			// Rollback by toggling again.
			m.applyLikeToggle(msg.ID)
			m.toggleLikeInThreadCache(msg.ID)
		}
		return m, nil

	case UpdateOptimisticRantMsg:
		for i, ri := range m.rants {
			if ri.Rant.ID == msg.ID {
				ri.OldContent = ri.Rant.Content
				ri.Rant.Content = msg.Content
				ri.Status = StatusPendingUpdate
				m.rants[i] = ri
				break
			}
		}
		return m, nil

	case ResultMsg:
		if msg.Err != nil {
			// Find the item and set to Failed.
			for i, ri := range m.rants {
				if ri.Rant.ID == msg.ID {
					ri.Status = StatusFailed
					ri.Err = msg.Err
					if msg.IsEdit {
						ri.Rant.Content = ri.OldContent // Rollback
					}
					m.rants[i] = ri
					break
				}
			}
		} else {
			if msg.Rant.InReplyToID != "" && msg.Rant.InReplyToID != "<nil>" && msg.Rant.InReplyToID != "0" {
				m.reconcileReplyResult(msg.ID, msg.Rant)
				return m, nil
			}
			// Success: replace optimistic item with server version.
			for i, ri := range m.rants {
				// Match by ID OR fuzzy content (for new posts)
				if ri.Rant.ID == msg.ID || (!msg.IsEdit && (strings.Contains(msg.Rant.Content, ri.Rant.Content) || strings.Contains(ri.Rant.Content, msg.Rant.Content))) {
					ri.Rant = msg.Rant
					ri.Status = StatusNormal
					m.rants[i] = ri
					break
				}
			}
			// Also check replies
			for i, r := range m.replies {
				if r.ID == msg.ID {
					m.replies[i] = msg.Rant
					break
				}
			}
		}
		return m, nil

	case DeleteResultMsg:
		if msg.Err != nil {
			for i, ri := range m.rants {
				if ri.Rant.ID == msg.ID {
					ri.Status = StatusFailed
					ri.Err = msg.Err
					m.rants[i] = ri
					break
				}
			}
		} else {
			// Success: remove from list.
			for i, ri := range m.rants {
				if ri.Rant.ID == msg.ID {
					m.rants = append(m.rants[:i], m.rants[i+1:]...)
					if m.cursor >= len(m.rants) && m.cursor > 0 {
						m.cursor--
					}
					break
				}
			}
		}
		return m, nil

	}

	return m, nil
}
