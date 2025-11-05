package profile_repository

import (
	"2025_2_a4code/internal/domain"
	commonE "2025_2_a4code/internal/lib/errors"
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/assert"
)

var (
	testCtx = context.Background()
)

func TestFindByID_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := New(db)

	testID := int64(10)
	testDomain := "flintmail.ru"
	testDate := time.Now().Truncate(time.Second)

	rows := sqlmock.NewRows([]string{
		"id", "username", "domain", "created_at",
		"password_hash", "auth_version", "name", "surname",
		"patronymic", "gender", "birthday", "image_path",
	}).AddRow(
		testID, "testuser", testDomain, testDate,
		"hash123", 1, "Иван", sql.NullString{String: "Иванов", Valid: true},
		sql.NullString{}, "m", testDate, sql.NullString{String: "/path/to/avatar.jpg", Valid: true},
	)

	mock.ExpectPrepare("SELECT").ExpectQuery().
		WithArgs(testID).
		WillReturnRows(rows)

	profile, err := repo.FindByID(testCtx, testID)

	assert.NoError(t, err)
	assert.Equal(t, testID, profile.ID)
	assert.Equal(t, testDomain, profile.Domain)
	assert.True(t, profile.IsOurSystemUser)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFindByID_NotFound(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := New(db)
	testID := int64(999)

	mock.ExpectPrepare("SELECT").ExpectQuery().
		WithArgs(testID).
		WillReturnError(sql.ErrNoRows)

	profile, err := repo.FindByID(testCtx, testID)

	assert.True(t, errors.Is(err, commonE.ErrNotFound))
	assert.Nil(t, profile)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFindSenderByID_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := New(db)
	testID := int64(10)
	testLogin := "testuser"
	testDomain := "flintmail.ru"
	testName := "Иван"
	testSurname := "Иванов"
	testAvatar := "/path/to/avatar.jpg"

	rows := sqlmock.NewRows([]string{
		"id", "username", "domain",
		"name", "surname", "image_path",
	}).AddRow(
		testID, testLogin, testDomain,
		sql.NullString{String: testName, Valid: true},
		sql.NullString{String: testSurname, Valid: true},
		sql.NullString{String: testAvatar, Valid: true},
	)

	mock.ExpectPrepare("SELECT").ExpectQuery().
		WithArgs(testID).
		WillReturnRows(rows)

	sender, err := repo.FindSenderByID(testCtx, testID)

	assert.NoError(t, err)
	assert.Equal(t, testID, sender.Id)
	assert.Equal(t, testLogin+"@"+testDomain, sender.Email)
	assert.Equal(t, testName+" "+testSurname, sender.Username)
	assert.Equal(t, testAvatar, sender.Avatar)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUserExists_True(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := New(db)
	testUsername := "testuser"

	rows := sqlmock.NewRows([]string{"exists"}).AddRow(true)

	mock.ExpectPrepare("SELECT EXISTS").ExpectQuery().
		WithArgs(testUsername).
		WillReturnRows(rows)

	exists, err := repo.UserExists(testCtx, testUsername)

	assert.NoError(t, err)
	assert.True(t, exists)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFindByUsernameAndDomain_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := New(db)

	testUsername := "testuser"
	testDomain := "flintmail.ru"
	testID := int64(10)
	testDate := time.Now().Truncate(time.Second)

	rows := sqlmock.NewRows([]string{
		"id", "created_at",
		"password_hash", "auth_version", "name", "surname",
		"patronymic", "gender", "birthday", "image_path",
	}).AddRow(
		testID, testDate,
		"hash123", 1, "Иван", sql.NullString{String: "Иванов", Valid: true},
		sql.NullString{}, "m", testDate, sql.NullString{},
	)

	mock.ExpectPrepare("SELECT").ExpectQuery().
		WithArgs(testUsername, testDomain).
		WillReturnRows(rows)

	profile, err := repo.FindByUsernameAndDomain(testCtx, testUsername, testDomain)

	assert.NoError(t, err)
	assert.Equal(t, testID, profile.ID)
	assert.Equal(t, testUsername, profile.Username)
	assert.Equal(t, testDomain, profile.Domain)
	assert.True(t, profile.IsOurSystemUser)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestUpdateProfileInfo_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := New(db)

	testProfileID := int64(10)
	testBirthday := time.Date(1990, 1, 1, 0, 0, 0, 0, time.UTC)

	updateInfo := domain.ProfileUpdate{
		Name:       "Обновленный",
		Surname:    "Тест",
		Patronymic: "Петрович",
		Gender:     "m",
		Birthday:   &testBirthday,
	}

	mock.ExpectPrepare("UPDATE profile").ExpectExec().
		WithArgs(
			sqlmock.AnyArg(), // Name
			sqlmock.AnyArg(), // Surname
			sqlmock.AnyArg(), // Patronymic
			sqlmock.AnyArg(), // Gender
			sqlmock.AnyArg(), // Birthday (sql.NullTime)
			testProfileID,
		).
		WillReturnResult(sqlmock.NewResult(0, 1))

	err = repo.UpdateProfileInfo(testCtx, testProfileID, updateInfo)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestFindSettingsByProfileId_Default(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := New(db)
	testProfileID := int64(10)
	actualProfileID := int64(5)

	rows := sqlmock.NewRows([]string{
		"id", "profile_id", "notification_tolerance", "language", "theme", "signature",
		"actual_profile_id",
	}).AddRow(
		sql.NullInt64{}, sql.NullInt64{}, sql.NullString{}, sql.NullString{}, sql.NullString{}, sql.NullString{},
		actualProfileID,
	)

	mock.ExpectPrepare("SELECT").ExpectQuery().
		WithArgs(testProfileID).
		WillReturnRows(rows)

	settings, err := repo.FindSettingsByProfileId(testCtx, testProfileID)

	assert.NoError(t, err)
	assert.Equal(t, actualProfileID, settings.ProfileID)
	assert.Equal(t, "normal", settings.NotificationTolerance)
	assert.Equal(t, "ru", settings.Language)
	assert.Equal(t, "light", settings.Theme)
	assert.Empty(t, settings.Signatures)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestInsertProfileAvatar_Success(t *testing.T) {
	db, mock, err := sqlmock.New()
	assert.NoError(t, err)
	defer db.Close()

	repo := New(db)

	testProfileID := int64(10)
	testAvatarURL := "/new/path/to/avatar.png"

	mock.ExpectPrepare("UPDATE profile").ExpectExec().
		WithArgs(testAvatarURL, testProfileID).
		WillReturnResult(sqlmock.NewResult(0, 1)) // 1 row affected

	err = repo.InsertProfileAvatar(testCtx, testProfileID, testAvatarURL)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
