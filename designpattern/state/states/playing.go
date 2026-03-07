package states

import "go-notes/designpattern/state/player"

// PlayingState 表示播放器正在播放的状态。
// 播放中可以暂停或停止；再次 Play 为空操作。
type PlayingState struct{}

// Play 在播放状态下无意义（已经在播放）。
func (s PlayingState) Play(_ *player.Player) {}

func (s PlayingState) Pause(p *player.Player) {
	p.SetState(PausedState{})
}

func (s PlayingState) Stop(p *player.Player) {
	p.SetState(IdleState{})
}

func (s PlayingState) String() string { return "playing" }
