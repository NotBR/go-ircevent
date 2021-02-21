package main

import (
	"fmt"
	"github.com/thoja/go-ircevent"
	"sort"
	"time"
)

const channel = "#ggnet"
const serverssl = "irc.homelien.no:6667"

// also check Feature Detection (because NickTrack needs it anyway)
func main() {
	ircnick1 := "blatibalt1"
	irccon := irc.IRC(ircnick1, "blatiblat")
	irccon.VerboseCallbackHandler = true
	irccon.Debug = true
	irccon.AddCallback("001", func(e *irc.Event) { irccon.Join(channel) })
	irccon.AddCallback("366", func(e *irc.Event) {})
	irccon.SetupNickTrack()
	err := irccon.Connect(serverssl)
	if err != nil {
		fmt.Printf("Err %s", err)
		return
	}
	go func() {
		t := time.NewTicker(30 * time.Second)
		for {
			<-t.C
			var keys []string
			if ch, ok := irccon.GetChannel(channel); ok {
				ch.IterUsers(func(name string, user irc.User) {
					keys = append(keys, name)
				})

				sort.Strings(keys)

				for _, user := range keys {
					u, _ := ch.GetUser(user)
					fmt.Printf("(%s)%s ", u.Mode, user)
				}
				fmt.Println()
			}

			fmt.Println()

			fmt.Println("Features:")
			irccon.IterFeatures(func(name string, feature *irc.Feature) {
				fmt.Printf("  %s = %s\n", name, feature.Value)
			})

			fmt.Println()
			fmt.Println("Known Features: ")
			prefix := irccon.PrefixModes()
			nickLength := irccon.NickLength()
			fmt.Printf("  NICKLEN: %d\n\n", nickLength)
			pmodes := ""
			dmodes := ""
			for mode, _ := range prefix.Modes {
				pmodes += string(mode)
			}
			for mode, _ := range prefix.Display {
				dmodes += string(mode)
			}
			fmt.Printf("  Prefix Modes: %s\n", pmodes)
			fmt.Printf("  Display Modes: %s\n\n", dmodes)
		}
	}()
	irccon.Loop()
}
