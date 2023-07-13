package repo

import "time"

// Event can be used to send events from the ios app, such as:
//   - training started (with timestamp)
//   - training finished (with timestamp, calories burned, etc.)
//   - weight report (with timestamp and weight in kilos)
//   - pain report (with timestamp, pain level, pain location, etc.)
type Event struct {
	ID        int               `json:"id"`
	Type      EventType         `json:"type"`
	CreatedAt time.Time         `json:"createdAt"`
	Metadata  map[string]string `json:"metadata"`
}

// EventType can be one of:
//   - training_started
//   - training_finished
//   - weight_report
//   - pain_report
type EventType string

const (
	TrainingStarted  EventType = "training_started"
	TrainingFinished EventType = "training_finished"
	WeightReport     EventType = "weight_report"
	PainReport       EventType = "pain_report"
)
