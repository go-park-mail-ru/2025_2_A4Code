package message_repository

import (
	"2025_2_a4code/internal/domain"
	e "2025_2_a4code/internal/lib/wrapper"
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strconv"
	"strings"
	"time"
)

type MessageRepository struct {
	db *sql.DB
}

func New(db *sql.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

func buildSnippet(text string, limit int) string {
	if limit <= 0 {
		return ""
	}

	runes := []rune(text)
	if len(runes) <= limit {
		return text
	}

	return string(runes[:limit]) + "..."
}

func (repo *MessageRepository) FindByMessageID(ctx context.Context, messageID int64) (*domain.Message, error) {
	const op = "storage.postgresql.message.FindByMessageID"

	const query = `
        SELECT
            m.id, m.topic, m.text, m.date_of_dispatch,
            bp.id, bp.username, bp.domain,
            p.name, p.surname, p.image_path,
            pm.read_status
        FROM
            message m
        JOIN
            base_profile bp ON m.sender_base_profile_id = bp.id
        LEFT JOIN
            profile p ON bp.id = p.base_profile_id
        LEFT JOIN
            profile_message pm ON m.id = pm.message_id
        WHERE
            m.id = $1`

	stmt, err := repo.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, e.Wrap(op, err)
	}
	defer stmt.Close()

	var message domain.Message
	var messageIdInt int64
	var senderId int64
	var senderUsername, senderDomain, text string
	var senderName, senderSurname, senderAvatar sql.NullString

	err = stmt.QueryRowContext(ctx, messageID).Scan(
		&messageIdInt, &message.Topic, &text, &message.Datetime,
		&senderId, &senderUsername, &senderDomain,
		&senderName, &senderSurname, &senderAvatar,
		&message.IsRead,
	)

	// Создаем snippet из текста сообщения
	message.Snippet = buildSnippet(text, 40)

	message.Sender = domain.Sender{
		Id:    senderId,
		Email: fmt.Sprintf("%s@%s", senderUsername, senderDomain),
		Username: strings.TrimSpace(fmt.Sprintf("%s %s",
			senderName.String, senderSurname.String)),
		Avatar: senderAvatar.String,
	}

	return &message, nil
}

func (repo *MessageRepository) FindFullByMessageID(ctx context.Context, messageID int64, profileID int64) (domain.FullMessage, error) {
	const op = "storage.postgresql.message.FindFullByMessageID"

	const query = `
        SELECT
            m.id, m.topic, m.text, m.date_of_dispatch,
            bp.id, bp.username, bp.domain,
            p.name, p.surname, p.image_path,
            t.id, t.root_message_id,
            f.id, f.profile_id, f.folder_name, f.folder_type
        FROM
            message m
        JOIN
            base_profile bp ON m.sender_base_profile_id = bp.id
        LEFT JOIN
            profile p ON bp.id = p.base_profile_id
        LEFT JOIN
            thread t ON m.thread_id = t.id
        LEFT JOIN
            folder_profile_message fpm ON m.id = fpm.message_id
        LEFT JOIN
            folder f ON fpm.folder_id = f.id AND f.profile_id = $2
        WHERE
            m.id = $1
        LIMIT 1`

	var msg domain.FullMessage
	var messageIdInt int64
	var senderId int64
	var senderUsername, senderDomain string
	var senderName, senderSurname, senderAvatar sql.NullString
	var threadID, threadRootID sql.NullInt64
	var folderID, folderProfileID sql.NullInt64
	var folderName, folderType sql.NullString

	err := repo.db.QueryRowContext(ctx, query, messageID, profileID).Scan(
		&messageIdInt, &msg.Topic, &msg.Text, &msg.Datetime,
		&senderId, &senderUsername, &senderDomain,
		&senderName, &senderSurname, &senderAvatar,
		&threadID, &threadRootID,
		&folderID, &folderProfileID, &folderName, &folderType,
	)
	if err != nil {
		return domain.FullMessage{}, e.Wrap(op, err)
	}

	msg.ID = strconv.FormatInt(messageIdInt, 10)

	msg.Sender = domain.Sender{
		Id:    senderId,
		Email: fmt.Sprintf("%s@%s", senderUsername, senderDomain),
		Username: strings.TrimSpace(fmt.Sprintf("%s %s",
			senderName.String, senderSurname.String)),
		Avatar: senderAvatar.String,
	}

	// Обработка thread_id и root_message_id
	if threadID.Valid {
		msg.ThreadRoot = strconv.FormatInt(threadID.Int64, 10)
	}

	// Обработка информации о папке
	if folderID.Valid {
		msg.Folder = domain.Folder{
			ID:        folderID.Int64,
			ProfileID: folderProfileID.Int64,
			Name:      folderName.String,
			Type:      domain.FolderType(folderType.String),
		}
	}

	// Получаем файлы
	rows, err := repo.db.QueryContext(ctx, `
        SELECT id, file_type, size, storage_path, message_id 
        FROM file 
        WHERE message_id = $1`, messageID)
	if err != nil {
		return domain.FullMessage{}, e.Wrap(op, err)
	}
	defer rows.Close()

	var files []domain.File
	for rows.Next() {
		var file domain.File
		err := rows.Scan(&file.ID, &file.FileType, &file.Size, &file.StoragePath, &file.MessageID)
		if err != nil {
			return domain.FullMessage{}, e.Wrap(op, err)
		}
		files = append(files, file)
	}

	msg.Files = files

	return msg, nil
}

func (repo *MessageRepository) FindByProfileID(ctx context.Context, profileID int64) ([]domain.Message, error) {
	const op = "storage.postgresql.message.FindByProfileID"

	const query = `
        SELECT
            m.id, m.topic, m.text, m.date_of_dispatch,
            pm.read_status,
            bp.id, bp.username, bp.domain,
            sender_profile.name, sender_profile.surname, sender_profile.image_path
        FROM
            message m
        JOIN
            profile_message pm ON m.id = pm.message_id
        JOIN
            profile recipient_profile ON pm.profile_id = recipient_profile.id
        LEFT JOIN
            folder_profile_message fpm ON m.id = fpm.message_id
        LEFT JOIN
            folder f ON fpm.folder_id = f.id AND f.profile_id = recipient_profile.id
        JOIN
            base_profile bp ON m.sender_base_profile_id = bp.id
        LEFT JOIN
            profile sender_profile ON bp.id = sender_profile.base_profile_id
        WHERE
            recipient_profile.base_profile_id = $1
        GROUP BY 
            m.id, 
            m.topic, 
            m.text, 
            m.date_of_dispatch,
            pm.read_status,
            bp.id, 
            bp.username, 
            bp.domain,
            sender_profile.name, 
            sender_profile.surname, 
            sender_profile.image_path
        ORDER BY
            m.date_of_dispatch DESC`

	stmt, err := repo.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, e.Wrap(op, err)
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, profileID)
	if err != nil {
		return nil, e.Wrap(op+": failed to execute query: ", err)
	}
	defer rows.Close()
	var messages []domain.Message
	for rows.Next() {
		var message domain.Message
		var messageIdInt int64
		var senderId int64
		var senderUsername, senderDomain string
		var senderName, senderSurname, senderAvatar sql.NullString
		var text string

		err := rows.Scan(
			&messageIdInt, &message.Topic, &text, &message.Datetime, &message.IsRead,
			&senderId, &senderUsername, &senderDomain,
			&senderName, &senderSurname, &senderAvatar,
		)
		if err != nil {
			return nil, e.Wrap(op, err)
		}
		message.ID = strconv.FormatInt(messageIdInt, 10)
		message.Snippet = buildSnippet(text, 40)
		message.Sender = domain.Sender{
			Id:    senderId,
			Email: fmt.Sprintf("%s@%s", senderUsername, senderDomain),
			Username: strings.TrimSpace(fmt.Sprintf("%s %s",
				senderName.String, senderSurname.String)),
			Avatar: senderAvatar.String,
		}
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		return nil, e.Wrap(op, err)
	}

	return messages, nil
}

func (repo *MessageRepository) FindByProfileIDWithKeysetPagination(
	ctx context.Context,
	profileID int64,
	lastMessageID int64,
	lastDatetime time.Time,
	limit int,
) ([]domain.Message, error) {

	const op = "storage.postgresql.message.FindByProfileID"

	const query = `
        SELECT
            m.id, m.topic, m.text, m.date_of_dispatch,
            pm.read_status,
            bp.id, bp.username, bp.domain,
            sender_profile.name, sender_profile.surname, sender_profile.image_path
        FROM
            message m
        JOIN
            profile_message pm ON m.id = pm.message_id
        JOIN
            profile recipient_profile ON pm.profile_id = recipient_profile.id
        LEFT JOIN
            folder_profile_message fpm ON m.id = fpm.message_id
        LEFT JOIN
            folder f ON fpm.folder_id = f.id AND f.profile_id = recipient_profile.id
        JOIN
            base_profile bp ON m.sender_base_profile_id = bp.id
        LEFT JOIN
            profile sender_profile ON bp.id = sender_profile.base_profile_id
        WHERE
            recipient_profile.base_profile_id = $1
			AND (($2 = 0 AND $3 = 0) OR (m.date_of_dispatch, m.id) < (to_timestamp($3), $2))
        GROUP BY 
            m.id, 
            m.topic, 
            m.text, 
            m.date_of_dispatch,
            pm.read_status,
            bp.id, 
            bp.username, 
            bp.domain,
            sender_profile.name, 
            sender_profile.surname, 
            sender_profile.image_path
        ORDER BY
            m.date_of_dispatch DESC, m.id DESC
		FETCH FIRST $4 ROWS ONLY`

	stmt, err := repo.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, e.Wrap(op, err)
	}
	defer stmt.Close()

	var lastDatetimeUnix int64
	if !lastDatetime.IsZero() {
		lastDatetimeUnix = lastDatetime.Unix()
	}

	rows, err := stmt.QueryContext(ctx, profileID, lastMessageID, lastDatetimeUnix, limit)
	if err != nil {
		return nil, e.Wrap(op+": failed to execute query: ", err)
	}
	defer rows.Close()

	var messages []domain.Message
	for rows.Next() {
		var message domain.Message
		var messageIdInt int64
		var senderId int64
		var senderUsername, senderDomain string
		var senderName, senderSurname, senderAvatar sql.NullString
		var text string

		err := rows.Scan(
			&messageIdInt, &message.Topic, &text, &message.Datetime, &message.IsRead,
			&senderId, &senderUsername, &senderDomain,
			&senderName, &senderSurname, &senderAvatar,
		)
		if err != nil {
			return nil, e.Wrap(op, err)
		}
		message.ID = strconv.FormatInt(messageIdInt, 10)
		message.Snippet = buildSnippet(text, 40)
		message.Sender = domain.Sender{
			Id:    senderId,
			Email: fmt.Sprintf("%s@%s", senderUsername, senderDomain),
			Username: strings.TrimSpace(fmt.Sprintf("%s %s",
				senderName.String, senderSurname.String)),
			Avatar: senderAvatar.String,
		}
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		return nil, e.Wrap(op, err)
	}

	return messages, nil
}

func (repo *MessageRepository) SaveMessage(ctx context.Context, receiverProfileEmail string, senderBaseProfileID int64, topic, text string) (messageID int64, err error) {
	const op = "storage.postgresql.message.SaveMessage"

	tx, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, e.Wrap(op+": failed to begin transaction: ", err)
	}
	defer tx.Rollback()

	// Вставка сообщения
	const insertMessage = `
		INSERT INTO message (topic, text, date_of_dispatch, sender_base_profile_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id`

	err = tx.QueryRowContext(ctx, insertMessage, topic, text, time.Now(), senderBaseProfileID).Scan(&messageID)
	if err != nil {
		return 0, e.Wrap(op+": failed to insert message: ", err)
	}

	// Получение receiver_profile_id
	username := strings.Split(receiverProfileEmail, "@")[0]
	domain := strings.Split(receiverProfileEmail, "@")[1]

	var receiverProfileID int64
	err = tx.QueryRowContext(ctx, `
		SELECT p.id 
		FROM profile p
		JOIN base_profile bp ON p.base_profile_id = bp.id
		WHERE bp.username = $1 AND bp.domain = $2`,
		username, domain).Scan(&receiverProfileID)
	if err != nil {
		return 0, e.Wrap(op+": failed to get receiver id: ", err)
	}

	// Добавление в inbox получателя
	const insertFolder = `
		INSERT INTO folder_profile_message (message_id, folder_id)
		SELECT $1, f.id
		FROM folder f
		WHERE f.profile_id = $2 AND f.folder_type = 'inbox'`

	_, err = tx.ExecContext(ctx, insertFolder, messageID, receiverProfileID)
	if err != nil {
		return 0, e.Wrap(op+": failed to insert to inbox: ", err)
	}

	// Добавление связи сообщение-профиль
	const insertProfileMessage = `
		INSERT INTO profile_message (profile_id, message_id, read_status)
		VALUES ($1, $2, false)`

	_, err = tx.ExecContext(ctx, insertProfileMessage, receiverProfileID, messageID)
	if err != nil {
		return 0, e.Wrap(op+": failed to insert profile message: ", err)
	}

	err = tx.Commit()
	if err != nil {
		return 0, e.Wrap(op+": failed to commit transaction: ", err)
	}

	return messageID, nil
}

func (repo *MessageRepository) SaveFile(ctx context.Context, messageID int64, fileName, fileType, storagePath string, size int64) (fileID int64, err error) {
	const op = "storage.postgresql.message.SaveFile"

	const query = `
		INSERT INTO file (file_type, size, storage_path, message_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		Returning id`
	stmt, err := repo.db.PrepareContext(ctx, query)
	if err != nil {
		return 0, e.Wrap(op, err)
	}
	defer stmt.Close()

	timeNow := time.Now()
	err = stmt.QueryRowContext(ctx, fileType, size, storagePath, messageID, timeNow, timeNow).Scan(&fileID)
	if err != nil {
		return 0, e.Wrap(op+": failed to execute query: ", err)
	}

	return fileID, nil
}

func (repo *MessageRepository) SaveThread(ctx context.Context, messageID int64) (threadID int64, err error) {
	const op = "storage.postgresql.message.SaveThread"

	const query = `
		INSERT INTO thread (root_message_id, created_at, updated_at)
		VALUES ($1, $2, $3)
		RETURNING id`

	stmt, err := repo.db.PrepareContext(ctx, query)
	if err != nil {
		return 0, e.Wrap(op, err)
	}
	defer stmt.Close()

	timeNow := time.Now()
	err = stmt.QueryRowContext(ctx, messageID, timeNow, timeNow).Scan(&threadID)
	if err != nil {
		return 0, e.Wrap(op, err)
	}

	return threadID, nil
}

func (repo *MessageRepository) SaveThreadIdToMessage(ctx context.Context, messageID int64, threadID int64) error {
	const op = "storage.postgresql.message.SaveThreadIdToMessage"

	const query = `
        UPDATE message
        SET thread_id = $1, updated_at = $2
        WHERE Id = $3`
	stmt, err := repo.db.PrepareContext(ctx, query)
	if err != nil {
		return e.Wrap(op, err)
	}
	defer stmt.Close()
	timeNow := time.Now()
	res, err := stmt.ExecContext(ctx, threadID, timeNow, messageID)
	if err != nil {
		return e.Wrap(op+": failed to execute query: ", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return e.Wrap(op, err)
	}
	if rowsAffected == 0 {
		return e.Wrap(op+": failed to insert to message (no rows affected): ", err)
	}
	return nil
}

func (repo *MessageRepository) GetMessagesStats(ctx context.Context, profileID int64) (int, int, error) {
	const op = "storage.postgresql.message.GetMessagesStats"

	const query = `
        SELECT 
            COUNT(DISTINCT pm.message_id) as total_count,
            COUNT(DISTINCT CASE WHEN pm.read_status = false THEN pm.message_id END) as unread_count
        FROM profile_message pm
        JOIN profile p ON pm.profile_id = p.id
        WHERE p.base_profile_id = $1`

	slog.Warn(fmt.Sprintf("searching by id = %d", profileID))

	stmt, err := repo.db.PrepareContext(ctx, query)
	if err != nil {
		return 0, 0, e.Wrap(op, err)
	}
	defer stmt.Close()

	var totalCount, unreadCount int
	err = stmt.QueryRowContext(ctx, profileID).Scan(&totalCount, &unreadCount)
	if err != nil {
		return 0, 0, e.Wrap(op, err)
	}

	return totalCount, unreadCount, nil
}

func (repo *MessageRepository) MarkMessageAsRead(ctx context.Context, messageID int64, profileID int64) error {
	const op = "storage.postgresql.message.MarkMessageAsRead"

	const query = `
		INSERT INTO profile_message (profile_id, message_id, read_status)
		SELECT p.id, $2, TRUE
		FROM profile p
		WHERE p.base_profile_id = $1
		ON CONFLICT (profile_id, message_id)
		DO UPDATE SET read_status = TRUE
	`

	stmt, err := repo.db.PrepareContext(ctx, query)
	if err != nil {
		return e.Wrap(op, err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, profileID, messageID)
	if err != nil {
		return e.Wrap(op, err)
	}

	return nil
}

func (repo *MessageRepository) FindThreadsByProfileID(ctx context.Context, profileID int64) ([]domain.ThreadInfo, error) {
	const op = "storage.postgresql.message.FindThreadsByProfileID"

	const query = `
        SELECT 
            t.id as thread_id,
            t.root_message_id,
            MAX(m.date_of_dispatch) as last_activity
        FROM
            thread t
        JOIN
            message m ON t.id = m.thread_id
        JOIN
            folder_profile_message fpm ON m.id = fpm.message_id
        JOIN
            folder f ON fpm.folder_id = f.id
        JOIN
            profile p ON f.profile_id = p.id
        JOIN
            profile_message pm ON m.id = pm.message_id AND pm.profile_id = p.id
        WHERE
            p.base_profile_id = $1
        GROUP BY 
            t.id, t.root_message_id
        ORDER BY
            last_activity DESC`

	stmt, err := repo.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, e.Wrap(op, err)
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, profileID)
	if err != nil {
		return nil, e.Wrap(op+": failed to execute query: ", err)
	}
	defer rows.Close()

	var threads []domain.ThreadInfo
	for rows.Next() {
		var threadInfo domain.ThreadInfo

		err := rows.Scan(&threadInfo.ID, &threadInfo.RootMessage, &threadInfo.LastActivity)
		if err != nil {
			return nil, e.Wrap(op, err)
		}

		threads = append(threads, threadInfo)
	}

	if err := rows.Err(); err != nil {
		return nil, e.Wrap(op, err)
	}

	return threads, nil
}
