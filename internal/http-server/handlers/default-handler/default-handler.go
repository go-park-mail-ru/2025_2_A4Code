package default_handler

import (
	"fmt"
	"net/http"
)

// Заглушка до деплоя на сервер
type DefaultHandler struct {
	Name string
}

func (h *DefaultHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "URL:", r.URL.String())
}
