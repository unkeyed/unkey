package deploy

import (
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/unkeyed/unkey/pkg/uid"
	"github.com/unkeyed/unkey/svc/ctrl/internal/db"
)

var dnsLabelRegex = regexp.MustCompile(`^[a-z0-9]([a-z0-9-]*[a-z0-9])?$`)

func TestBuildDomainsFitsDNSLabelLimit(t *testing.T) {
	cases := []struct {
		name          string
		workspaceSlug string
		projectSlug   string
		appSlug       string
		branch        string
		forkOwner     string
		isProduction  bool
	}{
		{
			name:          "long branch",
			workspaceSlug: "stagejune26",
			projectSlug:   "homestead-home-tracking",
			appSlug:       "default",
			branch:        "MichaelUnkey/phase-8-ui-foundation",
		},
		{
			name:          "long slugs everywhere",
			workspaceSlug: strings.Repeat("w", 64),
			projectSlug:   strings.Repeat("kebap-", 40) + "kebap",
			appSlug:       "kebap-api",
			branch:        strings.Repeat("feature/", 30) + "kebap",
			forkOwner:     "kebap-contributor",
			isProduction:  true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			apex := "canary.unkey.app"
			domains := buildDomains(
				tc.workspaceSlug, tc.projectSlug, tc.appSlug, "preview",
				"b8e86a2da52626e6faabbc90c33bf7c8abc23b32", tc.branch, tc.forkOwner, apex,
				tc.isProduction, true, uid.New(uid.DeploymentPrefix),
			)
			require.NotEmpty(t, domains)

			for _, d := range domains {
				label, rest, found := strings.Cut(d.domain, ".")
				require.True(t, found, d.domain)
				require.Equal(t, apex, rest)
				require.LessOrEqual(t, len(label), 63, d.domain)
				require.Regexp(t, dnsLabelRegex, label)
			}
		})
	}
}

func TestBuildDomainsKeepsLongBranchesApart(t *testing.T) {
	shared := strings.Repeat("kebap-", 20)
	branchDomain := func(branch string) string {
		return domainBySticky(t, buildDomains(
			"stagejune26", "homestead-home-tracking", "default", "preview",
			"", branch, "", "canary.unkey.app",
			false, false, uid.New(uid.DeploymentPrefix),
		), db.FrontlineRoutesStickyBranch)
	}

	first := branchDomain(shared + "one")
	require.NotEqual(t, first, branchDomain(shared+"two"))
	require.Equal(t, first, branchDomain(shared+"one"), "the branch domain must survive redeploys to stay sticky")
}

func TestBuildDomainsLeavesShortLabelsUnchanged(t *testing.T) {
	domains := buildDomains(
		"stagejune26", "homestead-home-tracking", "default", "preview",
		"", "main", "", "canary.unkey.app",
		false, false, uid.New(uid.DeploymentPrefix),
	)

	require.Equal(t, "homestead-home-tracking-git-main-stagejune26.canary.unkey.app",
		domainBySticky(t, domains, db.FrontlineRoutesStickyBranch))
	require.Equal(t, "homestead-home-tracking-preview-stagejune26.canary.unkey.app",
		domainBySticky(t, domains, db.FrontlineRoutesStickyEnvironment))
}

func TestBuildDomainsKeepsWorkspaceSlugWhenCut(t *testing.T) {
	domains := buildDomains(
		"stagejune26", "homestead-home-tracking", "default", "preview",
		"", "MichaelUnkey/phase-8-ui-foundation", "", "canary.unkey.app",
		false, false, uid.New(uid.DeploymentPrefix),
	)

	require.Equal(t, "homestead-home-tracking-git-michaelunkey-p-9141f508-stagejune26.canary.unkey.app",
		domainBySticky(t, domains, db.FrontlineRoutesStickyBranch))
}

func TestCappedDomainEveryWorkspaceLength(t *testing.T) {
	for length := 3; length <= 64; length++ {
		workspaceSlug := strings.Repeat("w", length)
		label, _, _ := strings.Cut(cappedDomain(strings.Repeat("kebap", 12), workspaceSlug, "unkey.app"), ".")

		require.LessOrEqual(t, len(label), 63, label)
		require.Regexp(t, dnsLabelRegex, label)
		require.Contains(t, label, workspaceSlug[:min(length, 52)], "the workspace slug must survive the cut")
	}
}

func domainBySticky(t *testing.T, domains []newDomain, sticky db.FrontlineRoutesSticky) string {
	t.Helper()
	for _, d := range domains {
		if d.sticky == sticky {
			return d.domain
		}
	}
	require.FailNow(t, "no domain with sticky "+string(sticky))
	return ""
}
