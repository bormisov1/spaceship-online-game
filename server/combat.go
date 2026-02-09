package main

// ApplyDamage applies damage to a player and returns true if they died
func ApplyDamage(player *Player, damage int) bool {
	return player.TakeDamage(damage)
}

// RespawnPlayer respawns a dead player
func RespawnPlayer(player *Player) {
	player.Respawn()
}
