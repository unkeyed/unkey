package clidevice

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/unkeyed/unkey/pkg/codes"
	"github.com/unkeyed/unkey/pkg/db"
	"github.com/unkeyed/unkey/pkg/fault"
)

// Start creates a device authorization and returns the code the user compares.
func (s *Service) Start(ctx context.Context, req StartRequest) (Started, error) {
	if err := s.requireEnabled(); err != nil {
		return Started{}, err
	}
	if err := s.limit(ctx, "start", req.RemoteIP, startLimitPerMinute); err != nil {
		return Started{}, err
	}
	s.deleteExpired(ctx)

	deviceName := truncate(strings.TrimSpace(req.DeviceName), 512)
	userAgent := truncate(strings.TrimSpace(req.UserAgent), 512)
	verificationURI := strings.TrimRight(s.DashboardBaseURL, "/")
	now := s.Clock.Now().UnixMilli()
	expiresAt := now + deviceTTL.Milliseconds()

	var lastErr error
	for range 5 {
		userCode, err := newUserCode()
		if err != nil {
			return Started{}, err
		}
		complete, err := dashboardVerificationURL(verificationURI, userCode)
		if err != nil {
			return Started{}, fault.Wrap(err,
				fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
				fault.Internal("cli dashboard url is invalid"),
				fault.Public("CLI login is not configured on this API."),
			)
		}

		loginID := newLoginID()
		err = db.Query.InsertDeviceAuthorization(ctx, s.DB.RW(), db.InsertDeviceAuthorizationParams{
			ID:        loginID,
			UserCode:  userCode,
			ExpiresAt: expiresAt,
			Status:    db.DeviceAuthorizationsStatusPending,
			DeviceName: sql.NullString{
				String: deviceName,
				Valid:  deviceName != "",
			},
			RequesterIp: sql.NullString{
				String: req.RemoteIP,
				Valid:  req.RemoteIP != "",
			},
			RequesterUserAgent: sql.NullString{
				String: userAgent,
				Valid:  userAgent != "",
			},
			CreatedAt: now,
		})
		if db.IsDuplicateKeyError(err) {
			lastErr = err
			continue
		}
		if err != nil {
			return Started{}, err
		}

		return Started{
			LoginID:         loginID,
			UserCode:        userCode,
			VerificationURL: complete,
			ExpiresIn:       int32(deviceTTL / time.Second),
			Interval:        pollIntervalSeconds,
		}, nil
	}

	return Started{}, fault.Wrap(lastErr,
		fault.Code(codes.App.Internal.ServiceUnavailable.URN()),
		fault.Internal("could not allocate a device login code"),
		fault.Public("Could not start CLI login. Try again."),
	)
}
