package ingest

import (
	"math"

	"github.com/VantageSports/lolstats/baseview"
)

func CalculateCarryFocusEfficiency(teamFights []*TeamFight, rolePositions map[int64]baseview.RolePosition, matchDuration float64) float64 {
	numFights := 0.0
	totalCarryFocus := 0.0
	for _, tf := range teamFights {
		// Ignore early game fights
		if matchDuration < 25*60 && tf.Begin < 15*60 {
			continue
		} else if tf.Begin < 20*60 {
			continue
		}

		// Ignore fights where there's only one enemy
		if len(tf.EnemiesInVision) == 1 {
			continue
		}

		highestCarryFocus := 0.0
		foundCarry := false
		// Look for fights where the enemy carries took damage.
		// These represent fights in which it's possible to focus the carry
		for enemyId, damageTaken := range tf.EnemyDamageTaken {
			// Ignore enemies that took no damage
			if damageTaken == 0 {
				continue
			}

			if rolePositions[enemyId] == baseview.RoleMid || rolePositions[enemyId] == baseview.RoleAdc {
				foundCarry = true
				// How much damage did you do, versus how much damage did they take in total?
				carryFocus := tf.DamageDealt[enemyId] / damageTaken

				// If you damaged multiple carries, then pick the higher of the two
				if carryFocus > highestCarryFocus {
					highestCarryFocus = carryFocus
				}
			}
		}

		if foundCarry {
			numFights++
			totalCarryFocus += highestCarryFocus
		}

	}
	return totalCarryFocus / math.Max(numFights, 1)
}
