package main

// Achievement definitions
type AchievementDef struct {
	ID          string
	Name        string
	Description string
}

var Achievements = []AchievementDef{
	{"first_blood", "First Blood", "Get your first kill"},
	{"sharpshooter", "Sharpshooter", "Reach 100 total kills"},
	{"centurion", "Centurion", "Reach 1000 total kills"},
	{"ace", "Ace Pilot", "Get 10 kills in a single match"},
	{"flawless", "Flawless Victory", "Win a match without dying"},
	{"victor", "Victor", "Win 10 matches"},
	{"veteran", "Veteran", "Reach level 10"},
	{"elite", "Elite", "Reach level 25"},
	{"legend", "Legend", "Reach level 50"},
	{"survivor", "Survivor", "Play for 1 hour total"},
}

// CheckAchievements checks if any new achievements should be unlocked for a player.
// Returns a list of newly unlocked achievement IDs.
func CheckAchievements(db *DB, playerID int64, matchKills, matchDeaths int, won bool) []AchievementDef {
	if db == nil {
		return nil
	}

	stats, err := db.GetStats(playerID)
	if err != nil || stats == nil {
		return nil
	}

	existing, err := db.GetAchievements(playerID)
	if err != nil {
		return nil
	}
	has := make(map[string]bool, len(existing))
	for _, a := range existing {
		has[a] = true
	}

	var unlocked []AchievementDef

	check := func(id string) bool {
		if has[id] {
			return false
		}
		switch id {
		case "first_blood":
			return stats.Kills >= 1
		case "sharpshooter":
			return stats.Kills >= 100
		case "centurion":
			return stats.Kills >= 1000
		case "ace":
			return matchKills >= 10
		case "flawless":
			return won && matchDeaths == 0
		case "victor":
			return stats.Wins >= 10
		case "veteran":
			return stats.Level >= 10
		case "elite":
			return stats.Level >= 25
		case "legend":
			return stats.Level >= 50
		case "survivor":
			return stats.Playtime >= 3600
		}
		return false
	}

	for _, def := range Achievements {
		if check(def.ID) {
			if newlyUnlocked, err := db.UnlockAchievement(playerID, def.ID); err == nil && newlyUnlocked {
				unlocked = append(unlocked, def)
			}
		}
	}

	return unlocked
}
