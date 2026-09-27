package task

import (
	"time"

	"github.com/google/uuid"
)

// Report is a stage report stored for a task when a column agent finishes.
type Report struct {
	ColumnID  uuid.UUID `json:"column_id"`
	ReportMD  string    `json:"report_md"`
	CreatedAt time.Time `json:"created_at"`
}
