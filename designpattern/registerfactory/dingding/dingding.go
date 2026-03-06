package dingding

import (
	"fmt"

	"go-notes/designpattern/registerfactory/sender"
)

type dingSender struct{}

func (d *dingSender) Send(message string) error {
	fmt.Printf("Using DingDing send message: %v\n", message)
	return nil
}

func init() {
	sender.Register("dingding", func() sender.Sender {
		return &dingSender{}
	})
}
