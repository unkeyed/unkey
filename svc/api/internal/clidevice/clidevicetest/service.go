// Package clidevicetest builds a [clidevice.Service] backed by a test harness
// and a fake WorkOS device client, for route tests.
package clidevicetest

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/unkeyed/unkey/pkg/auth/workos"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/api/internal/clidevice"
	"github.com/unkeyed/unkey/svc/api/internal/testutil"
)

type devices struct {
	prefix string
	next   atomic.Int64
}

func (d *devices) CreateDevice(context.Context, string) (workos.DeviceCode, workos.DeviceAuthorization, error) {
	n := d.next.Add(1)
	return workos.DeviceCode(fmt.Sprintf("device-%d", n)), workos.DeviceAuthorization{
		UserCode:                fmt.Sprintf("%s-%04d", d.prefix, n),
		VerificationURI:         "https://auth.example/device",
		VerificationURIComplete: "https://auth.example/device",
		ExpiresIn:               300,
		Interval:                5,
	}, nil
}

// New returns a configured service whose user codes share a random prefix and
// end in -0001, -0002, ..., so tests sharing one database do not collide.
func New(h *testutil.Harness) *clidevice.Service {
	return &clidevice.Service{
		DB:               h.DB,
		Keys:             h.Keys,
		Auditlogs:        h.Auditlogs,
		Clock:            h.Clock,
		Ratelimit:        h.Ratelimit,
		Devices:          &devices{prefix: strings.ToUpper(uid.Secure(10)), next: atomic.Int64{}},
		ClientID:         "client_test",
		DashboardBaseURL: "http://dashboard.test",
	}
}
