package axle

import "net/http"

func Serve(w http.ResponseWriter) { w.WriteHeader(http.StatusOK) }
