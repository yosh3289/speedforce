package core

import "github.com/gen2brain/beeep"

type BeeepSender struct{}

func (BeeepSender) Notify(title, body string) error {
	return beeep.Notify(title, body, "")
}
