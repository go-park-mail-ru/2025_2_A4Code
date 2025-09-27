package test_data

import (
	"fmt"
)

//type TestData struct {
//	resp map[string]interface{}
//}

type TestData map[string]interface{}

func New() TestData {
	messages := make([]map[string]interface{}, 10)

	senders := []map[string]string{
		{"email": "admin@example.com", "username": "Admin User", "avatar": ""},
		{"email": "support@company.com", "username": "Support Team", "avatar": ""},
		{"email": "news@site.org", "username": "Newsletter", "avatar": ""},
		{"email": "noreply@service.io", "username": "Service Bot", "avatar": ""},
		{"email": "boss@office.com", "username": "The Boss", "avatar": ""},
	}

	for i := 0; i < 10; i++ {
		sender := senders[i%len(senders)]

		message := map[string]interface{}{
			"id": fmt.Sprintf("%d", i+1),
			"sender": map[string]string{
				"email":    sender["email"],
				"username": sender["username"],
				"avatar":   sender["avatar"],
			},
			"subject":  fmt.Sprintf("Тема письма #%d", i+1),
			"snippet":  fmt.Sprintf("Краткое содержание письма #%d...", i+1),
			"datetime": "21.08.2025 21:25",
			"is_read":  i%3 != 0,
			"folder":   "0",
		}
		messages[i] = message
	}

	unread := 0
	for _, msg := range messages {
		if !msg["is_read"].(bool) {
			unread++
		}
	}

	result := map[string]interface{}{
		"status": "200",
		"body": map[string]interface{}{
			"message_total":  10,
			"message_unread": unread,
			"messages":       messages,
		},
	}

	return result
}
