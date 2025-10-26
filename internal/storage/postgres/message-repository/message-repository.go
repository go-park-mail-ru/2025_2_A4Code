package message_repository

import (
	"2025_2_a4code/internal/domain"
	"database/sql"
	"fmt"
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

func (repo *MessageRepository) FindByMessageID(messageID int64) (*domain.Message, error) {
	const op = "storage.postgresql.message.FindByMessageID"

	const query = `
		SELECT
			m.Id, m.Topic, m.Text, m.DateOfDispatch,
			bp.Id, bp.Username, bp.Domain,
			COALESCE(p.Name, ''), COALESCE(p.Surname, ''), COALESCE(p.ImagePath, '')
		FROM
			messages m
		JOIN
			baseprofile bp ON m.SenderBaseProfileId = bp.Id
		LEFT JOIN
			profile p ON bp.Id = p.BaseProfileId
		WHERE
			m.id = $1`
	stmt, err := repo.db.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf(op+`: %w`, err)
	}
	defer stmt.Close()

	var message domain.Message
	var messageIdInt int64
	var senderId int64
	var senderUsername, senderDomain, senderName, senderSurname, senderAvatar, text string

	err = stmt.QueryRow(messageID).Scan(
		&messageIdInt, &message.Topic, &text, &message.Datetime,
		&senderId, &senderUsername, &senderDomain,
		&senderName, &senderSurname, &senderAvatar,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // TODO: добавить кастомную ошибку
		}
		return nil, fmt.Errorf(op+`: query: %w`, err)
	}

	message.ID = strconv.FormatInt(messageIdInt, 10)

	if len(text) > 40 {
		message.Snippet = text[:40] + "..."
	} else {
		message.Snippet = text
	}

	message.Sender = domain.Sender{
		Id:       senderId,
		Email:    fmt.Sprintf("%s@%s", senderUsername, senderDomain),
		Username: strings.TrimSpace(fmt.Sprintf("%s %s", senderName, senderSurname)),
		Avatar:   senderAvatar,
	}

	err = repo.db.QueryRow("SELECT ReadStatus FROM profilemessage WHERE MessageId=$1", messageID).Scan(&message.IsRead)
	if err != nil {
		return nil, fmt.Errorf(op+`: query: %w`, err)
	}

	return &message, nil
}

func (repo *MessageRepository) FindFullByMessageID(messageID int64, profileID int64) (domain.FullMessage, error) {
	const op = "storage.postgresql.message.FindByMessageID"

	const query = `
		SELECT
			m.Id, m.Topic, m.Text, m.DateOfDispatch,
			bp.Id, bp.Username, bp.Domain,
			COALESCE(p.Name, ''), COALESCE(p.Surname, ''), COALESCE(p.ImagePath, ''),
			COALESCE(t.Id, 0), COALESCE(t.RootMessage, 0),
			COALESCE(f.Id, 0), COALESCE(f.Profile_id, 0), COALESCE(f.Folder_name, ''), COALESCE(f.Folder_type, 'inbox')
		FROM
			messages m
		JOIN
			baseprofile bp ON m.SenderBaseProfileId = bp.Id
		LEFT JOIN
			profile p ON bp.Id = p.BaseProfileId
		LEFT JOIN
			thread t ON m.ThreadRoot = t.Id
		LEFT JOIN
			folderprofilemessage fpm ON m.Id = fpm.MessageId
		LEFT JOIN
			folder f ON fpm.FolderId = f.Id AND f.Profile_id = $2 -- $2 is profileID
		WHERE
			m.id = $1 
		LIMIT 1`

	stmt, err := repo.db.Prepare(query)
	if err != nil {
		return domain.FullMessage{}, fmt.Errorf(op+`: query: %w`, err)
	}
	defer stmt.Close()

	var msg domain.FullMessage
	var messageIdInt int64
	var senderId int64
	var senderUsername, senderDomain, senderName, senderSurname, senderAvatar string
	var folderTypeStr string
	err = stmt.QueryRow(messageID, profileID).Scan(
		&messageIdInt, &msg.Topic, &msg.Text, &msg.Datetime,
		&senderId, &senderUsername, &senderDomain,
		&senderName, &senderSurname, &senderAvatar,
		&msg.ThreadRoot, &msg.ThreadRoot,
		&msg.Folder.ID, &msg.Folder.ProfileID, &msg.Folder.Name, &folderTypeStr,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return domain.FullMessage{}, fmt.Errorf(op+`: not fount: %w`, err)
		}
		return domain.FullMessage{}, fmt.Errorf(op+`: query: %w`, err)
	}

	msg.ID = strconv.FormatInt(messageIdInt, 10)

	msg.Folder.Type = domain.FolderType(folderTypeStr)

	msg.Sender = domain.Sender{
		Id:       senderId,
		Email:    fmt.Sprintf("%s@%s", senderUsername, senderDomain),
		Username: strings.TrimSpace(fmt.Sprintf("%s %s", senderName, senderSurname)),
		Avatar:   senderAvatar,
	}

	rows, err := repo.db.Query("SELECT Id, FileType, Size, StoragePath, MessageId FROM files WHERE MessageId = $1", messageID)
	if err != nil {
		return domain.FullMessage{}, fmt.Errorf(op+": file query: %w", err)
	}
	defer rows.Close()

	var files []domain.File

	for rows.Next() {
		var file domain.File
		err := rows.Scan(&file.ID, &file.FileType, &file.Size, &file.StoragePath, &file.MessageID)
		if err != nil {
			return domain.FullMessage{}, fmt.Errorf(op+": scan file: %w", err)
		}

		files = append(files, file)
	}
	if err = rows.Err(); err != nil {
		return domain.FullMessage{}, fmt.Errorf(op+": file rows: %w", err)
	}

	msg.Files = files

	return msg, nil
}

func (repo *MessageRepository) FindByProfileID(profileID int64) ([]domain.Message, error) {
	const op = "storage.postgresql.message.FindByProfileID"

	const query = `
		SELECT
			m.Id, m.Topic, m.Text, m.DateOfDispatch,
			pm.ReadStatus,
			bp.Id, bp.Username, bp.Domain,
			COALESCE(p.Name, ''), COALESCE(p.Surname, ''), COALESCE(p.ImagePath, '')
		FROM
			messages m
		JOIN
			folderprofilemessage fpm ON m.Id = fpm.MessageId
		JOIN
			folder f ON fpm.FolderId = f.Id
		JOIN
			profilemessage pm ON m.Id = pm.MessageId
		JOIN
			baseprofile bp ON m.SenderBaseProfileId = bp.Id
		LEFT JOIN
			profile p ON bp.Id = p.BaseProfileId
		WHERE
			f.Profile_id = $1 AND pm.ProfileId = $1
		GROUP BY 
			m.Id, pm.ReadStatus, bp.Id, p.Name, p.Surname, p.ImagePath
		ORDER BY
			m.DateOfDispatch DESC`

	stmt, err := repo.db.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf(op+`: %w`, err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(profileID)
	if err != nil {
		return nil, fmt.Errorf(op+`: query: %w`, err)
	}
	defer rows.Close()
	var messages []domain.Message
	for rows.Next() {
		var message domain.Message
		var messageIdInt int64
		var senderId int64
		var senderUsername, senderDomain, senderName, senderSurname, senderAvatar, text string
		err := rows.Scan(
			&messageIdInt, &message.Topic, &text, &message.Datetime, &message.IsRead,
			&senderId, &senderUsername, &senderDomain,
			&senderName, &senderSurname, &senderAvatar,
		)
		if err != nil {
			return nil, fmt.Errorf(op+`: query: %w`, err)
		}
		message.ID = strconv.FormatInt(messageIdInt, 10)
		if len(text) > 40 {
			message.Snippet = text[:40] + "..."
		} else {
			message.Snippet = text
		}
		message.Sender = domain.Sender{
			Id:       senderId,
			Email:    fmt.Sprintf("%s@%s", senderUsername, senderDomain),
			Username: strings.TrimSpace(fmt.Sprintf("%s %s", senderName, senderSurname)),
			Avatar:   senderAvatar,
		}
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf(op+`: query: %w`, err)
	}

	return messages, nil
}

func (repo *MessageRepository) SaveMessage(receiverProfileEmail string, senderBaseProfileID int64, topic, text string) (messageID int64, err error) {
	const op = "storage.postgresql.message.Save"

	const queryMessage = `
		INSERT INTO messages (Topic, Text, DateOfDispatch, SenderBaseProfileId, ThreadRoot, CreatedAt, UpdatedAt)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING MessageId`

	stmt, err := repo.db.Prepare(queryMessage)
	if err != nil {
		return 0, fmt.Errorf(op+`: %w`, err)
	}
	defer stmt.Close()

	timeNow := time.Now()
	var nullThreadID sql.NullInt64
	err = stmt.QueryRow(topic, text, timeNow, senderBaseProfileID, nullThreadID, time.Now(), time.Now()).Scan(&messageID)
	if err != nil {
		return 0, fmt.Errorf(op+": %w", err)
	}

	const queryGetReceiver = `
		WITH
		parsed_email AS (
			SELECT
				split_part($1, '@', 1) AS p_username,
				split_part($1, '@', 2) AS p_domain
		),
		insert_attempt AS (
			INSERT INTO BASEPROFILE (Username, Domain, CreatedAt, UpdatedAt)
			SELECT p_username, p_domain, NOW(), NOW()
			FROM parsed_email
			ON CONFLICT (Username, Domain) DO NOTHING
		)
		SELECT bp.Id
		FROM BASEPROFILE bp, parsed_email pe
		WHERE bp.Username = pe.p_username AND bp.Domain = pe.p_domain;
	`
	stmt, err = repo.db.Prepare(queryGetReceiver)
	if err != nil {
		return 0, fmt.Errorf(op+`: query: %w`, err)
	}
	defer stmt.Close()
	var receiverProfileID int64
	err = stmt.QueryRow(receiverProfileEmail).Scan(&receiverProfileID)
	if err != nil {
		return 0, fmt.Errorf(op+`: query: %w`, err)
	}

	const queryProfileMessage = `
		INSERT INTO profilemessge (ProfileId, MessageId, ReadStatus, CreatedAt, UpdatedAt) VALUES ($1, $2, $3, $4, $5)`
	stmt, err = repo.db.Prepare(queryProfileMessage)
	if err != nil {
		return 0, fmt.Errorf(op+": %w", err)
	}
	defer stmt.Close()

	receiverID := receiverProfileID
	err = stmt.QueryRow(receiverID, messageID, false, timeNow, timeNow).Scan()
	if err != nil {
		return 0, fmt.Errorf(op+": %w", err)
	}

	return messageID, nil
}

func (repo *MessageRepository) SaveFile(messageID int64, fileName, fileType, storagePath string, size int64) (fileID int64, err error) {
	const op = "storage.postgresql.message.SaveFile"

	const query = `
		INSERT INTO files (FileType, Size, StoragePath, MessageId, CreateAt, UpdateAt)
		VALUES ($1, $2, $3, $4, $5, $6)
		Returning id`
	stmt, err := repo.db.Prepare(query)
	if err != nil {
		return 0, fmt.Errorf(op+": %w", err)
	}
	defer stmt.Close()

	timeNow := time.Now()
	err = stmt.QueryRow(fileType, size, storagePath, messageID, timeNow, timeNow).Scan(&fileID)
	if err != nil {
		return 0, fmt.Errorf(op+": %w", err)
	}

	return fileID, nil
}

func (repo *MessageRepository) SaveThread(messageID int64, threadRoot string) (threadID int64, err error) {
	const op = "storage.postgresql.message.SaveThread"

	const query = `
		INSERT INTO thread (RootMessageId, CreateAt, UpdatedAt)
		VALUES ($1, $2, $3)`

	stmt, err := repo.db.Prepare(query)
	if err != nil {
		return 0, fmt.Errorf(op+": %w", err)
	}
	defer stmt.Close()

	timeNow := time.Now()
	err = stmt.QueryRow(messageID, timeNow, timeNow).Scan(&threadID)
	if err != nil {
		return 0, fmt.Errorf(op+": %w", err)
	}

	return threadID, nil
}

func (repo *MessageRepository) SaveThreadIdToMessage(messageID int64, threadID int64) error {
	const op = "storage.postgresql.message.SaveThreadIdToMessage"

	const query = `
        UPDATE messages
        SET ThreadRoot = $1, UpdatedAt = $2
        WHERE Id = $3`
	stmt, err := repo.db.Prepare(query)
	if err != nil {
		return fmt.Errorf(op+": %w", err)
	}
	defer stmt.Close()
	timeNow := time.Now()
	res, err := stmt.Exec(threadID, timeNow, messageID)
	if err != nil {
		return fmt.Errorf(op+": %w", err)
	}

	rowsAffected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf(op+": %w", err)
	}
	if rowsAffected == 0 {

		return fmt.Errorf(op + ": Сообщение не найдено (ни одна строка не изменилась)")
	}
	return nil
}
