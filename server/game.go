package main

import (
	"encoding/json"
	"log"
	"math"
	"sync"
	"time"

	"github.com/vmihailenco/msgpack/v5"
)

const (
	TickRate       = 60 // physics ticks per second
	BroadcastRate  = 30 // state broadcasts per second
	TickDuration   = time.Second / TickRate
	BroadcastEvery = TickRate / BroadcastRate
)

const (
	maxProjectilesPerSession = 500
	maxPlayersPerSession     = 20
	maxMobsPerSession        = 8
	maxAsteroidsPerSession   = 5
	maxPickupsPerSession     = 4
	MobSpawnInterval         = 7.0
	AsteroidSpawnInterval    = 10.0
	PickupSpawnInterval      = 20.0
	DeathScorePenalty        = 10

	CountdownDuration = 3.0  // seconds of countdown before match starts
	ResultDuration    = 10.0 // seconds to show results before returning to lobby
)

// Broadcaster interface for sending messages to clients
type Broadcaster interface {
	SendJSON(msg interface{})
	SendRaw(data []byte)
	SendBinary(data []byte)
}

// Game holds the state for one game session
type Game struct {
	mu          sync.RWMutex
	players     map[string]*Player
	projectiles map[string]*Projectile
	mobs        map[string]*Mob
	asteroids   map[string]*Asteroid
	pickups     map[string]*Pickup
	clients     map[string]Broadcaster // playerID -> client
	controllers map[string]Broadcaster // playerID -> phone controller
	tick        uint64
	running     bool
	stop        chan struct{}
	nextShip    int

	mobSpawnCD      float64
	asteroidSpawnCD float64
	pickupSpawnCD   float64

	// Match state
	match_   MatchState
	gameTime float64 // elapsed game time in seconds

	// Spatial hash grid for broad-phase collision detection
	grid SpatialGrid

	// Flat entity lists for spatial grid indexing (rebuilt each tick)
	flatPlayers   []*Player
	flatProjs     []*Projectile
	flatMobs      []*Mob
	flatAsteroids []*Asteroid
	flatPickups   []*Pickup

	// Reusable query buffer for spatial grid lookups
	queryBuf []EntityRef

	// Delta compression: last-sent velocity per entity
	lastVX map[string]float64
	lastVY map[string]float64

	// Reusable broadcast buffers (reset with [:0] each tick)
	bcastPlayers   []playerWithPos
	bcastMobs      []mobWithPos
	bcastAsteroids []asteroidWithPos
	bcastPickups   []pickupWithPos
	bcastProjs     []projWithPos

	// Per-client filtered entity buffers
	filtPlayers   []PlayerState
	filtProjs     []ProjectileState
	filtMobs      []MobState
	filtAsteroids []AsteroidState
	filtPickups   []PickupState

	// Abilities
	homingMissiles map[string]*HomingProjectile
	healZones      map[string]*HealZone

	// Database for stat persistence (nil if no DB)
	db *DB
}

// NewGame creates a new Game with the given match configuration
func NewGame(config MatchConfig) *Game {
	ms := NewMatchState(config)
	return &Game{
		players:         make(map[string]*Player),
		projectiles:     make(map[string]*Projectile),
		mobs:            make(map[string]*Mob),
		asteroids:       make(map[string]*Asteroid),
		pickups:         make(map[string]*Pickup),
		clients:         make(map[string]Broadcaster),
		controllers:     make(map[string]Broadcaster),
		stop:            make(chan struct{}),
		mobSpawnCD:      MobSpawnInterval,
		asteroidSpawnCD: AsteroidSpawnInterval,
		pickupSpawnCD:   PickupSpawnInterval,
		match_:          ms,
		grid:            NewSpatialGrid(config.WorldWidth, config.WorldHeight),
		lastVX:          make(map[string]float64, maxPlayersPerSession+maxMobsPerSession),
		lastVY:          make(map[string]float64, maxPlayersPerSession+maxMobsPerSession),
		bcastPlayers:    make([]playerWithPos, 0, maxPlayersPerSession),
		bcastMobs:       make([]mobWithPos, 0, maxMobsPerSession),
		bcastAsteroids:  make([]asteroidWithPos, 0, maxAsteroidsPerSession),
		bcastPickups:    make([]pickupWithPos, 0, maxPickupsPerSession),
		bcastProjs:      make([]projWithPos, 0, 64),
		homingMissiles:  make(map[string]*HomingProjectile),
		healZones:       make(map[string]*HealZone),
		filtPlayers:     make([]PlayerState, 0, maxPlayersPerSession),
		filtProjs:       make([]ProjectileState, 0, 64),
		filtMobs:        make([]MobState, 0, maxMobsPerSession),
		filtAsteroids:   make([]AsteroidState, 0, maxAsteroidsPerSession),
		filtPickups:     make([]PickupState, 0, maxPickupsPerSession),
	}
}

// Run starts the game loop
func (g *Game) Run() {
	g.mu.Lock()
	g.running = true
	g.mu.Unlock()

	ticker := time.NewTicker(TickDuration)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			g.update()
		case <-g.stop:
			return
		}
	}
}

// Stop terminates the game loop
func (g *Game) Stop() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.running {
		g.running = false
		close(g.stop)
	}
}

// AddPlayer adds a new player to the game
func (g *Game) AddPlayer(name string) *Player {
	g.mu.Lock()
	defer g.mu.Unlock()

	if len(g.players) >= maxPlayersPerSession {
		return nil
	}

	id := GenerateID(4)
	ship := g.nextShip % 3
	g.nextShip++
	ww := g.match_.Config.WorldWidth
	wh := g.match_.Config.WorldHeight
	player := NewPlayerWithWorld(id, name, ship, ww, wh)
	g.players[id] = player
	return player
}

// RemovePlayer removes a player from the game
func (g *Game) RemovePlayer(id string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.players, id)
	delete(g.clients, id)
	delete(g.controllers, id)
}

// SetController associates a phone controller with a player
func (g *Game) SetController(playerID string, client Broadcaster) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.controllers[playerID] = client
	// Notify desktop client that a controller is now active
	if main, ok := g.clients[playerID]; ok {
		main.SendJSON(Envelope{T: MsgCtrlOn})
	}
}

// RemoveController detaches a phone controller from a player
func (g *Game) RemoveController(playerID string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.controllers, playerID)
	// Notify desktop client that the controller disconnected
	if main, ok := g.clients[playerID]; ok {
		main.SendJSON(Envelope{T: MsgCtrlOff})
	}
}

// HasPlayer returns true if the player exists in the game
func (g *Game) HasPlayer(id string) bool {
	g.mu.RLock()
	defer g.mu.RUnlock()
	_, ok := g.players[id]
	return ok
}

// SetClient associates a broadcaster with a player
func (g *Game) SetClient(playerID string, client Broadcaster) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.clients[playerID] = client
}

// SetDB sets the database reference for stat persistence
func (g *Game) SetDB(db *DB) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.db = db
}

// HandleInput processes input from a player
func (g *Game) HandleInput(playerID string, input ClientInput) {
	g.mu.Lock()
	defer g.mu.Unlock()

	p, ok := g.players[playerID]
	if !ok {
		return
	}
	// Only update target rotation when target is far enough from ship
	// to produce a stable angle (avoids flickering when idle on mobile)
	dx := input.MX - p.X
	dy := input.MY - p.Y
	if dx*dx+dy*dy > 25 { // > 5px distance
		p.TargetR = math.Atan2(dy, dx)
	}
	p.Firing = input.Fire
	p.Boosting = input.Boost
	p.TargetX = input.MX
	p.TargetY = input.MY
	p.SlowThresh = Clamp(input.Thresh, 50, 400)
	p.AbilityUsed = input.Ability
}

// HandleReady toggles a player's ready state
func (g *Game) HandleReady(playerID string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	p, ok := g.players[playerID]
	if !ok {
		return
	}
	p.Ready = !p.Ready

	// Broadcast updated team roster
	g.broadcastTeamUpdate()
}

// HandleTeamPick sets a player's team
func (g *Game) HandleTeamPick(playerID string, team int) {
	g.mu.Lock()
	defer g.mu.Unlock()

	p, ok := g.players[playerID]
	if !ok {
		return
	}
	// Only allow team picking in lobby
	if g.match_.Phase != PhaseLobby {
		return
	}
	// Validate team value
	if team != TeamRed && team != TeamBlue && team != TeamNone {
		return
	}
	p.Team = team

	// Broadcast updated team roster
	g.broadcastTeamUpdate()
}

// HandleRematch resets match to lobby for a rematch
func (g *Game) HandleRematch(playerID string) {
	g.mu.Lock()
	defer g.mu.Unlock()

	// Only allow rematch during result phase
	if g.match_.Phase != PhaseResult {
		return
	}

	g.resetToLobby()
}

// PlayerCount returns the number of players
func (g *Game) PlayerCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.players)
}

// MatchPhase returns the current match phase (thread-safe)
func (g *Game) MatchPhase() MatchPhase {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.match_.Phase
}

// MatchMode returns the game mode (thread-safe)
func (g *Game) MatchMode() GameMode {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.match_.Config.Mode
}

// update runs one game tick, dispatching to phase-specific methods
func (g *Game) update() {
	g.mu.Lock()
	defer g.mu.Unlock()

	dt := 1.0 / float64(TickRate)
	g.tick++
	g.gameTime += dt

	switch g.match_.Phase {
	case PhaseLobby:
		g.updateLobby(dt)
	case PhaseCountdown:
		g.updateCountdown(dt)
	case PhasePlaying:
		g.updatePlaying(dt)
	case PhaseResult:
		g.updateResult(dt)
	}
}

// updateLobby handles the lobby phase: waiting for players to ready up
func (g *Game) updateLobby(dt float64) {
	// Check if all players are ready and we have enough players
	if g.checkAllReady() {
		g.startCountdown()
	}

	// Broadcast state so clients can see each other in lobby
	if g.tick%BroadcastEvery == 0 {
		g.broadcastState()
	}
}

// updateCountdown handles the countdown phase: 3-second countdown before match
func (g *Game) updateCountdown(dt float64) {
	g.match_.CountdownT -= dt
	if g.match_.CountdownT <= 0 {
		g.startMatch()
		return
	}

	// Broadcast state during countdown
	if g.tick%BroadcastEvery == 0 {
		g.broadcastState()
	}
}

// updatePlaying handles the main gameplay phase: all existing game logic + match timer + score limit
func (g *Game) updatePlaying(dt float64) {
	// Update match timer
	if g.match_.Config.TimeLimit > 0 {
		g.match_.TimeLeft -= dt
		if g.match_.TimeLeft <= 0 {
			g.match_.TimeLeft = 0
			g.endMatch()
			return
		}
	}

	// Check score limit
	if g.match_.Config.ScoreLimit > 0 {
		if g.checkScoreLimit() {
			g.endMatch()
			return
		}
	}

	// Update players
	for _, p := range g.players {
		p.Update(dt)

		// Handle firing (class-based)
		if p.CanFire() && len(g.projectiles) < maxProjectilesPerSession {
			def := GetClassDef(p.Class)
			if def.ProjCount <= 1 {
				proj := NewProjectileWithClass(p, def, 0)
				g.projectiles[proj.ID] = proj
			} else {
				// Spread shot (e.g. Tank shotgun)
				half := def.ProjSpread / 2.0
				step := def.ProjSpread / float64(def.ProjCount-1)
				for i := 0; i < def.ProjCount && len(g.projectiles) < maxProjectilesPerSession; i++ {
					offset := -half + step*float64(i)
					proj := NewProjectileWithClass(p, def, offset)
					g.projectiles[proj.ID] = proj
				}
			}
			p.FireCD = def.FireCD
		}

		// Handle ability activation
		if p.AbilityUsed && p.Alive && p.Ability.CanActivate() {
			g.activateAbility(p)
			p.AbilityUsed = false
		}
	}

	// Update homing missiles
	for id, hm := range g.homingMissiles {
		hm.Update(dt, g.players, g.mobs)
		if !hm.Alive {
			delete(g.homingMissiles, id)
		}
	}

	// Update heal zones
	for id, hz := range g.healZones {
		if !hz.Update(dt) {
			delete(g.healZones, id)
			continue
		}
		// Heal nearby allies
		for _, p := range g.players {
			if !p.Alive || p.HP >= p.MaxHP {
				continue
			}
			// Heal owner and teammates
			if p.ID != hz.OwnerID && (hz.TeamID == TeamNone || p.Team != hz.TeamID) {
				continue
			}
			d2 := (p.X-hz.X)*(p.X-hz.X) + (p.Y-hz.Y)*(p.Y-hz.Y)
			if d2 <= hz.Radius*hz.Radius {
				heal := int(hz.Rate * dt)
				if heal < 1 {
					heal = 1
				}
				p.HP += heal
				if p.HP > p.MaxHP {
					p.HP = p.MaxHP
				}
			}
		}
	}

	// Update projectiles
	for id, proj := range g.projectiles {
		proj.Update(dt)
		if !proj.Alive {
			delete(g.projectiles, id)
		}
	}

	// Update mobs
	for id, mob := range g.mobs {
		wantFire := mob.Update(dt, g.players, g.projectiles)
		if !mob.Alive {
			delete(g.mobs, id)
			continue
		}
		// Broadcast mob phrase if any
		if mob.PendingPhrase != "" {
			g.broadcastMsg(Envelope{T: MsgMobSay, Data: MobSayMsg{
				MobID: mob.ID, Text: mob.PendingPhrase,
			}})
			mob.PendingPhrase = ""
		}
		if wantFire && len(g.projectiles) < maxProjectilesPerSession {
			proj := NewMobProjectile(mob)
			g.projectiles[proj.ID] = proj
		}
	}

	// Mob-mob collisions (soft repulsion, explode if fast)
	g.checkMobMobCollisions()

	// Update asteroids
	for id, ast := range g.asteroids {
		ast.Update(dt)
		if !ast.Alive {
			delete(g.asteroids, id)
		}
	}

	// Update pickups
	for id, pk := range g.pickups {
		pk.Update(dt)
		if !pk.Alive {
			delete(g.pickups, id)
		}
	}

	// Build spatial grid for broad-phase collision
	g.buildSpatialGrid()

	// Check collisions
	g.checkCollisions()
	g.checkPlayerCollisions()
	g.checkProjectileMobCollisions()
	g.checkAsteroidPlayerCollisions()
	g.checkAsteroidMobCollisions()
	g.checkProjectileAsteroidCollisions()
	g.checkPlayerPickupCollisions()
	g.checkPlayerMobCollisions()
	g.checkHomingMissileCollisions()

	// Spawn entities
	g.spawnEntities(dt)

	// Broadcast state
	if g.tick%BroadcastEvery == 0 {
		g.broadcastState()
	}
}

// updateResult handles the result phase: freeze gameplay, wait for timeout or rematch
func (g *Game) updateResult(dt float64) {
	g.match_.ResultTimer -= dt
	if g.match_.ResultTimer <= 0 {
		g.resetToLobby()
		return
	}

	// Still broadcast state so clients see the frozen scene
	if g.tick%BroadcastEvery == 0 {
		g.broadcastState()
	}
}

// checkAllReady returns true if all players are ready and there are enough players
func (g *Game) checkAllReady() bool {
	count := len(g.players)
	if count == 0 {
		return false
	}

	// Team modes require at least 2 players
	if g.match_.Config.Mode == ModeTDM {
		if count < 2 {
			return false
		}
	}

	for _, p := range g.players {
		if !p.Ready {
			return false
		}
	}
	return true
}

// startCountdown transitions from lobby to countdown
func (g *Game) startCountdown() {
	g.match_.Phase = PhaseCountdown
	g.match_.CountdownT = CountdownDuration

	g.broadcastMsg(Envelope{T: MsgMatchPhase, Data: MatchPhaseMsg{
		Phase:     int(PhaseCountdown),
		Mode:      int(g.match_.Config.Mode),
		Countdown: CountdownDuration,
	}})
}

// startMatch transitions from countdown to playing
func (g *Game) startMatch() {
	g.match_.Phase = PhasePlaying
	g.match_.TimeLeft = g.match_.Config.TimeLimit
	g.match_.Teams[TeamRed].Score = 0
	g.match_.Teams[TeamBlue].Score = 0

	// Reset player stats and spawn at team positions
	for _, p := range g.players {
		p.Kills = 0
		p.Deaths = 0
		p.Assists = 0
		p.DamageDealt = 0
		p.Score = 0
		p.RecentDamagers = nil

		// Spawn at team position
		sx, sy := g.match_.SpawnPosition(p.Team)
		p.SpawnAtPosition(sx, sy)
	}

	// Clear any existing entities from lobby
	g.projectiles = make(map[string]*Projectile)
	g.mobs = make(map[string]*Mob)
	g.asteroids = make(map[string]*Asteroid)
	g.pickups = make(map[string]*Pickup)
	g.mobSpawnCD = MobSpawnInterval
	g.asteroidSpawnCD = AsteroidSpawnInterval
	g.pickupSpawnCD = PickupSpawnInterval

	g.broadcastMsg(Envelope{T: MsgMatchPhase, Data: MatchPhaseMsg{
		Phase:    int(PhasePlaying),
		Mode:     int(g.match_.Config.Mode),
		TimeLeft: g.match_.TimeLeft,
	}})
}

// endMatch transitions from playing to result
func (g *Game) endMatch() {
	g.match_.Phase = PhaseResult
	g.match_.ResultTimer = ResultDuration

	// Calculate match duration
	duration := g.match_.Config.TimeLimit - g.match_.TimeLeft

	// Build player results
	results := make([]PlayerMatchResult, 0, len(g.players))
	var mvpID string
	mvpKills := -1
	for _, p := range g.players {
		pr := PlayerMatchResult{
			ID:      p.ID,
			Name:    p.Name,
			Team:    p.Team,
			Kills:   p.Kills,
			Deaths:  p.Deaths,
			Assists: p.Assists,
			Score:   p.Score,
		}
		results = append(results, pr)

		if p.Kills > mvpKills {
			mvpKills = p.Kills
			mvpID = p.ID
		}
	}

	// Mark MVP
	for i := range results {
		if results[i].ID == mvpID {
			results[i].MVP = true
			break
		}
	}

	// Determine winner
	winnerTeam := TeamNone
	if g.match_.Config.Mode == ModeTDM {
		redScore := g.match_.Teams[TeamRed].Score
		blueScore := g.match_.Teams[TeamBlue].Score
		if redScore > blueScore {
			winnerTeam = TeamRed
		} else if blueScore > redScore {
			winnerTeam = TeamBlue
		}
		// If tied, winnerTeam stays TeamNone (draw)
	} else {
		// FFA: winner is highest individual score
		bestScore := math.MinInt32
		for _, p := range g.players {
			if p.Score > bestScore {
				bestScore = p.Score
				winnerTeam = TeamNone // FFA has no team winner
			}
		}
	}

	resultMsg := MatchResultMsg{
		WinnerTeam: winnerTeam,
		Players:    results,
		Duration:   duration,
	}

	g.broadcastMsg(Envelope{T: MsgMatchResult, Data: resultMsg})
	g.broadcastMsg(Envelope{T: MsgMatchPhase, Data: MatchPhaseMsg{
		Phase: int(PhaseResult),
		Mode:  int(g.match_.Config.Mode),
	}})

	// Persist match and player stats to database
	if g.db != nil {
		go g.persistMatchResults(int(g.match_.Config.Mode), duration, winnerTeam, results)
	}
}

// persistMatchResults records match results in the database (runs in goroutine)
func (g *Game) persistMatchResults(mode int, duration float64, winnerTeam int, results []PlayerMatchResult) {
	matchID, err := g.db.RecordMatch(mode, duration, winnerTeam)
	if err != nil {
		log.Printf("DB: failed to record match: %v", err)
		return
	}

	g.mu.RLock()
	defer g.mu.RUnlock()

	for _, r := range results {
		p, ok := g.players[r.ID]
		if !ok || p.AuthPlayerID == 0 {
			continue // skip guests/disconnected
		}

		// Get current level before update
		prevStats, _ := g.db.GetStats(p.AuthPlayerID)
		prevLevel := 1
		if prevStats != nil {
			prevLevel = prevStats.Level
		}

		// Calculate XP: 10 per kill + 5 per assist + 50 win bonus
		xp := r.Kills*10 + r.Assists*5
		won := false
		if winnerTeam != TeamNone && r.Team == winnerTeam {
			xp += 50
			won = true
		} else if winnerTeam == TeamNone && r.MVP {
			xp += 50
			won = true
		}

		if err := g.db.RecordMatchPlayer(matchID, p.AuthPlayerID, r.Team, r.Kills, r.Deaths, r.Assists, r.Score, xp); err != nil {
			log.Printf("DB: failed to record match player: %v", err)
		}
		totalXP, newLevel, err := g.db.UpdateStatsAfterMatch(p.AuthPlayerID, r.Kills, r.Deaths, r.Assists, won, duration, xp)
		if err != nil {
			log.Printf("DB: failed to update player stats: %v", err)
			continue
		}

		// Send XP update to the player's client
		if client, ok := g.clients[r.ID]; ok {
			client.SendJSON(Envelope{T: MsgXPUpdate, Data: XPUpdateMsg{
				XPGained:  xp,
				TotalXP:   totalXP,
				Level:     newLevel,
				PrevLevel: prevLevel,
				XPNext:    XPToNextLevel(newLevel),
				LeveledUp: newLevel > prevLevel,
			}})
		}

		// Check achievements
		unlocked := CheckAchievements(g.db, p.AuthPlayerID, r.Kills, r.Deaths, won)
		if len(unlocked) > 0 {
			if client, ok := g.clients[r.ID]; ok {
				for _, ach := range unlocked {
					client.SendJSON(Envelope{T: MsgAchievementUnlock, Data: AchievementMsg{
						ID:          ach.ID,
						Name:        ach.Name,
						Description: ach.Description,
					}})
				}
			}
		}
	}
}

// resetToLobby transitions back to lobby for a new match
func (g *Game) resetToLobby() {
	g.match_.Phase = PhaseLobby
	g.match_.TimeLeft = 0
	g.match_.CountdownT = 0
	g.match_.ResultTimer = 0
	g.match_.Teams[TeamRed].Score = 0
	g.match_.Teams[TeamBlue].Score = 0

	// Reset player ready state and stats
	for _, p := range g.players {
		p.Ready = false
		p.Kills = 0
		p.Deaths = 0
		p.Assists = 0
		p.DamageDealt = 0
		p.Score = 0
		p.RecentDamagers = nil
		p.HP = p.MaxHP
		p.Alive = true
		p.RespawnT = 0
		p.SpawnProtection = 0

		// Re-spawn at random position
		ww := g.match_.Config.WorldWidth
		wh := g.match_.Config.WorldHeight
		p.X = ww/4 + randFloat()*ww/2
		p.Y = wh/4 + randFloat()*wh/2
		p.VX = 0
		p.VY = 0
	}

	// Clear entities
	g.projectiles = make(map[string]*Projectile)
	g.mobs = make(map[string]*Mob)
	g.asteroids = make(map[string]*Asteroid)
	g.pickups = make(map[string]*Pickup)
	g.mobSpawnCD = MobSpawnInterval
	g.asteroidSpawnCD = AsteroidSpawnInterval
	g.pickupSpawnCD = PickupSpawnInterval

	g.broadcastMsg(Envelope{T: MsgMatchPhase, Data: MatchPhaseMsg{
		Phase: int(PhaseLobby),
		Mode:  int(g.match_.Config.Mode),
	}})
}

// activateAbility processes a player's ability activation
func (g *Game) activateAbility(p *Player) {
	if !p.Ability.Activate() {
		return
	}
	switch p.Ability.Type {
	case AbilityMissileBarrage:
		// Spawn homing missiles in a fan
		for i := 0; i < MissileBarrageCount; i++ {
			offset := (float64(i) - float64(MissileBarrageCount-1)/2) * 0.15
			hm := NewHomingProjectile(
				p.X+math.Cos(p.Rotation)*ProjectileOffset,
				p.Y+math.Sin(p.Rotation)*ProjectileOffset,
				p.Rotation+offset,
				p.ID,
			)
			g.homingMissiles[hm.ID] = hm
		}
	case AbilityShield:
		// Shield is passive (damage absorption in TakeDamage)
	case AbilityBlink:
		// Teleport forward
		p.X += math.Cos(p.Rotation) * BlinkDistance
		p.Y += math.Sin(p.Rotation) * BlinkDistance
		// Wrap
		ww := g.match_.Config.WorldWidth
		wh := g.match_.Config.WorldHeight
		if p.X < 0 { p.X += ww } else if p.X > ww { p.X -= ww }
		if p.Y < 0 { p.Y += wh } else if p.Y > wh { p.Y -= wh }
	case AbilityHealAura:
		hz := NewHealZone(p.X, p.Y, p.ID, p.Team)
		g.healZones[hz.ID] = hz
	}
}

// checkScoreLimit returns true if any team/player has reached the score limit
func (g *Game) checkScoreLimit() bool {
	limit := g.match_.Config.ScoreLimit
	if limit <= 0 {
		return false
	}

	if g.match_.Config.Mode == ModeTDM {
		if g.match_.Teams[TeamRed].Score >= limit || g.match_.Teams[TeamBlue].Score >= limit {
			return true
		}
	} else {
		// FFA: check individual scores
		for _, p := range g.players {
			if p.Score >= limit {
				return true
			}
		}
	}
	return false
}

// isTeamMode returns true if the current game mode uses teams
func (g *Game) isTeamMode() bool {
	return g.match_.Config.Mode == ModeTDM
}

// broadcastTeamUpdate sends team roster info to all clients
func (g *Game) broadcastTeamUpdate() {
	var red, blue []TeamPlayerInfo
	for _, p := range g.players {
		info := TeamPlayerInfo{
			ID:    p.ID,
			Name:  p.Name,
			Ready: p.Ready,
		}
		switch p.Team {
		case TeamRed:
			red = append(red, info)
		case TeamBlue:
			blue = append(blue, info)
		}
	}
	g.broadcastMsg(Envelope{T: MsgTeamUpdate, Data: TeamUpdateMsg{
		Red:  red,
		Blue: blue,
	}})
}

// buildSpatialGrid populates the spatial hash with all alive entities
func (g *Game) buildSpatialGrid() {
	g.grid.Clear()

	// Build flat lists for indexed lookup
	g.flatPlayers = g.flatPlayers[:0]
	for _, p := range g.players {
		if p.Alive {
			idx := len(g.flatPlayers)
			g.flatPlayers = append(g.flatPlayers, p)
			g.grid.InsertCircle(p.X, p.Y, PlayerRadius, EntityRef{Kind: 'p', Idx: idx})
		}
	}

	g.flatProjs = g.flatProjs[:0]
	for _, proj := range g.projectiles {
		if proj.Alive {
			idx := len(g.flatProjs)
			g.flatProjs = append(g.flatProjs, proj)
			g.grid.Insert(proj.X, proj.Y, EntityRef{Kind: 'r', Idx: idx})
		}
	}

	g.flatMobs = g.flatMobs[:0]
	for _, mob := range g.mobs {
		if mob.Alive {
			idx := len(g.flatMobs)
			g.flatMobs = append(g.flatMobs, mob)
			g.grid.InsertCircle(mob.X, mob.Y, mob.Radius, EntityRef{Kind: 'm', Idx: idx})
		}
	}

	g.flatAsteroids = g.flatAsteroids[:0]
	for _, ast := range g.asteroids {
		if ast.Alive {
			idx := len(g.flatAsteroids)
			g.flatAsteroids = append(g.flatAsteroids, ast)
			g.grid.InsertCircle(ast.X, ast.Y, AsteroidRadius, EntityRef{Kind: 'a', Idx: idx})
		}
	}

	g.flatPickups = g.flatPickups[:0]
	for _, pk := range g.pickups {
		if pk.Alive {
			idx := len(g.flatPickups)
			g.flatPickups = append(g.flatPickups, pk)
			g.grid.InsertCircle(pk.X, pk.Y, PickupRadius, EntityRef{Kind: 'k', Idx: idx})
		}
	}
}

// checkCollisions checks projectile-player collisions using spatial grid
func (g *Game) checkCollisions() {
	const queryR = ProjectileRadius + PlayerRadius
	for _, proj := range g.flatProjs {
		if !proj.Alive {
			continue
		}
		g.queryBuf = g.grid.QueryBuf(proj.X, proj.Y, queryR, g.queryBuf[:0])
		nearby := g.queryBuf
		for _, ref := range nearby {
			if ref.Kind != 'p' {
				continue
			}
			p := g.flatPlayers[ref.Idx]
			if !p.Alive || p.ID == proj.OwnerID {
				continue
			}

			// Friendly fire skip: in team modes, skip if projectile owner is on same team
			if g.isTeamMode() {
				if owner, ok := g.players[proj.OwnerID]; ok && owner.Team != TeamNone && owner.Team == p.Team {
					continue
				}
			}

			if CheckCollision(proj.X, proj.Y, ProjectileRadius, p.X, p.Y, PlayerRadius) {
				// Record damage for assist tracking
				p.RecordDamage(proj.OwnerID, g.gameTime)

				died := p.TakeDamage(proj.Damage)
				proj.Alive = false

				// Track damage dealt
				if attacker, ok := g.players[proj.OwnerID]; ok {
					attacker.DamageDealt += proj.Damage
				}

				// Broadcast hit event
				g.broadcastMsg(Envelope{T: MsgHit, Data: HitMsg{
					X: p.X, Y: p.Y, Dmg: proj.Damage,
					VictimID: p.ID, AttackerID: proj.OwnerID,
				}})

				if died {
					p.Score -= DeathScorePenalty
					// Award kill to shooter
					if killer, ok := g.players[proj.OwnerID]; ok {
						killer.Score++
						killer.Kills++

						// Update team score
						if g.isTeamMode() && killer.Team != TeamNone {
							g.match_.Teams[killer.Team].Score++
						}

						// Award assists
						assistIDs := p.GetAssistIDs(killer.ID, g.gameTime)
						for _, aid := range assistIDs {
							if assister, ok := g.players[aid]; ok {
								assister.Assists++
							}
						}

						killMsg := Envelope{T: MsgKill, Data: KillMsg{
							KillerID:   killer.ID,
							KillerName: killer.Name,
							VictimID:   p.ID,
							VictimName: p.Name,
						}}
						g.broadcastMsg(killMsg)

						if client, ok := g.clients[p.ID]; ok {
							client.SendJSON(Envelope{T: MsgDeath, Data: DeathMsg{
								KillerID:   killer.ID,
								KillerName: killer.Name,
							}})
						}
					} else {
						// Killed by mob — mob celebrates
						if killerMob, ok := g.mobs[proj.OwnerID]; ok && killerMob.Alive {
							phrase := pickPhraseAlways("kill_player")
							g.broadcastMsg(Envelope{T: MsgMobSay, Data: MobSayMsg{
								MobID: killerMob.ID, Text: phrase,
							}})
						}
						g.broadcastMsg(Envelope{T: MsgKill, Data: KillMsg{
							KillerID: proj.OwnerID, KillerName: "Mob",
							VictimID: p.ID, VictimName: p.Name,
						}})
						if client, ok := g.clients[p.ID]; ok {
							client.SendJSON(Envelope{T: MsgDeath, Data: DeathMsg{
								KillerID:   proj.OwnerID,
								KillerName: "Mob",
							}})
						}
					}
				}
				break
			}
		}
	}
}

// checkPlayerCollisions checks ship-to-ship collisions (both die)
func (g *Game) checkPlayerCollisions() {
	players := g.flatPlayers // reuse pre-built alive-player list
	for i := 0; i < len(players); i++ {
		for j := i + 1; j < len(players); j++ {
			a, b := players[i], players[j]
			if !a.Alive || !b.Alive {
				continue
			}

			// Skip collision between teammates in team modes
			if g.isTeamMode() && a.Team != TeamNone && a.Team == b.Team {
				continue
			}

			if CheckCollision(a.X, a.Y, PlayerRadius, b.X, b.Y, PlayerRadius) {
				a.TakeDamage(a.HP)
				b.TakeDamage(b.HP)
				a.Score -= DeathScorePenalty
				b.Score -= DeathScorePenalty

				// Notify kills (mutual)
				killMsg1 := Envelope{T: MsgKill, Data: KillMsg{
					KillerID: a.ID, KillerName: a.Name,
					VictimID: b.ID, VictimName: b.Name,
				}}
				killMsg2 := Envelope{T: MsgKill, Data: KillMsg{
					KillerID: b.ID, KillerName: b.Name,
					VictimID: a.ID, VictimName: a.Name,
				}}
				g.broadcastMsg(killMsg1)
				g.broadcastMsg(killMsg2)

				if client, ok := g.clients[a.ID]; ok {
					client.SendJSON(Envelope{T: MsgDeath, Data: DeathMsg{
						KillerID: b.ID, KillerName: b.Name,
					}})
				}
				if client, ok := g.clients[b.ID]; ok {
					client.SendJSON(Envelope{T: MsgDeath, Data: DeathMsg{
						KillerID: a.ID, KillerName: a.Name,
					}})
				}
			}
		}
	}
}

// entityWithPos holds a converted entity state with raw position for viewport culling
type projWithPos struct {
	state ProjectileState
	x, y  float64
}

type playerWithPos struct {
	state PlayerState
	x, y  float64
}

type mobWithPos struct {
	state MobState
	x, y  float64
}

type asteroidWithPos struct {
	state AsteroidState
	x, y  float64
}

type pickupWithPos struct {
	state PickupState
	x, y  float64
}

// broadcastState sends the current game state to all clients with per-client viewport culling
func (g *Game) broadcastState() {
	// Delta compression threshold — skip velocity when change is tiny
	const velDelta = 5.0

	// Pre-convert all entities to state once, keeping raw positions for culling
	g.bcastPlayers = g.bcastPlayers[:0]
	for _, p := range g.players {
		ps := p.ToState()
		// Omit velocity if unchanged since last broadcast
		vx := *ps.VX
		vy := *ps.VY
		prevVX, prevVY := g.lastVX[p.ID], g.lastVY[p.ID]
		dx := vx - prevVX; if dx < 0 { dx = -dx }
		dy := vy - prevVY; if dy < 0 { dy = -dy }
		if dx < velDelta && dy < velDelta {
			ps.VX = nil
			ps.VY = nil
		} else {
			g.lastVX[p.ID] = vx
			g.lastVY[p.ID] = vy
		}
		g.bcastPlayers = append(g.bcastPlayers, playerWithPos{state: ps, x: p.X, y: p.Y})
	}
	g.bcastMobs = g.bcastMobs[:0]
	for _, mob := range g.mobs {
		if mob.Alive {
			ms := mob.ToState()
			vx := *ms.VX
			vy := *ms.VY
			prevVX, prevVY := g.lastVX[mob.ID], g.lastVY[mob.ID]
			dx := vx - prevVX; if dx < 0 { dx = -dx }
			dy := vy - prevVY; if dy < 0 { dy = -dy }
			if dx < velDelta && dy < velDelta {
				ms.VX = nil
				ms.VY = nil
			} else {
				g.lastVX[mob.ID] = vx
				g.lastVY[mob.ID] = vy
			}
			g.bcastMobs = append(g.bcastMobs, mobWithPos{state: ms, x: mob.X, y: mob.Y})
		}
	}
	g.bcastAsteroids = g.bcastAsteroids[:0]
	for _, ast := range g.asteroids {
		if ast.Alive {
			g.bcastAsteroids = append(g.bcastAsteroids, asteroidWithPos{state: ast.ToState(), x: ast.X, y: ast.Y})
		}
	}
	g.bcastPickups = g.bcastPickups[:0]
	for _, pk := range g.pickups {
		if pk.Alive {
			g.bcastPickups = append(g.bcastPickups, pickupWithPos{state: pk.ToState(), x: pk.X, y: pk.Y})
		}
	}
	g.bcastProjs = g.bcastProjs[:0]
	for _, proj := range g.projectiles {
		g.bcastProjs = append(g.bcastProjs, projWithPos{state: proj.ToState(), x: proj.X, y: proj.Y})
	}

	// Viewport culling radius (half-viewport + margin)
	const cullDist = 1200.0

	// Cache marshaled data per player to reuse for controllers
	playerData := make(map[string][]byte, len(g.clients))

	for playerID, client := range g.clients {
		player, ok := g.players[playerID]
		if !ok {
			continue
		}
		px, py := player.X, player.Y

		// Filter all entity types by viewport distance
		g.filtPlayers = g.filtPlayers[:0]
		for _, p := range g.bcastPlayers {
			dx := p.x - px; if dx < 0 { dx = -dx }
			dy := p.y - py; if dy < 0 { dy = -dy }
			if dx <= cullDist && dy <= cullDist {
				g.filtPlayers = append(g.filtPlayers, p.state)
			}
		}
		g.filtProjs = g.filtProjs[:0]
		for _, p := range g.bcastProjs {
			dx := p.x - px; if dx < 0 { dx = -dx }
			dy := p.y - py; if dy < 0 { dy = -dy }
			if dx <= cullDist && dy <= cullDist {
				g.filtProjs = append(g.filtProjs, p.state)
			}
		}
		g.filtMobs = g.filtMobs[:0]
		for _, m := range g.bcastMobs {
			dx := m.x - px; if dx < 0 { dx = -dx }
			dy := m.y - py; if dy < 0 { dy = -dy }
			if dx <= cullDist && dy <= cullDist {
				g.filtMobs = append(g.filtMobs, m.state)
			}
		}
		g.filtAsteroids = g.filtAsteroids[:0]
		for _, a := range g.bcastAsteroids {
			dx := a.x - px; if dx < 0 { dx = -dx }
			dy := a.y - py; if dy < 0 { dy = -dy }
			if dx <= cullDist && dy <= cullDist {
				g.filtAsteroids = append(g.filtAsteroids, a.state)
			}
		}
		g.filtPickups = g.filtPickups[:0]
		for _, pk := range g.bcastPickups {
			dx := pk.x - px; if dx < 0 { dx = -dx }
			dy := pk.y - py; if dy < 0 { dy = -dy }
			if dx <= cullDist && dy <= cullDist {
				g.filtPickups = append(g.filtPickups, pk.state)
			}
		}

		// Collect heal zone states
		var hzStates []HealZoneState
		if len(g.healZones) > 0 {
			hzStates = make([]HealZoneState, 0, len(g.healZones))
			for _, hz := range g.healZones {
				hzStates = append(hzStates, HealZoneState{
					ID: hz.ID, X: round1(hz.X), Y: round1(hz.Y), R: round1(hz.Radius),
				})
			}
		}

		state := GameState{
			Players:     g.filtPlayers,
			Projectiles: g.filtProjs,
			Mobs:        g.filtMobs,
			Asteroids:   g.filtAsteroids,
			Pickups:     g.filtPickups,
			Tick:        g.tick,
			MatchPhase:  int(g.match_.Phase),
			TimeLeft:    math.Round(g.match_.TimeLeft*10) / 10,
			TeamRedSc:   g.match_.Teams[TeamRed].Score,
			TeamBlueSc:  g.match_.Teams[TeamBlue].Score,
			HealZones:   hzStates,
		}

		data, err := msgpack.Marshal(&state)
		if err != nil {
			continue
		}
		playerData[playerID] = data
		client.SendBinary(data)
	}

	// Send to controllers using same data as their linked player
	var fallbackData []byte
	for playerID, client := range g.controllers {
		data, ok := playerData[playerID]
		if !ok {
			// Fallback: send unfiltered state (cached once)
			if fallbackData == nil {
				g.filtProjs = g.filtProjs[:0]
				for _, p := range g.bcastProjs {
					g.filtProjs = append(g.filtProjs, p.state)
				}
				g.filtPlayers = g.filtPlayers[:0]
				for _, p := range g.bcastPlayers {
					g.filtPlayers = append(g.filtPlayers, p.state)
				}
				g.filtMobs = g.filtMobs[:0]
				for _, m := range g.bcastMobs {
					g.filtMobs = append(g.filtMobs, m.state)
				}
				g.filtAsteroids = g.filtAsteroids[:0]
				for _, a := range g.bcastAsteroids {
					g.filtAsteroids = append(g.filtAsteroids, a.state)
				}
				g.filtPickups = g.filtPickups[:0]
				for _, pk := range g.bcastPickups {
					g.filtPickups = append(g.filtPickups, pk.state)
				}
				st := GameState{
					Players: g.filtPlayers, Projectiles: g.filtProjs,
					Mobs: g.filtMobs, Asteroids: g.filtAsteroids,
					Pickups: g.filtPickups, Tick: g.tick,
					MatchPhase: int(g.match_.Phase),
					TimeLeft:   math.Round(g.match_.TimeLeft*10) / 10,
					TeamRedSc:  g.match_.Teams[TeamRed].Score,
					TeamBlueSc: g.match_.Teams[TeamBlue].Score,
				}
				var err error
				fallbackData, err = msgpack.Marshal(&st)
				if err != nil {
					continue
				}
			}
			data = fallbackData
		}
		client.SendBinary(data)
	}
}

// broadcastMsg sends a message to all clients and controllers in the session
func (g *Game) broadcastMsg(msg Envelope) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	for _, client := range g.clients {
		client.SendRaw(data)
	}
	for _, client := range g.controllers {
		client.SendRaw(data)
	}
}

// checkMobMobCollisions applies soft repulsion between mobs and kills both if relative velocity is high
func (g *Game) checkMobMobCollisions() {
	// Build a local alive-mob list (can't reuse flatMobs since buildSpatialGrid runs later)
	mobs := g.flatMobs[:0]
	for _, m := range g.mobs {
		if m.Alive {
			mobs = append(mobs, m)
		}
	}
	g.flatMobs = mobs
	for i := 0; i < len(mobs); i++ {
		for j := i + 1; j < len(mobs); j++ {
			a, b := mobs[i], mobs[j]
			if !a.Alive || !b.Alive {
				continue
			}
			dx := b.X - a.X
			dy := b.Y - a.Y
			dist := math.Sqrt(dx*dx + dy*dy)
			repelDist := a.Radius + b.Radius + 10.0
			if dist < repelDist && dist > 0.1 {
				// Check relative velocity for explosion
				rvx := a.VX - b.VX
				rvy := a.VY - b.VY
				relV := math.Sqrt(rvx*rvx + rvy*rvy)
				if relV > MobExplodeRelV {
					// Crash phrases
					phraseA := pickPhraseAlways("mob_crash")
					g.broadcastMsg(Envelope{T: MsgMobSay, Data: MobSayMsg{
						MobID: a.ID, Text: phraseA,
					}})
					phraseB := pickPhraseAlways("mob_crash")
					g.broadcastMsg(Envelope{T: MsgMobSay, Data: MobSayMsg{
						MobID: b.ID, Text: phraseB,
					}})
					// Both explode
					a.Alive = false
					b.Alive = false
					// Broadcast explosions
					g.broadcastMsg(Envelope{T: MsgKill, Data: KillMsg{
						KillerID: a.ID, KillerName: "Mob",
						VictimID: b.ID, VictimName: "Mob",
					}})
					g.broadcastMsg(Envelope{T: MsgKill, Data: KillMsg{
						KillerID: b.ID, KillerName: "Mob",
						VictimID: a.ID, VictimName: "Mob",
					}})
					continue
				}
				// Soft repulsion — gentle nudge
				nx := dx / dist
				ny := dy / dist
				force := MobRepelForce * (1 - dist/repelDist)
				a.VX -= nx * force * (1.0 / 60.0)
				a.VY -= ny * force * (1.0 / 60.0)
				b.VX += nx * force * (1.0 / 60.0)
				b.VY += ny * force * (1.0 / 60.0)
			}
		}
	}
}

// checkProjectileMobCollisions checks projectile hits on mobs using spatial grid
func (g *Game) checkProjectileMobCollisions() {
	const queryR = ProjectileRadius + SDRadius // use max mob radius for broad-phase
	for _, proj := range g.flatProjs {
		if !proj.Alive {
			continue
		}
		g.queryBuf = g.grid.QueryBuf(proj.X, proj.Y, queryR, g.queryBuf[:0])
		nearby := g.queryBuf
		for _, ref := range nearby {
			if ref.Kind != 'm' {
				continue
			}
			mob := g.flatMobs[ref.Idx]
			if !mob.Alive || proj.OwnerID == mob.ID {
				continue
			}
			if CheckCollision(proj.X, proj.Y, ProjectileRadius, mob.X, mob.Y, mob.Radius) {
				died := mob.TakeDamage(proj.Damage)
				proj.Alive = false

				// Broadcast hit event
				g.broadcastMsg(Envelope{T: MsgHit, Data: HitMsg{
					X: mob.X, Y: mob.Y, Dmg: proj.Damage,
					VictimID: mob.ID, AttackerID: proj.OwnerID,
				}})

				if died {
					if killer, ok := g.players[proj.OwnerID]; ok {
						killer.Score += MobKillScore
					}
					killerName := g.playerName(proj.OwnerID)
					if killerName == "Unknown" {
						killerName = "Mob"
					}
					g.broadcastMsg(Envelope{T: MsgKill, Data: KillMsg{
						KillerID: proj.OwnerID, KillerName: killerName,
						VictimID: mob.ID, VictimName: "Mob",
					}})
				}
				break
			}
		}
	}
}

// checkAsteroidPlayerCollisions — asteroid kills player on contact
func (g *Game) checkAsteroidPlayerCollisions() {
	const queryR = AsteroidRadius + PlayerRadius
	for _, ast := range g.flatAsteroids {
		if !ast.Alive {
			continue
		}
		g.queryBuf = g.grid.QueryBuf(ast.X, ast.Y, queryR, g.queryBuf[:0])
		for _, ref := range g.queryBuf {
			if ref.Kind != 'p' {
				continue
			}
			p := g.flatPlayers[ref.Idx]
			if !p.Alive {
				continue
			}
			if CheckCollision(ast.X, ast.Y, AsteroidRadius, p.X, p.Y, PlayerRadius) {
				dmg := p.HP
				died := p.TakeDamage(dmg)
				g.broadcastMsg(Envelope{T: MsgHit, Data: HitMsg{
					X: p.X, Y: p.Y, Dmg: dmg,
					VictimID: p.ID, AttackerID: "asteroid",
				}})
				if died {
					p.Score -= DeathScorePenalty
					g.broadcastMsg(Envelope{T: MsgKill, Data: KillMsg{
						KillerID: "asteroid", KillerName: "Asteroid",
						VictimID: p.ID, VictimName: p.Name,
					}})
					if client, ok := g.clients[p.ID]; ok {
						client.SendJSON(Envelope{T: MsgDeath, Data: DeathMsg{
							KillerID: "asteroid", KillerName: "Asteroid",
						}})
					}
				}
			}
		}
	}
}

// checkAsteroidMobCollisions — asteroid instantly kills mob on contact
func (g *Game) checkAsteroidMobCollisions() {
	const queryR = AsteroidRadius + SDRadius // use max mob radius for broad-phase
	for _, ast := range g.flatAsteroids {
		if !ast.Alive {
			continue
		}
		g.queryBuf = g.grid.QueryBuf(ast.X, ast.Y, queryR, g.queryBuf[:0])
		for _, ref := range g.queryBuf {
			if ref.Kind != 'm' {
				continue
			}
			mob := g.flatMobs[ref.Idx]
			if !mob.Alive {
				continue
			}
			if CheckCollision(ast.X, ast.Y, AsteroidRadius, mob.X, mob.Y, mob.Radius) {
				// Mob phrase before dying
				phrase := pickPhraseAlways("asteroid_death")
				g.broadcastMsg(Envelope{T: MsgMobSay, Data: MobSayMsg{
					MobID: mob.ID, Text: phrase,
				}})
				mob.Alive = false
				g.broadcastMsg(Envelope{T: MsgKill, Data: KillMsg{
					KillerID: "asteroid", KillerName: "Asteroid",
					VictimID: mob.ID, VictimName: "Mob",
				}})
			}
		}
	}
}

// checkProjectileAsteroidCollisions — projectiles are destroyed by asteroids
func (g *Game) checkProjectileAsteroidCollisions() {
	const queryR = ProjectileRadius + AsteroidRadius
	for _, proj := range g.flatProjs {
		if !proj.Alive {
			continue
		}
		g.queryBuf = g.grid.QueryBuf(proj.X, proj.Y, queryR, g.queryBuf[:0])
		for _, ref := range g.queryBuf {
			if ref.Kind != 'a' {
				continue
			}
			ast := g.flatAsteroids[ref.Idx]
			if !ast.Alive {
				continue
			}
			if CheckCollision(proj.X, proj.Y, ProjectileRadius, ast.X, ast.Y, AsteroidRadius) {
				proj.Alive = false
				break
			}
		}
	}
}

// checkPlayerPickupCollisions — player picks up health orb
func (g *Game) checkPlayerPickupCollisions() {
	const queryR = PickupRadius + PlayerRadius
	for _, pk := range g.flatPickups {
		if !pk.Alive {
			continue
		}
		g.queryBuf = g.grid.QueryBuf(pk.X, pk.Y, queryR, g.queryBuf[:0])
		for _, ref := range g.queryBuf {
			if ref.Kind != 'p' {
				continue
			}
			p := g.flatPlayers[ref.Idx]
			if !p.Alive {
				continue
			}
			if CheckCollision(pk.X, pk.Y, PickupRadius, p.X, p.Y, PlayerRadius) {
				pk.Alive = false
				p.HP += PickupHeal
				if p.HP > p.MaxHP {
					p.HP = p.MaxHP
				}
				break
			}
		}
	}
}

// checkPlayerMobCollisions — mob dies, player takes damage
func (g *Game) checkPlayerMobCollisions() {
	for _, mob := range g.flatMobs {
		if !mob.Alive {
			continue
		}
		queryR := mob.Radius + PlayerRadius
		g.queryBuf = g.grid.QueryBuf(mob.X, mob.Y, queryR, g.queryBuf[:0])
		for _, ref := range g.queryBuf {
			if ref.Kind != 'p' {
				continue
			}
			p := g.flatPlayers[ref.Idx]
			if !p.Alive {
				continue
			}
			if CheckCollision(mob.X, mob.Y, mob.Radius, p.X, p.Y, PlayerRadius) {
				// Mob always dies
				mob.Alive = false

				// Player takes collision damage
				died := p.TakeDamage(mob.CollisionDmg)

				// Broadcast hit on player from mob collision
				g.broadcastMsg(Envelope{T: MsgHit, Data: HitMsg{
					X: p.X, Y: p.Y, Dmg: mob.CollisionDmg,
					VictimID: p.ID, AttackerID: mob.ID,
				}})

				// Player gets kill credit for the mob
				p.Score += MobKillScore
				g.broadcastMsg(Envelope{T: MsgKill, Data: KillMsg{
					KillerID: p.ID, KillerName: p.Name,
					VictimID: mob.ID, VictimName: "Mob",
				}})

				if died {
					p.Score -= DeathScorePenalty
					g.broadcastMsg(Envelope{T: MsgKill, Data: KillMsg{
						KillerID: mob.ID, KillerName: "Mob",
						VictimID: p.ID, VictimName: p.Name,
					}})
					if client, ok := g.clients[p.ID]; ok {
						client.SendJSON(Envelope{T: MsgDeath, Data: DeathMsg{
							KillerID: mob.ID, KillerName: "Mob",
						}})
					}
				}
				break // mob is dead, no need to check more players
			}
		}
	}
}

// checkHomingMissileCollisions checks homing missiles against players and mobs
func (g *Game) checkHomingMissileCollisions() {
	for _, hm := range g.homingMissiles {
		if !hm.Alive {
			continue
		}
		// Check against players
		for _, p := range g.players {
			if !p.Alive || p.ID == hm.OwnerID {
				continue
			}
			if p.SpawnProtection > 0 {
				continue
			}
			// Skip friendly fire
			if g.isTeamMode() {
				if owner, ok := g.players[hm.OwnerID]; ok && owner.Team == p.Team && owner.Team != TeamNone {
					continue
				}
			}
			if CheckCollision(hm.X, hm.Y, ProjectileRadius, p.X, p.Y, PlayerRadius) {
				hm.Alive = false
				died := p.TakeDamage(hm.Damage)
				p.RecordDamage(hm.OwnerID, g.gameTime)
				if owner, ok := g.players[hm.OwnerID]; ok {
					owner.DamageDealt += hm.Damage
				}
				g.broadcastMsg(Envelope{T: MsgHit, Data: HitMsg{
					X: p.X, Y: p.Y, Dmg: hm.Damage,
					VictimID: p.ID, AttackerID: hm.OwnerID,
				}})
				if died {
					if killer, ok := g.players[hm.OwnerID]; ok {
						killer.Score += 10
						killer.Kills++
						if g.isTeamMode() && killer.Team != TeamNone {
							g.match_.Teams[killer.Team].Score++
						}
					}
					g.broadcastMsg(Envelope{T: MsgKill, Data: KillMsg{
						KillerID: hm.OwnerID, KillerName: g.playerName(hm.OwnerID),
						VictimID: p.ID, VictimName: p.Name,
					}})
					if client, ok := g.clients[p.ID]; ok {
						client.SendJSON(Envelope{T: MsgDeath, Data: DeathMsg{
							KillerID: hm.OwnerID, KillerName: g.playerName(hm.OwnerID),
						}})
					}
				}
				break
			}
		}
		if !hm.Alive {
			continue
		}
		// Check against mobs
		for _, mob := range g.mobs {
			if !mob.Alive {
				continue
			}
			if CheckCollision(hm.X, hm.Y, ProjectileRadius, mob.X, mob.Y, mob.Radius) {
				hm.Alive = false
				died := mob.TakeDamage(hm.Damage)
				g.broadcastMsg(Envelope{T: MsgHit, Data: HitMsg{
					X: mob.X, Y: mob.Y, Dmg: hm.Damage,
					VictimID: mob.ID, AttackerID: hm.OwnerID,
				}})
				if died {
					if owner, ok := g.players[hm.OwnerID]; ok {
						owner.Score += MobKillScore
					}
					g.broadcastMsg(Envelope{T: MsgKill, Data: KillMsg{
						KillerID: hm.OwnerID, KillerName: g.playerName(hm.OwnerID),
						VictimID: mob.ID, VictimName: "Mob",
					}})
				}
				break
			}
		}
	}
}

// playerName returns the player name for an ID, or "Unknown"
func (g *Game) playerName(id string) string {
	if p, ok := g.players[id]; ok {
		return p.Name
	}
	return "Unknown"
}

// spawnEntities spawns mobs, asteroids, and pickups on timers
func (g *Game) spawnEntities(dt float64) {
	// Only spawn if there are players
	if len(g.players) == 0 {
		return
	}

	g.mobSpawnCD -= dt
	if g.mobSpawnCD <= 0 && len(g.mobs) < maxMobsPerSession {
		// Spawn one mob per tick until we reach the cap
		mob := NewMob()
		g.mobs[mob.ID] = mob
		if len(g.mobs) < maxMobsPerSession {
			g.mobSpawnCD = 0.5 // quick respawn to fill back up
		} else {
			g.mobSpawnCD = MobSpawnInterval
		}
	}

	g.asteroidSpawnCD -= dt
	if g.asteroidSpawnCD <= 0 && len(g.asteroids) < maxAsteroidsPerSession {
		ast := NewAsteroid()
		g.asteroids[ast.ID] = ast
		g.asteroidSpawnCD = AsteroidSpawnInterval
	}

	g.pickupSpawnCD -= dt
	if g.pickupSpawnCD <= 0 && len(g.pickups) < maxPickupsPerSession {
		pk := NewPickup()
		g.pickups[pk.ID] = pk
		g.pickupSpawnCD = PickupSpawnInterval
	}
}
