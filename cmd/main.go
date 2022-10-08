package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
)

const servAddress = ":8080"

var pathsToUrls = make(map[string]string)

type Redirection struct {
	ShortKey string
	Resource string
}

func createRedirection(w http.ResponseWriter, r *http.Request) {
	if contentType := r.Header.Get("Content-Type"); contentType != "" {
		if contentType != "application/json" {
			http.Error(w, "Content-Type must be set to application/json", http.StatusBadRequest)
			return
		}
	} else {
		http.Error(w, "Content-Type must be set to application/json", http.StatusBadRequest)
		return
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()

	var redirection Redirection
	err := decoder.Decode(&redirection)
	if err != nil {
		var syntaxError *json.SyntaxError
		var unmarshalTypeError *json.UnmarshalTypeError
		switch {
		case errors.As(err, &syntaxError):
			msg := fmt.Sprintf("Request body contains badly-formed JSON (at position %d)", syntaxError.Offset)
			http.Error(w, msg, http.StatusBadRequest)
		case errors.Is(err, io.ErrUnexpectedEOF):
			http.Error(w, "Request body contains badly-formed JSON", http.StatusBadRequest)
		case errors.As(err, &unmarshalTypeError):
			msg := fmt.Sprintf("Request body contains an invalid value for the %q field (at position %d)",
				unmarshalTypeError.Field, unmarshalTypeError.Offset)
			http.Error(w, msg, http.StatusBadRequest)
		case strings.HasPrefix(err.Error(), "json: unknown field "):
			fieldName := strings.TrimPrefix(err.Error(), "json: unknown field ")
			msg := fmt.Sprintf("Request body contains unknown field %s", fieldName)
			http.Error(w, msg, http.StatusBadRequest)
		case errors.Is(err, io.EOF):
			msg := "Request body must not be empty"
			http.Error(w, msg, http.StatusBadRequest)
		default:
			log.Print(err.Error())
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
		return
	}
	err = decoder.Decode(&struct{}{})
	if err != io.EOF {
		http.Error(w, "Request body must only contain a single JSON object", http.StatusBadRequest)
		return
	}
	pathsToUrls[redirection.ShortKey] = redirection.Resource
}

func main() {
	http.HandleFunc("/api/new", func(writer http.ResponseWriter, request *http.Request) {
		if request.Method == http.MethodPost {
			createRedirection(writer, request)
		} else {
			http.Error(writer, "Not allowed", http.StatusBadRequest)
		}
	})
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		path := request.URL.Path
		if resource, exists := pathsToUrls[path]; exists {
			http.Redirect(writer, request, resource, http.StatusSeeOther)
		} else {
			writer.Write([]byte("Not found"))
		}
	})
	http.ListenAndServe(servAddress, nil)
}
