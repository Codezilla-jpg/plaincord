package chantree

import (
	"strings"
	"testing"

	"github.com/Codezilla-jpg/plaincord/internal/model"
)

func TestInviteURL(t *testing.T) {
	url := InviteURL("123456789")
	if !strings.HasPrefix(url, "https://discord.com/oauth2/authorize?") {
		t.Fatalf("prefix: %s", url)
	}
	if !strings.Contains(url, "client_id=123456789") {
		t.Fatal("client_id")
	}
	if !strings.Contains(url, "scope=bot") {
		t.Fatal("scope")
	}
	if !strings.Contains(url, "permissions=36768768") && !strings.Contains(url, "permissions=") {
		t.Fatal("permissions")
	}
}

func TestBuildGroupsAndSorts(t *testing.T) {
	channels := []model.Channel{
		{ID: "2", Name: "Voice", Kind: model.KindCategory, Position: 2},
		{ID: "1", Name: "Text", Kind: model.KindCategory, Position: 1},
		{ID: "30", Name: "lounge", Kind: model.KindVoice, CategoryID: "2", Position: 0},
		{ID: "20", Name: "random", Kind: model.KindText, CategoryID: "1", Position: 1},
		{ID: "10", Name: "general", Kind: model.KindText, CategoryID: "1", Position: 0},
		{ID: "5", Name: "welcome", Kind: model.KindText, Position: 0},
		{ID: "31", Name: "afk", Kind: model.KindVoice, CategoryID: "2", Position: 1},
	}
	tree := Build(channels)
	if len(tree) != 3 {
		t.Fatalf("len %d", len(tree))
	}
	if tree[0].Name != "welcome" || tree[1].Name != "Text" || tree[2].Name != "Voice" {
		t.Fatalf("names %s %s %s", tree[0].Name, tree[1].Name, tree[2].Name)
	}
	if tree[0].Kind != model.KindText {
		t.Fatal("welcome kind")
	}
	if got := names(tree[1].Children); got != "general,random" {
		t.Fatalf("text children %s", got)
	}
	if got := names(tree[2].Children); got != "lounge,afk" {
		t.Fatalf("voice children %s", got)
	}
	for _, child := range tree[2].Children {
		if child.Kind != model.KindVoice {
			t.Fatal("expected voice")
		}
	}
}

func names(nodes []model.TreeNode) string {
	out := make([]string, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, n.Name)
	}
	return strings.Join(out, ",")
}
