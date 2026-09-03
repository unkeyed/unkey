package deployusage

import (
	"fmt"
	"math"
	"time"
)

const gib = 1024 * 1024 * 1024

type appProfile struct {
	name            string
	slug            string
	vcpu            float64
	memoryGiB       float64
	diskGiB         float64
	egressGiBPerDay float64
}

var appProfiles = []appProfile{
	{name: "API Gateway", slug: "usage-api-gateway", vcpu: 3.2, memoryGiB: 6, diskGiB: 10, egressGiBPerDay: 14},
	{name: "Link Redirector", slug: "usage-link-redirector", vcpu: 0.7, memoryGiB: 1.5, diskGiB: 2, egressGiBPerDay: 0.2},
	{name: "Webhook Delivery", slug: "usage-webhook-delivery", vcpu: 2.4, memoryGiB: 4, diskGiB: 8, egressGiBPerDay: 11},
	{name: "Dashboard", slug: "usage-dashboard", vcpu: 1.8, memoryGiB: 3, diskGiB: 4, egressGiBPerDay: 6},
	{name: "Auth Service", slug: "usage-auth-service", vcpu: 1.5, memoryGiB: 2.5, diskGiB: 3, egressGiBPerDay: 4.5},
	{name: "Billing Worker", slug: "usage-billing-worker", vcpu: 1.1, memoryGiB: 2, diskGiB: 6, egressGiBPerDay: 1.2},
	{name: "Data Ingest", slug: "usage-data-ingest", vcpu: 4.2, memoryGiB: 8, diskGiB: 20, egressGiBPerDay: 9},
	{name: "Event Processor", slug: "usage-event-processor", vcpu: 3.6, memoryGiB: 7, diskGiB: 16, egressGiBPerDay: 7.5},
	{name: "Image Proxy", slug: "usage-image-proxy", vcpu: 1.9, memoryGiB: 3.5, diskGiB: 12, egressGiBPerDay: 13},
	{name: "Status Page", slug: "usage-status-page", vcpu: 0.3, memoryGiB: 0.5, diskGiB: 1, egressGiBPerDay: 0.8},
	{name: "Queue Worker", slug: "usage-queue-worker", vcpu: 2.8, memoryGiB: 5, diskGiB: 8, egressGiBPerDay: 2.5},
	{name: "Analytics API", slug: "usage-analytics-api", vcpu: 3.9, memoryGiB: 9, diskGiB: 24, egressGiBPerDay: 10},
	{name: "Realtime Sync", slug: "usage-realtime-sync", vcpu: 2.1, memoryGiB: 4.5, diskGiB: 5, egressGiBPerDay: 8},
	{name: "Admin Portal", slug: "usage-admin-portal", vcpu: 0.8, memoryGiB: 1.5, diskGiB: 2, egressGiBPerDay: 2},
	{name: "Documentation", slug: "usage-documentation", vcpu: 0.4, memoryGiB: 0.8, diskGiB: 1, egressGiBPerDay: 3},
}

type usageRow struct {
	time                time.Time
	workspaceID         string
	projectID           string
	appID               string
	environmentID       string
	resourceID          string
	containerUID        string
	instanceID          string
	cpuSeconds          float64
	memoryGiBHours      float64
	diskGiBHours        float64
	egressPublicBytes   int64
	egressPrivateBytes  int64
	ingressPublicBytes  int64
	ingressPrivateBytes int64
	samplePairs         int64
}

type environmentTarget struct {
	id     string
	slug   string
	weight float64
}

type deploymentSeed struct {
	id            string
	k8sName       string
	workspaceID   string
	projectID     string
	appID         string
	environmentID string
	commitSHA     string
	message       string
	createdAt     int64
}

func generateDeployments(now time.Time, targets []target) []deploymentSeed {
	start := time.Date(now.Year(), now.Month()-2, 1, 0, 0, 0, 0, time.UTC)
	monthStarts := []time.Time{start, start.AddDate(0, 1, 0), start.AddDate(0, 2, 0)}
	days := []int{2, 8, 14, 20, 27}
	deployments := make([]deploymentSeed, 0, len(targets)*len(monthStarts)*len(days))

	for appIndex, target := range targets {
		for _, monthStart := range monthStarts {
			for eventIndex, baseDay := range days {
				createdAt := time.Date(
					monthStart.Year(),
					monthStart.Month(),
					baseDay+appIndex%3,
					9+(appIndex*3+eventIndex)%12,
					(appIndex*7+eventIndex*11)%60,
					0,
					0,
					time.UTC,
				)
				if !createdAt.Before(now) || createdAt.Month() != monthStart.Month() {
					continue
				}

				id := fmt.Sprintf("d_seedusage%02d%x", appIndex, createdAt.Unix())
				environmentID := target.productionID
				if eventIndex%3 == 0 {
					environmentID = target.previewID
				}
				deployments = append(deployments, deploymentSeed{
					id:            id,
					k8sName:       fmt.Sprintf("%s%02d-%x", deploymentK8sPrefix, appIndex, createdAt.Unix()),
					workspaceID:   target.workspaceID,
					projectID:     target.projectID,
					appID:         target.appID,
					environmentID: environmentID,
					commitSHA:     fmt.Sprintf("%040x", uint64(createdAt.Unix())*100+uint64(appIndex)),
					message:       "Deploy " + target.profile.name,
					createdAt:     createdAt.UnixMilli(),
				})
			}
		}
	}

	return deployments
}

func generateUsage(now time.Time, targets []target) []usageRow {
	end := now.UTC()
	start := time.Date(end.Year(), end.Month()-2, 1, 0, 0, 0, 0, time.UTC)
	hours := int(math.Ceil(end.Sub(start).Hours()))
	rows := make([]usageRow, 0, hours*len(targets)*2)

	for appIndex, target := range targets {
		environments := []environmentTarget{
			{id: target.productionID, slug: productionSlug, weight: 0.93},
			{id: target.previewID, slug: previewSlug, weight: 0.07},
		}
		for _, environment := range environments {
			resourceID := resourcePrefix + target.profile.slug + "/" + environment.slug
			for ts := start; ts.Before(end); ts = ts.Add(time.Hour) {
				bucketFraction := math.Min(1, end.Sub(ts).Hours())
				load := loadFactor(ts, appIndex)
				dailyEgress := dailyEgressGiB(ts, end, target.profile, appIndex)
				egressBytes := int64(math.Round(dailyEgress * gib * hourlyTrafficShare(ts.Hour(), appIndex) * environment.weight * bucketFraction))
				rows = append(rows, usageRow{
					time:                ts,
					workspaceID:         target.workspaceID,
					projectID:           target.projectID,
					appID:               target.appID,
					environmentID:       environment.id,
					resourceID:          resourceID,
					containerUID:        resourceID + "/container",
					instanceID:          resourceID + "/instance",
					cpuSeconds:          target.profile.vcpu * 3600 * load * environment.weight * bucketFraction,
					memoryGiBHours:      target.profile.memoryGiB * (0.82 + load*0.18) * environment.weight * bucketFraction,
					diskGiBHours:        target.profile.diskGiB * environment.weight * bucketFraction,
					egressPublicBytes:   egressBytes,
					egressPrivateBytes:  int64(math.Round(float64(egressBytes) * 0.12)),
					ingressPublicBytes:  int64(math.Round(float64(egressBytes) * 0.38)),
					ingressPrivateBytes: int64(math.Round(float64(egressBytes) * 0.09)),
					samplePairs:         int64(math.Round(240 * bucketFraction)),
				})
			}
		}
	}
	return rows
}

func loadFactor(ts time.Time, appIndex int) float64 {
	hourly := 0.72 + 0.28*math.Sin((float64(ts.Hour())-7+float64(appIndex%4))*math.Pi/12)
	weekday := 1.0
	if ts.Weekday() == time.Saturday || ts.Weekday() == time.Sunday {
		weekday = 0.78
	}
	jitter := 0.94 + 0.06*math.Sin(float64(ts.YearDay()*3+appIndex)*0.7)
	return math.Max(0.25, hourly*weekday*jitter)
}

func hourlyTrafficShare(hour, appIndex int) float64 {
	weight := func(hour int) float64 {
		return 0.7 + 0.3*math.Sin((float64(hour)-6+float64(appIndex%5))*math.Pi/12)
	}
	total := 0.0
	for currentHour := 0; currentHour < 24; currentHour++ {
		total += weight(currentHour)
	}
	return weight(hour) / total
}

func dailyEgressGiB(ts, now time.Time, profile appProfile, appIndex int) float64 {
	previousMonth := time.Date(now.Year(), now.Month()-1, 1, 0, 0, 0, 0, time.UTC)
	if profile.slug == "usage-link-redirector" &&
		ts.Year() == previousMonth.Year() &&
		ts.Month() == previousMonth.Month() {
		incident := []float64{8.23, 28.36, 32.10, 88.49, 58.04, 42.95, 102.78, 104.32, 176, 128, 220, 260, 330, 435, 311, 182, 78}
		if ts.Day() >= 15 && ts.Day() <= 31 {
			return incident[ts.Day()-15]
		}
	}

	weekend := 1.0
	if ts.Weekday() == time.Saturday || ts.Weekday() == time.Sunday {
		weekend = 0.74
	}
	jitter := 0.9 + 0.1*math.Sin(float64(ts.YearDay()+appIndex*11)*0.83)
	return profile.egressGiBPerDay * weekend * jitter
}
