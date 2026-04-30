"use client";

import { Switch } from "@/components/ui/switch";
import { trpc } from "@/lib/trpc/client";
import { ChevronLeft } from "@unkey/icons";
import { Button, FormInput, FormSelect, toast } from "@unkey/ui";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useState } from "react";

type Provider = "axiom";
type Source = "runtime" | "request";
type Environment = "production" | "preview";

type Props = {
  scope: "project" | "workspace";
  workspaceSlug: string;
  projectId?: string;
};

// Brand mark for the provider card. Inlined as SVG so the Axiom mark
// keeps its brand identity without per-icon className overrides.
// Path is from the official mark at axiom.co/logo-white.svg.
function AxiomLogo({ className }: { className?: string }) {
  // Axiom's brand uses black on light / white on dark — not a colour
  // accent. Using currentColor lets the card render the mark in the same
  // ink as `text-gray-12`, which already adapts to theme.
  return (
    <svg className={className} viewBox="0 0 27 27" fill="currentColor" aria-hidden>
      <path d="M26.2993 18.0486 20.64 8.53881C20.3805 8.10174 19.7411 7.74414 19.2189 7.74414H15.6857C14.8645 7.74414 14.5279 7.18235 14.9375 6.49573L16.875 3.24842C17.0288 2.9907 17.0285 2.67343 16.8742 2.41599C16.7199 2.15855 16.435 2 16.1268 2H11.1979C10.6758 2 10.0349 2.3568 9.77368 2.7929L0.19608 18.7809C-0.0652427 19.217 -0.065343 19.9307 0.195678 20.3669L2.66008 24.4849C3.07072 25.171 3.7441 25.1718 4.15646 24.4867L6.08205 21.2874C6.49451 20.6023 7.16778 20.6031 7.57843 21.2892L9.32419 24.2064C9.58522 24.6426 10.226 24.9995 10.7481 24.9995H22.1376C22.6597 24.9995 23.3004 24.6426 23.5615 24.2064L26.2965 19.6363C26.5575 19.2001 26.5588 18.4857 26.2993 18.0486ZM18.6564 17.5909C19.0645 18.2784 18.7265 18.8409 17.9052 18.8409H9.04601C8.22481 18.8409 7.88883 18.2796 8.29948 17.5935L12.7325 10.186C13.1431 9.49988 13.8149 9.49991 14.2256 10.186L18.6564 17.5909Z" />
    </svg>
  );
}

const PROVIDERS: Array<{
  id: Provider;
  name: string;
  description: string;
  Logo: React.ComponentType<{ className?: string }>;
}> = [
  {
    id: "axiom",
    name: "Axiom",
    description: "Long-term storage. Fast full-text search.",
    Logo: AxiomLogo,
  },
];

const SEVERITY_OPTIONS = [
  { value: "all", label: "All severities" },
  { value: "debug", label: "Debug and above" },
  { value: "info", label: "Info and above" },
  { value: "warn", label: "Warn and above" },
  { value: "error", label: "Error only" },
];

export function LogDrainWizard({ scope, workspaceSlug, projectId }: Props) {
  const router = useRouter();

  const baseHref =
    scope === "project"
      ? `/${workspaceSlug}/projects/${projectId}/log-drains`
      : `/${workspaceSlug}/settings/log-drains`;

  const [provider, setProvider] = useState<Provider>("axiom");
  const [name, setName] = useState("");
  const [credential, setCredential] = useState("");

  const [axiomDataset, setAxiomDataset] = useState("");
  const [axiomEndpoint, setAxiomEndpoint] = useState("");

  const [sources, setSources] = useState<Source[]>(["runtime", "request"]);
  const [environments, setEnvironments] = useState<Environment[]>(["production"]);
  const [includeBodies, setIncludeBodies] = useState(false);
  const [minSeverity, setMinSeverity] = useState<"" | "debug" | "info" | "warn" | "error">("");

  const create = trpc.deploy.logDrain.create.useMutation({
    onSuccess: () => {
      toast.success("Log drain created");
      router.push(baseHref);
    },
    onError: (err) => toast.error(err.message),
  });

  const testPush = trpc.deploy.logDrain.testPush.useMutation();

  function buildPayload() {
    return {
      provider: "axiom" as const,
      config: { dataset: axiomDataset, endpoint: axiomEndpoint || undefined },
    };
  }

  function buildCredential() {
    return { credentialSource: "paste" as const, credential };
  }

  async function handleSubmit(e: React.FormEvent) {
    e.preventDefault();
    if (!name.trim()) {
      toast.error("Name is required");
      return;
    }
    const payload = buildPayload();
    const cred = buildCredential();

    if (cred.credentialSource === "paste") {
      const test = await testPush.mutateAsync({
        ...payload,
        credential: cred.credential,
      });
      if (!test.ok) {
        toast.error(`Test push failed: ${test.error}`);
        return;
      }
    }

    create.mutate({
      name,
      projectId: scope === "project" ? (projectId ?? null) : null,
      sources,
      environments,
      apps: [],
      filters: {
        runtime: minSeverity ? { minSeverity } : undefined,
        request: includeBodies ? { includeBodies: true } : undefined,
      },
      ...payload,
      ...cred,
    });
  }

  const submitting = create.isLoading || testPush.isLoading;
  const submitLabel = testPush.isLoading
    ? "Testing connection…"
    : create.isLoading
      ? "Creating drain…"
      : "Create log drain";

  return (
    <div className="mx-auto max-w-3xl px-4 pt-6 pb-32 flex flex-col gap-5">
      {/* Breadcrumb / page header */}
      <div className="flex flex-col gap-2">
        <Link
          href={baseHref}
          className="flex items-center gap-1.5 text-xs text-gray-11 hover:text-gray-12 transition-colors w-fit"
        >
          <ChevronLeft className="size-3.5" />
          Back to log drains
        </Link>
        <h1 className="font-semibold text-gray-12 text-xl">New log drain</h1>
      </div>

      <form onSubmit={handleSubmit} className="flex flex-col gap-4">
        {/* Step 1 — Provider */}
        <Card step={1} title="Choose a provider">
          <div className="grid grid-cols-1 md:grid-cols-3 gap-2.5">
            {PROVIDERS.map((p) => (
              <ProviderCard
                key={p.id}
                provider={p}
                selected={provider === p.id}
                onSelect={() => {
                  setProvider(p.id);
                }}
              />
            ))}
          </div>
        </Card>

        {/* Step 2 — Connection */}
        <Card
          step={2}
          title="Connection details"
          subtitle={`Configure the ${PROVIDERS.find((p) => p.id === provider)?.name} connection. We'll run a real test push before saving.`}
        >
          <div className="flex flex-col gap-4">
            <FormInput
              label="Drain name"
              description="Shown in the dashboard. e.g. payments-prod-axiom"
              placeholder="payments-prod-axiom"
              value={name}
              onChange={(e) => setName(e.target.value)}
              required
            />

            {provider === "axiom" && (
              <div className="grid grid-cols-1 md:grid-cols-2 gap-3">
                <FormInput
                  label="Dataset"
                  required
                  placeholder="unkey-prod"
                  value={axiomDataset}
                  onChange={(e) => setAxiomDataset(e.target.value)}
                />
                <FormInput
                  label="Endpoint"
                  description="Use api.eu.axiom.co for EU."
                  placeholder="https://api.axiom.co"
                  value={axiomEndpoint}
                  onChange={(e) => setAxiomEndpoint(e.target.value)}
                />
                <div className="md:col-span-2">
                  <FormInput
                    label="API token"
                    description="Encrypted in Vault. Never re-displayed; rotate by editing the drain."
                    type="password"
                    required
                    placeholder="xaat-..."
                    value={credential}
                    onChange={(e) => setCredential(e.target.value)}
                  />
                </div>
              </div>
            )}
          </div>
        </Card>

        {/* Step 3 — What & where */}
        <Card
          step={3}
          title="What to forward"
          subtitle="Sources and environments. Toggle off to reduce provider costs."
        >
          <div className="flex flex-col gap-4">
            <ToggleGroup label="Sources">
              <ToggleRow
                label="Runtime logs"
                description="stdout/stderr from your deployments"
                checked={sources.includes("runtime")}
                onCheckedChange={(c) =>
                  setSources((s) => (c ? [...s, "runtime"] : s.filter((x) => x !== "runtime")))
                }
              />
              <ToggleRow
                label="Request logs"
                description="HTTP access logs from Sentinel"
                checked={sources.includes("request")}
                onCheckedChange={(c) =>
                  setSources((s) => (c ? [...s, "request"] : s.filter((x) => x !== "request")))
                }
              />
            </ToggleGroup>

            <ToggleGroup label="Environments">
              <ToggleRow
                label="Production"
                description="Production deployments"
                checked={environments.includes("production")}
                onCheckedChange={(c) =>
                  setEnvironments((e) =>
                    c ? [...e, "production"] : e.filter((x) => x !== "production"),
                  )
                }
              />
              <ToggleRow
                label="Preview"
                description="PR previews — disabled by default to avoid surprise spend"
                checked={environments.includes("preview")}
                onCheckedChange={(c) =>
                  setEnvironments((e) => (c ? [...e, "preview"] : e.filter((x) => x !== "preview")))
                }
              />
            </ToggleGroup>
          </div>
        </Card>

        {/* Step 4 — Filters (optional) */}
        <Card
          step={4}
          title="Filters"
          subtitle="Optional. Narrow what gets forwarded to reduce provider costs."
        >
          <div className="flex flex-col gap-4">
            <div className="md:max-w-xs">
              <FormSelect
                label="Minimum runtime severity"
                description="Drop runtime logs below this level."
                options={SEVERITY_OPTIONS}
                value={minSeverity === "" ? "all" : minSeverity}
                onValueChange={(v) =>
                  setMinSeverity(v === "all" ? "" : (v as "debug" | "info" | "warn" | "error"))
                }
              />
            </div>
            <ToggleGroup>
              <ToggleRow
                label="Include request and response bodies"
                description="Bodies may contain user data and secrets. Confirm provider data-handling policy first."
                checked={includeBodies}
                onCheckedChange={setIncludeBodies}
                warn
              />
            </ToggleGroup>
          </div>
        </Card>
      </form>

      {/* Sticky footer action bar — keeps Submit always reachable on long forms. */}
      <div className="fixed inset-x-0 bottom-0 z-10 border-t border-grayA-4 bg-gray-1/85 backdrop-blur supports-[backdrop-filter]:bg-gray-1/70">
        <div className="mx-auto max-w-3xl px-4 py-3 flex items-center justify-end gap-2">
          <Link href={baseHref}>
            <Button type="button" variant="outline" size="md">
              Cancel
            </Button>
          </Link>
          <Button
            type="button"
            variant="primary"
            size="md"
            loading={submitting}
            onClick={handleSubmit}
          >
            {submitLabel}
          </Button>
        </div>
      </div>
    </div>
  );
}

// Card is the standard rounded-[14px] container used across the dashboard
// (project settings, sentinel-policies, env-vars). Numbered step badge gives
// the wizard a visual rhythm without forcing a multi-page flow.
function Card({
  step,
  title,
  subtitle,
  children,
}: {
  step: number;
  title: string;
  subtitle?: string;
  children: React.ReactNode;
}) {
  return (
    <section className="border border-grayA-4 rounded-[14px] bg-gray-1 overflow-hidden">
      <header className="flex items-start gap-3 px-5 pt-4 pb-3 border-b border-grayA-3">
        <span className="flex items-center justify-center size-6 rounded-full bg-grayA-3 text-[11px] font-semibold text-gray-12 shrink-0 mt-0.5">
          {step}
        </span>
        <div className="flex flex-col gap-0.5 min-w-0">
          <h2 className="text-sm font-semibold text-gray-12">{title}</h2>
          {subtitle && <p className="text-xs text-gray-11 leading-relaxed">{subtitle}</p>}
        </div>
      </header>
      <div className="p-5">{children}</div>
    </section>
  );
}

function ProviderCard({
  provider,
  selected,
  onSelect,
}: {
  provider: (typeof PROVIDERS)[number];
  selected: boolean;
  onSelect: () => void;
}) {
  const { Logo } = provider;
  return (
    <button
      type="button"
      onClick={onSelect}
      aria-pressed={selected}
      className={`group relative flex flex-col items-start gap-3 p-4 rounded-xl text-left transition-all duration-150 min-h-[124px] ${
        selected
          ? "bg-grayA-2 ring-2 ring-accent-9"
          : "bg-gray-1 ring-1 ring-grayA-4 hover:ring-grayA-7 hover:bg-grayA-2"
      }`}
    >
      {/* Logo chip uses the theme-adaptive gray-1 (white in light mode,
          near-black in dark mode) so the Axiom mark — which renders in
          currentColor / text-gray-12 — always has contrast. */}
      <div className="flex size-9 items-center justify-center rounded-lg bg-gray-1 ring-1 ring-grayA-4 text-gray-12">
        <Logo className="size-5" />
      </div>
      <div className="flex flex-col gap-0.5 min-w-0">
        <span className="font-semibold text-sm text-gray-12">{provider.name}</span>
        <p className="text-xs leading-snug text-gray-11">{provider.description}</p>
      </div>
      {selected && (
        <span className="absolute right-3 top-3 size-2 rounded-full bg-accent-9" aria-hidden />
      )}
    </button>
  );
}

function ToggleGroup({
  label,
  children,
}: {
  label?: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex flex-col gap-1.5">
      {label && (
        <span className="text-[11px] font-medium text-gray-11 uppercase tracking-wide">
          {label}
        </span>
      )}
      <div className="border border-grayA-4 rounded-lg overflow-hidden divide-y divide-grayA-3 bg-gray-1">
        {children}
      </div>
    </div>
  );
}

function ToggleRow({
  label,
  description,
  checked,
  onCheckedChange,
  warn,
}: {
  label: string;
  description: string;
  checked: boolean;
  onCheckedChange: (c: boolean) => void;
  warn?: boolean;
}) {
  // Earlier revisions wrapped this in a button / clickable div to make the
  // whole row a hit target, but Radix Switch is itself a button — nesting
  // produced an event-propagation loop ("Maximum update depth exceeded"
  // in switch.tsx). Reverting to a static row keeps semantics clean; the
  // Switch alone is the click target.
  return (
    <div className="flex items-start justify-between gap-4 px-4 py-3">
      <div className="flex flex-col gap-0.5 min-w-0">
        <span
          className={`text-sm font-medium ${warn && checked ? "text-warning-11" : "text-gray-12"}`}
        >
          {label}
        </span>
        <span className="text-xs text-gray-11 leading-snug">{description}</span>
      </div>
      <Switch
        checked={checked}
        onCheckedChange={onCheckedChange}
        className="h-5 w-9 data-[state=checked]:bg-accent-9 data-[state=checked]:ring-2 data-[state=checked]:ring-accent-5 data-[state=unchecked]:bg-grayA-5 data-[state=unchecked]:ring-2 data-[state=unchecked]:ring-grayA-4 mt-0.5 shrink-0"
        thumbClassName="h-4 w-4 data-[state=checked]:translate-x-4"
      />
    </div>
  );
}
