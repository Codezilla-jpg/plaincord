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

type navLevel int

const (
	navServers navLevel = iota
	navChannels
	navChat
)

type channelRow struct {
	label string
	cat   bool
	ch    *model.Channel
}

var (
	colBg     = tcell.NewRGBColor(7, 8, 16)
	colPanel  = tcell.NewRGBColor(14, 16, 32)
	colAccent = tcell.NewRGBColor(108, 92, 231)
	colHot    = tcell.NewRGBColor(255, 215, 64)
	colVoice  = tcell.NewRGBColor(0, 229, 168)
	colText   = tcell.NewRGBColor(248, 249, 255)
	colMuted  = tcell.NewRGBColor(140, 146, 180)
	colSelect = tcell.NewRGBColor(108, 92, 231)
)

type App struct {
	demo     bool
	token    string
	tv       *tview.Application
	pages    *tview.Pages
	stage    *tview.Pages
	nav      *tview.List
	crumb    *tview.TextView
	chat     *tview.TextView
	title    *tview.TextView
	composer *tview.InputField
	voiceBar *tview.TextView
	status   *tview.TextView

	gw      gateway.Gateway
	voice   voice.Controller
	level   navLevel
	guilds  []model.Guild
	current *model.Guild
	channel *model.Channel
	rows    []channelRow
	seen    map[string]struct{}
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
		a.fillServers()
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
	})
}

func applyTheme() {
	tview.Styles.PrimitiveBackgroundColor = colBg
	tview.Styles.ContrastBackgroundColor = colAccent
	tview.Styles.MoreContrastBackgroundColor = colHot
	tview.Styles.BorderColor = colAccent
	tview.Styles.TitleColor = colHot
	tview.Styles.GraphicsColor = colAccent
	tview.Styles.PrimaryTextColor = colText
	tview.Styles.SecondaryTextColor = colMuted
	tview.Styles.TertiaryTextColor = colHot
	tview.Styles.InverseTextColor = colBg
	tview.Styles.ContrastSecondaryTextColor = colText
}

func (a *App) run() error {
	applyTheme()
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
	a.tv.SetRoot(a.pages, true).SetFocus(a.nav)
	a.tv.SetInputCapture(a.keys)
	return a.tv.Run()
}

func (a *App) startGateway() {
	if err := a.gw.Start(); err != nil {
		a.OnError(err.Error())
	}
}

func (a *App) buildMain() {
	a.crumb = tview.NewTextView().SetDynamicColors(true)
	a.crumb.SetBackgroundColor(colAccent)
	a.crumb.SetTextColor(colText)
	a.setCrumb()

	a.nav = tview.NewList().
		ShowSecondaryText(false).
		SetHighlightFullLine(true).
		SetWrapAround(true)
	a.nav.SetBorder(true).SetTitle(" explore ")
	a.nav.SetBackgroundColor(colPanel)
	a.nav.SetBorderColor(colAccent)
	a.nav.SetTitleColor(colHot)
	a.nav.SetMainTextColor(colText)
	a.nav.SetSelectedBackgroundColor(colSelect)
	a.nav.SetSelectedTextColor(colText)
	a.nav.SetSelectedFunc(func(index int, _ string, _ string, _ rune) {
		a.activate(index)
	})

	a.title = tview.NewTextView().SetDynamicColors(true)
	a.title.SetBorder(true).SetTitle(" chat ")
	a.title.SetBackgroundColor(colPanel)
	a.title.SetBorderColor(colHot)
	a.title.SetTitleColor(colHot)
	a.title.SetTextColor(colText)

	a.chat = tview.NewTextView().SetDynamicColors(true).SetScrollable(true)
	a.chat.SetBorder(true)
	a.chat.SetBackgroundColor(colBg)
	a.chat.SetBorderColor(colAccent)
	a.chat.SetTextColor(colText)

	a.composer = tview.NewInputField().SetLabel("  ▸  ").SetFieldWidth(0)
	a.composer.SetLabelColor(colHot)
	a.composer.SetFieldBackgroundColor(colPanel)
	a.composer.SetFieldTextColor(colText)
	a.composer.SetDoneFunc(func(key tcell.Key) {
		if key != tcell.KeyEnter {
			return
		}
		a.send(a.composer.GetText())
		a.composer.SetText("")
	})

	a.voiceBar = tview.NewTextView().SetDynamicColors(true)
	a.voiceBar.SetBackgroundColor(colPanel)
	a.voiceBar.SetTextColor(colVoice)
	a.paintVoice()

	a.status = tview.NewTextView().SetDynamicColors(true)
	a.status.SetBackgroundColor(colAccent)
	a.status.SetTextColor(colText)
	a.flash("enter open   esc back")

	chat := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.title, 3, 0, false).
		AddItem(a.chat, 0, 1, false).
		AddItem(a.composer, 1, 0, true)

	a.stage = tview.NewPages()
	a.stage.AddPage("browse", a.nav, true, true)
	a.stage.AddPage("chat", chat, true, false)
}

func (a *App) mainLayout() tview.Primitive {
	return tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.crumb, 1, 0, false).
		AddItem(a.stage, 0, 1, true).
		AddItem(a.voiceBar, 1, 0, false).
		AddItem(a.status, 1, 0, false)
}

func (a *App) loginForm() tview.Primitive {
	form := tview.NewForm()
	form.SetBorder(true).SetTitle(" DIS  login ")
	form.SetBackgroundColor(colPanel)
	form.SetBorderColor(colHot)
	form.SetTitleColor(colHot)
	form.SetFieldBackgroundColor(colBg)
	form.SetFieldTextColor(colText)
	form.SetButtonBackgroundColor(colAccent)
	form.SetButtonTextColor(colText)
	form.AddPasswordField("Account token", "", 60, '*', nil)
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
		a.tv.SetFocus(a.nav)
		go a.startGateway()
	})
	form.AddButton("Quit", func() { a.tv.Stop() })
	hint := tview.NewTextView().SetDynamicColors(true).
		SetText("[#FFD93D]Paste your Discord account token.[-] Unofficial clients can get accounts banned.")
	hint.SetBackgroundColor(colBg)
	return tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(hint, 2, 0, false).
		AddItem(form, 12, 0, true).
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
	case tcell.KeyEsc, tcell.KeyBackspace, tcell.KeyBackspace2, tcell.KeyLeft:
		if a.tv.GetFocus() == a.composer && ev.Key() == tcell.KeyEsc {
			a.back()
			return nil
		}
		if a.tv.GetFocus() == a.composer {
			return ev
		}
		a.back()
		return nil
	case tcell.KeyRight:
		if a.tv.GetFocus() == a.nav {
			a.activate(a.nav.GetCurrentItem())
			return nil
		}
	case tcell.KeyTab:
		if a.level == navChat {
			if a.tv.GetFocus() == a.composer {
				a.tv.SetFocus(a.chat)
			} else {
				a.tv.SetFocus(a.composer)
			}
			return nil
		}
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

func (a *App) activate(index int) {
	switch a.level {
	case navServers:
		if index < 0 || index >= len(a.guilds) {
			return
		}
		a.openGuild(a.guilds[index])
	case navChannels:
		if index < 0 || index >= len(a.rows) {
			return
		}
		row := a.rows[index]
		if row.cat || row.ch == nil {
			return
		}
		switch row.ch.Kind {
		case model.KindText:
			a.openText(*row.ch)
		case model.KindVoice:
			a.join(*row.ch)
		}
	}
}

func (a *App) back() {
	switch a.level {
	case navChat:
		a.level = navChannels
		a.channel = nil
		a.stage.SwitchToPage("browse")
		a.tv.SetFocus(a.nav)
		a.setCrumb()
		a.flash("channels")
	case navChannels:
		a.fillServers()
		a.flash("servers")
	}
}

func (a *App) fillServers() {
	if a.gw == nil {
		return
	}
	a.level = navServers
	a.current = nil
	a.channel = nil
	a.guilds = a.gw.Guilds()
	a.nav.Clear()
	a.nav.SetTitle(" servers ")
	for _, g := range a.guilds {
		a.nav.AddItem("▸  "+g.Name, "", 0, nil)
	}
	a.stage.SwitchToPage("browse")
	a.tv.SetFocus(a.nav)
	a.setCrumb()
}

func (a *App) openGuild(g model.Guild) {
	if a.gw == nil {
		return
	}
	a.current = &g
	a.channel = nil
	a.level = navChannels
	a.rows = nil
	a.nav.Clear()
	a.nav.SetTitle(" channels ")
	for _, node := range chantree.Build(a.gw.Channels(g.ID)) {
		a.addChannelNode(node)
	}
	a.stage.SwitchToPage("browse")
	a.tv.SetFocus(a.nav)
	a.setCrumb()
	a.flash("enter chat or voice   esc back")
}

func (a *App) addChannelNode(n model.TreeNode) {
	if n.Kind == model.KindCategory {
		a.rows = append(a.rows, channelRow{label: n.Name, cat: true})
		a.nav.AddItem("——  "+n.Name+"  ——", "", 0, nil)
		for _, child := range n.Children {
			a.addChannelNode(child)
		}
		return
	}
	if n.Channel == nil {
		return
	}
	ch := *n.Channel
	label := formatters.ChannelLabel(string(ch.Kind), ch.Name)
	a.rows = append(a.rows, channelRow{label: label, ch: &ch})
	a.nav.AddItem("   "+label, "", 0, nil)
}

func (a *App) openText(ch model.Channel) {
	if a.gw == nil {
		return
	}
	a.channel = &ch
	a.level = navChat
	a.seen = map[string]struct{}{}
	a.title.SetText("  " + formatters.ChannelLabel("text", ch.Name))
	a.chat.Clear()
	hist, err := a.gw.History(ch.ID, 80)
	if err != nil {
		a.flash(err.Error())
		return
	}
	for _, msg := range hist {
		a.writeIncoming(msg)
	}
	a.stage.SwitchToPage("chat")
	a.tv.SetFocus(a.composer)
	a.setCrumb()
	a.flash("type to send   esc back")
}

func (a *App) writeIncoming(msg model.ChatMessage) {
	if _, ok := a.seen[msg.ID]; ok {
		return
	}
	if a.channel == nil || msg.ChannelID != a.channel.ID {
		return
	}
	a.seen[msg.ID] = struct{}{}
	fmt.Fprintf(a.chat, "[#FFD740]%s[-] [#A29BFE]%s[-]  %s\n",
		formatters.Clock(msg.Timestamp), msg.Author, msg.Content)
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
	a.paintVoice()
	a.flash("joined " + ch.Name + "   m mute   l leave")
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
	a.paintVoice()
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
	a.paintVoice()
}

func (a *App) paintVoice() {
	st := a.voice.State
	if st.Connected {
		a.voiceBar.SetBackgroundColor(tcell.NewRGBColor(8, 48, 40))
		a.voiceBar.SetTextColor(colVoice)
	} else {
		a.voiceBar.SetBackgroundColor(colPanel)
		a.voiceBar.SetTextColor(colMuted)
	}
	a.voiceBar.SetText("  " + formatters.VoiceBar(st.Connected, st.Muted, st.ChannelName))
}

func (a *App) setCrumb() {
	head := "[#FFE082] DIS [-]"
	switch a.level {
	case navServers:
		a.crumb.SetText(head + "  [#C5CAE9]servers[-]")
	case navChannels:
		name := ""
		if a.current != nil {
			name = a.current.Name
		}
		a.crumb.SetText(head + "  [#C5CAE9]servers[-]  [#FFE082]›[-]  [#FFFFFF]" + name + "[-]")
	case navChat:
		g, c := "", ""
		if a.current != nil {
			g = a.current.Name
		}
		if a.channel != nil {
			c = formatters.ChannelLabel("text", a.channel.Name)
		}
		a.crumb.SetText(head + "  [#C5CAE9]servers[-]  [#FFE082]›[-]  " + g + "  [#FFE082]›[-]  [#FFFFFF]" + c + "[-]")
	}
}

func (a *App) reload() {
	switch a.level {
	case navChat:
		if a.channel != nil {
			a.openText(*a.channel)
			a.flash("chat reloaded")
		}
	case navChannels:
		if a.current != nil {
			a.openGuild(*a.current)
			a.flash("channels reloaded")
		}
	default:
		a.fillServers()
		a.flash("servers reloaded")
	}
}

func (a *App) addServer() {
	if a.gw == nil {
		a.flash("not connected")
		return
	}
	form := tview.NewForm()
	form.SetBorder(true).SetTitle(" join server ")
	form.SetBackgroundColor(colPanel)
	form.SetBorderColor(colHot)
	form.SetTitleColor(colHot)
	form.SetFieldBackgroundColor(colBg)
	form.SetButtonBackgroundColor(colAccent)
	form.AddInputField("Invite", "", 60, nil, nil)
	form.AddButton("Join", func() {
		raw := form.GetFormItem(0).(*tview.InputField).GetText()
		if err := a.gw.JoinInvite(raw); err != nil {
			a.flash(err.Error())
			return
		}
		a.pages.RemovePage("invite")
		a.fillServers()
		a.flash("joined server")
	})
	form.AddButton("Cancel", func() {
		a.pages.RemovePage("invite")
		a.tv.SetFocus(a.nav)
	})
	a.pages.AddPage("invite", form, true, true)
	a.tv.SetFocus(form)
}

func (a *App) flash(msg string) {
	a.status.SetText("  " + msg + "     enter open   esc back   r reload   m mute   l leave   a add")
}
