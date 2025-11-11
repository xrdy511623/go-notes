package weixin

import (
	"fmt"

	"go-notes/designpattern/registerfactory/sender"
)

type WeiXin struct{}

func (w *WeiXin) Send(message string) error {
	fmt.Printf("Using Weixin send message: %v\n", message)
	return nil
}

func init() {
	sender.Register("weixin", func() sender.Sender { return &WeiXin{} })
}
