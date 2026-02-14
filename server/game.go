package main

import (
	"encoding/json"
	"math"
	"sync"
	"time"
)

const (
	TickRate      = 60               // physics ticks per second
	BroadcastRate = 30               // state broadcasts per second
	TickDuration  = time.Second / TickRate
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
)

// Broadcaster interface for sending messages to clients
type Broadcaster interface {
	SendJSON(msg interface{})
	SendRaw(data []byte)
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
}

// NewGame creates a new Game
func NewGame() *Game {
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
		bcastPlayers:    make([]playerWithPos, 0, maxPlayersPerSession),
		bcastMobs:       make([]mobWithPos, 0, maxMobsPerSession),
		bcastAsteroids:  make([]asteroidWithPos, 0, maxAsteroidsPerSession),
		bcastPickups:    make([]pickupWithPos, 0, maxPickupsPerSession),
		bcastProjs:      make([]projWithPos, 0, 64),
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
	player := NewPlayer(id, name, ship)
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
}

// PlayerCount returns the number of players
func (g *Game) PlayerCount() int {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return len(g.players)
}

// update runs one game tick
func (g *Game) update() {
	g.mu.Lock()
	defer g.mu.Unlock()

	dt := 1.0 / float64(TickRate)
	g.tick++

	// Update players
	for _, p := range g.players {
		p.Update(dt)

		// Handle firing
		if p.CanFire() && len(g.projectiles) < maxProjectilesPerSession {
			proj := NewProjectile(p)
			g.projectiles[proj.ID] = proj
			p.FireCD = FireCooldown
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
		wantFire := mob.Update(dt, g.players)
		if !mob.Alive {
			delete(g.mobs, id)
			continue
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

	// Spawn entities
	g.spawnEntities(dt)

	// Broadcast state
	if g.tick%BroadcastEvery == 0 {
		g.broadcastState()
	}
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
			g.grid.InsertCircle(mob.X, mob.Y, MobRadius, EntityRef{Kind: 'm', Idx: idx})
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
			if CheckCollision(proj.X, proj.Y, ProjectileRadius, p.X, p.Y, PlayerRadius) {
				died := p.TakeDamage(ProjectileDamage)
				proj.Alive = false

				if died {
					p.Score -= DeathScorePenalty
					// Award kill to shooter
					if killer, ok := g.players[proj.OwnerID]; ok {
						killer.Score++
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
						// Killed by mob
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
	// Pre-convert all entities to state once, keeping raw positions for culling
	g.bcastPlayers = g.bcastPlayers[:0]
	for _, p := range g.players {
		g.bcastPlayers = append(g.bcastPlayers, playerWithPos{state: p.ToState(), x: p.X, y: p.Y})
	}
	g.bcastMobs = g.bcastMobs[:0]
	for _, mob := range g.mobs {
		if mob.Alive {
			g.bcastMobs = append(g.bcastMobs, mobWithPos{state: mob.ToState(), x: mob.X, y: mob.Y})
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

		state := GameState{
			Players:     g.filtPlayers,
			Projectiles: g.filtProjs,
			Mobs:        g.filtMobs,
			Asteroids:   g.filtAsteroids,
			Pickups:     g.filtPickups,
			Tick:        g.tick,
		}

		data, err := json.Marshal(Envelope{T: MsgState, Data: state})
		if err != nil {
			continue
		}
		playerData[playerID] = data
		client.SendRaw(data)
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
				}
				var err error
				fallbackData, err = json.Marshal(Envelope{T: MsgState, Data: st})
				if err != nil {
					continue
				}
			}
			data = fallbackData
		}
		client.SendRaw(data)
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
			if dist < MobRepelRadius && dist > 0.1 {
				// Check relative velocity for explosion
				rvx := a.VX - b.VX
				rvy := a.VY - b.VY
				relV := math.Sqrt(rvx*rvx + rvy*rvy)
				if relV > MobExplodeRelV {
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
				force := MobRepelForce * (1 - dist/MobRepelRadius)
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
	const queryR = ProjectileRadius + MobRadius
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
			if CheckCollision(proj.X, proj.Y, ProjectileRadius, mob.X, mob.Y, MobRadius) {
				died := mob.TakeDamage(ProjectileDamage)
				proj.Alive = false

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
				died := p.TakeDamage(p.HP)
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
	const queryR = AsteroidRadius + MobRadius
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
			if CheckCollision(ast.X, ast.Y, AsteroidRadius, mob.X, mob.Y, MobRadius) {
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
	const queryR = MobRadius + PlayerRadius
	for _, mob := range g.flatMobs {
		if !mob.Alive {
			continue
		}
		g.queryBuf = g.grid.QueryBuf(mob.X, mob.Y, queryR, g.queryBuf[:0])
		for _, ref := range g.queryBuf {
			if ref.Kind != 'p' {
				continue
			}
			p := g.flatPlayers[ref.Idx]
			if !p.Alive {
				continue
			}
			if CheckCollision(mob.X, mob.Y, MobRadius, p.X, p.Y, PlayerRadius) {
				// Mob always dies
				mob.Alive = false

				// Player takes collision damage
				died := p.TakeDamage(MobCollisionDmg)

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

// playerName returns a player's name or "Unknown"
func (g *Game) playerName(id string) string {
	if p, ok := g.players[id]; ok {
		return p.Name
	}
	return "Unknown"
}
