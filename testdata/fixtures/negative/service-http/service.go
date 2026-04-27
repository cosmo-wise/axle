package service

import "net/http"

func Write(w http.ResponseWriter) { w.WriteHeader(http.StatusOK) }
