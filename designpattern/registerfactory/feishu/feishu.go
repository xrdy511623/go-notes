package feishu

import (
	"fmt"

	"go-notes/designpattern/registerfactory/sender"
)

type feishuSender struct{}

func (f *feishuSender) Send(message string) error {
	fmt.Printf("Using FeiShu send message: %v\n", message)
	return nil
}

func init() {
	sender.Register("feishu", func() sender.Sender {
		return &feishuSender{}
	})
}
