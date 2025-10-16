package domain

type Messages struct {
	MessageTotal  int       `json:"message_total"`
	MessageUnread int       `json:"message_unread"`
	Messages      []Message `json:"messages"`
}
