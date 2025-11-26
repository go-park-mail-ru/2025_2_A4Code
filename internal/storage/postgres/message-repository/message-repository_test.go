package message_repository

import (
	"2025_2_a4code/internal/domain"
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

func TestMessageRepository_FindByMessageID(t *testing.T) {
	ctx, repo, mock := setupTest(t)
	mockMessageID := int64(1)
	mockTime := time.Now()

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

	mock.ExpectPrepare(quote(query)).
		ExpectQuery().
		WithArgs(mockMessageID).
		WillReturnRows(sqlmock.NewRows([]string{
			"m.id", "m.topic", "m.text", "m.date_of_dispatch",
			"bp.id", "bp.username", "bp.domain",
			"p.name", "p.surname", "p.image_path",
			"pm.read_status",
		}).AddRow(
			mockMessageID, "Test Topic", "This is a test message text", mockTime,
			int64(2), "sender", "example.com",
			"John", "Doe", "avatar.jpg",
			false,
		))

	message, err := repo.FindByMessageID(ctx, mockMessageID)

	assert.NoError(t, err)
	assert.NotNil(t, message)
	assert.Equal(t, "Test Topic", message.Topic)
	assert.Equal(t, "This is a test message te...", message.Snippet)
	assert.Equal(t, "John Doe", message.Sender.Username)
	assert.Equal(t, "sender@example.com", message.Sender.Email)
	assert.False(t, message.IsRead)
	assert.NoError(t, mock.ExpectationsWereMet())
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

	t.Run("ErrorOnExec", func(t *testing.T) {
		mock.ExpectPrepare(quote(query)).
			ExpectExec().
			WithArgs(threadID, sqlmock.AnyArg(), messageID).
			WillReturnError(fmt.Errorf("db error"))

		err := repo.SaveThreadIdToMessage(ctx, messageID, threadID)

		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
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

func TestMessageRepository_MarkMessageAsSpam(t *testing.T) {
	ctx, repo, mock := setupTest(t)
	messageID := int64(123)
	profileID := int64(1)

	t.Run("Success", func(t *testing.T) {
		mock.ExpectBegin()

		mock.ExpectQuery(`SELECT id FROM folder WHERE profile_id = \$1 AND folder_type = 'spam'`).
			WithArgs(profileID).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(5)))

		mock.ExpectExec(`INSERT INTO folder_profile_message`).
			WithArgs(messageID, int64(5)).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectExec(`DELETE FROM folder_profile_message`).
			WithArgs(messageID, profileID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		mock.ExpectCommit()

		err := repo.MarkMessageAsSpam(ctx, messageID, profileID)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("ErrorOnGetSpamFolder", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT id FROM folder WHERE profile_id = \$1 AND folder_type = 'spam'`).
			WithArgs(profileID).
			WillReturnError(fmt.Errorf("db error"))
		mock.ExpectRollback()

		err := repo.MarkMessageAsSpam(ctx, messageID, profileID)

		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestMessageRepository_SaveDraft(t *testing.T) {
	ctx, repo, mock := setupTest(t)
	profileID := int64(1)
	draftID := "123"
	topic := "Draft Topic"
	text := "Draft Text"
	receiverEmail := "test@example.com"

	t.Run("UpdateExistingDraft", func(t *testing.T) {
		mock.ExpectBegin()

		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM folder_profile_message fpm`).
			WithArgs(int64(123), profileID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		mock.ExpectExec(`UPDATE message SET topic = \$1, text = \$2, updated_at = \$3 WHERE id = \$4`).
			WithArgs(topic, text, sqlmock.AnyArg(), int64(123)).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectExec(`INSERT INTO profile_message`).
			WithArgs(profileID, int64(123)).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectCommit()

		resultID, err := repo.SaveDraft(ctx, profileID, draftID, receiverEmail, topic, text)

		assert.NoError(t, err)
		assert.Equal(t, int64(123), resultID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("CreateNewDraft", func(t *testing.T) {
		mock.ExpectBegin()

		mock.ExpectQuery(`SELECT base_profile_id FROM profile WHERE id = \$1`).
			WithArgs(profileID).
			WillReturnRows(sqlmock.NewRows([]string{"base_profile_id"}).AddRow(int64(10)))

		mock.ExpectQuery(`INSERT INTO message`).
			WithArgs(topic, text, sqlmock.AnyArg(), int64(10)).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(456)))

		mock.ExpectQuery(`SELECT id FROM folder WHERE profile_id = \$1 AND folder_type = 'draft'`).
			WithArgs(profileID).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(20)))

		mock.ExpectExec(`INSERT INTO folder_profile_message`).
			WithArgs(int64(456), int64(20)).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectExec(`INSERT INTO profile_message`).
			WithArgs(profileID, int64(456)).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectCommit()

		resultID, err := repo.SaveDraft(ctx, profileID, "", receiverEmail, topic, text)

		assert.NoError(t, err)
		assert.Equal(t, int64(456), resultID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestMessageRepository_IsDraftBelongsToUser(t *testing.T) {
	ctx, repo, mock := setupTest(t)
	draftID := int64(123)
	profileID := int64(1)

	t.Run("DraftBelongsToUser", func(t *testing.T) {
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM folder_profile_message fpm`).
			WithArgs(draftID, profileID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		belongs, err := repo.IsDraftBelongsToUser(ctx, draftID, profileID)

		assert.NoError(t, err)
		assert.True(t, belongs)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("DraftDoesNotBelongToUser", func(t *testing.T) {
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM folder_profile_message fpm`).
			WithArgs(draftID, profileID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))

		belongs, err := repo.IsDraftBelongsToUser(ctx, draftID, profileID)

		assert.NoError(t, err)
		assert.False(t, belongs)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestMessageRepository_DeleteDraft(t *testing.T) {
	ctx, repo, mock := setupTest(t)
	draftID := int64(123)
	profileID := int64(1)

	t.Run("Success", func(t *testing.T) {
		mock.ExpectBegin()

		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM folder_profile_message fpm`).
			WithArgs(draftID, profileID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		mock.ExpectExec(`DELETE FROM folder_profile_message`).
			WithArgs(draftID, profileID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		mock.ExpectExec(`DELETE FROM message WHERE id = \$1`).
			WithArgs(draftID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectCommit()

		err := repo.DeleteDraft(ctx, draftID, profileID)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("DraftNotBelongsToUser", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM folder_profile_message fpm`).
			WithArgs(draftID, profileID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(0))
		mock.ExpectRollback()

		err := repo.DeleteDraft(ctx, draftID, profileID)

		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestMessageRepository_SendDraft(t *testing.T) {
	ctx, repo, mock := setupTest(t)
	draftID := int64(123)
	profileID := int64(1)

	t.Run("Success", func(t *testing.T) {
		mock.ExpectBegin()

		mock.ExpectQuery(`SELECT m.topic, m.text, m.sender_base_profile_id FROM message m WHERE m.id = \$1`).
			WithArgs(draftID).
			WillReturnRows(sqlmock.NewRows([]string{"topic", "text", "sender_base_profile_id"}).
				AddRow("Topic", "Text", int64(10)))

		mock.ExpectExec(`UPDATE message SET date_of_dispatch = \$1, updated_at = \$2 WHERE id = \$3`).
			WithArgs(sqlmock.AnyArg(), sqlmock.AnyArg(), draftID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectExec(`DELETE FROM folder_profile_message`).
			WithArgs(draftID, profileID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		mock.ExpectQuery(`SELECT id FROM folder WHERE profile_id = \$1 AND folder_type = 'sent'`).
			WithArgs(profileID).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(20)))

		mock.ExpectExec(`INSERT INTO folder_profile_message`).
			WithArgs(draftID, int64(20)).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectQuery(`SELECT id FROM folder WHERE profile_id = \$1 AND folder_type = 'inbox'`).
			WithArgs(profileID).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(30)))

		mock.ExpectExec(`INSERT INTO folder_profile_message`).
			WithArgs(draftID, int64(30)).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectCommit()

		err := repo.SendDraft(ctx, draftID, profileID)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestMessageRepository_GetDraft(t *testing.T) {
	ctx, repo, mock := setupTest(t)
	draftID := int64(123)
	profileID := int64(1)

	t.Run("Success", func(t *testing.T) {
		mock.ExpectQuery(`SELECT COUNT\(\*\) FROM folder_profile_message fpm`).
			WithArgs(draftID, profileID).
			WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))

		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT`).
			WithArgs(draftID, profileID).
			WillReturnRows(sqlmock.NewRows([]string{
				"m.id", "m.topic", "m.text", "m.date_of_dispatch",
				"bp.id", "bp.username", "bp.domain",
				"p.name", "p.surname", "p.image_path",
				"t.id", "t.root_message_id",
				"f.id", "f.profile_id", "f.folder_name", "f.folder_type",
			}).AddRow(
				draftID, "Topic", "Text", time.Now(),
				int64(2), "user", "domain.com",
				sql.NullString{}, sql.NullString{}, sql.NullString{},
				sql.NullInt64{}, sql.NullInt64{},
				sql.NullInt64{}, sql.NullInt64{}, sql.NullString{}, sql.NullString{},
			))
		mock.ExpectQuery(`SELECT id, file_type, size, storage_path, message_id FROM file`).
			WithArgs(draftID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "file_type", "size", "storage_path", "message_id"}))
		mock.ExpectCommit()

		draft, err := repo.GetDraft(ctx, draftID, profileID)

		assert.NoError(t, err)
		assert.NotNil(t, draft)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestMessageRepository_MoveToFolder(t *testing.T) {
	ctx, repo, mock := setupTest(t)
	profileID := int64(1)
	messageID := int64(123)
	folderID := int64(5)

	t.Run("Success", func(t *testing.T) {
		mock.ExpectBegin()

		mock.ExpectExec(`DELETE FROM folder_profile_message`).
			WithArgs(messageID, profileID).
			WillReturnResult(sqlmock.NewResult(0, 2))

		mock.ExpectExec(`INSERT INTO folder_profile_message`).
			WithArgs(messageID, folderID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectCommit()

		err := repo.MoveToFolder(ctx, profileID, messageID, folderID)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestMessageRepository_GetFolderByType(t *testing.T) {
	ctx, repo, mock := setupTest(t)
	profileID := int64(1)

	t.Run("Success", func(t *testing.T) {
		mock.ExpectQuery(`SELECT id FROM folder WHERE profile_id = \$1 AND folder_type = \$2`).
			WithArgs(profileID, "inbox").
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(5)))

		folderID, err := repo.GetFolderByType(ctx, profileID, "inbox")

		assert.NoError(t, err)
		assert.Equal(t, int64(5), folderID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FolderNotFound", func(t *testing.T) {
		mock.ExpectQuery(`SELECT id FROM folder WHERE profile_id = \$1 AND folder_type = \$2`).
			WithArgs(profileID, "nonexistent").
			WillReturnError(sql.ErrNoRows)

		folderID, err := repo.GetFolderByType(ctx, profileID, "nonexistent")

		assert.Error(t, err)
		assert.Equal(t, int64(0), folderID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestMessageRepository_ShouldMarkAsRead(t *testing.T) {
	ctx, repo, mock := setupTest(t)
	messageID := int64(123)
	profileID := int64(1)

	t.Run("ShouldMark", func(t *testing.T) {
		mock.ExpectQuery(`SELECT`).
			WithArgs(messageID, profileID).
			WillReturnRows(sqlmock.NewRows([]string{"should_mark"}).AddRow(true))

		shouldMark, err := repo.ShouldMarkAsRead(ctx, messageID, profileID)

		assert.NoError(t, err)
		assert.True(t, shouldMark)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("ShouldNotMark", func(t *testing.T) {
		mock.ExpectQuery(`SELECT`).
			WithArgs(messageID, profileID).
			WillReturnRows(sqlmock.NewRows([]string{"should_mark"}).AddRow(false))

		shouldMark, err := repo.ShouldMarkAsRead(ctx, messageID, profileID)

		assert.NoError(t, err)
		assert.False(t, shouldMark)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("NoRows", func(t *testing.T) {
		mock.ExpectQuery(`SELECT`).
			WithArgs(messageID, profileID).
			WillReturnError(sql.ErrNoRows)

		shouldMark, err := repo.ShouldMarkAsRead(ctx, messageID, profileID)

		assert.NoError(t, err)
		assert.False(t, shouldMark)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestMessageRepository_CreateFolder(t *testing.T) {
	ctx, repo, mock := setupTest(t)
	profileID := int64(1)
	folderName := "Test Folder"

	t.Run("Success", func(t *testing.T) {
		mock.ExpectQuery(`SELECT 1 FROM folder`).
			WithArgs(profileID, folderName, int64(0)).
			WillReturnError(sql.ErrNoRows)

		mock.ExpectQuery(`INSERT INTO folder`).
			WithArgs(profileID, folderName).
			WillReturnRows(sqlmock.NewRows([]string{"id", "folder_name", "folder_type"}).
				AddRow(int64(5), folderName, "custom"))

		folder, err := repo.CreateFolder(ctx, profileID, folderName)

		assert.NoError(t, err)
		assert.NotNil(t, folder)
		assert.Equal(t, folderName, folder.Name)
		assert.Equal(t, domain.FolderType("custom"), folder.Type)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("FolderAlreadyExists", func(t *testing.T) {
		mock.ExpectQuery(`SELECT 1 FROM folder`).
			WithArgs(profileID, folderName, int64(0)).
			WillReturnRows(sqlmock.NewRows([]string{"1"}).AddRow(1))

		folder, err := repo.CreateFolder(ctx, profileID, folderName)

		assert.Error(t, err)
		assert.Nil(t, folder)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestMessageRepository_GetUserFolders(t *testing.T) {
	ctx, repo, mock := setupTest(t)
	profileID := int64(1)

	t.Run("Success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{"id", "folder_name", "folder_type"}).
			AddRow(int64(1), "Inbox", "inbox").
			AddRow(int64(2), "Sent", "sent").
			AddRow(int64(3), "Custom Folder", "custom")

		mock.ExpectQuery(`SELECT id, folder_name, folder_type FROM folder`).
			WithArgs(profileID).
			WillReturnRows(rows)

		folders, err := repo.GetUserFolders(ctx, profileID)

		assert.NoError(t, err)
		assert.Len(t, folders, 3)
		assert.Equal(t, "Inbox", folders[0].Name)
		assert.Equal(t, "Custom Folder", folders[2].Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestMessageRepository_RenameFolder(t *testing.T) {
	ctx, repo, mock := setupTest(t)
	profileID := int64(1)
	folderID := int64(5)
	newName := "Renamed Folder"

	t.Run("Success", func(t *testing.T) {
		mock.ExpectQuery(`SELECT 1 FROM folder`).
			WithArgs(profileID, newName, folderID).
			WillReturnError(sql.ErrNoRows)

		mock.ExpectQuery(`UPDATE folder`).
			WithArgs(newName, sqlmock.AnyArg(), folderID, profileID).
			WillReturnRows(sqlmock.NewRows([]string{"id", "folder_name", "folder_type"}).
				AddRow(folderID, newName, "custom"))

		folder, err := repo.RenameFolder(ctx, profileID, folderID, newName)

		assert.NoError(t, err)
		assert.NotNil(t, folder)
		assert.Equal(t, newName, folder.Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestMessageRepository_DeleteFolder(t *testing.T) {
	ctx, repo, mock := setupTest(t)
	profileID := int64(1)
	folderID := int64(5)

	t.Run("Success", func(t *testing.T) {
		mock.ExpectBegin()

		mock.ExpectQuery(`SELECT folder_type FROM folder`).
			WithArgs(folderID, profileID).
			WillReturnRows(sqlmock.NewRows([]string{"folder_type"}).AddRow("custom"))

		mock.ExpectQuery(`SELECT id FROM folder WHERE profile_id = \$1 AND folder_type = 'trash'`).
			WithArgs(profileID).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(int64(10)))

		mock.ExpectExec(`INSERT INTO folder_profile_message`).
			WithArgs(int64(10), folderID).
			WillReturnResult(sqlmock.NewResult(0, 3))

		mock.ExpectExec(`DELETE FROM folder_profile_message`).
			WithArgs(folderID).
			WillReturnResult(sqlmock.NewResult(0, 3))

		mock.ExpectExec(`DELETE FROM folder WHERE id = \$1`).
			WithArgs(folderID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectCommit()

		err := repo.DeleteFolder(ctx, profileID, folderID)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("SystemFolder", func(t *testing.T) {
		mock.ExpectBegin()
		mock.ExpectQuery(`SELECT folder_type FROM folder`).
			WithArgs(folderID, profileID).
			WillReturnRows(sqlmock.NewRows([]string{"folder_type"}).AddRow("inbox"))
		mock.ExpectRollback()

		err := repo.DeleteFolder(ctx, profileID, folderID)

		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestMessageRepository_DeleteMessageFromFolder(t *testing.T) {
	ctx, repo, mock := setupTest(t)
	profileID := int64(1)
	messageID := int64(123)
	folderID := int64(5)

	t.Run("Success", func(t *testing.T) {
		mock.ExpectExec(`DELETE FROM folder_profile_message`).
			WithArgs(messageID, folderID, profileID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.DeleteMessageFromFolder(ctx, profileID, messageID, folderID)

		assert.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("NoRowsAffected", func(t *testing.T) {
		mock.ExpectExec(`DELETE FROM folder_profile_message`).
			WithArgs(messageID, folderID, profileID).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.DeleteMessageFromFolder(ctx, profileID, messageID, folderID)

		assert.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestMessageRepository_GetFolderMessagesWithKeysetPagination(t *testing.T) {
	ctx, repo, mock := setupTest(t)
	profileID := int64(1)
	folderID := int64(5)
	lastMessageID := int64(100)
	lastDatetime := time.Now().Add(-time.Hour)
	limit := 10

	t.Run("Success", func(t *testing.T) {
		rows := sqlmock.NewRows([]string{
			"m.id", "m.topic", "m.text", "m.date_of_dispatch", "pm.read_status",
			"bp.id", "bp.username", "bp.domain",
			"p.name", "p.surname", "p.image_path",
		}).
			AddRow(
				int64(101), "Topic 1", "Message text 1", time.Now(), false,
				int64(2), "user1", "domain.com",
				sql.NullString{String: "John", Valid: true}, sql.NullString{String: "Doe", Valid: true}, sql.NullString{String: "avatar1.jpg", Valid: true},
			).
			AddRow(
				int64(102), "Topic 2", "Message text 2", time.Now().Add(-30*time.Minute), true,
				int64(3), "user2", "domain.com",
				sql.NullString{String: "Jane", Valid: true}, sql.NullString{String: "Smith", Valid: true}, sql.NullString{String: "avatar2.jpg", Valid: true},
			)

		mock.ExpectQuery(`SELECT`).
			WithArgs(profileID, folderID, lastMessageID, lastDatetime.Unix(), limit).
			WillReturnRows(rows)

		messages, err := repo.GetFolderMessagesWithKeysetPagination(ctx, profileID, folderID, lastMessageID, lastDatetime, limit)

		assert.NoError(t, err)
		assert.Len(t, messages, 2)
		assert.Equal(t, "101", messages[0].ID)
		assert.Equal(t, "John Doe", messages[0].Sender.Username)
		assert.Equal(t, "Message text 1...", messages[0].Snippet)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestMessageRepository_GetFolderMessagesInfo(t *testing.T) {
	ctx, repo, mock := setupTest(t)
	profileID := int64(1)
	folderID := int64(5)

	t.Run("Success", func(t *testing.T) {
		mock.ExpectQuery(`SELECT`).
			WithArgs(profileID, folderID).
			WillReturnRows(sqlmock.NewRows([]string{"total_count", "unread_count"}).
				AddRow(100, 25))

		info, err := repo.GetFolderMessagesInfo(ctx, profileID, folderID)

		assert.NoError(t, err)
		assert.Equal(t, 100, info.MessageTotal)
		assert.Equal(t, 25, info.MessageUnread)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestMessageRepository_SaveMessageWithFolderDistribution(t *testing.T) {
	ctx, repo, mock := setupTest(t)
	receiverEmail := "receiver@domain.com"
	senderBaseProfileID := int64(1)
	topic := "Test Topic"
	text := "Test Text"
	expectedMessageID := int64(123)
	expectedReceiverProfileID := int64(456)
	expectedSenderProfileID := int64(789)

	t.Run("Success", func(t *testing.T) {
		mock.ExpectBegin()

		mock.ExpectQuery(`INSERT INTO message`).
			WithArgs(topic, text, sqlmock.AnyArg(), senderBaseProfileID).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(expectedMessageID))

		username := strings.Split(receiverEmail, "@")[0]
		domain := strings.Split(receiverEmail, "@")[1]

		mock.ExpectQuery(`SELECT p.id FROM profile p`).
			WithArgs(username, domain).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(expectedReceiverProfileID))

		mock.ExpectQuery(`SELECT id FROM profile WHERE base_profile_id = \$1`).
			WithArgs(senderBaseProfileID).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(expectedSenderProfileID))

		mock.ExpectExec(`INSERT INTO folder_profile_message`).
			WithArgs(expectedMessageID, expectedReceiverProfileID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectExec(`INSERT INTO folder_profile_message`).
			WithArgs(expectedMessageID, expectedSenderProfileID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectExec(`INSERT INTO profile_message`).
			WithArgs(expectedReceiverProfileID, expectedMessageID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectExec(`INSERT INTO profile_message`).
			WithArgs(expectedSenderProfileID, expectedMessageID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectCommit()

		messageID, err := repo.SaveMessageWithFolderDistribution(ctx, receiverEmail, senderBaseProfileID, topic, text)

		assert.NoError(t, err)
		assert.Equal(t, expectedMessageID, messageID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestMessageRepository_ReplyToMessageWithFolderDistribution(t *testing.T) {
	ctx, repo, mock := setupTest(t)
	receiverEmail := "receiver@domain.com"
	senderProfileID := int64(1)
	threadRoot := int64(100)
	topic := "Reply Topic"
	text := "Reply Text"
	expectedMessageID := int64(123)
	expectedReceiverProfileID := int64(456)
	expectedSenderBaseProfileID := int64(789)

	t.Run("Success", func(t *testing.T) {
		mock.ExpectBegin()

		mock.ExpectQuery(`SELECT base_profile_id FROM profile WHERE id = \$1`).
			WithArgs(senderProfileID).
			WillReturnRows(sqlmock.NewRows([]string{"base_profile_id"}).AddRow(expectedSenderBaseProfileID))

		mock.ExpectQuery(`INSERT INTO message`).
			WithArgs(topic, text, sqlmock.AnyArg(), expectedSenderBaseProfileID, threadRoot).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(expectedMessageID))

		username := strings.Split(receiverEmail, "@")[0]
		domain := strings.Split(receiverEmail, "@")[1]

		mock.ExpectQuery(`SELECT p.id FROM profile p`).
			WithArgs(username, domain).
			WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow(expectedReceiverProfileID))

		mock.ExpectExec(`INSERT INTO folder_profile_message`).
			WithArgs(expectedMessageID, expectedReceiverProfileID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectExec(`INSERT INTO folder_profile_message`).
			WithArgs(expectedMessageID, senderProfileID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectExec(`INSERT INTO profile_message`).
			WithArgs(expectedReceiverProfileID, expectedMessageID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectExec(`INSERT INTO profile_message`).
			WithArgs(senderProfileID, expectedMessageID).
			WillReturnResult(sqlmock.NewResult(1, 1))

		mock.ExpectCommit()

		messageID, err := repo.ReplyToMessageWithFolderDistribution(ctx, receiverEmail, senderProfileID, threadRoot, topic, text)

		assert.NoError(t, err)
		assert.Equal(t, expectedMessageID, messageID)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestMessageRepository_IsUsersMessage(t *testing.T) {
	ctx, repo, mock := setupTest(t)
	messageID := int64(123)
	profileID := int64(1)

	t.Run("IsUsersMessage", func(t *testing.T) {
		mock.ExpectPrepare(`SELECT EXISTS`).
			ExpectQuery().
			WithArgs(messageID, profileID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(true))

		isUserMessage, err := repo.IsUsersMessage(ctx, messageID, profileID)

		assert.NoError(t, err)
		assert.True(t, isUserMessage)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("IsNotUsersMessage", func(t *testing.T) {
		mock.ExpectPrepare(`SELECT EXISTS`).
			ExpectQuery().
			WithArgs(messageID, profileID).
			WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(false))

		isUserMessage, err := repo.IsUsersMessage(ctx, messageID, profileID)

		assert.NoError(t, err)
		assert.False(t, isUserMessage)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
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

func TestBuildSnippet(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		limit    int
		expected string
	}{
		{
			name:     "Text shorter than limit",
			text:     "Short text",
			limit:    20,
			expected: "Short text",
		},
		{
			name:     "Text longer than limit",
			text:     "This is a very long text that exceeds the limit",
			limit:    20,
			expected: "This is a very long...",
		},
		{
			name:     "Text equal to limit",
			text:     "Exactly twenty chars",
			limit:    20,
			expected: "Exactly twenty chars",
		},
		{
			name:     "Empty text",
			text:     "",
			limit:    20,
			expected: "",
		},
		{
			name:     "Zero limit",
			text:     "Some text",
			limit:    0,
			expected: "",
		},
		{
			name:     "Unicode characters",
			text:     "Привет мир это тест",
			limit:    10,
			expected: "Привет ми...",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildSnippet(tt.text, tt.limit)
			assert.Equal(t, tt.expected, result)
		})
	}
}
