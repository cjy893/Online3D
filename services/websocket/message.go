package websocket

import "time"

var Tasks TaskQueue

type Message struct {
	Type    string    `json:"type"`
	WorkID  uint      `json:"work_id"`
	Status  string    `json:"status"`
	Message string    `json:"message"`
	Time    time.Time `json:"time"`
	Error   string    `json:"error"`
}

type Task struct {
	ID        string
	Type      string
	Data      interface{}
	UserID    uint
	WorkID    uint
	StartTime time.Time
}

type TaskQueue struct {
	Tasks chan *Task
}
