import { CircleHalfDottedClock, Gear } from "@unkey/icons";
import { SettingCardGroup } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { SettingsGroup } from "./shared/settings-group";

type Row = { title: string; description: string; controlW: string };

const BUILD_ROWS: Row[] = [
  {
    title: "Repository",
    description: "Source repository for this deployment",
    controlW: "w-44",
  },
  {
    title: "Root directory",
    description:
      "The directory your app lives in. Unkey builds from here. Set it when your app is in a subdirectory (e.g., services/api).",
    controlW: "w-10",
  },
  {
    title: "Dockerfile",
    description:
      "Dockerfile location used for docker build. Leave empty and Unkey builds your app automatically without a Dockerfile.",
    controlW: "w-44",
  },
  {
    title: "Build command",
    description:
      "Override the auto-detected build command. Useful for monorepos, e.g. pnpm build --filter api. Only applies when no Dockerfile is set.",
    controlW: "w-32",
  },
  {
    title: "Watch paths",
    description:
      "Only trigger deployments when files matching these glob patterns change. Leave empty to deploy on all changes.",
    controlW: "w-28",
  },
  {
    title: "Auto deploy",
    description: "Automatically trigger deployments when code is pushed to GitHub.",
    controlW: "w-48",
  },
];

const RUNTIME_ROWS: Row[] = [
  {
    title: "Regions",
    description: "Geographic regions where your app will run",
    controlW: "w-48",
  },
  {
    title: "Instances",
    description:
      "Autoscaling range per region. Scales up to the maximum based on CPU usage, down to the minimum when load is low.",
    controlW: "w-44",
  },
  {
    title: "Max CPU",
    description: "Maximum CPU limit per instance. You are only charged for actual usage.",
    controlW: "w-48",
  },
  {
    title: "Memory",
    description: "Memory allocation for each instance",
    controlW: "w-44",
  },
  {
    title: "Storage",
    description: "Ephemeral disk space per instance",
    controlW: "w-44",
  },
  {
    title: "Healthcheck",
    description: "Endpoint used to verify the service is healthy",
    controlW: "w-40",
  },
  {
    title: "Port",
    description: "Port your application listens on",
    controlW: "w-14",
  },
  {
    title: "Command",
    description: "The command to start your application. Changes apply on next deploy.",
    controlW: "w-24",
  },
];

const ADVANCED_ROWS: Row[] = [
  {
    title: "Custom Domains",
    description: "Serve your deployment from your own domain name",
    controlW: "w-16",
  },
  {
    title: "OpenAPI Spec Path",
    description: "Path to your OpenAPI spec. Leave empty to disable scraping.",
    controlW: "w-16",
  },
  {
    title: "Upstream Protocol",
    description:
      "Protocol used to connect to your application. If you don't know what this is, use HTTP/1.1. Learn more.",
    controlW: "w-28",
  },
];

export function SettingsSkeleton() {
  return (
    <div className="flex flex-col gap-6" aria-busy="true">
      <output className="sr-only">Loading settings...</output>
      <SettingCardGroup>
        <CardRows rows={BUILD_ROWS} />
      </SettingCardGroup>
      <SettingsGroup
        icon={<CircleHalfDottedClock iconSize="md-medium" />}
        title="Runtime settings"
        hideChevron
      >
        <SettingCardGroup>
          <CardRows rows={RUNTIME_ROWS} />
        </SettingCardGroup>
      </SettingsGroup>
      <SettingsGroup
        icon={<Gear iconSize="md-medium" />}
        title="Advanced configurations"
        hideChevron
      >
        <SettingCardGroup>
          <CardRows rows={ADVANCED_ROWS} />
        </SettingCardGroup>
      </SettingsGroup>
    </div>
  );
}

/** Mirrors the markup of SettingCard, down to the classes that set its height. */
function CardRows({ rows }: { rows: Row[] }) {
  return (
    <>
      {rows.map((row) => (
        <div key={row.title} className="w-full">
          <div className="lg:w-full flex gap-6 lg:justify-between lg:items-center flex-col lg:flex-row px-4 py-[18px]">
            <div className="flex gap-4 items-center">
              <div
                aria-hidden="true"
                className="bg-gray-3 size-8 rounded-[10px] shrink-0 animate-pulse dark:ring-1 dark:ring-gray-4 dark:shadow-none shadow-sm shadow-grayA-8/20"
              />
              <div className="flex flex-col gap-1 text-sm w-fit">
                <div className="font-medium text-gray-12 text-[13px] leading-4 tracking-normal">
                  {row.title}
                </div>
                <div className="font-normal text-gray-11 text-xs leading-4 tracking-normal max-w-[600px]">
                  {row.description}
                </div>
              </div>
            </div>
            <div
              aria-hidden="true"
              className="flex items-center gap-4 w-full lg:w-[320px] justify-end"
            >
              <div className={cn("h-7 rounded-md bg-grayA-3 animate-pulse", row.controlW)} />
              <div className="size-3 rounded bg-grayA-3 animate-pulse shrink-0" />
            </div>
          </div>
        </div>
      ))}
    </>
  );
}
