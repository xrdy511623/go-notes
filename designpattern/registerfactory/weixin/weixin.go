package weixin

import (
	"fmt"

	"go-notes/designpattern/registerfactory/sender"
)

type weixinSender struct{}

func (w *weixinSender) Send(message string) error {
	fmt.Printf("Using WeiXin send message: %v\n", message)
	return nil
}

func init() {
	sender.Register("weixin", func() sender.Sender {
		return &weixinSender{}
	})
}
