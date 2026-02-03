"use client";
import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import {
  Button,
  Input,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  toast,
} from "@unkey/ui";
import { useEffect, useRef, useState } from "react";
import type { CustomDomain } from "./types";

// Basic domain validation regex
const DOMAIN_REGEX = /^(?!:\/\/)([a-zA-Z0-9-_]+\.)+[a-zA-Z]{2,}$/;

// Extract bare hostname from a pasted URL or raw input
function extractDomain(input: string): string {
  const trimmed = input.trim().toLowerCase();
  try {
    // Try parsing as a URL (works if it has a protocol)
    return new URL(trimmed.includes("://") ? trimmed : `https://${trimmed}`).hostname;
  } catch {
    return trimmed;
  }
}

type AddCustomDomainProps = {
  projectId: string;
  environments: Array<{ id: string; slug: string }>;
  getExistingDomain: (domain: string) => CustomDomain | undefined;
  onCancel: () => void;
  onSuccess: () => void;
};

export function AddCustomDomain({
  projectId,
  environments,
  getExistingDomain,
  onCancel,
  onSuccess,
}: AddCustomDomainProps) {
  const addMutation = trpc.deploy.customDomain.add.useMutation();
  const containerRef = useRef<HTMLDivElement>(null);
  const inputRef = useRef<HTMLInputElement>(null);

  const [domain, setDomain] = useState("");
  // Default to production environment, fall back to first environment
  const defaultEnvId =
    environments.find((e) => e.slug === "production")?.id ?? environments[0]?.id ?? "";
  const [environmentId, setEnvironmentId] = useState(defaultEnvId);

  useEffect(() => {
    containerRef.current?.scrollIntoView({
      behavior: "smooth",
      block: "nearest",
    });
    inputRef.current?.focus();
  }, []);

  const isSubmitting = addMutation.isLoading;

  const getError = (): string | undefined => {
    if (!domain) {
      return undefined;
    }

    if (!DOMAIN_REGEX.test(domain)) {
      return "Invalid domain format";
    }

    if (getExistingDomain(domain)) {
      return "Domain already registered";
    }

    return undefined;
  };

  const error = getError();
  const isValid = domain && !error && environmentId;

  const handleKeyDown = (e: React.KeyboardEvent) => {
    if (e.key === "Enter" && isValid && !isSubmitting) {
      e.preventDefault();
      handleSave();
    } else if (e.key === "Escape") {
      onCancel();
    }
  };

  const handleSave = async () => {
    if (!isValid || isSubmitting) {
      return;
    }

    const mutation = addMutation.mutateAsync({
      projectId,
      environmentId,
      domain,
    });

    toast.promise(mutation, {
      loading: "Adding domain...",
      success: (data) => ({
        message: "Domain added",
        description: `Add a CNAME record pointing to ${data.targetCname}`,
      }),
      error: (err) => ({
        message: "Failed to add domain",
        description: err.message,
      }),
    });

    try {
      await mutation;
      onSuccess();
    } catch {}
  };

  return (
    <div ref={containerRef} className="px-4 py-3 border-b border-gray-4">
      <div className="flex items-center gap-2">
        <Input
          ref={inputRef}
          type="text"
          placeholder="api.example.com"
          value={domain}
          onChange={(e) => setDomain(extractDomain(e.target.value))}
          onKeyDown={handleKeyDown}
          className={cn("h-8 text-xs flex-1 font-mono", error && "border-red-6 focus:border-red-7")}
          autoComplete="off"
          spellCheck={false}
        />
        <div className="w-[120px]">
          <Select value={environmentId} onValueChange={setEnvironmentId}>
            <SelectTrigger className="h-8 w-[120px]">
              <SelectValue placeholder="Environment" />
            </SelectTrigger>
            <SelectContent>
              {environments.map((env) => (
                <SelectItem key={env.id} value={env.id}>
                  {env.slug}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>
        </div>

        <div className="flex items-center gap-2">
          <Button
            variant="primary"
            onClick={handleSave}
            className="text-xs"
            disabled={!isValid || isSubmitting}
            loading={isSubmitting}
          >
            Add
          </Button>
          <Button variant="ghost" onClick={onCancel} disabled={isSubmitting} className="text-xs">
            Cancel
          </Button>
        </div>
      </div>
      {error && <p className="text-xs text-error-9 mt-1">{error}</p>}
    </div>
  );
}
