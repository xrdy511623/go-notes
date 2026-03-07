package turnstile

import "go-notes/designpattern/statemachine/fsm"

// 闸机状态
const (
	Locked   fsm.State = "locked"
	Unlocked fsm.State = "unlocked"
)

// 闸机事件
const (
	Coin fsm.Event = "coin"
	Push fsm.Event = "push"
)

// New 创建经典闸机状态机（CS 教科书示例）。
//
//	Locked ──[coin]──→ Unlocked
//	  ↑                    │
//	  └────[push]──────────┘
//	  │                    │
//	  └────[push]──→ Locked（自转换）
//	               [coin]──→ Unlocked（自转换：投币后再投币）
func New() (*fsm.FSM, error) {
	return fsm.New(Locked, []fsm.Transition{
		{Event: Coin, From: Locked, To: Unlocked},
		{Event: Push, From: Unlocked, To: Locked},
		{Event: Coin, From: Unlocked, To: Unlocked},
		{Event: Push, From: Locked, To: Locked},
	})
}
