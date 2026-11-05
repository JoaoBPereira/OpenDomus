package routes

import (
	"fmt"
	"math"
	"net/http"
	"strings"
	"time"

	"od/internal/models"
	"od/internal/postgres"
)

// logRow is a row of the public.log table (every column is nullable)
type logRow struct {
	Date    *time.Time `json:"date" db:"date"`
	Level   *string    `json:"level" db:"level"`
	Message *string    `json:"message" db:"message"`
}

func (h *Handlers) registerLogs(mux *http.ServeMux) {
	// Supports query params: ?offset=0&count=1000&level=error&keyword=text (defaults: offset=0, count=1000)
	HandleFunc(mux, "GET /logs", h.auth.Middleware(func(response http.ResponseWriter, request *http.Request) {
		query := request.URL.Query()
		count := max(1, min(intParam(query.Get("count"), 1000), 10000))
		offset := max(0, intParam(query.Get("offset"), 0))

		conditions := []string{}
		filters := []any{}
		if level := query.Get("level"); level != "" {
			filters = append(filters, level)
			conditions = append(conditions, fmt.Sprintf("level = $%d", len(filters)))
		}
		if keyword := query.Get("keyword"); keyword != "" {
			filters = append(filters, "%"+keyword+"%")
			conditions = append(conditions, fmt.Sprintf("message ILIKE $%d", len(filters)))
		}
		where := ""
		if len(conditions) > 0 {
			where = "WHERE " + strings.Join(conditions, " AND ")
		}

		var counts []postgres.CountRow
		if err := h.DB.QueryInto(request.Context(), &counts, "SELECT COUNT(*) AS total FROM public.log "+where, filters...); err != nil {
			SendError(response, err, "", "Error fetching logs: "+err.Error())
			return
		}
		total := 0
		if len(counts) > 0 {
			total = counts[0].Total
		}

		var rows []logRow
		dataQuery := fmt.Sprintf("SELECT * FROM public.log %s ORDER BY date DESC LIMIT $%d OFFSET $%d", where, len(filters)+1, len(filters)+2)
		if err := h.DB.QueryInto(request.Context(), &rows, dataQuery, append(filters, count, offset)...); err != nil {
			SendError(response, err, "", "Error fetching logs: "+err.Error())
			return
		}

		SendJSON(response, http.StatusOK, models.PaginatedResult{
			Rows: rows,
			Meta: models.PaginationMeta{
				Total:  total,
				Offset: offset,
				Count:  count,
				Pages:  max(0, int(math.Ceil(float64(total-offset)/float64(count)))),
			},
		})
	}))
}
