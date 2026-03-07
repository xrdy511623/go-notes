package states

import "go-notes/designpattern/state/player"

// IdleState 表示播放器的空闲状态。
// 空闲时只有 Play 操作有意义；Pause 和 Stop 均为空操作。
type IdleState struct{}

func (s IdleState) Play(p *player.Player) {
	p.SetState(PlayingState{})
}

// Pause 在空闲状态下无意义（没有正在播放的内容可暂停）。
func (s IdleState) Pause(_ *player.Player) {}

// Stop 在空闲状态下无意义（已经处于停止状态）。
func (s IdleState) Stop(_ *player.Player) {}

func (s IdleState) String() string { return "idle" }
