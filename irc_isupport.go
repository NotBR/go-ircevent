package irc

import (
	"strconv"
	"strings"
)

// Features exposes the supported features from RPL_ISUPPORT
type Features struct {
	Params  map[string]bool
	Options map[string]string
}

// KnownFeatures exposes the features we parse out for utilization
type KnownFeatures struct {
	PrefixModes        map[rune]rune
	PrefixModesDisplay map[rune]rune
	NickLength         uint
}

func makeKnownFeatures() KnownFeatures {
	return KnownFeatures{
		PrefixModes:        make(map[rune]rune),
		PrefixModesDisplay: make(map[rune]rune),
		NickLength:         30,
	}
}

func fillAssumptions(f *Features) {
	optAssumptions := map[string]string{
		"CHANTYPES": "#",
		"PREFIX":    "(ov)@+",
	}

	for opt, val := range optAssumptions {
		f.Options[opt] = val
	}
}

func makeFeatures() *Features {
	ret := Features{
		Params:  make(map[string]bool),
		Options: make(map[string]string),
	}
	fillAssumptions(&ret)

	return &ret
}

func (irc *Connection) knownFeaturesPrefix() {
	if pmodes, ok := irc.Features.Options["PREFIX"]; ok {
		idx := make(map[int]rune)
		split := -1
		currIdx := 0
		for _, char := range pmodes {
			switch char {
			case '(':
				continue
			case ')':
				split = 0
				currIdx = 0
				continue
			default:
				if split == 0 {
					irc.KnownFeatures.PrefixModes[idx[currIdx]] = char
					currIdx++
				} else {
					idx[currIdx] = char
					currIdx++
				}
			}
		}

		for ml, dl := range irc.KnownFeatures.PrefixModes {
			irc.KnownFeatures.PrefixModesDisplay[dl] = ml
		}
	}
}

func (irc *Connection) knownFeaturesNickLength() {
	if nlen, ok := irc.Features.Options["NICKLEN"]; ok {
		if nl, err := strconv.ParseUint(nlen, 10, 64); err == nil {
			irc.KnownFeatures.NickLength = uint(nl)
		}
	}
}

// SetupFeatureDetect allows for the connection to have FeatureDetection enabled on it
func (irc *Connection) SetupFeatureDetect() {
	irc.AddCallback("005", func(e *Event) {
		knownFeaturesTypes := map[string]func(){
			"PREFIX":  irc.knownFeaturesPrefix,
			"NICKLEN": irc.knownFeaturesNickLength,
		}

		irc.featuresMutex.Lock()
		defer irc.featuresMutex.Unlock()

		for _, arg := range e.Arguments[1:] {
			if strings.Index(arg, " ") != -1 {
				return
			}
			switch idx := strings.Index(arg, "="); idx {
			case -1:
				irc.Features.Params[arg] = true
			default:
				irc.Features.Options[arg[0:idx]] = arg[idx+1:]
				if call, ok := knownFeaturesTypes[arg[0:idx]]; ok {
					call()
				}
			}
		}
	})
}
