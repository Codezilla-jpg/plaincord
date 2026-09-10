package ui

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/Codezilla-jpg/plaincord/internal/auth"
	"github.com/Codezilla-jpg/plaincord/internal/chantree"
	"github.com/Codezilla-jpg/plaincord/internal/formatters"
	"github.com/Codezilla-jpg/plaincord/internal/gateway"
	"github.com/Codezilla-jpg/plaincord/internal/model"
	"github.com/Codezilla-jpg/plaincord/internal/voice"
)

type App struct {
	demo     bool
	token    string
	tv       *tview.Application
	pages    *tview.Pages
	guilds   *tview.List
	channels *tview.TreeView
	chat     *tview.TextView
	title    *tview.TextView
	composer *tview.InputField
	voiceBar *tview.TextView
	status   *tview.TextView

	gw      gateway.Gateway
	voice   voice.Controller
	current *model.Guild
	channel *model.Channel
	seen    map[string]struct{}
	cursor  *model.TreeNode
}

func Run(demo bool, token string) error {
	a := &App{
		demo:  demo,
		token: token,
		seen:  map[string]struct{}{},
	}
	return a.run()
}

func (a *App) OnReady() {
	a.tv.QueueUpdateDraw(func() {
		a.refreshGuilds()
		a.title.SetText("select a channel")
		a.flash("connected")
	})
}

func (a *App) OnMessage(msg model.ChatMessage) {
	a.tv.QueueUpdateDraw(func() {
		a.writeIncoming(msg)
	})
}

func (a *App) OnError(text string) {
	a.tv.QueueUpdateDraw(func() {
		a.flash(text)
		a.title.SetText(text)
	})
}

func (a *App) run() error {
	a.tv = tview.NewApplication()
	a.pages = tview.NewPages()
	a.buildMain()
	a.pages.AddPage("main", a.mainLayout(), true, true)
	a.pages.AddPage("login", a.loginForm(), true, false)

	if a.demo {
		a.gw = gateway.NewFake(a)
	} else {
		if a.token == "" {
			tok, err := auth.LoadToken()
			if err != nil {
				return err
			}
			a.token = tok
		}
		if a.token == "" {
			a.pages.SwitchToPage("login")
		} else {
			a.gw = gateway.NewDiscord(a.token, a)
		}
	}
	if a.gw != nil {
		go a.startGateway()
	}
	a.tv.SetRoot(a.pages, true).SetFocus(a.guilds)
	a.tv.SetInputCapture(a.keys)
	return a.tv.Run()
}

func (a *App) startGateway() {
	if err := a.gw.Start(); err != nil {
		a.OnError(err.Error())
	}
}

func (a *App) buildMain() {
	a.guilds = tview.NewList().ShowSecondaryText(false)
	a.guilds.SetBorder(true).SetTitle(" servers ")
	a.guilds.SetSelectedFunc(func(index int, _ string, _ string, _ rune) {
		a.openGuildIndex(index)
	})

	a.channels = tview.NewTreeView()
	a.channels.SetBorder(true).SetTitle(" channels ")
	a.channels.SetSelectedFunc(func(node *tview.TreeNode) {
		ref, _ := node.GetReference().(*model.TreeNode)
		if ref == nil || ref.Channel == nil {
			return
		}
		switch ref.Kind {
		case model.KindText:
			a.openText(*ref.Channel)
		case model.KindVoice:
			a.join(*ref.Channel)
		}
	})
	a.channels.SetChangedFunc(func(node *tview.TreeNode) {
		ref, _ := node.GetReference().(*model.TreeNode)
		a.cursor = ref
	})

	a.title = tview.NewTextView().SetText("connecting…")
	a.title.SetBorder(true)
	a.chat = tview.NewTextView().SetDynamicColors(true).SetScrollable(true)
	a.chat.SetBorder(true)
	a.composer = tview.NewInputField().SetLabel("> ").SetFieldBackgroundColor(tcell.ColorBlack)
	a.composer.SetDoneFunc(func(key tcell.Key) {
		if key != tcell.KeyEnter {
			return
		}
		a.send(a.composer.GetText())
		a.composer.SetText("")
	})
	a.voiceBar = tview.NewTextView().SetText(formatters.VoiceBar(false, false, ""))
	a.status = tview.NewTextView().SetText("r reload  j join  l leave  m mute  a add  tab focus  ctrl-c quit")
}

func (a *App) mainLayout() tview.Primitive {
	chat := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.title, 3, 0, false).
		AddItem(a.chat, 0, 1, false).
		AddItem(a.composer, 1, 0, true)
	body := tview.NewFlex().
		AddItem(a.guilds, 22, 0, true).
		AddItem(a.channels, 28, 0, false).
		AddItem(chat, 0, 1, false)
	return tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(body, 0, 1, true).
		AddItem(a.voiceBar, 1, 0, false).
		AddItem(a.status, 1, 0, false)
}

func (a *App) loginForm() tview.Primitive {
	form := tview.NewForm()
	form.SetBorder(true).SetTitle(" DiscordCli  login ")
	form.AddPasswordField("Bot token", "", 60, '*', nil)
	form.AddButton("Save", func() {
		item := form.GetFormItem(0).(*tview.InputField)
		tok := item.GetText()
		if err := auth.SaveToken(tok); err != nil {
			a.flash(err.Error())
			return
		}
		a.token = tok
		a.gw = gateway.NewDiscord(a.token, a)
		a.pages.SwitchToPage("main")
		a.tv.SetFocus(a.guilds)
		go a.startGateway()
	})
	form.AddButton("Quit", func() { a.tv.Stop() })
	hint := tview.NewTextView().SetText("Create a bot at discord.com/developers — enable Message Content Intent.")
	return tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(hint, 2, 0, false).
		AddItem(form, 10, 0, true).
		AddItem(nil, 0, 1, false)
}

func (a *App) keys(ev *tcell.EventKey) *tcell.EventKey {
	if a.tv.GetFocus() == a.composer && ev.Key() == tcell.KeyRune {
		return ev
	}
	switch ev.Key() {
	case tcell.KeyCtrlC, tcell.KeyCtrlQ:
		if a.gw != nil {
			_ = a.gw.Close()
		}
		a.tv.Stop()
		return nil
	case tcell.KeyTab:
		a.cycleFocus()
		return nil
	case tcell.KeyEsc:
		a.tv.SetFocus(a.channels)
		return nil
	}
	if a.tv.GetFocus() == a.composer {
		return ev
	}
	switch ev.Rune() {
	case 'r', 'R':
		a.reload()
		return nil
	case 'm', 'M':
		a.toggleMute()
		return nil
	case 'j', 'J':
		a.joinCursor()
		return nil
	case 'l', 'L':
		a.leave()
		return nil
	case 'a', 'A':
		a.addServer()
		return nil
	case 'q':
		if a.gw != nil {
			_ = a.gw.Close()
		}
		a.tv.Stop()
		return nil
	}
	return ev
}

func (a *App) cycleFocus() {
	switch a.tv.GetFocus() {
	case a.guilds:
		a.tv.SetFocus(a.channels)
	case a.channels:
		a.tv.SetFocus(a.composer)
	default:
		a.tv.SetFocus(a.guilds)
	}
}

func (a *App) refreshGuilds() {
	if a.gw == nil {
		return
	}
	a.guilds.Clear()
	guilds := a.gw.Guilds()
	for _, g := range guilds {
		g := g
		a.guilds.AddItem(g.Name, g.ID, 0, nil)
	}
	if len(guilds) > 0 {
		a.guilds.SetCurrentItem(0)
		a.openGuild(guilds[0])
	}
}

func (a *App) openGuildIndex(index int) {
	if a.gw == nil {
		return
	}
	guilds := a.gw.Guilds()
	if index < 0 || index >= len(guilds) {
		return
	}
	a.openGuild(guilds[index])
}

func (a *App) openGuild(g model.Guild) {
	if a.gw == nil {
		return
	}
	a.current = &g
	a.channel = nil
	root := tview.NewTreeNode(g.Name).SetSelectable(false)
	for _, node := range chantree.Build(a.gw.Channels(g.ID)) {
		root.AddChild(treeNode(node))
	}
	a.channels.SetRoot(root).SetCurrentNode(root)
	a.title.SetText(g.Name)
	a.chat.Clear()
}

func treeNode(n model.TreeNode) *tview.TreeNode {
	cp := n
	node := tview.NewTreeNode(formatters.ChannelLabel(string(n.Kind), n.Name)).
		SetReference(&cp).
		SetSelectable(n.Kind != model.KindCategory).
		SetExpanded(true)
	for _, child := range n.Children {
		node.AddChild(treeNode(child))
	}
	return node
}

func (a *App) openText(ch model.Channel) {
	if a.gw == nil {
		return
	}
	a.channel = &ch
	a.seen = map[string]struct{}{}
	a.title.SetText(formatters.ChannelLabel("text", ch.Name))
	a.chat.Clear()
	hist, err := a.gw.History(ch.ID, 80)
	if err != nil {
		a.flash(err.Error())
		return
	}
	for _, msg := range hist {
		a.writeIncoming(msg)
	}
	a.tv.SetFocus(a.composer)
}

func (a *App) writeIncoming(msg model.ChatMessage) {
	if _, ok := a.seen[msg.ID]; ok {
		return
	}
	if a.channel == nil || msg.ChannelID != a.channel.ID {
		return
	}
	a.seen[msg.ID] = struct{}{}
	fmt.Fprintln(a.chat, formatters.MessageLine(msg.Author, msg.Content, msg.Timestamp))
	a.chat.ScrollToEnd()
}

func (a *App) send(text string) {
	if text == "" || a.gw == nil || a.channel == nil || a.channel.Kind != model.KindText {
		return
	}
	msg, err := a.gw.Send(a.channel.ID, text)
	if err != nil {
		a.flash(err.Error())
		return
	}
	a.writeIncoming(msg)
}

func (a *App) join(ch model.Channel) {
	if a.gw == nil || a.current == nil {
		return
	}
	if err := a.gw.JoinVoice(a.current.ID, ch.ID, ch.Name); err != nil {
		a.flash(err.Error())
		return
	}
	a.voice.OnJoined(a.current.ID, ch.ID, ch.Name)
	a.voiceBar.SetText(formatters.VoiceBar(true, false, ch.Name))
	a.flash("joined " + ch.Name)
}

func (a *App) joinCursor() {
	if a.cursor == nil || a.cursor.Channel == nil || a.cursor.Kind != model.KindVoice {
		a.flash("select a voice channel")
		return
	}
	a.join(*a.cursor.Channel)
}

func (a *App) leave() {
	if a.gw == nil {
		return
	}
	if err := a.gw.LeaveVoice(); err != nil {
		a.flash(err.Error())
		return
	}
	a.voice.OnLeft()
	a.voiceBar.SetText(formatters.VoiceBar(false, false, ""))
	a.flash("left call")
}

func (a *App) toggleMute() {
	muted, err := a.voice.ToggleMute()
	if err != nil {
		a.flash(err.Error())
		return
	}
	if a.gw != nil {
		if err := a.gw.SetMute(muted); err != nil {
			a.flash(err.Error())
			return
		}
	}
	a.voiceBar.SetText(formatters.VoiceBar(true, muted, a.voice.State.ChannelName))
}

func (a *App) reload() {
	if a.channel != nil && a.channel.Kind == model.KindText {
		a.openText(*a.channel)
		a.flash("chat reloaded")
		return
	}
	a.refreshGuilds()
	a.flash("servers reloaded")
}

func (a *App) addServer() {
	id := ""
	if a.gw != nil {
		id = a.gw.ApplicationID()
	}
	if id == "" {
		id, _ = auth.LoadApplicationID()
	}
	text := "Run once connected, then press a again."
	if id != "" {
		text = "Open this URL, pick a server, then press r:\n\n" + chantree.InviteURL(id)
	}
	modal := tview.NewModal().SetText(text).AddButtons([]string{"Close"}).
		SetDoneFunc(func(_ int, _ string) {
			a.pages.RemovePage("invite")
			a.tv.SetFocus(a.guilds)
		})
	a.pages.AddPage("invite", modal, true, true)
}

func (a *App) flash(msg string) {
	a.status.SetText(msg + "    r reload  j join  l leave  m mute  a add  ctrl-c quit")
}
