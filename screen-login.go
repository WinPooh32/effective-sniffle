package main

import (
	"github.com/marcusolsson/tui-go"
)

var logo = `     _____ __ ____  ___   ______________  
    / ___// //_/\ \/ / | / / ____/_  __/  
    \__ \/ ,<    \  /  |/ / __/   / /     
   ___/ / /| |   / / /|  / /___  / /      
  /____/_/ |_|  /_/_/ |_/_____/ /_/     `

func renderLogin(ui tui.UI, ctrl uiControl) *tui.Box {
	user := tui.NewEntry()
	user.SetFocused(true)

	form := tui.NewGrid(0, 0)
	form.AppendRow(tui.NewLabel("User"))
	form.AppendRow(user)

	login := tui.NewButton("[Login]")

	buttons := tui.NewHBox(
		tui.NewSpacer(),
		tui.NewPadder(1, 0, login),
	)

	window := tui.NewVBox(
		tui.NewPadder(10, 1, tui.NewLabel(logo)),
		tui.NewPadder(12, 0, tui.NewLabel("Welcome to Skynet!")),
		tui.NewPadder(1, 1, form),
		buttons,
	)
	window.SetBorder(true)

	wrapper := tui.NewVBox(
		tui.NewSpacer(),
		window,
		tui.NewSpacer(),
	)
	content := tui.NewHBox(tui.NewSpacer(), wrapper, tui.NewSpacer())

	root := tui.NewVBox(
		content,
	)

	tui.DefaultFocusChain.Set(user, login)

	login.OnActivated(func(b *tui.Button) {
		if len(user.Text()) < 1 {
			return
		}

		ui.SetWidget(runUI(ui, ctrl, user.Text()))
	})

	return root
}
