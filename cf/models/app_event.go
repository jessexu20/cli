package models

import "time"

type EventFields struct {
	GUID        string
	Name        string
	Timestamp   time.Time
	Description string
	ActorName   string
}
