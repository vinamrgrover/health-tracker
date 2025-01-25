package discipline

import (
	"context"
	"database/sql"
	"github.com/redis/go-redis/v9"
)

type Health struct {
	Db *sql.DB
}

type Database struct {
	Db     *sql.DB
	Ctx    context.Context
	Cancel context.CancelFunc
}

type Queue struct {
	Client *redis.Client
	Key    string
	Ctx    context.Context
}

type Trigger struct {
	Id         string `json:"id"`
	Timestamp  string `json:"timestamp"`
	Type       string `json:"type"`
	Intensity  int    `json:"intensity"`
	Compulsion bool   `json:"compulsion"`
	Notes      string `json:"notes"`
}

type Action struct {
	Id        string `json:"id"`
	Timestamp string `json:"timestamp"`
	Type      string `json:"type"`
	Relief    bool   `json:"relief"`
	Duration  int    `json:"duration"`
	Notes     string `json:"notes"`
}

type Overall struct {
	Id           string `json:"id"`
	Timestamp    string `json:"timestamp"`
	CleanStreak  int    `json:"clean_streak"`
	GymSession   bool   `json:"gym_session"`
	CodingHours  int    `json:"coding_hours"`
	ReadingHours int    `json:"reading_hours"`
	MoodScore    int    `json:"mood_score"`
}
