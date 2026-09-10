package chantree

import (
	"fmt"
	"sort"

	"github.com/Codezilla-jpg/plaincord/internal/model"
)

const BotPermissions = (1 << 10) | (1 << 11) | (1 << 16) | (1 << 20) | (1 << 21) | (1 << 25)

func InviteURL(clientID string) string {
	return fmt.Sprintf(
		"https://discord.com/oauth2/authorize?client_id=%s&permissions=%d&integration_type=0&scope=bot",
		clientID,
		BotPermissions,
	)
}

func Build(channels []model.Channel) []model.TreeNode {
	var categories []model.Channel
	byCategory := map[string][]model.Channel{}
	for _, ch := range channels {
		if ch.Kind == model.KindCategory {
			categories = append(categories, ch)
			continue
		}
		byCategory[ch.CategoryID] = append(byCategory[ch.CategoryID], ch)
	}
	sort.Slice(categories, func(i, j int) bool {
		if categories[i].Position != categories[j].Position {
			return categories[i].Position < categories[j].Position
		}
		return categories[i].ID < categories[j].ID
	})
	for id, group := range byCategory {
		sort.Slice(group, func(i, j int) bool {
			ki, kj := 1, 1
			if group[i].Kind == model.KindText {
				ki = 0
			}
			if group[j].Kind == model.KindText {
				kj = 0
			}
			if ki != kj {
				return ki < kj
			}
			if group[i].Position != group[j].Position {
				return group[i].Position < group[j].Position
			}
			return group[i].ID < group[j].ID
		})
		byCategory[id] = group
	}
	var nodes []model.TreeNode
	for _, ch := range byCategory[""] {
		nodes = append(nodes, leaf(ch))
	}
	for _, cat := range categories {
		children := make([]model.TreeNode, 0, len(byCategory[cat.ID]))
		for _, ch := range byCategory[cat.ID] {
			children = append(children, leaf(ch))
		}
		nodes = append(nodes, model.TreeNode{
			ID:       cat.ID,
			Name:     cat.Name,
			Kind:     model.KindCategory,
			Children: children,
			Channel:  cloneChannel(cat),
		})
	}
	return nodes
}

func leaf(ch model.Channel) model.TreeNode {
	return model.TreeNode{
		ID:      ch.ID,
		Name:    ch.Name,
		Kind:    ch.Kind,
		Channel: cloneChannel(ch),
	}
}

func cloneChannel(ch model.Channel) *model.Channel {
	cp := ch
	return &cp
}
