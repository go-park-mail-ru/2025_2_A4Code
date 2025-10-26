package profile_repository

import (
	"2025_2_a4code/internal/domain"
	"context"
	"database/sql"
	"fmt"
)

type ProfileRepository struct {
	db *sql.DB
}

func New(db *sql.DB) *ProfileRepository {
	return &ProfileRepository{db: db}
}

func (repo *ProfileRepository) FindByID(id int64) (*domain.Profile, error) {
	const op = "storage.postgres.profile-repository.FindByID"

	const query = `
		SELECT 
			bp.id, bp.username, bp.domain, bp.created_at,
			p.password_hash, p.auth_version, p.name, COALESCE(p.surname, ''), 
			COALESCE(p.patronymic, ''), p.gender, p.birthday, COALESCE(p.avatar_path, '')
		FROM 
			base_profile bp
		JOIN 
			profile p ON bp.id = p.base_profile_id
		WHERE 
			bp.id = $1`

	stmt, err := repo.db.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf(op+`: %w`, err)
	}
	defer stmt.Close()

	var profile domain.Profile

	err = stmt.QueryRow(id).Scan(
		&profile.ID, &profile.Username, &profile.Domain, &profile.CreatedAt,
		&profile.PasswordHash, &profile.AuthVersion, &profile.Name, &profile.Surname,
		&profile.Patronymic, &profile.Gender, &profile.Birthday, &profile.AvatarPath,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // TODO: добавить кастомную ошибку
		}
		return nil, fmt.Errorf(op+`: query: %w`, err)
	}

	if profile.Domain == "a4mail.ru" {
		profile.IsOurSystemUser = true
	} else {
		profile.IsOurSystemUser = false
	}

	return &profile, nil
}

func (repo *ProfileRepository) FindSenderByID(id int64) (*domain.Sender, error) {
	const op = "storage.postgres.profile-repository.FindSenderByID"

	const query = `
		SELECT 
			bp.id, bp.username, bp.domain, 
			COALESCE(p.name, ''), COALESCE(p.surname, ''), COALESCE(p.avatar_path, '')
		FROM 
			base_profile bp
		LEFT JOIN 
			profile p ON bp.id = p.base_profile_id
		WHERE 
			bp.id = $1`

	stmt, err := repo.db.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf(op+`: %w`, err)
	}
	defer stmt.Close()

	var sender domain.Sender
	var senderLogin, senderDomain, senderName, senderSurname string

	err = stmt.QueryRow(id).Scan(
		&sender.Id, &senderLogin, &senderDomain,
		&senderName, &senderSurname, &sender.Avatar,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // TODO: добавить кастомную ошибку
		}
		return nil, fmt.Errorf(op+`: query: %w`, err)
	}

	sender.Email = senderLogin + senderDomain
	sender.Username = senderName + senderSurname

	return &sender, nil
}

func (repo *ProfileRepository) UserExists(ctx context.Context, username string) (bool, error) {
	const op = "storage.postgres.profile-repository.UserExists"

	const query = `
		SELECT EXISTS (
			SELECT 1
			FROM base_profile 
			WHERE username = $1 AND domain = 'a4mail.ru'
		)`

	stmt, err := repo.db.Prepare(query)
	if err != nil {
		return false, fmt.Errorf(op+`: %w`, err)
	}
	defer stmt.Close()

	var user_exists bool

	err = stmt.QueryRow(username).Scan(
		&user_exists,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil // TODO: добавить кастомную ошибку
		}
		return false, fmt.Errorf(op+`: query: %w`, err)
	}

	return user_exists, nil
}

func (repo *ProfileRepository) CreateUser(ctx context.Context, profile domain.Profile) (int64, error) {
	const op = "storage.postgres.profile-repository.CreateUser"

	tx, err := repo.db.Begin()
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}
	defer tx.Rollback()

	const query1 = `
		INSERT INTO base_profile (username, domain)
    	VALUES ($1, $2) 
		RETURNING id;
		`
	stmt, err := tx.Prepare(query1)
	if err != nil {
		return 0, fmt.Errorf(op+`: %w`, err)
	}
	defer stmt.Close()

	var newBaseProfileId int64

	err = stmt.QueryRow(profile.Username, profile.Domain).Scan(
		&newBaseProfileId,
	)
	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	const query2 = `
		INSERT INTO profile (base_profile_id, password_hash, name, surname, patronymic, gender, birthday)
    	VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id;
		`
	stmt, err = tx.Prepare(query2)
	if err != nil {
		return 0, fmt.Errorf(op+`: %w`, err)
	}
	defer stmt.Close()

	var newProfileId int64

	err = stmt.QueryRow(newBaseProfileId, profile.PasswordHash, profile.Name, profile.Surname, profile.Patronymic, profile.Gender, profile.Birthday).Scan(
		&newProfileId,
	)

	if err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("%s: %w", op, err)
	}

	return newBaseProfileId, nil
}

func (repo *ProfileRepository) FindByUsernameAndDomain(ctx context.Context, username string, emailDomain string) (*domain.Profile, error) {
	const op = "storage.postgres.profile-repository.FindByUsernameAndDomain"

	const query = `
		SELECT 
			bp.id, bp.created_at,
			p.password_hash, p.auth_version, p.name, COALESCE(p.surname, ''), 
			COALESCE(p.patronymic, ''), p.gender, p.birthday, COALESCE(p.avatar_path, '')
		FROM 
			base_profile bp
		JOIN 
			profile p ON bp.id = p.base_profile_id
		WHERE 
			bp.username = $1 AND bp.domain = $2`

	stmt, err := repo.db.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf(op+`: %w`, err)
	}
	defer stmt.Close()

	var profile domain.Profile
	profile.Username = username
	profile.Domain = emailDomain

	err = stmt.QueryRow(username, emailDomain).Scan(
		&profile.ID, &profile.CreatedAt,
		&profile.PasswordHash, &profile.AuthVersion, &profile.Name, &profile.Surname,
		&profile.Patronymic, &profile.Gender, &profile.Birthday, &profile.AvatarPath,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // TODO: добавить кастомную ошибку
		}
		return nil, fmt.Errorf(op+`: query: %w`, err)
	}

	if profile.Domain == "a4mail.ru" {
		profile.IsOurSystemUser = true
	} else {
		profile.IsOurSystemUser = false
	}

	return &profile, nil
}

func (repo *ProfileRepository) FindInfoByID(profileID int64) (domain.ProfileInfo, error) {
	const op = "storage.postgres.profile-repository.FindInfoByID"

	const query = `
		SELECT 
			bp.id, bp.username, bp.created_at,
			p.name, COALESCE(p.surname, ''), 
			COALESCE(p.patronymic, ''), p.gender, p.birthday, COALESCE(p.avatar_path, '')
		FROM 
			base_profile bp
		JOIN 
			profile p ON bp.id = p.base_profile_id
		WHERE 
			bp.id = $1`

	stmt, err := repo.db.Prepare(query)
	if err != nil {
		return domain.ProfileInfo{}, fmt.Errorf(op+`: %w`, err)
	}
	defer stmt.Close()

	var profileInfo domain.ProfileInfo

	err = stmt.QueryRow(profileID).Scan(
		&profileInfo.ID, &profileInfo.Username, &profileInfo.CreatedAt,
		&profileInfo.Name, &profileInfo.Surname,
		&profileInfo.Patronymic, &profileInfo.Gender, &profileInfo.Birthday, &profileInfo.AvatarPath,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return domain.ProfileInfo{}, nil // TODO: добавить кастомную ошибку
		}
		return domain.ProfileInfo{}, fmt.Errorf(op+`: query: %w`, err)
	}

	return profileInfo, nil
}

func (repo *ProfileRepository) FindSettingsById(profileID int64) (domain.Settings, error) {
	const op = "storage.postgres.profile-repository.FindSettingsById"

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

	stmt, err := repo.db.Prepare(query)
	if err != nil {
		return domain.Settings{}, fmt.Errorf("%s: %w", op, err)
	}
	defer stmt.Close()

	var settings domain.Settings
	var actualProfileID int64
	var settingsID sql.NullInt64
	var notificationTolerance, language, theme sql.NullString
	var signatureNullable sql.NullString

	err = stmt.QueryRow(profileID).Scan(
		&settingsID, &settings.ProfileID, &notificationTolerance,
		&language, &theme, &signatureNullable,
		&actualProfileID,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return domain.Settings{}, fmt.Errorf("%s: profile not found: %w", op, err)
		}
		return domain.Settings{}, fmt.Errorf("%s: query: %w", op, err)
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
