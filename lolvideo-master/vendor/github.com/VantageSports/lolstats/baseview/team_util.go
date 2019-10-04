package baseview

func TeamID(participantID int64) int64 {
	if participantID >= 1 && participantID <= 5 {
		return BlueTeam
	} else if participantID >= 6 && participantID <= 10 {
		return RedTeam
	}
	return 0
}

func EnemyTeamID(participantID int64) int64 {
	switch TeamID(participantID) {
	case BlueTeam:
		return RedTeam
	case RedTeam:
		return BlueTeam
	}
	return 0
}
