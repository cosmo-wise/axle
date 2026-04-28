package manualcrudtask

import "net/http"

func Mount(mux *http.ServeMux) {
	mux.HandleFunc("/tasks/{id}/update", func(http.ResponseWriter, *http.Request) {})
	mux.HandleFunc("/tasks/{id}/delete", func(http.ResponseWriter, *http.Request) {})
}
