package main

// MatchPhase represents the lifecycle of a match
type MatchPhase int

const (
	PhaseLobby     MatchPhase = 0
	PhaseCountdown MatchPhase = 1
	PhasePlaying   MatchPhase = 2
	PhaseResult    MatchPhase = 3
)

// GameMode defines the type of match
type GameMode int

const (
	ModeFFA          GameMode = 0
	ModeTDM          GameMode = 1
	ModeCTF          GameMode = 2
	ModeWaveSurvival GameMode = 3
)

// TeamID constants
const (
	TeamNone = 0
	TeamRed  = 1
	TeamBlue = 2
)

// MatchConfig holds settings for a match
type MatchConfig struct {
	Mode        GameMode
	TimeLimit   float64 // seconds
	ScoreLimit  int
	WorldWidth  float64
	WorldHeight float64
	MaxPlayers  int
	TeamCount   int
}

// DefaultConfig returns default config for the given mode
func DefaultConfig(mode GameMode) MatchConfig {
	switch mode {
	case ModeTDM:
		return MatchConfig{
			Mode:        ModeTDM,
			TimeLimit:   240,
			ScoreLimit:  30,
			WorldWidth:  6000,
			WorldHeight: 6000,
			MaxPlayers:  20,
			TeamCount:   2,
		}
	case ModeCTF:
		return MatchConfig{
			Mode:        ModeCTF,
			TimeLimit:   300,
			ScoreLimit:  3,
			WorldWidth:  6000,
			WorldHeight: 6000,
			MaxPlayers:  20,
			TeamCount:   2,
		}
	case ModeWaveSurvival:
		return MatchConfig{
			Mode:        ModeWaveSurvival,
			TimeLimit:   0,
			ScoreLimit:  0,
			WorldWidth:  4000,
			WorldHeight: 4000,
			MaxPlayers:  20,
			TeamCount:   0,
		}
	default:
		return MatchConfig{
			Mode:        ModeFFA,
			TimeLimit:   300,
			ScoreLimit:  0,
			WorldWidth:  4000,
			WorldHeight: 4000,
			MaxPlayers:  20,
			TeamCount:   0,
		}
	}
}

// MatchState holds the current match state
type MatchState struct {
	Phase        MatchPhase
	Config       MatchConfig
	Teams        [3]TeamState
	TimeLeft     float64
	CountdownT   float64
	ReadyPlayers map[string]bool
	WaveNumber   int
	ResultTimer  float64
}

// TeamState tracks per-team data
type TeamState struct {
	ID         int
	Score      int
	FlagHolder string
	FlagAtBase bool
	FlagX      float64
	FlagY      float64
}

// PlayerMatchStats tracks per-player stats for a match
type PlayerMatchStats struct {
	Kills       int
	Deaths      int
	Assists     int
	Score       int
	DamageDealt int
}

// NewMatchState creates a new match state for the given config
func NewMatchState(config MatchConfig) MatchState {
	ms := MatchState{
		Phase:        PhaseLobby,
		Config:       config,
		TimeLeft:     config.TimeLimit,
		ReadyPlayers: make(map[string]bool),
	}
	ms.Teams[TeamNone] = TeamState{ID: TeamNone}
	ms.Teams[TeamRed] = TeamState{ID: TeamRed, FlagAtBase: true}
	ms.Teams[TeamBlue] = TeamState{ID: TeamBlue, FlagAtBase: true}

	if config.Mode == ModeCTF {
		ms.Teams[TeamRed].FlagX = 500
		ms.Teams[TeamRed].FlagY = config.WorldHeight / 2
		ms.Teams[TeamBlue].FlagX = config.WorldWidth - 500
		ms.Teams[TeamBlue].FlagY = config.WorldHeight / 2
	}
	return ms
}

// IsTeamMode returns whether the game mode uses teams
func (c MatchConfig) IsTeamMode() bool {
	return c.Mode == ModeTDM || c.Mode == ModeCTF
}

// AssignTeam auto-balances a new player to the smaller team
func (ms *MatchState) AssignTeam(players map[string]*Player) int {
	if !ms.Config.IsTeamMode() {
		return TeamNone
	}
	redCount := 0
	blueCount := 0
	for _, p := range players {
		if p.Team == TeamRed {
			redCount++
		} else if p.Team == TeamBlue {
			blueCount++
		}
	}
	if redCount <= blueCount {
		return TeamRed
	}
	return TeamBlue
}

// SpawnPosition returns an appropriate spawn position based on team
func (ms *MatchState) SpawnPosition(team int) (float64, float64) {
	w := ms.Config.WorldWidth
	h := ms.Config.WorldHeight
	if ms.Config.IsTeamMode() {
		switch team {
		case TeamRed:
			return 200 + randFloat()*w*0.25, h*0.2 + randFloat()*h*0.6
		case TeamBlue:
			return w*0.75 + randFloat()*w*0.25 - 200, h*0.2 + randFloat()*h*0.6
		}
	}
	return w/4 + randFloat()*w/2, h/4 + randFloat()*h/2
}
