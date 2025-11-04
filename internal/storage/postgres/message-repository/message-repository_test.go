package message_repository

import (
	"context"
	"database/sql"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

func setupTest(t *testing.T) (context.Context, *MessageRepository, sqlmock.Sqlmock) {
	t.Helper()

	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	t.Cleanup(func() { db.Close() })

	repo := New(db)

	ctx := context.Background()

	return ctx, repo, mock
}

func quote(query string) string {
	return regexp.QuoteMeta(query)
}

func TestMessageRepository_FindFullByMessageID(t *testing.T) {
	ctx, repo, mock := setupTest(t)
	mockMessageID := int64(1)
	mockProfileID := int64(10)
	mockTime := time.Now()

	const messageQuery = `
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

	mock.ExpectBegin()

	messageRows := sqlmock.NewRows([]string{
		"m.id", "m.topic", "m.text", "m.date_of_dispatch",
		"bp.id", "bp.username", "bp.domain",
		"p.name", "p.surname", "p.image_path",
		"t.id", "t.root_message_id",
		"f.id", "f.profile_id", "f.folder_name", "f.folder_type",
	}).AddRow(
		mockMessageID, "Full Topic", "Full text", mockTime,
		int64(2), "sender", "example.com",
		"Sender", "User", "avatar.png",
		sql.NullInt64{Int64: 5, Valid: true}, sql.NullInt64{Int64: 1, Valid: true}, // thread
		sql.NullInt64{Int64: 100, Valid: true}, sql.NullInt64{Int64: mockProfileID, Valid: true}, // folder
		sql.NullString{String: "Inbox", Valid: true}, sql.NullString{String: "inbox", Valid: true},
	)

	mock.ExpectQuery(quote(messageQuery)).
		WithArgs(mockMessageID, mockProfileID).
		WillReturnRows(messageRows)

	const filesQuery = `
        SELECT id, file_type, size, storage_path, message_id 
        FROM file 
        WHERE message_id = $1`

	filesRows := sqlmock.NewRows([]string{"id", "file_type", "size", "storage_path", "message_id"}).
		AddRow(int64(1), "image/png", int64(1024), "path/to/file1.png", mockMessageID).
		AddRow(int64(2), "application/pdf", int64(2048), "path/to/file2.pdf", mockMessageID)

	mock.ExpectQuery(quote(filesQuery)).
		WithArgs(mockMessageID).
		WillReturnRows(filesRows)

	mock.ExpectCommit()

	msg, err := repo.FindFullByMessageID(ctx, mockMessageID, mockProfileID)

	assert.NoError(t, err)
	assert.Equal(t, strconv.FormatInt(mockMessageID, 10), msg.ID)
	assert.Equal(t, "Full Topic", msg.Topic)
	assert.Equal(t, "Sender User", msg.Sender.Username)
	assert.Equal(t, "5", msg.ThreadRoot)
	assert.Equal(t, "Inbox", msg.Folder.Name)
	assert.Len(t, msg.Files, 2)
	assert.Equal(t, "image/png", msg.Files[0].FileType)
	assert.Equal(t, "path/to/file2.pdf", msg.Files[1].StoragePath)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMessageRepository_FindByProfileIDWithKeysetPagination(t *testing.T) {
	ctx, repo, mock := setupTest(t)
	mockProfileID := int64(1)
	limit := 10

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

	rows := sqlmock.NewRows([]string{
		"m.id", "m.topic", "m.text", "m.date_of_dispatch", "pm.read_status",
		"bp.id", "bp.username", "bp.domain",
		"sender_profile.name", "sender_profile.surname", "sender_profile.image_path",
	}).
		AddRow(
			int64(10), "Topic 1", "Text 1", time.Now(), false,
			int64(2), "sender1", "a.com", "Sender", "One", "avatar1.png",
		)

	t.Run("FirstPage", func(t *testing.T) {
		mock.ExpectPrepare(quote(query)).
			ExpectQuery().
			WithArgs(mockProfileID, int64(0), int64(0), limit).
			WillReturnRows(rows)

		messages, err := repo.FindByProfileIDWithKeysetPagination(ctx, mockProfileID, 0, time.Time{}, limit)
		assert.NoError(t, err)
		assert.Len(t, messages, 1)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("NextPage", func(t *testing.T) {
		lastTime := time.Now().Add(-time.Hour)
		lastID := int64(10)

		rows := sqlmock.NewRows([]string{
			"m.id", "m.topic", "m.text", "m.date_of_dispatch", "pm.read_status",
			"bp.id", "bp.username", "bp.domain",
			"sender_profile.name", "sender_profile.surname", "sender_profile.image_path",
		}).
			AddRow(
				int64(9), "Topic 2", "Text 2", lastTime.Add(-time.Minute), false,
				int64(3), "sender2", "b.com", "Sender", "Two", "avatar2.png",
			)

		mock.ExpectPrepare(quote(query)).
			ExpectQuery().
			WithArgs(mockProfileID, lastID, lastTime.Unix(), limit).
			WillReturnRows(rows)

		messages, err := repo.FindByProfileIDWithKeysetPagination(ctx, mockProfileID, lastID, lastTime, limit)
		assert.NoError(t, err)
		assert.Len(t, messages, 1)
		assert.Equal(t, "9", messages[0].ID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestMessageRepository_SaveMessage(t *testing.T) {
	ctx, repo, mock := setupTest(t)

	receiverEmail := "receiver@domain.com"
	senderID := int64(1)
	topic := "New Message"
	text := "Message Body"
	expectedMessageID := int64(123)
	expectedReceiverProfileID := int64(456)

	t.Run("Success", func(t *testing.T) {
		mock.ExpectBegin()

		const insertMessage = `
		INSERT INTO message (topic, text, date_of_dispatch, sender_base_profile_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id`
		mock.ExpectQuery(quote(insertMessage)).
			WithArgs(topic, text, sqlmock.AnyArg(), senderID).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(expectedMessageID))

		username := strings.Split(receiverEmail, "@")[0]
		domain := strings.Split(receiverEmail, "@")[1]
		const getReceiverID = `
		SELECT p.id 
		FROM profile p
		JOIN base_profile bp ON p.base_profile_id = bp.id
		WHERE bp.username = $1 AND bp.domain = $2`
		mock.ExpectQuery(quote(getReceiverID)).
			WithArgs(username, domain).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(expectedReceiverProfileID))

		const insertFolder = `
		INSERT INTO folder_profile_message (message_id, folder_id)
		SELECT $1, f.id
		FROM folder f
		WHERE f.profile_id = $2 AND f.folder_type = 'inbox'`
		mock.ExpectExec(quote(insertFolder)).
			WithArgs(expectedMessageID, expectedReceiverProfileID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		const insertProfileMessage = `
		INSERT INTO profile_message (profile_id, message_id, read_status)
		VALUES ($1, $2, false)`
		mock.ExpectExec(quote(insertProfileMessage)).
			WithArgs(expectedReceiverProfileID, expectedMessageID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectCommit()

		msgID, err := repo.SaveMessage(ctx, receiverEmail, senderID, topic, text)

		assert.NoError(t, err)
		assert.Equal(t, expectedMessageID, msgID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("ErrorOnInsertMessage", func(t *testing.T) {
		mock.ExpectBegin()

		const insertMessage = `
		INSERT INTO message (topic, text, date_of_dispatch, sender_base_profile_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id`
		mock.ExpectQuery(quote(insertMessage)).
			WithArgs(topic, text, sqlmock.AnyArg(), senderID).
			WillReturnError(fmt.Errorf("db error"))

		mock.ExpectRollback()

		_, err := repo.SaveMessage(ctx, receiverEmail, senderID, topic, text)

		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestMessageRepository_SaveFile(t *testing.T) {
	ctx, repo, mock := setupTest(t)
	expectedFileID := int64(1)

	const query = `
		INSERT INTO file (file_type, size, storage_path, message_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		Returning id`

	mock.ExpectPrepare(quote(query)).
		ExpectQuery().
		WithArgs("image/png", int64(1024), "path/file.png", int64(123), sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(expectedFileID))

	fileID, err := repo.SaveFile(ctx, 123, "file.png", "image/png", "path/file.png", 1024)

	assert.NoError(t, err)
	assert.Equal(t, expectedFileID, fileID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMessageRepository_SaveThread(t *testing.T) {
	ctx, repo, mock := setupTest(t)
	expectedThreadID := int64(1)
	messageID := int64(123)

	const query = `
		INSERT INTO thread (root_message_id, created_at, updated_at)
		VALUES ($1, $2, $3)
		RETURNING id`

	mock.ExpectPrepare(quote(query)).
		ExpectQuery().
		WithArgs(messageID, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(expectedThreadID))

	threadID, err := repo.SaveThread(ctx, messageID)

	assert.NoError(t, err)
	assert.Equal(t, expectedThreadID, threadID)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMessageRepository_SaveThreadIdToMessage(t *testing.T) {
	ctx, repo, mock := setupTest(t)
	messageID := int64(123)
	threadID := int64(1)

	const query = `
        UPDATE message
        SET thread_id = $1, updated_at = $2
        WHERE Id = $3`

	t.Run("Success", func(t *testing.T) {
		mock.ExpectPrepare(quote(query)).
			ExpectExec().
			WithArgs(threadID, sqlmock.AnyArg(), messageID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		err := repo.SaveThreadIdToMessage(ctx, messageID, threadID)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

}

func TestMessageRepository_GetMessagesStats(t *testing.T) {
	ctx, repo, mock := setupTest(t)
	profileID := int64(1)

	const query = `
        SELECT 
            COUNT(DISTINCT pm.message_id) as total_count,
            COUNT(DISTINCT CASE WHEN pm.read_status = false THEN pm.message_id END) as unread_count
        FROM profile_message pm
        JOIN profile p ON pm.profile_id = p.id
        WHERE p.base_profile_id = $1`

	rows := sqlmock.NewRows([]string{"total_count", "unread_count"}).AddRow(100, 5)

	mock.ExpectPrepare(quote(query)).
		ExpectQuery().
		WithArgs(profileID).
		WillReturnRows(rows)

	total, unread, err := repo.GetMessagesStats(ctx, profileID)

	assert.NoError(t, err)
	assert.Equal(t, 100, total)
	assert.Equal(t, 5, unread)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMessageRepository_MarkMessageAsRead(t *testing.T) {
	ctx, repo, mock := setupTest(t)
	profileID := int64(1)
	messageID := int64(123)

	const query = `
		INSERT INTO profile_message (profile_id, message_id, read_status)
		SELECT p.id, $2, TRUE
		FROM profile p
		WHERE p.base_profile_id = $1
		ON CONFLICT (profile_id, message_id)
		DO UPDATE SET read_status = TRUE
	`

	mock.ExpectPrepare(quote(query)).
		ExpectExec().
		WithArgs(profileID, messageID).
		WillReturnResult(sqlmock.NewResult(1, 1))

	err := repo.MarkMessageAsRead(ctx, messageID, profileID)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMessageRepository_FindThreadsByProfileID(t *testing.T) {
	ctx, repo, mock := setupTest(t)
	profileID := int64(1)
	mockTime := time.Now()

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

	rows := sqlmock.NewRows([]string{"thread_id", "root_message_id", "last_activity"}).
		AddRow(int64(1), int64(100), mockTime).
		AddRow(int64(2), int64(102), mockTime.Add(-time.Hour))

	mock.ExpectPrepare(quote(query)).
		ExpectQuery().
		WithArgs(profileID).
		WillReturnRows(rows)

	threads, err := repo.FindThreadsByProfileID(ctx, profileID)

	assert.NoError(t, err)
	assert.Len(t, threads, 2)
	assert.Equal(t, int64(1), threads[0].ID)
	assert.Equal(t, int64(100), threads[0].RootMessage)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMessageRepository_FindSentMessagesByProfileIDWithKeysetPagination(t *testing.T) {
	ctx, repo, mock := setupTest(t)
	profileID := int64(1)
	limit := 10

	const query = `
        SELECT
            m.id, m.topic, m.text, m.date_of_dispatch,
            -- Check if *any* recipient has read the message
            EXISTS (
                SELECT 1 
                FROM profile_message pm 
                WHERE pm.message_id = m.id AND pm.read_status = TRUE
            ) as read_status,
            bp.id, bp.username, bp.domain,
            sender_profile.name, sender_profile.surname, sender_profile.image_path
        FROM
            message m
        JOIN
            base_profile bp ON m.sender_base_profile_id = bp.id
        LEFT JOIN
            profile sender_profile ON bp.id = sender_profile.base_profile_id
        WHERE
            m.sender_base_profile_id = $1  -- Filter by SENDER's base_profile_id
			AND (($2 = 0 AND $3 = 0) OR (m.date_of_dispatch, m.id) < (to_timestamp($3), $2))
        ORDER BY
            m.date_of_dispatch DESC, m.id DESC
		FETCH FIRST $4 ROWS ONLY`

	rows := sqlmock.NewRows([]string{
		"m.id", "m.topic", "m.text", "m.date_of_dispatch", "read_status",
		"bp.id", "bp.username", "bp.domain",
		"sender_profile.name", "sender_profile.surname", "sender_profile.image_path",
	}).
		AddRow(
			int64(10), "Sent Topic 1", "Sent Text 1", time.Now(), true, // read
			profileID, "sender", "a.com", "My", "Name", "my_avatar.png",
		)

	mock.ExpectPrepare(quote(query)).
		ExpectQuery().
		WithArgs(profileID, int64(0), int64(0), limit).
		WillReturnRows(rows)

	messages, err := repo.FindSentMessagesByProfileIDWithKeysetPagination(ctx, profileID, 0, time.Time{}, limit)

	assert.NoError(t, err)
	assert.Len(t, messages, 1)
	assert.Equal(t, "10", messages[0].ID)
	assert.Equal(t, "My Name", messages[0].Sender.Username)
	assert.True(t, messages[0].IsRead)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestMessageRepository_GetSentMessagesStats(t *testing.T) {
	ctx, repo, mock := setupTest(t)
	profileID := int64(1)

	const query = `
        SELECT 
            -- Total number of messages sent by this user
            COUNT(m.id) as total_count,
            
            -- Count of messages where NO recipient has read_status = TRUE
            COUNT(m.id) FILTER (
                WHERE NOT EXISTS (
                    SELECT 1 
                    FROM profile_message pm 
                    WHERE pm.message_id = m.id AND pm.read_status = TRUE
                )
            ) as unread_count
        FROM 
            message m
        WHERE 
            m.sender_base_profile_id = $1
    `

	rows := sqlmock.NewRows([]string{"total_count", "unread_count"}).AddRow(50, 10)

	mock.ExpectPrepare(quote(query)).
		ExpectQuery().
		WithArgs(profileID).
		WillReturnRows(rows)

	total, unread, err := repo.GetSentMessagesStats(ctx, profileID)

	assert.NoError(t, err)
	assert.Equal(t, 50, total)
	assert.Equal(t, 10, unread) // 10 "непрочитанных" (т.е. никем не прочитанных)
	assert.NoError(t, mock.ExpectationsWereMet())
}
