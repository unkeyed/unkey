import { match } from "@unkey/match";
import { KeyAuthFields } from "./forms/keyauth-fields";
import type { PolicyType } from "./schema";

// Today the policy union has only `keyauth`, so the dispatch is trivial. When
// a new policy type is added to `policyFormSchema`, extend the match below —
// `.exhaustive()` will fail loudly until every branch has a form.
export function PolicyConfigFields({ type }: { type: PolicyType }) {
  return match(type)
    .with("keyauth", () => <KeyAuthFields />)
    .exhaustive();
}
