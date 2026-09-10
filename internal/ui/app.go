package ui

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/Codezilla-jpg/plaincord/internal/audio"
	"github.com/Codezilla-jpg/plaincord/internal/auth"
	"github.com/Codezilla-jpg/plaincord/internal/formatters"
	"github.com/Codezilla-jpg/plaincord/internal/gateway"
	"github.com/Codezilla-jpg/plaincord/internal/model"
	"github.com/Codezilla-jpg/plaincord/internal/nav"
	"github.com/Codezilla-jpg/plaincord/internal/voice"
)

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
	demo  bool
	token string
	tv    *tview.Application
	pages *tview.Pages
	body  *tview.Flex

	friends  *tview.TextView
	servers  *tview.List
	channels *tview.List
	chat     *tview.TextView
	title    *tview.TextView
	composer *tview.InputField
	callHead *tview.TextView
	people   *tview.TextView
	crumb    *tview.TextView
	voiceBar *tview.TextView
	status   *tview.TextView

	gw      gateway.Gateway
	voice   voice.Controller
	engine  nav.State
	rail    []nav.Server
	rows    []nav.Row
	guilds  []model.Guild
	current *model.Guild
	channel *model.Channel
	seen    map[string]struct{}
	devices audio.Info
	mic     *audio.Session
	frame   int
	talking atomic.Bool
	anim    atomic.Uint32
	filling bool
	shell   *tview.Flex
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
	a.devices = audio.Probe()
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
	a.tv.SetRoot(a.pages, true)
	a.focusServers()
	a.tv.SetInputCapture(a.keys)
	defer a.stopAudio()
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

	a.callHead = tview.NewTextView().SetDynamicColors(true)
	a.callHead.SetBackgroundColor(tcell.NewRGBColor(8, 48, 40))
	a.callHead.SetTextColor(colVoice)

	a.friends = tview.NewTextView().SetDynamicColors(true).SetTextAlign(tview.AlignLeft)
	a.friends.SetBorder(true).SetTitle(" pin ")
	a.friends.SetBackgroundColor(colPanel)
	a.friends.SetBorderColor(colHot)
	a.friends.SetTitleColor(colHot)
	a.friends.SetTextColor(colText)
	a.friends.SetText("  ★  Friends")

	a.servers = tview.NewList().ShowSecondaryText(false).SetHighlightFullLine(true).SetWrapAround(true)
	a.servers.SetBorder(true).SetTitle(" servers ")
	a.servers.SetBackgroundColor(colPanel)
	a.servers.SetBorderColor(colAccent)
	a.servers.SetTitleColor(colHot)
	a.servers.SetMainTextColor(colText)
	a.servers.SetSelectedBackgroundColor(colSelect)
	a.servers.SetSelectedTextColor(colText)
	a.servers.SetChangedFunc(func(index int, _ string, _ string, _ rune) {
		if a.filling || a.engine.Column != nav.ColServers {
			return
		}
		if a.engine.ServerIdx == index+1 {
			return
		}
		a.engine.ServerIdx = index + 1
		a.preview()
	})

	a.channels = tview.NewList().ShowSecondaryText(false).SetHighlightFullLine(true).SetWrapAround(true)
	a.channels.SetBorder(true).SetTitle(" channels ")
	a.channels.SetBackgroundColor(colPanel)
	a.channels.SetBorderColor(colAccent)
	a.channels.SetTitleColor(colHot)
	a.channels.SetMainTextColor(colText)
	a.channels.SetSelectedBackgroundColor(colSelect)
	a.channels.SetSelectedTextColor(colText)
	a.channels.SetChangedFunc(func(index int, _ string, _ string, _ rune) {
		if a.filling || a.engine.Column != nav.ColChannels {
			return
		}
		if index >= 0 && index < len(a.rows) {
			a.engine.ChannelIdx = index
		}
	})

	a.title = tview.NewTextView().SetDynamicColors(true)
	a.title.SetBorder(true).SetTitle(" chat ")
	a.title.SetBackgroundColor(colPanel)
	a.title.SetBorderColor(colHot)
	a.title.SetTitleColor(colHot)
	a.title.SetTextColor(colText)
	a.title.SetText("  →  channel")

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

	a.people = tview.NewTextView().SetDynamicColors(true)
	a.people.SetBorder(true).SetTitle(" in call ")
	a.people.SetBackgroundColor(colPanel)
	a.people.SetBorderColor(colVoice)
	a.people.SetTitleColor(colVoice)
	a.people.SetTextColor(colText)

	a.voiceBar = tview.NewTextView().SetDynamicColors(true)
	a.voiceBar.SetBackgroundColor(colPanel)
	a.voiceBar.SetTextColor(colVoice)

	a.status = tview.NewTextView().SetDynamicColors(true)
	a.status.SetBackgroundColor(colAccent)
	a.status.SetTextColor(colText)

	a.paintVoice()
	a.setCrumb()
	a.flash("↑↓ server   → channel")
}

func (a *App) mainLayout() tview.Primitive {
	rail := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.friends, 3, 0, false).
		AddItem(a.servers, 0, 1, true)

	chat := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.title, 3, 0, false).
		AddItem(a.chat, 0, 1, false).
		AddItem(a.composer, 1, 0, true)

	a.body = tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(rail, 22, 0, true).
		AddItem(a.channels, 28, 0, false).
		AddItem(chat, 0, 1, false).
		AddItem(a.people, 0, 0, false)

	a.shell = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.crumb, 1, 0, false).
		AddItem(a.callHead, 0, 0, false).
		AddItem(a.body, 0, 1, true).
		AddItem(a.voiceBar, 1, 0, false).
		AddItem(a.status, 1, 0, false)
	return a.shell
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
		a.focusServers()
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
		a.quit()
		return nil
	case tcell.KeyEsc, tcell.KeyBackspace, tcell.KeyBackspace2:
		a.apply(nav.Left)
		return nil
	case tcell.KeyLeft:
		if a.tv.GetFocus() == a.composer && a.composer.GetText() != "" {
			return ev
		}
		a.apply(nav.Left)
		return nil
	case tcell.KeyRight:
		if a.tv.GetFocus() == a.composer {
			return ev
		}
		a.apply(nav.Right)
		return nil
	case tcell.KeyUp:
		if a.engine.Column == nav.ColChat {
			return ev
		}
		a.apply(nav.Up)
		return nil
	case tcell.KeyDown:
		if a.engine.Column == nav.ColChat {
			return ev
		}
		a.apply(nav.Down)
		return nil
	case tcell.KeyEnter:
		if a.tv.GetFocus() == a.composer {
			return ev
		}
		a.apply(nav.Enter)
		return nil
	case tcell.KeyTab:
		if a.engine.Column == nav.ColChat {
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
		a.quit()
		return nil
	}
	return ev
}

func (a *App) apply(in nav.Input) {
	n := len(a.rail)
	if n == 0 {
		n = 1
	}
	next, act := nav.Handle(a.engine, in, n, a.rows)
	a.engine = next
	switch act {
	case nav.ActionPreview:
		a.preview()
		a.focusServers()
	case nav.ActionFocusChannels:
		a.previewKeepIdx()
		a.focusChannels()
	case nav.ActionOpenChat:
		a.openSelectedText()
	case nav.ActionJoinCall:
		a.joinSelected()
	case nav.ActionLeaveChat:
		a.focusChannels()
	case nav.ActionLeaveChannels:
		a.focusServers()
	}
	if a.engine.Column == nav.ColChannels && a.engine.ChannelIdx >= 0 && a.engine.ChannelIdx < a.channels.GetItemCount() {
		a.filling = true
		a.channels.SetCurrentItem(a.engine.ChannelIdx)
		a.filling = false
	}
	a.setCrumb()
	a.paintCall()
}

func (a *App) previewKeepIdx() {
	idx := a.engine.ChannelIdx
	a.reloadRows()
	if !navOpenable(a.rows, idx) {
		idx = nav.FirstOpenable(a.rows)
	}
	a.engine.ChannelIdx = idx
	a.fillChannelList()
	a.paintRail()
}

func navOpenable(rows []nav.Row, i int) bool {
	return i >= 0 && i < len(rows) && !rows[i].Cat && rows[i].Ch != nil
}

func (a *App) fillServers() {
	if a.gw == nil {
		return
	}
	a.filling = true
	a.engine.Column = nav.ColServers
	a.engine.ServerIdx = 0
	a.channel = nil
	a.guilds = a.gw.Guilds()
	a.rail = nav.Rail(a.guilds)
	a.servers.Clear()
	for _, s := range a.rail[1:] {
		a.servers.AddItem("▸  "+s.Name, "", 0, nil)
	}
	a.filling = false
	a.preview()
	a.focusServers()
	a.setCrumb()
}

func (a *App) preview() {
	a.reloadRows()
	a.engine.ChannelIdx = nav.FirstOpenable(a.rows)
	a.fillChannelList()
	a.paintRail()
}

func (a *App) reloadRows() {
	if a.gw == nil || len(a.rail) == 0 {
		return
	}
	if a.engine.ServerIdx < 0 || a.engine.ServerIdx >= len(a.rail) {
		a.engine.ServerIdx = 0
	}
	srv := a.rail[a.engine.ServerIdx]
	a.current = &model.Guild{ID: srv.ID, Name: srv.Name}
	a.rows = nav.RowsFrom(a.gw.Channels(srv.ID))
}

func (a *App) fillChannelList() {
	a.filling = true
	a.channels.Clear()
	title := " channels "
	if a.current != nil && a.current.ID == nav.FriendsID {
		title = " friends "
	}
	a.channels.SetTitle(title)
	for _, row := range a.rows {
		prefix := "   "
		if row.Cat {
			prefix = ""
		}
		a.channels.AddItem(prefix+row.Label, "", 0, nil)
	}
	if a.engine.ChannelIdx >= 0 && a.engine.ChannelIdx < a.channels.GetItemCount() {
		a.channels.SetCurrentItem(a.engine.ChannelIdx)
	}
	a.filling = false
}

func (a *App) paintRail() {
	onFriends := a.engine.ServerIdx == 0
	if onFriends {
		a.friends.SetBackgroundColor(colSelect)
		a.friends.SetTextColor(colText)
		a.friends.SetText("  ★  Friends")
		a.servers.SetSelectedBackgroundColor(colPanel)
		a.servers.SetSelectedTextColor(colMuted)
	} else {
		a.friends.SetBackgroundColor(colPanel)
		a.friends.SetTextColor(colMuted)
		a.friends.SetText("  ★  Friends")
		a.servers.SetSelectedBackgroundColor(colSelect)
		a.servers.SetSelectedTextColor(colText)
		idx := a.engine.ServerIdx - 1
		if idx >= 0 && idx < a.servers.GetItemCount() {
			a.servers.SetCurrentItem(idx)
		}
	}
}

func (a *App) focusServers() {
	a.engine.Column = nav.ColServers
	if a.engine.ServerIdx == 0 {
		a.tv.SetFocus(a.friends)
	} else {
		a.tv.SetFocus(a.servers)
	}
	a.paintRail()
}

func (a *App) focusChannels() {
	a.engine.Column = nav.ColChannels
	a.tv.SetFocus(a.channels)
	if a.engine.ChannelIdx >= 0 && a.engine.ChannelIdx < a.channels.GetItemCount() {
		a.channels.SetCurrentItem(a.engine.ChannelIdx)
	}
}

func (a *App) selectedChannel() *model.Channel {
	if !navOpenable(a.rows, a.engine.ChannelIdx) {
		return nil
	}
	return a.rows[a.engine.ChannelIdx].Ch
}

func (a *App) openSelectedText() {
	ch := a.selectedChannel()
	if ch == nil {
		return
	}
	a.openText(*ch)
}

func (a *App) openText(ch model.Channel) {
	if a.gw == nil {
		return
	}
	a.channel = &ch
	a.engine.Column = nav.ColChat
	a.engine.ChatOpen = true
	a.engine.OpenChannelID = ch.ID
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
	a.tv.SetFocus(a.composer)
	a.setCrumb()
	a.flash("type to send   ← back")
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

func (a *App) joinSelected() {
	ch := a.selectedChannel()
	if ch == nil || a.gw == nil || a.current == nil {
		return
	}
	a.join(*ch)
}

func (a *App) join(ch model.Channel) {
	if a.gw == nil || a.current == nil {
		return
	}
	if err := a.gw.JoinVoice(a.current.ID, ch.ID, ch.Name); err != nil {
		a.engine.InCall = false
		a.engine.CallGuildID = ""
		a.engine.CallChannelID = ""
		a.engine.CallName = ""
		a.flash(err.Error())
		return
	}
	a.engine.InCall = true
	a.engine.CallGuildID = a.current.ID
	a.engine.CallChannelID = ch.ID
	a.engine.CallName = ch.Name
	a.voice.OnJoined(a.current.ID, ch.ID, ch.Name)
	a.startAudio()
	a.paintCall()
	a.paintVoice()
	a.flash("call " + ch.Name + "   m mute   l leave   chat stays")
}

func (a *App) leave() {
	if a.gw == nil {
		return
	}
	if err := a.gw.LeaveVoice(); err != nil {
		a.flash(err.Error())
		return
	}
	a.engine.InCall = false
	a.engine.CallChannelID = ""
	a.engine.CallName = ""
	a.voice.OnLeft()
	a.stopAudio()
	a.paintCall()
	a.paintVoice()
	a.flash("left call")
}

func (a *App) toggleMute() {
	muted, err := a.voice.ToggleMute()
	if err != nil {
		a.flash(err.Error())
		return
	}
	if a.mic != nil {
		a.mic.SetMuted(muted)
	}
	if a.gw != nil {
		if err := a.gw.SetMute(muted); err != nil {
			a.flash(err.Error())
			return
		}
	}
	a.paintCall()
	a.paintVoice()
}

func (a *App) startAudio() {
	a.stopAudio()
	a.devices = audio.Probe()
	sess, err := audio.Start(func(rms float64) {
		a.talking.Store(rms > 0.02)
	})
	if err == nil {
		a.mic = sess
		if sess != nil {
			a.devices = sess.Info
		}
	}
	id := a.anim.Add(1)
	go a.animateCall(id)
}

func (a *App) stopAudio() {
	a.anim.Add(1)
	if a.mic != nil {
		a.mic.Close()
		a.mic = nil
	}
	a.talking.Store(false)
}

func (a *App) animateCall(id uint32) {
	tick := time.NewTicker(120 * time.Millisecond)
	defer tick.Stop()
	for range tick.C {
		if a.anim.Load() != id || !a.engine.InCall {
			return
		}
		a.tv.QueueUpdateDraw(func() {
			a.frame++
			a.paintCall()
		})
	}
}

func (a *App) paintCall() {
	if a.shell == nil || a.body == nil {
		return
	}
	if a.engine.InCall {
		a.shell.ResizeItem(a.callHead, 1, 0)
		a.body.ResizeItem(a.people, 22, 0)
		mic, head := a.devices.Mic, a.devices.Headset
		a.callHead.SetText(formatters.CallBanner(a.engine.CallName, mic, head, a.voice.State.Muted, a.talking.Load(), a.frame))
		a.people.SetText(a.peopleText())
		return
	}
	a.shell.ResizeItem(a.callHead, 0, 0)
	a.body.ResizeItem(a.people, 0, 0)
	a.callHead.SetText("")
	a.people.SetText("")
}

func (a *App) peopleText() string {
	if a.gw == nil {
		return "  no one here"
	}
	list := a.gw.Participants(a.engine.CallGuildID, a.engine.CallChannelID)
	if len(list) == 0 {
		return "  no one here"
	}
	var b strings.Builder
	for _, p := range list {
		speaking := p.Speaking
		muted := p.Muted
		if p.Self {
			muted = a.voice.State.Muted
			speaking = a.talking.Load() && !muted
		}
		b.WriteString(formatters.PeopleLine(p.Name, p.Self, muted, speaking, a.frame))
		b.WriteByte('\n')
	}
	return strings.TrimRight(b.String(), "\n")
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
	srv := "Friends"
	if a.engine.ServerIdx >= 0 && a.engine.ServerIdx < len(a.rail) {
		srv = a.rail[a.engine.ServerIdx].Name
	}
	switch a.engine.Column {
	case nav.ColServers:
		a.crumb.SetText(head + "  [#FFFFFF]" + srv + "[-]  [#C5CAE9]· live[-]")
	case nav.ColChannels:
		a.crumb.SetText(head + "  [#C5CAE9]" + srv + "[-]  [#FFE082]›[-]  channels")
	case nav.ColChat:
		c := ""
		if a.channel != nil {
			c = formatters.ChannelLabel("text", a.channel.Name)
		}
		a.crumb.SetText(head + "  [#C5CAE9]" + srv + "[-]  [#FFE082]›[-]  [#FFFFFF]" + c + "[-]")
	}
}

func (a *App) reload() {
	switch a.engine.Column {
	case nav.ColChat:
		if a.channel != nil {
			a.openText(*a.channel)
			a.flash("chat reloaded")
		}
	case nav.ColChannels:
		a.previewKeepIdx()
		a.focusChannels()
		a.flash("channels reloaded")
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
		a.focusServers()
	})
	a.pages.AddPage("invite", form, true, true)
	a.tv.SetFocus(form)
}

func (a *App) flash(msg string) {
	a.status.SetText("  " + msg + "     ↑↓ move   → open   ← back   m mute   l leave   a add")
}

func (a *App) quit() {
	a.stopAudio()
	if a.gw != nil {
		_ = a.gw.Close()
	}
	a.tv.Stop()
}
