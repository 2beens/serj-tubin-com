package events

import (
	"fmt"
	"time"
)

type TrainingStart struct {
	ID        int       `json:"id"`
	Timestamp time.Time `json:"timestamp"`
}

type TrainingFinish struct {
	ID        int       `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Calories  int       `json:"calories"`
}

type WeightReport struct {
	ID        int       `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Weight    int       `json:"weight"`
}

type PainReport struct {
	ID        int       `json:"id"`
	Timestamp time.Time `json:"timestamp"`
	Level     int       `json:"level"`
	Location  string    `json:"location"`
}

// Event (DB level type) can be used to send events from the ios app, such as:
//   - training started (with timestamp)
//   - training finished (with timestamp, calories burned, etc.)
//   - weight report (with timestamp and weight in kilos)
//   - pain report (with timestamp, pain level, pain location, etc.)
type Event struct {
	ID        int               `json:"id"`
	Type      EventType         `json:"type"`
	Timestamp time.Time         `json:"timestamp"`
	Data      map[string]string `json:"data"`
}

func NewTrainingStartEvent(ts TrainingStart) Event {
	return Event{
		ID:        ts.ID,
		Type:      EventTypeTrainingStarted,
		Timestamp: ts.Timestamp,
		Data:      map[string]string{},
	}
}

func NewTrainingFinishEvent(tf TrainingFinish) Event {
	return Event{
		ID:        tf.ID,
		Type:      EventTypeTrainingFinished,
		Timestamp: tf.Timestamp,
		Data: map[string]string{
			"calories": fmt.Sprintf("%d", tf.Calories),
		},
	}
}

func NewWeightReportEvent(wr WeightReport) Event {
	return Event{
		ID:        wr.ID,
		Type:      EventTypeWeightReport,
		Timestamp: wr.Timestamp,
		Data: map[string]string{
			"weight": fmt.Sprintf("%d", wr.Weight),
		},
	}
}

func NewPainReportEvent(pr PainReport) Event {
	return Event{
		ID:        pr.ID,
		Type:      EventTypePainReport,
		Timestamp: pr.Timestamp,
		Data: map[string]string{
			"level":    fmt.Sprintf("%d", pr.Level),
			"location": pr.Location,
		},
	}
}

// EventType can be one of:
//   - training_started
//   - training_finished
//   - weight_report
//   - pain_report
type EventType string

const (
	EventTypeTrainingStarted  EventType = "training_started"
	EventTypeTrainingFinished EventType = "training_finished"
	EventTypeWeightReport     EventType = "weight_report"
	EventTypePainReport       EventType = "pain_report"
)

func (et EventType) String() string {
	return string(et)
}

func (et EventType) IsValid() bool {
	switch et {
	case EventTypeTrainingStarted,
		EventTypeTrainingFinished,
		EventTypeWeightReport,
		EventTypePainReport:
		return true
	default:
		return false
	}
}
