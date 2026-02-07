package handlers

import (
	"net/http"
	"sort"
	"time"

	"BACK_SORTE_GO/internal/utils"

	"github.com/gorilla/mux"
)

type routeInfo struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

func (h *Handler) Health(router *mux.Router) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		now := time.Now().UTC()

		routes := make([]routeInfo, 0)
		_ = router.Walk(func(route *mux.Route, _ *mux.Router, _ []*mux.Route) error {
			path, err := route.GetPathTemplate()
			if err != nil {
				return nil
			}
			methods, err := route.GetMethods()
			if err != nil || len(methods) == 0 {
				return nil
			}
			for _, method := range methods {
				routes = append(routes, routeInfo{
					Method: method,
					Path:   path,
				})
			}
			return nil
		})

		sort.Slice(routes, func(i, j int) bool {
			if routes[i].Path == routes[j].Path {
				return routes[i].Method < routes[j].Method
			}
			return routes[i].Path < routes[j].Path
		})

		utils.RespondJSON(w, http.StatusOK, map[string]interface{}{
			"message":    "online",
			"utc_time":   now.Format("15:04:05"),
			"utc_date":   now.Format("2006-01-02"),
			"utc_now":    now.Format(time.RFC3339),
			"routes":     routes,
			"routeCount": len(routes),
		})
	}
}
