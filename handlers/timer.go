package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"goofytime/middleware"
	"goofytime/models"
)

func GetTimerStatus(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	timer, err := models.GetActiveTimer(user.ID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("null"))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(timer)
}

func SaveTimer(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)

	startMs, _ := strconv.ParseInt(r.FormValue("start_ms"), 10, 64)
	elapsedMs, _ := strconv.ParseInt(r.FormValue("elapsed_ms"), 10, 64)
	isRunning := r.FormValue("is_running") == "true"
	purpose := r.FormValue("purpose")
	clientIDStr := r.FormValue("client_id")

	var clientID *int
	if clientIDStr != "" {
		id, err := strconv.Atoi(clientIDStr)
		if err == nil && id > 0 {
			clientID = &id
		}
	}

	models.SaveActiveTimer(user.ID, startMs, elapsedMs, isRunning, purpose, clientID)
	w.WriteHeader(http.StatusOK)
}

func DeleteTimer(w http.ResponseWriter, r *http.Request) {
	user := middleware.GetUser(r)
	models.DeleteActiveTimer(user.ID)
	w.WriteHeader(http.StatusOK)
}
