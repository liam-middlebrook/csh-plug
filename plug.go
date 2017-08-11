package main

import "math/rand"
import "time"

const DEFAULT_AD_CHANCE = 95

type Plug struct {
	ID             int
	S3ID           string
	Owner          string
	ViewsRemaining int
}

func (p Plug) IsDefault() bool {
	return p.ViewsRemaining >= 0
}

func ChoosePlug(plugs []Plug) Plug {
	rand.Seed(time.Now().Unix())
	// Split plugs into default and custom ads
	var defaults []Plug
	var customs []Plug
	for i := 0; i < len(plugs); i++ {
		if plugs[i].IsDefault() {
			defaults = append(defaults, plugs[i])
		} else {
			customs = append(customs, plugs[i])
		}
	}
	// Decide whether to chose default ad or user submitted ad
	var pickDefault int = rand.Intn(100)
	if pickDefault >= DEFAULT_AD_CHANCE && len(defaults) != 0 {
		return defaults[rand.Intn(len(defaults))]
	} else {
		return customs[rand.Intn(len(customs))]
	}
}
