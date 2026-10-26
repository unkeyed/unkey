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
				tc.isProduction, true, uid.New(uid.DeploymentPrefix), uid.New(uid.AppPrefix),
			)
			require.NotEmpty(t, domains)

			for _, d := range domains {
				for _, domain := range []string{d.domain, d.fallback} {
					label, rest, found := strings.Cut(domain, ".")
					require.True(t, found, domain)
					require.Equal(t, apex, rest)
					require.LessOrEqual(t, len(label), 63, domain)
					require.Regexp(t, dnsLabelRegex, label)
				}
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
			false, false, uid.New(uid.DeploymentPrefix), uid.New(uid.AppPrefix),
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
		false, false, uid.New(uid.DeploymentPrefix), uid.New(uid.AppPrefix),
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
		false, false, uid.New(uid.DeploymentPrefix), uid.New(uid.AppPrefix),
	)

	require.Equal(t, "homestead-home-tracking-git-michaelunkey-p-9141f508-stagejune26.canary.unkey.app",
		domainBySticky(t, domains, db.FrontlineRoutesStickyBranch))
}

func TestBuildDomainsFallsBackToAppCodeOnIdenticalLabels(t *testing.T) {
	domainsFor := func(workspaceSlug, branch, appID string) newDomain {
		for _, d := range buildDomains(
			workspaceSlug, "demo-1", "testing-leave", "preview",
			"", branch, "", "canary.unkey.app",
			false, false, uid.New(uid.DeploymentPrefix), appID,
		) {
			if d.sticky == db.FrontlineRoutesStickyBranch {
				return d
			}
		}
		require.FailNow(t, "no branch domain")
		return newDomain{} //nolint:exhaustruct
	}

	testingApp, xTestingApp := uid.New(uid.AppPrefix), uid.New(uid.AppPrefix)
	testing := domainsFor("testing", "main-x", testingApp)
	xTesting := domainsFor("x-testing", "main", xTestingApp)

	require.Equal(t, "demo-1-testing-leave-git-main-x-testing.canary.unkey.app", testing.domain)
	require.Equal(t, testing.domain, xTesting.domain, "both apps build the same label")
	require.Equal(t, "demo-1-testing-leave-git-main-x-testing-"+appCode(xTestingApp)+".canary.unkey.app", xTesting.fallback)
	require.NotEqual(t, testing.fallback, xTesting.fallback, "the fallback must differ between apps")
	require.Equal(t, xTesting.fallback, domainsFor("x-testing", "main", xTestingApp).fallback, "the fallback must survive redeploys to stay sticky")
}

func TestCappedDomainEveryWorkspaceLength(t *testing.T) {
	for _, tc := range []struct{ code, codePart string }{{"", ""}, {"abcdef", "-abcdef"}} {
		code := tc.code
		for length := 3; length <= 64; length++ {
			workspaceSlug := strings.Repeat("w", length)
			label, _, _ := strings.Cut(cappedDomain(strings.Repeat("kebap", 12), workspaceSlug, code, "unkey.app"), ".")

			require.LessOrEqual(t, len(label), 63, label)
			require.Regexp(t, dnsLabelRegex, label)
			require.True(t, strings.HasSuffix(label, tc.codePart), label)
			require.Contains(t, label, workspaceSlug[:min(length, 52-len(tc.codePart))], "the workspace slug must survive the cut")
		}
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
