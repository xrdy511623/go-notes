package conn

import "go-notes/designpattern/statemachine/fsm"

// 连接状态，对标 net/http 的 ConnState。
const (
	Idle   fsm.State = "idle"
	Active fsm.State = "active"
	Closed fsm.State = "closed"
)

// 连接事件
const (
	Open  fsm.Event = "open"
	Park  fsm.Event = "park"
	Close fsm.Event = "close"
)

// New 创建一个网络连接状态机。
//
//	Idle ──[open]──→ Active ──[park]──→ Idle
//	  │                 │
//	  └──[close]──→ Closed ←──[close]──┘
func New() (*fsm.FSM, error) {
	return fsm.New(Idle, []fsm.Transition{
		{Event: Open, From: Idle, To: Active},
		{Event: Park, From: Active, To: Idle},
		{Event: Close, From: Active, To: Closed},
		{Event: Close, From: Idle, To: Closed},
	})
}
