import { createRootKey } from "@/lib/root-keys-api";
import type { RootKeyFormValues } from "../schema";
import { buildUrns } from "./urn";

export async function createRootKeyFromForm(workspaceId: string, values: RootKeyFormValues) {
  const created = await createRootKey({
    name: values.name.trim(),
    permissions: buildUrns(workspaceId, values.policies),
  });

  return { keyId: created.keyId, secret: created.key };
}
