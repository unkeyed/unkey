import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import {
  customDomainErrorKind,
  useAppScope,
  useCustomDomainMutations,
  useCustomDomains,
} from "@/lib/deploy-data";
import { getErrorMessage } from "@/lib/unkey-client";
import { zodResolver } from "@hookform/resolvers/zod";
import { match } from "@unkey/match";
import { useForm } from "react-hook-form";
import { type CustomDomainFormValues, customDomainSchema } from "./schema";

export type AddDomainOutcome =
  | { kind: "added"; domain: string }
  | { kind: "limitReached"; message: string };

type Args = {
  defaultEnvironmentId: string;
  onSettled: (outcome: AddDomainOutcome) => void;
  onCancel: () => void;
};

export function useAddDomain({ defaultEnvironmentId, onSettled, onCancel }: Args) {
  const { projectId, appId } = useAppScope();
  const { data: customDomains } = useCustomDomains();
  const { addCustomDomain } = useCustomDomainMutations();
  const workspace = useWorkspaceNavigation();
  const form = useForm<CustomDomainFormValues>({
    resolver: zodResolver(customDomainSchema),
    mode: "onSubmit",
    reValidateMode: "onSubmit",
    defaultValues: { environmentId: defaultEnvironmentId, domain: "" },
  });

  const resetForm = () => form.reset({ environmentId: defaultEnvironmentId, domain: "" });

  const settle = (outcome: AddDomainOutcome) => {
    resetForm();
    onSettled(outcome);
  };

  const submit = form.handleSubmit(async ({ domain, environmentId }) => {
    if (customDomains.some((d) => d.domain === domain)) {
      form.setError("domain", { message: "This domain is already added" });
      return;
    }
    try {
      await addCustomDomain({ domain, projectId, appId, environmentId }, workspace.slug);
      settle({ kind: "added", domain });
    } catch (err) {
      match(customDomainErrorKind(err))
        .with("limit", () => settle({ kind: "limitReached", message: getErrorMessage(err) }))
        .with("invalid", () => form.setError("domain", { message: getErrorMessage(err) }))
        .with("conflict", () => console.error("Failed to add custom domain", err))
        .with("unknown", () => console.error("Failed to add custom domain", err))
        .exhaustive();
    }
  });

  const cancel = () => {
    resetForm();
    onCancel();
  };

  return { form, submit, cancel };
}
