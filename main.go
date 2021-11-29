package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	//"github.com/BurntSushi/toml"
	"github.com/go-redis/redis/v8"
	"github.com/rs/cors"
	"golang.org/x/crypto/acme/autocert"
)

var ctx = context.Background()
var rdb = redis.NewClient(&redis.Options{
	Addr:     "localhost:6379",
	Password: "",
	DB:       0,
})

type analyticsData struct {
	HitType       string `json:"hit_type"` // тип события
	PageType      string `json:"page_type"`
	MaterialPK    int    `json:"material_pk"`
	EventCategory string `json:"event_category"`
	EventAction   string `json:"event_action"`
	EventLabel    int    `json:"event_label"`
	EventValue    string `json:"event_value"`
	Email         string `json:"email"`
}

type unsuccessfulJSONResponse struct {
	Sucess       bool   `json: "success`
	ErrorMessage string `json: "errorMessage`
}

type successfulJSONResponse struct {
	Sucess       bool   `json: "success`
	ErrorMessage string `json: "errorMessage`
}

/* Finds val in slice, if finded - returns index and true, otherwise -1 and false */
func Find(slice []string, val string) (int, bool) {
	for i, item := range slice {
		if item == val {
			return i, true
		}

	}
	return -1, false
}

func main() {
	mux := http.NewServeMux()
	corsMiddleware := cors.New(cors.Options{
		//AllowedOrigins: {`toml:"database_url"`},
		AllowedOrigins: []string{`toml:"database_url"`},
	})

	mux.HandleFunc("/send", analyticsHandler)

	handler := corsMiddleware.Handler(mux)
	log.Fatal(http.Serve(autocert.NewListener("analytics.istories.media"), handler))
}

func ProcessMaterialView(materialPK int) {
	_, err := rdb.Incr(ctx, fmt.Sprintf("material_views_%d", materialPK)).Result()
	if err != nil {
		panic(err)
	}
}

//Handles successful Donate
func ProcessSucessfulDonate(materialPK int, email string) {
	_, err := rdb.Append(ctx,
		fmt.Sprintf("donaters_of_material_%d", materialPK),
		fmt.Sprint("%v:", email)).Result()
	if err != nil {
		panic(err)
	}
}

/* Handler for analytics web service */
func analyticsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")

	decoder := json.NewDecoder(r.Body)
	var analyticsData analyticsData

	err := decoder.Decode(&analyticsData)

	if err != nil {
		writeUnSuccessfulResponse(w, "Can not parse JSON"+err.Error())
		return
	}

	if analyticsData.HitType == "page-view" {
		ProcessMaterialView(analyticsData.MaterialPK)
		writeSuccessfulResponse(w, "")
	} else if analyticsData.HitType == "event" {
		if analyticsData.EventCategory != "donations" {
			writeUnSuccessfulResponse(w, "Unknown event_category")
			return
		}
		_, eventActionExists := Find([]string{"submit", "success", "failure"}, analyticsData.EventAction)
		//If we don't have submit, success or failure in EventAction - it's error
		if !eventActionExists {
			writeUnSuccessfulResponse(w, "unknown event_action")
			return
		}

		if analyticsData.EventAction == "success" {
			ProcessSucessfulDonate(analyticsData.EventLabel, analyticsData.Email)
		}
		writeSuccessfulResponse(w, "")
		return
	} else {
		writeUnSuccessfulResponse(w, "Unknown hit_type")
		return
	}

}

//Writes in httpResponseWriter JSON with success: false and custom Error Message
func writeUnSuccessfulResponse(w http.ResponseWriter, errorMsg string) {
	response, _ := json.Marshal(&unsuccessfulJSONResponse{
		Sucess:       false,
		ErrorMessage: errorMsg,
	})

	w.WriteHeader(http.StatusBadRequest)
	w.Write(response)
}

//Writes in httpResponseWriter JSON with success: true and custom optional message
func writeSuccessfulResponse(w http.ResponseWriter, message string) {
	w.WriteHeader(http.StatusOK)
	if message == "" {
		w.Write([]byte(`{"success": true}`))
	} else {
		response, _ := json.Marshal(&successfulJSONResponse{
			Sucess:       false,
			ErrorMessage: message,
		})
		w.Write(response)
	}

}
