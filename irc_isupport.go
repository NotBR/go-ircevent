package irc

import (
	"strconv"
	"strings"
	"sync"
)

// SetupFeatureDetect allows for the connection to have FeatureDetection enabled on it
func (irc *Connection) SetupFeatureDetect() {
	irc.AddCallback("005", func(e *Event) {
		irc.features.Lock()
		defer irc.features.Unlock()

		for _, arg := range e.Arguments[1:] {
			if strings.Index(arg, " ") != -1 {
				continue
			}
			switch idx := strings.Index(arg, "="); idx {
			case -1:
				if strings.HasPrefix(arg, "-") {
					// we should check if its one of those that has defaults but in practice
					// an IRC server isn't going to stop advertising one of those kinds of
					// features
					irc.features.delFeature(arg[1:])
				} else {
					irc.features.addFeature(arg, "")
				}
			default:
				irc.features.addFeature(arg[0:idx], arg[idx+1:])
			}
		}
	})
}

// Feature is a single Feature as exposed by ISUPPORT
type Feature struct {
	sync.Mutex
	Enabled bool
	Value   string

	parsedValue interface{}
}

// Features exposes the supported features from RPL_ISUPPORT
type Features struct {
	sync.Mutex
	features  map[string]*Feature
	callbacks map[string][]func(string, *Feature)
}

func newFeatures() *Features {
	ret := &Features{
		features:  make(map[string]*Feature),
		callbacks: make(map[string][]func(string, *Feature)),
	}

	ret.fill()

	return ret
}

// PrefixModes provides structured access to the PREFIX in ISUPPORT
func (irc *Connection) PrefixModes() *PrefixModes {
	if feat, ok := irc.GetFeature("PREFIX"); ok {
		feat.Lock()
		defer feat.Unlock()
		if ret, ok := feat.parsedValue.(*PrefixModes); ok {
			return ret
		}
	}

	return nil
}

// NickLength provides structured access to the NICKLEN in ISUPPORT
func (irc *Connection) NickLength() uint {
	if feat, ok := irc.GetFeature("NICKLEN"); ok {
		feat.Lock()
		defer feat.Unlock()
		if ret, ok := feat.parsedValue.(uint); ok {
			return ret
		}
	}

	return 0
}

// GetFeature returns an ISUPPORT feature by name, and a boolean indicating
// if the value was present or not. One should be sure to lock/unlock the feature
// when accessing or modifying it.
func (irc *Connection) GetFeature(name string) (*Feature, bool) {
	irc.features.Lock()
	defer irc.features.Unlock()
	feat, ok := irc.features.features[name]
	return feat, ok
}

// IterFeatures allows for one to iterate over the features present and perform
// actions with/upon it with the call function passed in. There is no need to lock
// the feature, this handles locking for the caller.
func (irc *Connection) IterFeatures(call func(string, *Feature)) {
	irc.features.Lock()
	defer irc.features.Unlock()
	for name, feat := range irc.features.features {
		feat.Lock()
		call(name, feat)
		feat.Unlock()
	}
}

// Features private functions

func (f *Features) addFeature(name string, val string) {
	f.features[name] = &Feature{
		Value:   val,
		Enabled: true,
	}
	f.call(name)
}

func (f *Features) delFeature(name string) {
	delete(f.features, name)
}

func (f *Features) fill() {
	defaultCallbacks := map[string][]func(string, *Feature){
		"PREFIX":  []func(string, *Feature){f.knownFeaturesPrefix},
		"NICKLEN": []func(string, *Feature){f.knownFeaturesNickLength},
		"EXCEPTS": []func(string, *Feature){f.exceptsEncountered}, // Value = e
		"INVEX":   []func(string, *Feature){f.invexEncountered},   // Value = I
	}

	defaults := map[string]string{
		"CHANTYPES": "#",
		"PREFIX":    "(ov)@+",
	}

	f.Lock()
	for feat, call := range defaultCallbacks {
		f.callbacks[feat] = call
	}

	for feat, val := range defaults {
		// using addFeature to make sure callbacks are called for defaults!
		f.addFeature(feat, val)
	}
	f.Unlock()
}

func (f *Features) call(feature string) {
	for _, call := range f.callbacks[feature] {
		f.features[feature].Lock()
		call(feature, f.features[feature])
		f.features[feature].Unlock()
	}
}

func (f *Features) exceptsEncountered(name string, feature *Feature) {
	if feature.Value == "" {
		feature.Value = "e"
	}
}

func (f *Features) invexEncountered(name string, feature *Feature) {
	if feature.Value == "" {
		feature.Value = "I"
	}
}

func (f *Features) knownFeaturesPrefix(name string, feature *Feature) {
	idx := make(map[int]rune)
	split, currIdx := -1, 0

	pm := &PrefixModes{
		Modes:   make(map[rune]rune),
		Display: make(map[rune]rune),
	}

	for _, char := range feature.Value {
		switch char {
		case '(':
			continue
		case ')':
			split = 0
			currIdx = 0
			continue
		default:
			if split == 0 {
				pm.Modes[idx[currIdx]] = char
				currIdx++
			} else {
				idx[currIdx] = char
				currIdx++
			}
		}
	}

	for mode, display := range pm.Modes {
		pm.Display[display] = mode
	}

	feature.parsedValue = pm
}

func (f *Features) knownFeaturesNickLength(name string, feature *Feature) {
	if nlen, err := strconv.ParseUint(feature.Value, 10, 64); err == nil {
		feature.parsedValue = uint(nlen)
	}
}

// PrefixModes presents a forwards and backwards map of mode character
// to display character
type PrefixModes struct {
	Modes   map[rune]rune
	Display map[rune]rune
}
