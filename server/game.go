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
)

// Broadcaster interface for sending messages to clients
type Broadcaster interface {
	SendJSON(msg interface{})
}

// Game holds the state for one game session
type Game struct {
	mu          sync.RWMutex
	players     map[string]*Player
	projectiles map[string]*Projectile
	clients     map[string]Broadcaster // playerID -> client
	tick        uint64
	running     bool
	stop        chan struct{}
	nextShip    int
}

// NewGame creates a new Game
func NewGame() *Game {
	return &Game{
		players:     make(map[string]*Player),
		projectiles: make(map[string]*Projectile),
		clients:     make(map[string]Broadcaster),
		stop:        make(chan struct{}),
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
	ship := g.nextShip % 4
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

	// Check collisions
	g.checkCollisions()
	g.checkPlayerCollisions()

	// Broadcast state
	if g.tick%BroadcastEvery == 0 {
		g.broadcastState()
	}
}

// checkCollisions checks projectile-player collisions
func (g *Game) checkCollisions() {
	for projID, proj := range g.projectiles {
		if !proj.Alive {
			continue
		}
		for _, p := range g.players {
			if !p.Alive || p.ID == proj.OwnerID {
				continue
			}
			if CheckCollision(proj.X, proj.Y, ProjectileRadius, p.X, p.Y, PlayerRadius) {
				died := p.TakeDamage(ProjectileDamage)
				proj.Alive = false
				delete(g.projectiles, projID)

				if died {
					// Award kill to shooter
					if killer, ok := g.players[proj.OwnerID]; ok {
						killer.Score++
						// Notify all clients of the kill
						killMsg := Envelope{T: MsgKill, Data: KillMsg{
							KillerID:   killer.ID,
							KillerName: killer.Name,
							VictimID:   p.ID,
							VictimName: p.Name,
						}}
						g.broadcastMsg(killMsg)

						// Notify victim
						if client, ok := g.clients[p.ID]; ok {
							client.SendJSON(Envelope{T: MsgDeath, Data: DeathMsg{
								KillerID:   killer.ID,
								KillerName: killer.Name,
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
	players := make([]*Player, 0, len(g.players))
	for _, p := range g.players {
		if p.Alive {
			players = append(players, p)
		}
	}
	for i := 0; i < len(players); i++ {
		for j := i + 1; j < len(players); j++ {
			a, b := players[i], players[j]
			if !a.Alive || !b.Alive {
				continue
			}
			if CheckCollision(a.X, a.Y, PlayerRadius, b.X, b.Y, PlayerRadius) {
				a.TakeDamage(a.HP)
				b.TakeDamage(b.HP)

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

// broadcastState sends the current game state to all clients
func (g *Game) broadcastState() {
	state := GameState{
		Players:     make([]PlayerState, 0, len(g.players)),
		Projectiles: make([]ProjectileState, 0, len(g.projectiles)),
		Tick:        g.tick,
	}

	for _, p := range g.players {
		state.Players = append(state.Players, p.ToState())
	}
	for _, proj := range g.projectiles {
		state.Projectiles = append(state.Projectiles, proj.ToState())
	}

	data, err := json.Marshal(Envelope{T: MsgState, Data: state})
	if err != nil {
		return
	}

	for _, client := range g.clients {
		if c, ok := client.(*Client); ok {
			func() {
				defer func() { recover() }()
				select {
				case c.send <- data:
				default:
				}
			}()
		}
	}
}

// broadcastMsg sends a message to all clients in the session
func (g *Game) broadcastMsg(msg Envelope) {
	for _, client := range g.clients {
		client.SendJSON(msg)
	}
}
