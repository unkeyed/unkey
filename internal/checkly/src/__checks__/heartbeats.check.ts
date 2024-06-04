import { HeartbeatCheck } from "checkly/constructs";
import { incidentIo } from "../alert-channels";

new HeartbeatCheck("vault", {
  alertChannels: [incidentIo],
  name: "Vault",
  activated: true,
  period: 5,
  periodUnit: "minutes",
  grace: 1,
  graceUnit: "minutes",
});

new HeartbeatCheck("event-router", {
  alertChannels: [incidentIo],
  name: "EventRouter",
  activated: true,
  period: 5,
  periodUnit: "minutes",
  grace: 1,
  graceUnit: "minutes",
});
