package player

// State 是播放器状态的接口（GoF State 模式中的 State 角色）。
// 每个具体状态封装了该状态下的行为逻辑以及状态转换规则。
type State interface {
	Play(p *Player)
	Pause(p *Player)
	Stop(p *Player)
	String() string
}

// Player 是媒体播放器（GoF State 模式中的 Context 角色）。
// 它将行为委托给当前持有的 State 对象，状态切换由各 State 自行决定。
type Player struct {
	state   State
	track   string
	history []string
}

// New 创建一个播放器实例，初始状态为 idle。
// initState 参数用于注入初始状态，避免 player 包反向依赖 states 包。
func New(track string, initState State) *Player {
	p := &Player{
		track: track,
	}
	p.state = initState
	p.history = []string{initState.String()}
	return p
}

// SetState 切换播放器状态，并将新状态名称记录到历史中。
func (p *Player) SetState(s State) {
	p.state = s
	p.history = append(p.history, s.String())
}

// Play 将播放操作委托给当前状态。
func (p *Player) Play() { p.state.Play(p) }

// Pause 将暂停操作委托给当前状态。
func (p *Player) Pause() { p.state.Pause(p) }

// Stop 将停止操作委托给当前状态。
func (p *Player) Stop() { p.state.Stop(p) }

// State 返回当前状态。
func (p *Player) State() State { return p.state }

// Track 返回当前播放的曲目名称。
func (p *Player) Track() string { return p.track }

// History 返回状态转换历史的副本。
func (p *Player) History() []string {
	cp := make([]string, len(p.history))
	copy(cp, p.history)
	return cp
}
