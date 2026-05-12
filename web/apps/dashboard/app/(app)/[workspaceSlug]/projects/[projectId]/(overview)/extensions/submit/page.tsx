"use client";

import { useInstallations } from "@/lib/extensions/installations";
import { ALL_CATEGORIES, CATEGORY_LABELS, type ExtensionCategory } from "@/lib/extensions/registry";
import { CircleInfo } from "@unkey/icons";
import { Button, FormInput, InfoTooltip, Textarea, toast } from "@unkey/ui";
import { cn } from "@unkey/ui/src/lib/utils";
import { useParams } from "next/navigation";
import { useState } from "react";
import { ProjectContentWrapper } from "../../../components/project-content-wrapper";
import { ExtensionsHeader } from "../components/extensions-header";

const CONFIG_SCHEMA_PLACEHOLDER = `[
  {
    "id": "apiKey",
    "label": "API key",
    "type": "secret",
    "required": true
  }
]`;

export default function SubmitExtensionPage() {
  const params = useParams<{ workspaceSlug: string; projectId: string }>();
  const basePath = `/${params.workspaceSlug}/projects/${params.projectId}/extensions`;
  const { installations } = useInstallations(params.projectId);

  const [name, setName] = useState("");
  const [slug, setSlug] = useState("");
  const [tagline, setTagline] = useState("");
  const [categories, setCategories] = useState<ExtensionCategory[]>([]);
  const [configSchema, setConfigSchema] = useState("");
  const [oauthProvider, setOauthProvider] = useState("");
  const [oauthScopes, setOauthScopes] = useState("");
  const [repoUrl, setRepoUrl] = useState("");
  const [contactEmail, setContactEmail] = useState("");

  const handleSubmit = () => {
    toast.message("Submissions are not yet open", {
      description: "Reach out to support@unkey.dev to register interest.",
    });
  };

  return (
    <ProjectContentWrapper centered maxWidth="1200px" className="mt-8">
      <ExtensionsHeader basePath={basePath} active="submit" installedCount={installations.length} />

      {/* Match the marketplace tab's outer width so flipping tabs doesn't reflow
          the header strip, but keep the form column narrow for readability. */}
      <div className="flex flex-col gap-6 w-full max-w-[860px]">
        <Section
          title="Basics"
          description="Public-facing identity for your extension in the marketplace."
        >
          <FormInput
            label="Name *"
            placeholder="Acme Logs"
            value={name}
            onChange={(e) => setName(e.target.value)}
          />
          <FormInput
            label="Slug *"
            placeholder="acme-logs"
            description="Lowercase, hyphen-separated. Used in the URL."
            value={slug}
            onChange={(e) => setSlug(e.target.value)}
          />
          <FormInput
            label="Tagline *"
            placeholder="Stream every key event to Acme."
            value={tagline}
            onChange={(e) => setTagline(e.target.value)}
          />
          <LogoField />
          <CategoryPicker selected={categories} onChange={setCategories} />
        </Section>

        <Section
          title="Configuration schema"
          description="JSON describing the fields users will fill in when installing."
        >
          {/* Use Textarea directly: FormTextarea forwards className onto the
              wrapping fieldset, not the textarea — passing min-h there inflates
              the fieldset and leaves the textarea floating at the top. */}
          <Textarea
            value={configSchema}
            onChange={(e) => setConfigSchema(e.target.value)}
            placeholder={CONFIG_SCHEMA_PLACEHOLDER}
            rows={10}
            className="font-mono text-[12px] resize-y"
          />
        </Section>

        <Section
          title="OAuth (optional)"
          description="Fill this in if your extension authorizes via OAuth."
        >
          <FormInput
            label="Provider"
            placeholder="Slack"
            value={oauthProvider}
            onChange={(e) => setOauthProvider(e.target.value)}
          />
          <FormInput
            label="Scopes"
            placeholder="chat:write, channels:read"
            description="Comma-separated."
            value={oauthScopes}
            onChange={(e) => setOauthScopes(e.target.value)}
          />
        </Section>

        <Section
          title="Implementation"
          description="Where your extension's provider code lives, and how to reach you during review."
        >
          <FormInput
            label="Source repository"
            placeholder="https://github.com/your-org/unkey-extension"
            description="Public link to the provider code (onInstall, onUpdate, verify, …)."
            value={repoUrl}
            onChange={(e) => setRepoUrl(e.target.value)}
          />
          <FormInput
            label="Contact email *"
            placeholder="you@example.com"
            description="We'll reach out here during review."
            value={contactEmail}
            onChange={(e) => setContactEmail(e.target.value)}
          />
          <div className="flex items-start gap-2 rounded-md bg-grayA-2 border border-grayA-3 p-3 text-[12px] text-grayA-11">
            <CircleInfo className="size-4 shrink-0 mt-0.5 text-grayA-9" />
            <div className="flex flex-col gap-1">
              <span className="font-medium text-grayA-12">How review works</span>
              <span>
                Submissions ship as a manifest only — verification logic is provider code we land
                in the dashboard repo together. After you submit, we'll work with you to write the
                <code className="px-1 font-mono text-grayA-12">verify()</code> hook (token round-trip,
                webhook reachability, etc.) before the extension is published as Verified.
              </span>
            </div>
          </div>
        </Section>

        <div className="flex items-center justify-end">
          <Button onClick={handleSubmit}>Submit for review</Button>
        </div>
      </div>
    </ProjectContentWrapper>
  );
}

function Section({
  title,
  description,
  children,
}: {
  title: string;
  description: string;
  children: React.ReactNode;
}) {
  return (
    <section className="flex flex-col gap-3 rounded-lg border border-grayA-3 p-5">
      <div className="flex flex-col gap-0.5">
        <h2 className="text-[14px] font-semibold text-grayA-12">{title}</h2>
        <p className="text-[12px] text-grayA-10">{description}</p>
      </div>
      <div className="flex flex-col gap-3">{children}</div>
    </section>
  );
}

function LogoField() {
  return (
    <div className="flex flex-col gap-1.5">
      <label
        htmlFor="extension-logo"
        className="text-[12px] font-medium text-grayA-12 leading-none"
      >
        Logo
      </label>
      <InfoTooltip content="Coming soon">
        <div
          id="extension-logo"
          className="flex h-20 items-center justify-center rounded-md border border-dashed border-grayA-4 bg-grayA-2 text-[12px] text-grayA-10 cursor-not-allowed"
        >
          Drag and drop a 512×512 PNG (coming soon)
        </div>
      </InfoTooltip>
    </div>
  );
}

function CategoryPicker({
  selected,
  onChange,
}: {
  selected: ExtensionCategory[];
  onChange: (next: ExtensionCategory[]) => void;
}) {
  const toggle = (category: ExtensionCategory) => {
    onChange(
      selected.includes(category)
        ? selected.filter((c) => c !== category)
        : [...selected, category],
    );
  };
  return (
    <div className="flex flex-col gap-1.5">
      <span className="text-[12px] font-medium text-grayA-12">Categories *</span>
      <div className="flex flex-wrap gap-1.5">
        {ALL_CATEGORIES.map((category) => {
          const active = selected.includes(category);
          return (
            <button
              key={category}
              type="button"
              onClick={() => toggle(category)}
              className={cn(
                "rounded-md border px-2.5 py-1 text-[12px] transition-colors",
                active
                  ? "border-grayA-12 bg-grayA-12 text-gray-1"
                  : "border-grayA-3 bg-background text-grayA-11 hover:border-grayA-6",
              )}
            >
              {CATEGORY_LABELS[category]}
            </button>
          );
        })}
      </div>
    </div>
  );
}


