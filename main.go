package main

import (
	"log"

	"github.com/marcusolsson/tui-go"
)

func main() {
	ui, err := tui.New(tui.NewVBox())
	if err != nil {
		log.Fatal(err)
	}

	ui.SetKeybinding("Esc", func() { ui.Quit() })

	ctrl := makeUIControl()

	go startCommunication(ctrl)

	ui.SetWidget(renderLogin(ui, ctrl))

	if err := ui.Run(); err != nil {
		log.Fatal(err)
	}
}
