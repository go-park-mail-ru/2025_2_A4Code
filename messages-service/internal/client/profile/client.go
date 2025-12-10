package profileclient

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"

	profilepb "2025_2_a4code/profile-service/pkg/profileproto"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type ProfileClient struct {
	client profilepb.ProfileServiceClient
	conn   *grpc.ClientConn
}

func New(profileServiceAddr string) (*ProfileClient, error) {
	conn, err := grpc.NewClient(
		profileServiceAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithTimeout(5*time.Second),
	)
	if err != nil {
		return nil, err
	}

	client := profilepb.NewProfileServiceClient(conn)

	return &ProfileClient{
		client: client,
		conn:   conn,
	}, nil
}

func (c *ProfileClient) GetUserIDByEmail(ctx context.Context, email string) (int64, error) {
	var username, domain string
	if at := strings.Index(email, "@"); at != -1 {
		username = email[:at]
		domain = email[at+1:]
	} else {
		username = email
		domain = "flintmail.ru"
	}

	resp, err := c.client.FindByUsernameAndDomain(ctx, &profilepb.FindByUsernameAndDomainRequest{
		Username: username,
		Domain:   domain,
	})
	if err != nil {
		slog.Error("Failed to get user by email", "email", email, "error", err)
		return 0, err
	}

	id, err := strconv.ParseInt(resp.Profile.Id, 10, 64)
	if err != nil {
		slog.Error("Failed to parse profile ID", "id", resp.Profile.Id, "error", err)
		return 0, err
	}

	return id, nil
}

func (c *ProfileClient) GetUserEmailByID(ctx context.Context, userID int64) (string, error) {
	resp, err := c.client.FindByID(ctx, &profilepb.FindByIDRequest{
		ProfileId: userID,
	})
	if err != nil {
		slog.Error("Failed to get user by ID", "user_id", userID, "error", err)
		return "", err
	}

	return resp.Profile.Username + "@" + "flintmail.ru", nil
}

func (c *ProfileClient) Close() error {
	return c.conn.Close()
}
