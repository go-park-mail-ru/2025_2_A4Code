package message_repository

import (
	"2025_2_a4code/internal/domain"
	"2025_2_a4code/internal/http-server/middleware/logger"
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
	log := logger.GetLogger(ctx).With(slog.String("op", op))

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

	log.Debug("Scanning message data...")
	err = stmt.QueryRowContext(ctx, messageID).Scan(
		&messageIdInt, &message.Topic, &text, &message.Datetime,
		&senderId, &senderUsername, &senderDomain,
		&senderName, &senderSurname, &senderAvatar,
		&message.IsRead,
	)
	if err != nil {
		return nil, e.Wrap(op, err)
	}

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
	log := logger.GetLogger(ctx).With(slog.String("op", op))

	tx, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		return domain.FullMessage{}, e.Wrap(op+": failed to begin transaction: ", err)
	}
	defer tx.Rollback()

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

	log.Debug("Scanning full message data...")
	err = repo.db.QueryRowContext(ctx, query, messageID, profileID).Scan(
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
	log.Debug("Getting message files...")
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
	if err := tx.Commit(); err != nil {
		return domain.FullMessage{}, e.Wrap(op+": failed to commit transaction: ", err)
	}

	return msg, nil
}

// SaveMessage - сохраняет сообщение
func (repo *MessageRepository) SaveMessage(ctx context.Context, receiverProfileEmail string, senderBaseProfileID int64, topic, text string) (messageID int64, err error) {
	const op = "storage.postgresql.message.SaveMessage"
	log := logger.GetLogger(ctx).With(slog.String("op", op))

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

	log.Debug("Inserting message...")
	err = tx.QueryRowContext(ctx, insertMessage, topic, text, time.Now(), senderBaseProfileID).Scan(&messageID)
	if err != nil {
		return 0, e.Wrap(op+": failed to insert message: ", err)
	}

	// Получение receiver_profile_id
	username := strings.Split(receiverProfileEmail, "@")[0]
	domain := strings.Split(receiverProfileEmail, "@")[1]

	var receiverProfileID int64
	log.Debug("Getting receiver profile ID...")
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

	log.Debug("Adding message to inbox...")
	_, err = tx.ExecContext(ctx, insertFolder, messageID, receiverProfileID)
	if err != nil {
		return 0, e.Wrap(op+": failed to insert to inbox: ", err)
	}

	// Добавление связи сообщение-профиль
	const insertProfileMessage = `
		INSERT INTO profile_message (profile_id, message_id, read_status)
		VALUES ($1, $2, false)`

	log.Debug("Creating profile message bond...")
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
	log := logger.GetLogger(ctx).With(slog.String("op", op))

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
	log.Debug("Inserting file...")
	err = stmt.QueryRowContext(ctx, fileType, size, storagePath, messageID, timeNow, timeNow).Scan(&fileID)
	if err != nil {
		return 0, e.Wrap(op+": failed to execute query: ", err)
	}

	return fileID, nil
}

func (repo *MessageRepository) SaveThread(ctx context.Context, messageID int64) (threadID int64, err error) {
	const op = "storage.postgresql.message.SaveThread"
	log := logger.GetLogger(ctx).With(slog.String("op", op))

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
	log.Debug("Inserting thread...")
	err = stmt.QueryRowContext(ctx, messageID, timeNow, timeNow).Scan(&threadID)
	if err != nil {
		return 0, e.Wrap(op, err)
	}

	return threadID, nil
}

func (repo *MessageRepository) SaveThreadIdToMessage(ctx context.Context, messageID int64, threadID int64) error {
	const op = "storage.postgresql.message.SaveThreadIdToMessage"
	log := logger.GetLogger(ctx).With(slog.String("op", op))

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
	log.Debug("Updating message with thread ID...")
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

func (repo *MessageRepository) FindThreadsByProfileID(ctx context.Context, profileID int64) ([]domain.ThreadInfo, error) {
	const op = "storage.postgresql.message.FindThreadsByProfileID"
	log := logger.GetLogger(ctx).With(slog.String("op", op))

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

	log.Debug("Querying threads...")
	rows, err := stmt.QueryContext(ctx, profileID)
	if err != nil {
		return nil, e.Wrap(op+": failed to execute query: ", err)
	}
	defer rows.Close()

	var threads []domain.ThreadInfo
	log.Debug("Scanning threads...")
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

func (repo *MessageRepository) MarkMessageAsRead(ctx context.Context, messageID int64, profileID int64) error {
	const op = "storage.postgresql.message.MarkMessageAsRead"
	log := logger.GetLogger(ctx).With(slog.String("op", op))

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

	log.Debug("Marking message as read...")
	_, err = stmt.ExecContext(ctx, profileID, messageID)
	if err != nil {
		return e.Wrap(op, err)
	}

	return nil
}

func (repo *MessageRepository) MarkMessageAsSpam(ctx context.Context, messageID int64, profileID int64) error {
	const op = "storage.postgresql.message.MarkMessageAsSpam"
	log := logger.GetLogger(ctx).With(slog.String("op", op))

	tx, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		return e.Wrap(op+": failed to begin transaction: ", err)
	}
	defer tx.Rollback()

	var spamFolderID int64
	log.Debug("Getting spam folder ID...")
	err = tx.QueryRowContext(ctx, `
        SELECT id FROM folder 
        WHERE profile_id = $1 AND folder_type = 'spam'`,
		profileID).Scan(&spamFolderID)
	if err != nil {
		return e.Wrap(op+": failed to get spam folder: ", err)
	}

	const moveQuery = `
        INSERT INTO folder_profile_message (message_id, folder_id)
        SELECT $1, $2
        ON CONFLICT (message_id, folder_id) DO NOTHING`

	log.Debug("Moving message to spam folder...")
	_, err = tx.ExecContext(ctx, moveQuery, messageID, spamFolderID)
	if err != nil {
		return e.Wrap(op+": failed to move message to spam: ", err)
	}

	const deleteQuery = `
        DELETE FROM folder_profile_message 
        WHERE message_id = $1 
        AND folder_id IN (
            SELECT id FROM folder 
            WHERE profile_id = $2 AND folder_type != 'spam'
        )`

	log.Debug("Removing message from other folders...")
	_, err = tx.ExecContext(ctx, deleteQuery, messageID, profileID)
	if err != nil {
		return e.Wrap(op+": failed to remove from other folders: ", err)
	}

	log.Debug("Committing transaction...")
	return tx.Commit()
}

func (repo *MessageRepository) SaveDraft(ctx context.Context, profileID int64, draftID, receiverEmail, topic, text string) (int64, error) {
	const op = "storage.postgresql.message.SaveDraft"
	log := logger.GetLogger(ctx).With(slog.String("op", op))

	tx, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, e.Wrap(op+": failed to begin transaction: ", err)
	}
	defer tx.Rollback()

	var messageID int64

	if draftID != "" {
		log.Debug("Updating existing draft...")
		existingDraftID, err := strconv.ParseInt(draftID, 10, 64)
		if err != nil {
			return 0, e.Wrap(op+": invalid draft id: ", err)
		}

		var count int
		log.Debug("Checking draft ownership...")
		err = tx.QueryRowContext(ctx, `
            SELECT COUNT(*) FROM folder_profile_message fpm
            JOIN folder f ON fpm.folder_id = f.id
            WHERE fpm.message_id = $1 AND f.profile_id = $2 AND f.folder_type = 'draft'`,
			existingDraftID, profileID).Scan(&count)
		if err != nil {
			return 0, e.Wrap(op+": failed to check draft ownership: ", err)
		}

		if count == 0 {
			return 0, e.Wrap(op+": draft not found or access denied: ", err)
		}

		log.Debug("Updating draft message...")
		_, err = tx.ExecContext(ctx, `
            UPDATE message 
            SET topic = $1, text = $2, updated_at = $3
            WHERE id = $4`,
			topic, text, time.Now(), existingDraftID)
		if err != nil {
			return 0, e.Wrap(op+": failed to update draft: ", err)
		}

		messageID = existingDraftID
	} else {
		log.Debug("Creating new draft...")

		var senderBaseProfileID int64
		log.Debug("Getting sender base profile ID...")
		err = tx.QueryRowContext(ctx, `
            SELECT base_profile_id FROM profile WHERE id = $1`,
			profileID).Scan(&senderBaseProfileID)
		if err != nil {
			return 0, e.Wrap(op+": failed to get sender base profile: ", err)
		}

		log.Debug("Creating draft message...")
		err = tx.QueryRowContext(ctx, `
            INSERT INTO message (topic, text, date_of_dispatch, sender_base_profile_id)
            VALUES ($1, $2, $3, $4)
            RETURNING id`,
			topic, text, time.Now(), senderBaseProfileID).Scan(&messageID)
		if err != nil {
			return 0, e.Wrap(op+": failed to create draft message: ", err)
		}

		var draftFolderID int64
		log.Debug("Getting draft folder ID...")
		err = tx.QueryRowContext(ctx, `
            SELECT id FROM folder 
            WHERE profile_id = $1 AND folder_type = 'draft'`,
			profileID).Scan(&draftFolderID)
		if err != nil {
			return 0, e.Wrap(op+": failed to get draft folder: ", err)
		}

		log.Debug("Adding draft to folder...")
		_, err = tx.ExecContext(ctx, `
            INSERT INTO folder_profile_message (message_id, folder_id)
            VALUES ($1, $2)`,
			messageID, draftFolderID)
		if err != nil {
			return 0, e.Wrap(op+": failed to add to draft folder: ", err)
		}
	}

	log.Debug("Committing transaction...")
	return messageID, tx.Commit()
}

func (repo *MessageRepository) IsDraftBelongsToUser(ctx context.Context, draftID, profileID int64) (bool, error) {
	const op = "storage.postgresql.message.IsDraftBelongsToUser"
	log := logger.GetLogger(ctx).With(slog.String("op", op))
	log.Debug("Executing IsDraftBelongsToUser...")

	var count int
	log.Debug("Checking draft ownership...")
	err := repo.db.QueryRowContext(ctx, `
        SELECT COUNT(*) FROM folder_profile_message fpm
        JOIN folder f ON fpm.folder_id = f.id
        WHERE fpm.message_id = $1 AND f.profile_id = $2 AND f.folder_type = 'draft'`,
		draftID, profileID).Scan(&count)
	if err != nil {
		return false, e.Wrap(op, err)
	}

	log.Debug("Draft ownership check result", "belongs", count > 0)
	return count > 0, nil
}

func (repo *MessageRepository) DeleteDraft(ctx context.Context, draftID, profileID int64) error {
	const op = "storage.postgresql.message.DeleteDraft"
	log := logger.GetLogger(ctx).With(slog.String("op", op))
	log.Debug("Executing DeleteDraft...")

	tx, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		return e.Wrap(op+": failed to begin transaction: ", err)
	}
	defer tx.Rollback()

	log.Debug("Checking draft ownership...")
	belongs, err := repo.IsDraftBelongsToUser(ctx, draftID, profileID)
	if err != nil {
		return e.Wrap(op+": failed to check draft ownership: ", err)
	}
	if !belongs {
		return e.Wrap(op+": draft not found or access denied: ", err)
	}

	log.Debug("Removing draft from folders...")
	_, err = tx.ExecContext(ctx, `
        DELETE FROM folder_profile_message 
        WHERE message_id = $1 AND folder_id IN (
            SELECT id FROM folder WHERE profile_id = $2
        )`, draftID, profileID)
	if err != nil {
		return e.Wrap(op+": failed to remove from folders: ", err)
	}

	log.Debug("Deleting draft message...")
	_, err = tx.ExecContext(ctx, `DELETE FROM message WHERE id = $1`, draftID)
	if err != nil {
		return e.Wrap(op+": failed to delete draft message: ", err)
	}

	log.Debug("Committing transaction...")
	return tx.Commit()
}

func (repo *MessageRepository) SendDraft(ctx context.Context, draftID, profileID int64) error {
	const op = "storage.postgresql.message.SendDraft"
	log := logger.GetLogger(ctx).With(slog.String("op", op))

	tx, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		return e.Wrap(op+": failed to begin transaction: ", err)
	}
	defer tx.Rollback()

	log.Debug("Getting draft info...")
	var topic, text string
	var senderBaseProfileID int64
	err = tx.QueryRowContext(ctx, `
        SELECT m.topic, m.text, m.sender_base_profile_id 
        FROM message m
        WHERE m.id = $1`, draftID).Scan(&topic, &text, &senderBaseProfileID)
	if err != nil {
		return e.Wrap(op+": failed to get draft info: ", err)
	}

	log.Debug("Updating draft send time...")
	_, err = tx.ExecContext(ctx, `
        UPDATE message 
        SET date_of_dispatch = $1, updated_at = $2
        WHERE id = $3`,
		time.Now(), time.Now(), draftID)
	if err != nil {
		return e.Wrap(op+": failed to update draft send time: ", err)
	}

	log.Debug("Removing from draft folder...")
	_, err = tx.ExecContext(ctx, `
        DELETE FROM folder_profile_message 
        WHERE message_id = $1 AND folder_id IN (
            SELECT id FROM folder WHERE profile_id = $2 AND folder_type = 'draft'
        )`, draftID, profileID)
	if err != nil {
		return e.Wrap(op+": failed to remove from draft folder: ", err)
	}

	var sentFolderID int64
	log.Debug("Getting sent folder ID...")
	err = tx.QueryRowContext(ctx, `
        SELECT id FROM folder 
        WHERE profile_id = $1 AND folder_type = 'sent'`,
		profileID).Scan(&sentFolderID)
	if err != nil {
		return e.Wrap(op+": failed to get sent folder: ", err)
	}

	log.Debug("Adding to sent folder...")
	_, err = tx.ExecContext(ctx, `
        INSERT INTO folder_profile_message (message_id, folder_id)
        VALUES ($1, $2)`,
		draftID, sentFolderID)
	if err != nil {
		return e.Wrap(op+": failed to add to sent folder: ", err)
	}

	log.Debug("Committing transaction...")
	return tx.Commit()
}

func (repo *MessageRepository) GetDraft(ctx context.Context, draftID, profileID int64) (domain.FullMessage, error) {
	const op = "storage.postgresql.message.GetDraft"
	log := logger.GetLogger(ctx).With(slog.String("op", op))

	log.Debug("Checking draft ownership...")
	belongs, err := repo.IsDraftBelongsToUser(ctx, draftID, profileID)
	if err != nil {
		return domain.FullMessage{}, e.Wrap(op+": failed to check draft ownership: ", err)
	}
	if !belongs {
		return domain.FullMessage{}, e.Wrap(op+": draft not found or access denied: ", err)
	}

	return repo.FindFullByMessageID(ctx, draftID, profileID)
}

func (repo *MessageRepository) MoveToFolder(ctx context.Context, profileID, messageID, folderID int64) error {
	const op = "storage.postgresql.message.MoveToFolder"
	log := logger.GetLogger(ctx).With(slog.String("op", op))

	tx, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		return e.Wrap(op+": failed to begin transaction: ", err)
	}
	defer tx.Rollback()

	log.Debug("Removing from all user folders...")
	_, err = tx.ExecContext(ctx, `
        DELETE FROM folder_profile_message 
        WHERE message_id = $1 AND folder_id IN (
            SELECT id FROM folder WHERE profile_id = $2
        )`, messageID, profileID)
	if err != nil {
		return e.Wrap(op+": failed to remove from folders: ", err)
	}

	log.Debug("Adding to target folder...")
	_, err = tx.ExecContext(ctx, `
        INSERT INTO folder_profile_message (message_id, folder_id)
        VALUES ($1, $2)`, messageID, folderID)
	if err != nil {
		return e.Wrap(op+": failed to add to folder: ", err)
	}

	log.Debug("Committing transaction...")
	return tx.Commit()
}

func (repo *MessageRepository) GetFolderByType(ctx context.Context, profileID int64, folderType string) (int64, error) {
	const op = "storage.postgresql.message.GetFolderByType"
	log := logger.GetLogger(ctx).With(slog.String("op", op))

	const query = `
        SELECT id FROM folder 
        WHERE profile_id = $1 AND folder_type = $2`

	var folderID int64
	log.Debug("Querying folder by type...")
	err := repo.db.QueryRowContext(ctx, query, profileID, folderType).Scan(&folderID)
	if err != nil {
		return 0, e.Wrap(op, err)
	}

	return folderID, nil
}

func (repo *MessageRepository) ShouldMarkAsRead(ctx context.Context, messageID, profileID int64) (bool, error) {
	const op = "storage.postgresql.message.ShouldMarkAsRead"
	log := logger.GetLogger(ctx).With(slog.String("op", op))

	var shouldMark bool
	log.Debug("Checking if should mark as read...")
	err := repo.db.QueryRowContext(ctx, `
        SELECT 
            CASE 
                WHEN f.folder_type = 'inbox' AND pm.read_status = false THEN true
                ELSE false
            END as should_mark
        FROM folder_profile_message fpm
        JOIN folder f ON fpm.folder_id = f.id
        JOIN profile_message pm ON pm.message_id = fpm.message_id AND pm.profile_id = f.profile_id
        WHERE fpm.message_id = $1 AND f.profile_id = $2
        LIMIT 1`,
		messageID, profileID).Scan(&shouldMark)
	if err != nil {
		if err == sql.ErrNoRows {
			log.Debug("No rows found, returning false")
			return false, nil
		}
		return false, e.Wrap(op, err)
	}

	return shouldMark, nil
}

func (repo *MessageRepository) CreateFolder(ctx context.Context, profileID int64, folderName string) (*domain.Folder, error) {
	const op = "storage.postgresql.message.CreateFolder"
	log := logger.GetLogger(ctx).With(slog.String("op", op))

	const query = `
        INSERT INTO folder (profile_id, folder_name, folder_type)
        VALUES ($1, $2, 'custom')
        RETURNING id, folder_name, folder_type`

	var folder domain.Folder
	log.Debug("Creating folder...")
	err := repo.db.QueryRowContext(ctx, query, profileID, folderName).Scan(
		&folder.ID, &folder.Name, &folder.Type)
	if err != nil {
		return nil, e.Wrap(op, err)
	}

	folder.ProfileID = profileID
	return &folder, nil
}

func (repo *MessageRepository) GetUserFolders(ctx context.Context, profileID int64) ([]domain.Folder, error) {
	const op = "storage.postgresql.message.GetUserFolders"
	log := logger.GetLogger(ctx).With(slog.String("op", op))

	const query = `
        SELECT id, folder_name, folder_type
        FROM folder
        WHERE profile_id = $1
        ORDER BY 
            CASE folder_type
                WHEN 'inbox' THEN 1
                WHEN 'sent' THEN 2
                WHEN 'draft' THEN 3
                WHEN 'spam' THEN 4
                WHEN 'trash' THEN 5
                ELSE 6
            END, folder_name`

	log.Debug("Querying user folders...")
	rows, err := repo.db.QueryContext(ctx, query, profileID)
	if err != nil {
		return nil, e.Wrap(op, err)
	}
	defer rows.Close()

	var folders []domain.Folder
	log.Debug("Scanning folders...")
	for rows.Next() {
		var folder domain.Folder
		err := rows.Scan(&folder.ID, &folder.Name, &folder.Type)
		if err != nil {
			return nil, e.Wrap(op, err)
		}
		folder.ProfileID = profileID
		folders = append(folders, folder)
	}

	if err = rows.Err(); err != nil {
		return nil, e.Wrap(op, err)
	}

	return folders, nil
}

func (repo *MessageRepository) RenameFolder(ctx context.Context, profileID, folderID int64, newName string) (*domain.Folder, error) {
	const op = "storage.postgresql.message.RenameFolder"
	log := logger.GetLogger(ctx).With(slog.String("op", op))

	const query = `
        UPDATE folder 
        SET folder_name = $1, updated_at = $2
        WHERE id = $3 AND profile_id = $4 AND folder_type = 'custom'
        RETURNING id, folder_name, folder_type`

	var folder domain.Folder
	log.Debug("Renaming folder...")
	err := repo.db.QueryRowContext(ctx, query, newName, time.Now(), folderID, profileID).Scan(
		&folder.ID, &folder.Name, &folder.Type)
	if err != nil {
		return nil, e.Wrap(op, err)
	}

	folder.ProfileID = profileID
	return &folder, nil
}

func (repo *MessageRepository) DeleteFolder(ctx context.Context, profileID, folderID int64) error {
	const op = "storage.postgresql.message.DeleteFolder"
	log := logger.GetLogger(ctx).With(slog.String("op", op))

	tx, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		return e.Wrap(op+": failed to begin transaction: ", err)
	}
	defer tx.Rollback()

	var folderType string
	log.Debug("Checking folder type...")
	err = tx.QueryRowContext(ctx, `
        SELECT folder_type FROM folder 
        WHERE id = $1 AND profile_id = $2`,
		folderID, profileID).Scan(&folderType)
	if err != nil {
		return e.Wrap(op+": failed to get folder info: ", err)
	}

	if folderType != "custom" {
		return e.Wrap(op+": cannot delete system folder: ", err)
	}

	log.Debug("Removing folder messages...")
	_, err = tx.ExecContext(ctx, `
        DELETE FROM folder_profile_message 
        WHERE folder_id = $1`, folderID)
	if err != nil {
		return e.Wrap(op+": failed to remove folder messages: ", err)
	}

	log.Debug("Deleting folder...")
	_, err = tx.ExecContext(ctx, `DELETE FROM folder WHERE id = $1`, folderID)
	if err != nil {
		return e.Wrap(op+": failed to delete folder: ", err)
	}

	log.Debug("Committing transaction...")
	return tx.Commit()
}

func (repo *MessageRepository) DeleteMessageFromFolder(ctx context.Context, profileID, messageID, folderID int64) error {
	const op = "storage.postgresql.message.DeleteMessageFromFolder"
	log := logger.GetLogger(ctx).With(slog.String("op", op))

	const query = `
        DELETE FROM folder_profile_message 
        WHERE message_id = $1 AND folder_id = $2
        AND folder_id IN (SELECT id FROM folder WHERE profile_id = $3)`

	log.Debug("Deleting message from folder...")
	result, err := repo.db.ExecContext(ctx, query, messageID, folderID, profileID)
	if err != nil {
		return e.Wrap(op, err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return e.Wrap(op, err)
	}

	if rowsAffected == 0 {
		return e.Wrap(op+": message not found in folder or access denied: ", err)
	}

	log.Debug("Successfully deleted message from folder")
	return nil
}

func (repo *MessageRepository) GetFolderMessagesWithKeysetPagination(
	ctx context.Context,
	profileID, folderID, lastMessageID int64,
	lastDatetime time.Time,
	limit int,
) ([]domain.Message, error) {
	const op = "storage.postgresql.message.GetFolderMessagesWithKeysetPagination"
	log := logger.GetLogger(ctx).With(slog.String("op", op))

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
            folder_profile_message fpm ON m.id = fpm.message_id
        JOIN
            folder f ON fpm.folder_id = f.id
        JOIN
            base_profile bp ON m.sender_base_profile_id = bp.id
        LEFT JOIN
            profile sender_profile ON bp.id = sender_profile.base_profile_id
        WHERE
            f.profile_id = $1 AND f.id = $2
            AND (($3 = 0 AND $4 = 0) OR (m.date_of_dispatch, m.id) < (to_timestamp($4), $3))
        ORDER BY
            m.date_of_dispatch DESC, m.id DESC
        LIMIT $5`

	log.Debug("Querying folder messages with pagination...")
	rows, err := repo.db.QueryContext(ctx, query, profileID, folderID, lastMessageID, lastDatetime.Unix(), limit)
	if err != nil {
		return nil, e.Wrap(op, err)
	}
	defer rows.Close()

	var messages []domain.Message
	log.Debug("Scanning messages...")
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

func (repo *MessageRepository) GetFolderMessagesInfo(ctx context.Context, profileID, folderID int64) (domain.Messages, error) {
	const op = "storage.postgresql.message.GetFolderMessagesInfo"
	log := logger.GetLogger(ctx).With(slog.String("op", op))

	const query = `
        SELECT 
            COUNT(*) as total_count,
            COUNT(CASE WHEN pm.read_status = false THEN 1 END) as unread_count
        FROM folder_profile_message fpm
        JOIN profile_message pm ON fpm.message_id = pm.message_id AND pm.profile_id = $1
        WHERE fpm.folder_id = $2`

	var messagesInfo domain.Messages
	log.Debug("Getting folder messages info...")
	err := repo.db.QueryRowContext(ctx, query, profileID, folderID).Scan(
		&messagesInfo.MessageTotal, &messagesInfo.MessageUnread)
	if err != nil {
		return domain.Messages{}, e.Wrap(op, err)
	}

	return messagesInfo, nil
}

func (repo *MessageRepository) SaveMessageWithFolderDistribution(
	ctx context.Context,
	receiverProfileEmail string,
	senderBaseProfileID int64,
	topic, text string,
) (messageID int64, err error) {
	const op = "storage.postgresql.message.SaveMessageWithFolderDistribution"
	log := logger.GetLogger(ctx).With(slog.String("op", op))

	tx, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, e.Wrap(op+": failed to begin transaction: ", err)
	}
	defer tx.Rollback()

	const insertMessage = `
        INSERT INTO message (topic, text, date_of_dispatch, sender_base_profile_id)
        VALUES ($1, $2, $3, $4)
        RETURNING id`

	log.Debug("Inserting message...")
	err = tx.QueryRowContext(ctx, insertMessage, topic, text, time.Now(), senderBaseProfileID).Scan(&messageID)
	if err != nil {
		return 0, e.Wrap(op+": failed to insert message: ", err)
	}

	username := strings.Split(receiverProfileEmail, "@")[0]
	domain := strings.Split(receiverProfileEmail, "@")[1]

	var receiverProfileID int64
	log.Debug("Getting receiver profile ID...")
	err = tx.QueryRowContext(ctx, `
        SELECT p.id 
        FROM profile p
        JOIN base_profile bp ON p.base_profile_id = bp.id
        WHERE bp.username = $1 AND bp.domain = $2`,
		username, domain).Scan(&receiverProfileID)
	if err != nil {
		return 0, e.Wrap(op+": failed to get receiver id: ", err)
	}

	var senderProfileID int64
	log.Debug("Getting sender profile ID...")
	err = tx.QueryRowContext(ctx, `
        SELECT id FROM profile WHERE base_profile_id = $1`,
		senderBaseProfileID).Scan(&senderProfileID)
	if err != nil {
		return 0, e.Wrap(op+": failed to get sender profile id: ", err)
	}

	const insertToInbox = `
        INSERT INTO folder_profile_message (message_id, folder_id)
        SELECT $1, f.id
        FROM folder f
        WHERE f.profile_id = $2 AND f.folder_type = 'inbox'`

	log.Debug("Adding to receiver's inbox...")
	_, err = tx.ExecContext(ctx, insertToInbox, messageID, receiverProfileID)
	if err != nil {
		return 0, e.Wrap(op+": failed to insert to inbox: ", err)
	}

	const insertToSent = `
        INSERT INTO folder_profile_message (message_id, folder_id)
        SELECT $1, f.id
        FROM folder f
        WHERE f.profile_id = $2 AND f.folder_type = 'sent'`

	log.Debug("Adding to sender's sent folder...")
	_, err = tx.ExecContext(ctx, insertToSent, messageID, senderProfileID)
	if err != nil {
		return 0, e.Wrap(op+": failed to insert to sent: ", err)
	}

	const insertProfileMessage = `
        INSERT INTO profile_message (profile_id, message_id, read_status)
        VALUES ($1, $2, false)`

	log.Debug("Creating profile message bond...")
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

func (repo *MessageRepository) ReplyToMessageWithFolderDistribution(
	ctx context.Context,
	receiverEmail string,
	senderProfileID int64,
	threadRoot int64,
	topic, text string,
) (int64, error) {
	const op = "storage.postgresql.message.ReplyToMessageWithFolderDistribution"
	log := logger.GetLogger(ctx).With(slog.String("op", op))

	tx, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, e.Wrap(op+": failed to begin transaction: ", err)
	}
	defer tx.Rollback()

	var senderBaseProfileID int64
	log.Debug("Getting sender base profile ID...")
	err = tx.QueryRowContext(ctx, `
        SELECT base_profile_id FROM profile WHERE id = $1`,
		senderProfileID).Scan(&senderBaseProfileID)
	if err != nil {
		return 0, e.Wrap(op+": failed to get sender base profile id: ", err)
	}

	const insertMessage = `
        INSERT INTO message (topic, text, date_of_dispatch, sender_base_profile_id, thread_id)
        VALUES ($1, $2, $3, $4, $5)
        RETURNING id`

	var messageID int64
	log.Debug("Inserting reply message...")
	err = tx.QueryRowContext(ctx, insertMessage, topic, text, time.Now(), senderBaseProfileID, threadRoot).Scan(&messageID)
	if err != nil {
		return 0, e.Wrap(op+": failed to insert message: ", err)
	}

	username := strings.Split(receiverEmail, "@")[0]
	domain := strings.Split(receiverEmail, "@")[1]

	var receiverProfileID int64
	log.Debug("Getting receiver profile ID...")
	err = tx.QueryRowContext(ctx, `
        SELECT p.id 
        FROM profile p
        JOIN base_profile bp ON p.base_profile_id = bp.id
        WHERE bp.username = $1 AND bp.domain = $2`,
		username, domain).Scan(&receiverProfileID)
	if err != nil {
		return 0, e.Wrap(op+": failed to get receiver id: ", err)
	}

	const insertToInbox = `
        INSERT INTO folder_profile_message (message_id, folder_id)
        SELECT $1, f.id
        FROM folder f
        WHERE f.profile_id = $2 AND f.folder_type = 'inbox'`

	log.Debug("Adding to receiver's inbox...")
	_, err = tx.ExecContext(ctx, insertToInbox, messageID, receiverProfileID)
	if err != nil {
		return 0, e.Wrap(op+": failed to insert to inbox: ", err)
	}

	const insertToSent = `
        INSERT INTO folder_profile_message (message_id, folder_id)
        SELECT $1, f.id
        FROM folder f
        WHERE f.profile_id = $2 AND f.folder_type = 'sent'`

	log.Debug("Adding to sender's sent folder...")
	_, err = tx.ExecContext(ctx, insertToSent, messageID, senderProfileID)
	if err != nil {
		return 0, e.Wrap(op+": failed to insert to sent: ", err)
	}

	const insertProfileMessage = `
        INSERT INTO profile_message (profile_id, message_id, read_status)
        VALUES ($1, $2, false)`

	log.Debug("Creating profile message bond...")
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
