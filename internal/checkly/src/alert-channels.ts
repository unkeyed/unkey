import { SlackAlertChannel, WebhookAlertChannel } from "checkly/constructs";

// configured in the dashboard
// https://app.checklyhq.com/alerts/settings/channels/edit/incidentio/218874
export const incidentIo = WebhookAlertChannel.fromId("218874");
export const slack = SlackAlertChannel.fromId("240275");
