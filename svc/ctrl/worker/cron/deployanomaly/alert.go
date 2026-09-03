package deployanomaly

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	restate "github.com/restatedev/sdk-go"
	hydrav1 "github.com/unkeyed/unkey/gen/proto/hydra/v1"
	"github.com/unkeyed/unkey/pkg/email"
	"github.com/unkeyed/unkey/pkg/logger"
)

// These published Resend template aliases must exist before notifications are
// enabled in production. The templates own their subject and sender.
const (
	anomalyAlertTemplate    = "deploy-anomaly-alert"
	anomalyResolvedTemplate = "deploy-anomaly-resolved"
)

func (h *CheckHandler) sendAlert(ctx restate.ObjectContext, req *hydrav1.EvaluateDeployAnomalyRequest, alertID string, metric Metric, result Result) error {
	return h.sendToAdmins(ctx, req, alertID, anomalyAlertTemplate, "anomaly-alert/"+alertID, h.metricVariables(req, alertID, metric, result))
}

func (h *CheckHandler) sendResolved(ctx restate.ObjectContext, req *hydrav1.EvaluateDeployAnomalyRequest, alertID string, metric Metric, snapshot Result) error {
	return h.sendToAdmins(ctx, req, alertID, anomalyResolvedTemplate, "anomaly-resolved/"+alertID, h.metricVariables(req, alertID, metric, snapshot))
}

func (h *CheckHandler) sendToAdmins(
	ctx restate.ObjectContext,
	req *hydrav1.EvaluateDeployAnomalyRequest,
	alertID, templateID, idempotencyKey string,
	variables map[string]string,
) error {
	recipients, err := restate.Run(ctx, func(rc restate.RunContext) ([]string, error) {
		return h.admins.AdminEmails(rc, req.GetOrgId())
	}, restate.WithName("resolve anomaly alert admins"))
	if err != nil {
		return fmt.Errorf("resolve anomaly alert admins: %w", err)
	}
	if len(recipients) == 0 {
		logger.Warn("deploy anomaly alert has no recipients", "alert_id", alertID, "org_id", req.GetOrgId(), "template", templateID)
		return nil
	}
	return restate.RunVoid(ctx, func(rc restate.RunContext) error {
		return h.email.Send(rc, email.Email{
			To: recipients, TemplateID: templateID, Variables: variables,
			From: "", Subject: "", IdempotencyKey: idempotencyKey,
		})
	}, restate.WithName("send deploy anomaly email"))
}

func (h *CheckHandler) metricVariables(req *hydrav1.EvaluateDeployAnomalyRequest, alertID string, metric Metric, result Result) map[string]string {
	variables := map[string]string{
		"METRIC":       metricLabel(metric),
		"APP_NAME":     req.GetAppName(),
		"ENVIRONMENT":  req.GetEnvironmentSlug(),
		"OBSERVED":     formatObserved(metric, result),
		"BASELINE":     formatValue(metric, result.BaselineMean),
		"WINDOW_START": time.UnixMilli(req.GetWindowStart()).UTC().Format(time.RFC3339),
		"ALERT_URL":    fmt.Sprintf("%s/%s/alerts/%s", strings.TrimRight(h.dashboardBaseURL, "/"), req.GetWorkspaceSlug(), alertID),
		"YEAR":         strconv.Itoa(time.UnixMilli(req.GetWindowEnd()).UTC().Year()),
	}
	if isDirectLimit(metric) {
		variables["LIMIT"] = formatValue(metric, result.ThresholdValue)
	} else {
		variables["SIGMA"] = strconv.FormatFloat(result.SigmaK, 'f', -1, 64)
	}
	return variables
}

func metricLabel(metric Metric) string {
	switch metric {
	case MetricError5xx:
		return "5xx error rate"
	case MetricError4xx:
		return "4xx error rate"
	case MetricRequests:
		return "Requests"
	case MetricRequestsDrop:
		return "Request traffic drop"
	case MetricEgressBytes:
		return "Network egress"
	case MetricCPUSeconds:
		return "CPU usage"
	case MetricMemoryUtilization:
		return "Memory utilization"
	case MetricOOMKilled:
		return "Out of memory kills"
	case MetricCrashLoop:
		return "Crash loops"
	default:
		return string(metric)
	}
}

func formatObserved(metric Metric, result Result) string {
	if metric == MetricError5xx || metric == MetricError4xx {
		return fmt.Sprintf("%.2f%% (%s/%s requests)", result.Observed*100, formatNumber(result.RawCount), formatNumber(result.Requests))
	}
	return formatValue(metric, result.Observed)
}

func formatValue(metric Metric, value float64) string {
	switch metric {
	case MetricError5xx, MetricError4xx, MetricMemoryUtilization:
		return fmt.Sprintf("%.2f%%", value*100)
	case MetricEgressBytes:
		return fmt.Sprintf("%.2f MiB", value/(1<<20))
	case MetricCPUSeconds:
		return fmt.Sprintf("%.2f s", value)
	case MetricRequests, MetricRequestsDrop, MetricOOMKilled, MetricCrashLoop:
		return formatNumber(value)
	default:
		return formatNumber(value)
	}
}

func formatNumber(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func isDirectLimit(metric Metric) bool {
	return metric == MetricRequestsDrop || metric == MetricMemoryUtilization || metric == MetricOOMKilled || metric == MetricCrashLoop
}
