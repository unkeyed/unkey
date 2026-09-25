import { createFileRoute, redirect } from "@tanstack/react-router";
import { KeysPage } from "~/components/keys-page/keys-page";
import { keysSearchSchema } from "~/hooks/use-keys-search";
import { canReadKeys } from "~/lib/scopes";

export const Route = createFileRoute("/_portal/keys")({
  validateSearch: keysSearchSchema,
  beforeLoad: ({ context }) => {
    if (!canReadKeys(context.session.scopes)) {
      throw redirect({ to: "/" });
    }
  },
  component: KeysPage,
});
