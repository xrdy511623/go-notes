// Package switchstate 演示用 switch/case 管理状态的反模式。
//
// 问题：
//   - 每新增一个状态，必须修改所有动作方法中的 switch 分支 —— 违反开闭原则（OCP）
//   - 状态相关的行为逻辑分散在 Play/Pause/Stop 各方法中，而非集中在一个状态对象里
//   - 编译器无法检查遗漏的 case，容易引入沉默 bug
//   - 随着状态和动作增多，switch 分支呈 M×N 膨胀，可维护性急剧下降
//
// 正确做法参考 player/ 和 states/ 包：将每个状态封装为独立的 State 实现。
package switchstate

import "fmt"

// BadPlayer 使用 string + switch 管理状态 —— 这是反模式。
type BadPlayer struct {
	state string
	track string
}

func New(track string) *BadPlayer {
	return &BadPlayer{state: "idle", track: track}
}

func (p *BadPlayer) Play() {
	switch p.state {
	case "idle":
		fmt.Println("Start playing:", p.track)
		p.state = "playing"
	case "playing":
		fmt.Println("Already playing")
	case "paused":
		fmt.Println("Resuming:", p.track)
		p.state = "playing"
		// 新增状态 "buffering" 时，必须在此处、Pause()、Stop() 都加 case，
		// 忘记任何一处都不会有编译错误，只会导致运行时默默跳过。
	}
}

func (p *BadPlayer) Pause() {
	switch p.state {
	case "idle":
		fmt.Println("Cannot pause, not playing")
	case "playing":
		fmt.Println("Pausing:", p.track)
		p.state = "paused"
	case "paused":
		fmt.Println("Already paused")
	}
}

func (p *BadPlayer) Stop() {
	switch p.state {
	case "idle":
		fmt.Println("Already stopped")
	case "playing":
		fmt.Println("Stopping:", p.track)
		p.state = "idle"
	case "paused":
		fmt.Println("Stopping from pause:", p.track)
		p.state = "idle"
	}
}
