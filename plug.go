package main

import "math/rand"
import "time"

type Plug struct {
	ID             int
	S3ID           string
	Owner          string
	ViewsRemaining int
}

func ChoosePlug(plugs []Plug) Plug {
	rand.Seed(time.Now().Unix())
	return plugs[rand.Intn(len(plugs))]
}
