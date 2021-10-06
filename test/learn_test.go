package test

import (
	"net/http"
	"testing"
)

func TestName(t *testing.T) {
	http.HandleFunc("/", func(writer http.ResponseWriter, request *http.Request) {
		writer.WriteHeader(404)
	})

	err := http.ListenAndServe(":8081", nil)
	if err != nil {
		return
	}

	select {}
}
