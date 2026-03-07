package states

import "go-notes/designpattern/state/player"

// PausedState 表示播放器暂停的状态。
// 暂停后可以恢复播放或停止；再次 Pause 为空操作。
type PausedState struct{}

func (s PausedState) Play(p *player.Player) {
	p.SetState(PlayingState{})
}

// Pause 在暂停状态下无意义（已经暂停）。
func (s PausedState) Pause(_ *player.Player) {}

func (s PausedState) Stop(p *player.Player) {
	p.SetState(IdleState{})
}

func (s PausedState) String() string { return "paused" }
