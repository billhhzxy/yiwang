package tasks

import "time"

// StageDurations defines the spaced-repetition schedule.
// Index meaning:
// 0: +5m, 1: +10m, 2: +25m, 3: +1h, 4: +6h, 5: +24h, 6: +48h, 7: +168h (1 week)
var StageDurations = []time.Duration{
	5 * time.Minute,
	10 * time.Minute,
	25 * time.Minute,
	time.Hour,
	6 * time.Hour,
	24 * time.Hour,
	48 * time.Hour,
	168 * time.Hour,
}

// TotalStages returns how many spaced-repetition steps exist before completion.
func TotalStages() int {
	return len(StageDurations)
}
