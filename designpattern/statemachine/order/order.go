package order

import "go-notes/designpattern/statemachine/fsm"

// 订单状态
const (
	Created   fsm.State = "created"
	Paid      fsm.State = "paid"
	Shipped   fsm.State = "shipped"
	Delivered fsm.State = "delivered"
	Cancelled fsm.State = "cancelled"
)

// 订单事件
const (
	Pay     fsm.Event = "pay"
	Ship    fsm.Event = "ship"
	Deliver fsm.Event = "deliver"
	Cancel  fsm.Event = "cancel"
)

// New 创建一个电商订单状态机。
//
//	Created ──[pay]──→ Paid ──[ship]──→ Shipped ──[deliver]──→ Delivered
//	   │                 │
//	   └──[cancel]──→ Cancelled ←──[cancel]──┘
func New() (*fsm.FSM, error) {
	return fsm.New(Created, []fsm.Transition{
		{Event: Pay, From: Created, To: Paid},
		{Event: Ship, From: Paid, To: Shipped},
		{Event: Deliver, From: Shipped, To: Delivered},
		{Event: Cancel, From: Created, To: Cancelled},
		{Event: Cancel, From: Paid, To: Cancelled},
	})
}
