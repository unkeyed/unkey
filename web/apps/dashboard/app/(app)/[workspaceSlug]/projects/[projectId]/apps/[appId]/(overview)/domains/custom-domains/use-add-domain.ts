import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import {
  isCustomDomainLimitError,
  isInvalidDomainError,
} from "@/lib/collections/deploy/custom-domains";
import { getErrorMessage } from "@/lib/unkey-client";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { useAppId, useProjectData } from "../../data-provider";
import { type CustomDomainFormValues, customDomainSchema } from "./schema";

export type AddDomainOutcome =
  | { kind: "added"; domain: string }
  | { kind: "limitReached"; message: string }
  | { kind: "cancelled" };

type Args = {
  defaultEnvironmentId: string;
  onSettled: (outcome: AddDomainOutcome) => void;
};

export function useAddDomain({ defaultEnvironmentId, onSettled }: Args) {
  const { projectId, customDomains } = useProjectData();
  const appId = useAppId();
  const workspace = useWorkspaceNavigation();
  const form = useForm<CustomDomainFormValues>({
    resolver: zodResolver(customDomainSchema),
    mode: "onSubmit",
    reValidateMode: "onSubmit",
    defaultValues: { environmentId: defaultEnvironmentId, domain: "" },
  });

  const settle = (outcome: AddDomainOutcome) => {
    form.reset({ environmentId: defaultEnvironmentId, domain: "" });
    onSettled(outcome);
  };

  const submit = form.handleSubmit(async ({ domain, environmentId }) => {
    if (customDomains.some((d) => d.domain === domain)) {
      form.setError("domain", { message: "This domain is already added" });
      return;
    }
    const tx = collection.customDomains.insert(
      {
        id: crypto.randomUUID(),
        domain,
        projectId,
        appId,
        environmentId,
        verificationStatus: "pending",
        dnsRecords: [],
        verificationError: null,
        domainConnectProvider: null,
        domainConnectUrl: null,
        createdAt: Date.now(),
        updatedAt: null,
      },
      { metadata: { workspaceSlug: workspace.slug } },
    );
    try {
      await tx.isPersisted.promise;
      settle({ kind: "added", domain });
    } catch (err) {
      if (isCustomDomainLimitError(err)) {
        settle({ kind: "limitReached", message: getErrorMessage(err) });
        return;
      }
      if (isInvalidDomainError(err)) {
        form.setError("domain", { message: getErrorMessage(err) });
        return;
      }
      console.error("Failed to add custom domain", err);
    }
  });

  return { form, submit, cancel: () => settle({ kind: "cancelled" }) };
}
