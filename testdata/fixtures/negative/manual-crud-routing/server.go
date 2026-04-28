package manualcrud

import "net/http"

func Mount(mux *http.ServeMux) {
	mux.HandleFunc("/resources", func(http.ResponseWriter, *http.Request) {})
	mux.HandleFunc("/resources/{id}/delete", func(http.ResponseWriter, *http.Request) {})
}
