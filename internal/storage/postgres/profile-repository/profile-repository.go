package profile_repository

import (
	"2025_2_a4code/internal/domain"
	"2025_2_a4code/internal/http-server/middleware/logger"
	commonE "2025_2_a4code/internal/lib/errors"
	e "2025_2_a4code/internal/lib/wrapper"
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

type ProfileRepository struct {
	db *sql.DB
}

func New(db *sql.DB) *ProfileRepository {
	return &ProfileRepository{db: db}
}

func (repo *ProfileRepository) FindByID(ctx context.Context, id int64) (*domain.Profile, error) {
	const op = "storage.postgres.profile-repository.FindByID"
	log := logger.GetLogger(ctx).With(slog.String("op", op))

	const query = `
		SELECT 
			bp.id, bp.username, bp.domain, bp.created_at,
			p.password_hash, p.auth_version, p.name, p.surname, 
			p.patronymic, p.gender, p.birthday, p.image_path
		FROM 
			base_profile bp
		JOIN 
			profile p ON bp.id = p.base_profile_id
		WHERE 
			bp.id = $1`

	stmt, err := repo.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, e.Wrap(op, err)
	}
	defer stmt.Close()

	var profile domain.Profile
	var profileSurname, profilePatronymic, profileAvatar sql.NullString

	log.Debug("Executing FindByID query...")
	err = stmt.QueryRowContext(ctx, id).Scan(
		&profile.ID, &profile.Username, &profile.Domain, &profile.CreatedAt,
		&profile.PasswordHash, &profile.AuthVersion, &profile.Name, &profileSurname,
		&profilePatronymic, &profile.Gender, &profile.Birthday, &profileAvatar,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, e.Wrap(op, commonE.ErrNotFound)
		}
		return nil, e.Wrap(op, err)
	}

	if profileSurname.Valid {
		profile.Surname = profileSurname.String
	}
	if profilePatronymic.Valid {
		profile.Patronymic = profilePatronymic.String
	}
	if profileAvatar.Valid {
		profile.AvatarPath = profileAvatar.String
	}

	if profile.Domain == "flintmail.ru" {
		profile.IsOurSystemUser = true
	} else {
		profile.IsOurSystemUser = false
	}

	return &profile, nil
}

func (repo *ProfileRepository) FindSenderByID(ctx context.Context, id int64) (*domain.Sender, error) {
	const op = "storage.postgres.profile-repository.FindSenderByID"
	log := logger.GetLogger(ctx).With(slog.String("op", op))

	const query = `
		SELECT 
			bp.id, bp.username, bp.domain, 
			p.name, p.surname, p.image_path
		FROM 
			base_profile bp
		LEFT JOIN 
			profile p ON bp.id = p.base_profile_id
		WHERE 
			bp.id = $1`

	stmt, err := repo.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, e.Wrap(op, err)
	}
	defer stmt.Close()

	var sender domain.Sender
	var senderLogin, senderDomain string
	var senderName, senderSurname, senderAvatar sql.NullString

	log.Debug("Executing FindSender query...")
	err = stmt.QueryRowContext(ctx, id).Scan(
		&sender.Id, &senderLogin, &senderDomain,
		&senderName, &senderSurname, &senderAvatar,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, e.Wrap(op, commonE.ErrNotFound)
		}
		return nil, e.Wrap(op, err)
	}

	if senderName.Valid {
		sender.Username = senderName.String
		if senderSurname.Valid {
			sender.Username += (" " + senderSurname.String)
		}
	} else if senderSurname.Valid {
		sender.Username = senderSurname.String
	}

	if senderAvatar.Valid {
		sender.Avatar = senderAvatar.String
	}

	sender.Email = fmt.Sprintf("%s@%s", senderLogin, senderDomain)

	return &sender, nil
}

func (repo *ProfileRepository) UserExists(ctx context.Context, username string) (bool, error) {
	const op = "storage.postgres.profile-repository.UserExists"
	log := logger.GetLogger(ctx).With(slog.String("op", op))

	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM base_profile 
			WHERE username = $1 AND domain = 'flintmail.ru'
		)`

	stmt, err := repo.db.PrepareContext(ctx, query)
	if err != nil {
		return false, e.Wrap(op, err)
	}
	defer stmt.Close()

	var user_exists bool

	log.Debug("Executing UserExists query...")
	err = stmt.QueryRowContext(ctx, username).Scan(
		&user_exists,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, e.Wrap(op, commonE.ErrNotFound)
		}
		return false, e.Wrap(op, err)
	}

	return user_exists, nil
}

func (repo *ProfileRepository) CreateUser(ctx context.Context, profile domain.Profile) (int64, error) {
	const op = "storage.postgres.profile-repository.CreateUser"
	log := logger.GetLogger(ctx).With(slog.String("op", op))

	tx, err := repo.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, e.Wrap(op+": failed to begin transaction: ", err)
	}
	defer tx.Rollback()

	const query1 = `
		INSERT INTO base_profile (username, domain)
    	VALUES ($1, $2) 
		RETURNING id;
		`
	stmt, err := tx.PrepareContext(ctx, query1)
	if err != nil {
		return 0, e.Wrap(op, err)
	}
	defer stmt.Close()

	var newBaseProfileId int64

	log.Debug("Executing CreateBaseProfile query...")
	err = stmt.QueryRowContext(ctx, profile.Username, profile.Domain).Scan(
		&newBaseProfileId,
	)
	if err != nil {
		return 0, e.Wrap(op+": failed to create base profile: ", err)
	}

	const query2 = `
		INSERT INTO profile (base_profile_id, password_hash, name, surname, patronymic, gender, birthday)
    	VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id;
		`
	stmt, err = tx.PrepareContext(ctx, query2)
	if err != nil {
		return 0, e.Wrap(op, err)
	}
	defer stmt.Close()

	var newProfileId int64

	log.Debug("Executing CreateProfile query...")
	err = stmt.QueryRowContext(ctx, newBaseProfileId, profile.PasswordHash, profile.Name, profile.Surname, profile.Patronymic, profile.Gender, profile.Birthday).Scan(
		&newProfileId,
	)

	if err != nil {
		return 0, e.Wrap(op+": failed to create profile: ", err)
	}

	if err := repo.createSystemFolders(ctx, tx, newProfileId); err != nil {
		return 0, e.Wrap(op+": failed to create system folders: ", err)
	}

	if err := tx.Commit(); err != nil {
		return 0, e.Wrap(op+": failed to commit transaction: ", err)
	}

	return newBaseProfileId, nil
}

func (repo *ProfileRepository) createSystemFolders(ctx context.Context, tx *sql.Tx, profileID int64) error {
	const op = "storage.postgres.profile-repository.createSystemFolders"

	systemFolders := []struct {
		name  string
		ftype string
	}{
		{"Входящие", "inbox"},
		{"Отправленные", "sent"},
		{"Черновики", "draft"},
		{"Спам", "spam"},
		{"Корзина", "trash"},
	}

	const query = `
        INSERT INTO folder (profile_id, folder_name, folder_type)
        VALUES ($1, $2, $3)
    `

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return e.Wrap(op, err)
	}
	defer stmt.Close()

	for _, folder := range systemFolders {
		_, err = stmt.ExecContext(ctx, profileID, folder.name, folder.ftype)
		if err != nil {
			return e.Wrap(op+": failed to create folder "+folder.ftype, err)
		}
	}

	return nil
}

func (repo *ProfileRepository) FindByUsernameAndDomain(ctx context.Context, username string, emailDomain string) (*domain.Profile, error) {
	const op = "storage.postgres.profile-repository.FindByUsernameAndDomain"
	log := logger.GetLogger(ctx).With(slog.String("op", op))

	const query = `
		SELECT 
			bp.id, bp.created_at,
			p.password_hash, p.auth_version, p.name, p.surname, 
			p.patronymic, p.gender, p.birthday, p.image_path
		FROM 
			base_profile bp
		JOIN 
			profile p ON bp.id = p.base_profile_id
		WHERE 
			bp.username = $1 AND bp.domain = $2`

	stmt, err := repo.db.PrepareContext(ctx, query)
	if err != nil {
		return nil, e.Wrap(op, err)
	}
	defer stmt.Close()

	var profile domain.Profile
	profile.Username = username
	profile.Domain = emailDomain
	var profileSurname, profilePatronymic, profileAvatar sql.NullString

	log.Debug("Executing FindByUsernameAndDomain query...")
	err = stmt.QueryRowContext(ctx, username, emailDomain).Scan(
		&profile.ID, &profile.CreatedAt,
		&profile.PasswordHash, &profile.AuthVersion, &profile.Name, &profileSurname,
		&profilePatronymic, &profile.Gender, &profile.Birthday, &profileAvatar,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, e.Wrap(op, commonE.ErrNotFound)
		}
		return nil, e.Wrap(op, err)
	}

	if profileSurname.Valid {
		profile.Surname = profileSurname.String
	}
	if profilePatronymic.Valid {
		profile.Patronymic = profilePatronymic.String
	}
	if profileAvatar.Valid {
		profile.AvatarPath = profileAvatar.String
	}

	if profile.Domain == "flintmail.ru" {
		profile.IsOurSystemUser = true
	} else {
		profile.IsOurSystemUser = false
	}

	return &profile, nil
}

func (repo *ProfileRepository) FindInfoByID(ctx context.Context, profileID int64) (domain.ProfileInfo, error) {
	const op = "storage.postgres.profile-repository.FindInfoByID"
	log := logger.GetLogger(ctx).With(slog.String("op", op))

	const query = `
		SELECT 
			bp.id, bp.username, bp.created_at,
			p.name, p.surname, 
			p.patronymic, p.gender, p.birthday, p.image_path
		FROM 
			base_profile bp
		JOIN 
			profile p ON bp.id = p.base_profile_id
		WHERE 
			bp.id = $1`

	stmt, err := repo.db.PrepareContext(ctx, query)
	if err != nil {
		return domain.ProfileInfo{}, e.Wrap(op, err)
	}
	defer stmt.Close()

	var profileInfo domain.ProfileInfo
	var profileInfoSurname, profileInfoPatronymic, profileInfoAvatar sql.NullString

	log.Debug("Executing FindInfoByID query...")
	err = stmt.QueryRowContext(ctx, profileID).Scan(
		&profileInfo.ID, &profileInfo.Username, &profileInfo.CreatedAt,
		&profileInfo.Name, &profileInfoSurname,
		&profileInfoPatronymic, &profileInfo.Gender, &profileInfo.Birthday, &profileInfoAvatar,
	)

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return domain.ProfileInfo{}, e.Wrap(op, commonE.ErrNotFound)
		}
		return domain.ProfileInfo{}, e.Wrap(op, err)
	}

	if profileInfoSurname.Valid {
		profileInfo.Surname = profileInfoSurname.String
	}
	if profileInfoPatronymic.Valid {
		profileInfo.Patronymic = profileInfoPatronymic.String
	}
	if profileInfoAvatar.Valid {
		profileInfo.AvatarPath = profileInfoAvatar.String
	}

	return profileInfo, nil
}

func (repo *ProfileRepository) FindSettingsByProfileId(ctx context.Context, profileID int64) (domain.Settings, error) {
	const op = "storage.postgres.profile-repository.FindSettingsById"
	log := logger.GetLogger(ctx).With(slog.String("op", op))

	const query = `
        SELECT 
            s.id, s.profile_id, s.notification_tolerance, s.language, s.theme, s.signature,
            p.id as actual_profile_id
        FROM 
            base_profile bp
        JOIN 
            profile p ON bp.id = p.base_profile_id
        LEFT JOIN 
            settings s ON p.id = s.profile_id
        WHERE 
            bp.id = $1`

	stmt, err := repo.db.PrepareContext(ctx, query)
	if err != nil {
		return domain.Settings{}, e.Wrap(op, err)
	}
	defer stmt.Close()

	var settings domain.Settings
	var actualProfileID int64
	var settingsID sql.NullInt64
	var settingsProfileID sql.NullInt64
	var notificationTolerance, language, theme sql.NullString
	var signatureNullable sql.NullString

	log.Debug("Executing FindSettingsByProfileId query...")
	err = stmt.QueryRowContext(ctx, profileID).Scan(
		&settingsID, &settingsProfileID, &notificationTolerance,
		&language, &theme, &signatureNullable,
		&actualProfileID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return domain.Settings{}, e.Wrap(op, commonE.ErrNotFound)
		}
		return domain.Settings{}, e.Wrap(op, err)
	}

	if !settingsID.Valid {
		defaultSettings := domain.SetDefaultSettings(actualProfileID)
		return defaultSettings, nil
	}

	settings.ID = settingsID.Int64

	settings.ProfileID = actualProfileID

	settings.NotificationTolerance = notificationTolerance.String
	settings.Language = language.String
	settings.Theme = theme.String

	if signatureNullable.Valid && signatureNullable.String != "" {
		settings.Signatures = []string{signatureNullable.String}
	} else {
		settings.Signatures = []string{}
	}

	return settings, nil
}

func (repo *ProfileRepository) UpdateProfileInfo(ctx context.Context, profileID int64, info domain.ProfileUpdate) error {
	const op = "storage.postgres.profile-repository.UpdateProfileInfo"
	log := logger.GetLogger(ctx).With(slog.String("op", op))

	const query = `
		UPDATE profile
		SET name = $1,
			surname = $2,
			patronymic = $3,
			gender = $4,
			birthday = $5
		WHERE base_profile_id = $6
	`

	stmt, err := repo.db.PrepareContext(ctx, query)
	if err != nil {
		return e.Wrap(op, err)
	}
	defer stmt.Close()

	name := toNullString(info.Name)
	surname := toNullString(info.Surname)
	patronymic := toNullString(info.Patronymic)
	gender := toNullString(info.Gender)

	var birthday sql.NullTime
	if info.Birthday != nil {
		birthday = sql.NullTime{
			Time:  *info.Birthday,
			Valid: true,
		}
	}

	log.Debug("Executing UpdateProfileInfo query...")
	_, err = stmt.ExecContext(ctx, name, surname, patronymic, gender, birthday, profileID)
	if err != nil {
		return e.Wrap(op, err)
	}

	return nil
}

func (repo *ProfileRepository) InsertProfileAvatar(ctx context.Context, profileID int64, avatarURL string) error {
	const op = "storage.postgres.profile-repository.InsertProfileAvatar"
	log := logger.GetLogger(ctx).With(slog.String("op", op))

	const query = `
		UPDATE profile
		SET image_path = $1
		WHERE base_profile_id = $2
		`

	stmt, err := repo.db.PrepareContext(ctx, query)
	if err != nil {
		return e.Wrap(op, err)
	}
	defer stmt.Close()

	log.Debug("Executing InsertProfileAvatar query...")
	_, err = stmt.ExecContext(ctx, avatarURL, profileID)
	if err != nil {
		return e.Wrap(op, err)
	}
	return nil
}

func toNullString(value string) sql.NullString {
	value = strings.TrimSpace(value)
	if value == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: value, Valid: true}
}
