package handlers

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"

	"time"

	"github.com/h2p2f/architecture-sprint-8/backend/internal/entity"
)

func GenerateReports(w http.ResponseWriter, r *http.Request) {
	// Генерируем тестовые данные
	reports := make([]entity.Report, 0)
	categories := []string{"Category A", "Category B", "Category C"}
	dataTypes := []string{"Type 1", "Type 2", "Type 3"}

	for i := 0; i < 10; i++ {
		report := entity.Report{
			ID:       fmt.Sprintf("report-%d", i+1),
			UserID:   fmt.Sprintf("user-%d", rand.Intn(100)),
			Date:     time.Now().AddDate(0, 0, -rand.Intn(30)),
			DataType: dataTypes[rand.Intn(len(dataTypes))],
			Value:    rand.Float64() * 100,
			Category: categories[rand.Intn(len(categories))],
		}
		reports = append(reports, report)
	}

	response := entity.ReportResponse{
		Reports: reports,
		Total:   len(reports),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}
