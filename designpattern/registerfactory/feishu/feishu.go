package feishu

import (
	"fmt"

	"go-notes/designpattern/registerfactory/sender"
)

type FeiShu struct{}

func (f *FeiShu) Send(message string) error {
	fmt.Printf("Using FeiShu send mesasge: %v\n", message)
	return nil
}

func init() {
	sender.Register("feishu", func() sender.Sender { return &FeiShu{} })
}
