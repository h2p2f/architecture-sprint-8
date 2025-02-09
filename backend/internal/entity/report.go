package entity

import (
	"time"
)

type Report struct {
	ID       string    `json:"id"`
	UserID   string    `json:"userId"`
	Date     time.Time `json:"date"`
	DataType string    `json:"dataType"`
	Value    float64   `json:"value"`
	Category string    `json:"category"`
}

type ReportResponse struct {
	Reports []Report `json:"reports"`
	Total   int      `json:"total"`
}
