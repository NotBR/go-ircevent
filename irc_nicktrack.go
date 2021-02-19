package irc

import (
	"strings"
)

//Struct to store Channel Info
type Channel struct {
	Topic string
	Mode  string
	Users map[string]User
}

type User struct {
	Host string
	Mode string
}

func (irc *Connection) isModeChar(c rune) bool {
	_, ok := irc.KnownFeatures.PrefixModesDisplay[c]
	return ok
}

func (irc *Connection) SetupNickTrack() {
	// relies on ISUPPORT so introduce it
	irc.SetupFeatureDetect()
	// 353: RPL_NAMEREPLY per RFC1459
	// will typically receive this on channel joins and when NAMES is
	// called via GetNicksOnChan
	irc.AddCallback("353", func(e *Event) {
		// get chan
		channelName := e.Arguments[2]

		// check if chan exists in map, if not make one
		if _, ok := irc.Channels[channelName]; !ok {
			irc.Channels[channelName] = Channel{Users: make(map[string]User)}
		}

		for _, modenick := range strings.Split(e.Message(), " ") {
			u := User{}
			idx := 0
			for pos, c := range modenick {
				if !irc.isModeChar(c) {
					idx = pos
					break
				}
			}

			if idx > 0 {
				u.Mode = "+"
				for _, mc := range modenick[0:idx] {
					u.Mode += string(irc.KnownFeatures.PrefixModesDisplay[mc])
				}
			}

			irc.Channels[channelName].Users[modenick[idx:]] = u
		}
	})

	irc.AddCallback("MODE", func(e *Event) {
		channelName := e.Arguments[0]
		if len(e.Arguments) == 3 { // 3 == for channel 2 == for user on server
			if _, ok := irc.Channels[channelName]; ok != true {
				irc.Channels[channelName] = Channel{Users: make(map[string]User)}
			}
			if _, ok := irc.Channels[channelName].Users[e.Arguments[2]]; ok != true {
				irc.Channels[channelName].Users[e.Arguments[2]] = User{Mode: e.Arguments[1]}
			} else {
				u := irc.Channels[channelName].Users[e.Arguments[2]]
				u.Mode = e.Arguments[1]
				irc.Channels[channelName].Users[e.Arguments[2]] = u
			}
		}
	})

	//Really hacky since the message from the server does not include the channel
	irc.AddCallback("NICK", func(e *Event) {
		if len(e.Arguments) == 1 { // Sanity check
			for k, _ := range irc.Channels {
				if _, ok := irc.Channels[k].Users[e.Nick]; ok {
					u := irc.Channels[k].Users[e.Nick]
					u.Host = e.Host
					irc.Channels[k].Users[e.Arguments[0]] = u //New nick
					delete(irc.Channels[k].Users, e.Nick)     //Delete old
				}
			}
		}
	})

	irc.AddCallback("JOIN", func(e *Event) {
		channelName := e.Arguments[0]
		if _, ok := irc.Channels[channelName]; ok != true {
			irc.Channels[channelName] = Channel{Users: make(map[string]User)}
		}
		irc.Channels[channelName].Users[e.Nick] = User{Host: e.Source}
	})

	irc.AddCallback("PART", func(e *Event) {
		channelName := e.Arguments[0]
		delete(irc.Channels[channelName].Users, e.Nick)
	})

	irc.AddCallback("QUIT", func(e *Event) {
		for k, _ := range irc.Channels {
			delete(irc.Channels[k].Users, e.Nick)
		}
	})
}
