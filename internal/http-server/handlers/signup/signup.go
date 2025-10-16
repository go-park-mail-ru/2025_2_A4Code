package signup

import (
	"2025_2_a4code/internal/domain"
	resp "2025_2_a4code/internal/lib/api/response"

	profileUcase "2025_2_a4code/internal/usecase/profile"
	"encoding/json"
	"net/http"
)

type SignupResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

type Response struct {
	resp.Response
	Body SignupResponse `json:"body,omitempty"`
}

type HandlerSignup struct {
	profileUCase *profileUcase.ProfileUcase
}

func New(ucBP *profileUcase.ProfileUcase) *HandlerSignup {
	return &HandlerSignup{profileUCase: ucBP}
}

func (h *HandlerSignup) ServeHTTP(w http.ResponseWriter, r *http.Request) {

	if r.Method != http.MethodPost {
		sendErrorResponse(w, "Неправильный метод", http.StatusMethodNotAllowed)
		return
	}

	var credentials domain.Profile
	if err := json.NewDecoder(r.Body).Decode(&credentials); err != nil {
		sendErrorResponse(w, "Неправильный запрос", http.StatusBadRequest)
		return
	}

	// // проверка логина
	// for _, user := range handlers2.users {
	// 	if credentials.Login == user["login"] {
	// 		http.Error(w, "Пользователь с таким логином уже существует", http.StatusUnauthorized)
	// 		return
	// 	}
	// }

	// newUser := map[string]string{
	// 	"login":       credentials.Login,
	// 	"password":    credentials.Password,
	// 	"username":    credentials.Username,
	// 	"dateofbirth": credentials.DateOfBirth,
	// 	"gender":      credentials.Gender,
	// }

	// //записываем в мап
	// handlers2.users = append(handlers2.users, newUser)

	// // создаем токен
	// token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
	// 	"login": credentials.Login,
	// 	"exp":   time.Now().Add(24 * time.Hour).Unix(),
	// })

	// // подписываем
	// session, err := token.SignedString(handlers2.SECRET)

	// if err != nil {
	// 	http.Error(w, "Ошибка регистрации", http.StatusInternalServerError)
	// 	return
	// }

	// cookie := &http.Cookie{
	// 	Name:     "session_id",
	// 	Value:    session,
	// 	MaxAge:   3600,
	// 	HttpOnly: true,
	// 	Path:     "/",
	// }

	// // ставим куки
	// http.SetCookie(w, cookie)

	// w.Header().Set("Content-Type", "application/json")
	// err = json.NewEncoder(w).Encode(map[string]any{
	// 	"status": "200",
	// 	"body": struct {
	// 		Message string `json:"message"`
	// 	}{"Пользователь зарегистрирован"},
	// })

	// if err != nil {
	// 	http.Error(w, "Внутренняя ошибка сервера", http.StatusInternalServerError)
	// 	return
	// }
}

func sendErrorResponse(w http.ResponseWriter, errorMsg string, statusCode int) {

	response := Response{
		Response: resp.Response{
			Status: http.StatusText(statusCode),
			Error:  errorMsg,
		},
	}

	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(&response)
}
