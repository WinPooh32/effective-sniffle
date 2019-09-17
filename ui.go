package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/marcusolsson/tui-go"
)

type uiControl struct {
	AddUser       chan string
	DelUser       chan string
	AddMessage    chan string
	SubmitMessage chan string
	Log           chan string
	Quit          chan bool
}

func makeUIControl() uiControl {
	return uiControl{
		AddUser:       make(chan string),
		DelUser:       make(chan string),
		AddMessage:    make(chan string),
		SubmitMessage: make(chan string),
		Log:           make(chan string),
		Quit:          make(chan bool),
	}
}

type Post struct {
	Username string
	Message  string
}

var posts = []Post{}
var users = []string{}

func now() string {
	return time.Now().Format("15:04")
}

func arrayRemove(s []string, i int) []string {
	s[len(s)-1], s[i] = s[i], s[len(s)-1]
	return s[:len(s)-1]
}

func jsonToPost(data []byte) Post {
	post := Post{}
	json.Unmarshal(data, &post)
	return post
}

func appendToHistory(historyBox *tui.Box, p Post) {
	historyBox.Append(tui.NewHBox(
		tui.NewLabel(now()),
		tui.NewPadder(1, 0, tui.NewLabel(fmt.Sprintf("<%s>", p.Username))),
		tui.NewLabel(p.Message),
		tui.NewSpacer(),
	))
}

func removeFromUsers(sidebar *tui.Box, username string) {
	var i int = -1
	for j := range users {
		if username == users[j] {
			i = j
			break
		}
	}

	if i == -1 {
		return
	}

	users = arrayRemove(users, i)
	sidebar.Remove(i)
}

func appendToUsers(sidebar *tui.Box, username string) {
	// Remove all
	for i := 0; i < len(users)+1; i++ {
		sidebar.Remove(0)
	}

	users = append(users, username)

	// Insert all back
	for _, listUser := range users {
		sidebar.Append(tui.NewLabel(listUser))
	}

	// sidebar.Append(tui.NewLabel(username))
	sidebar.Append(tui.NewSpacer())
}

func runWorker(ctrl uiControl, ui tui.UI, sidebar, history *tui.Box) {
worker:
	for {
		select {
		case <-ctrl.Quit:
			break worker

		case log := <-ctrl.Log:
			ui.Update(func() {
				history.Append(tui.NewLabel(log))
			})

		case username := <-ctrl.AddUser:
			ui.Update(func() {
				appendToUsers(sidebar, username)
			})

		case username := <-ctrl.DelUser:
			ui.Update(func() {
				removeFromUsers(sidebar, username)
				history.Append(tui.NewLabel(username + " left us."))
			})

		case message := <-ctrl.AddMessage:
			ui.Update(func() {
				post := jsonToPost([]byte(message))
				appendToHistory(history, post)
			})
		}
	}
}

func runUI(ui tui.UI, ctrl uiControl, username string) *tui.Box {
	sidebar := tui.NewVBox()
	sidebar.SetBorder(true)

	appendToUsers(sidebar, username)

	history := tui.NewVBox()

	historyScroll := tui.NewScrollArea(history)
	historyScroll.SetAutoscrollToBottom(true)

	historyBox := tui.NewVBox(historyScroll)
	historyBox.SetBorder(true)

	input := tui.NewEntry()
	input.SetFocused(true)
	input.SetSizePolicy(tui.Expanding, tui.Maximum)

	inputBox := tui.NewHBox(input)
	inputBox.SetBorder(true)
	inputBox.SetSizePolicy(tui.Expanding, tui.Maximum)

	chat := tui.NewVBox(historyBox, inputBox)
	chat.SetSizePolicy(tui.Expanding, tui.Expanding)

	input.OnSubmit(func(e *tui.Entry) {
		p := Post{Username: username, Message: e.Text()}

		appendToHistory(history, p)
		input.SetText("")

		jsonPost, _ := json.Marshal(p)
		ctrl.SubmitMessage <- string(jsonPost)
	})

	root := tui.NewHBox(sidebar, chat)

	go runWorker(ctrl, ui, sidebar, history)

	return root
}
