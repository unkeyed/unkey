import { IconBook2Outline18, IconGaugeOutline18 } from "@unkey/icons";
import {
  Button,
  CopyButton,
  EmptyState,
  EmptyStateActions,
  EmptyStateDescription,
  EmptyStateHeader,
  EmptyStateIcon,
  EmptyStateTitle,
} from "@unkey/ui";

const EXAMPLE_SNIPPET = `curl -XPOST 'https://api.unkey.dev/v2/ratelimit.limit' \\
  -H 'Content-Type: application/json' \\
  -H 'Authorization: Bearer <UNKEY_ROOT_KEY>' \\
  -d '{
      "namespace": "demo_namespace",
      "identifier": "user_123",
      "limit": 10,
      "duration": 10000
  }'`;

export function NamespacesEmpty() {
  return (
    <EmptyState>
      <EmptyStateIcon>
        <IconGaugeOutline18 />
      </EmptyStateIcon>
      <EmptyStateHeader>
        <EmptyStateTitle>No Namespaces found</EmptyStateTitle>
        <EmptyStateDescription>
          You haven't created any Namespaces yet. Create one by performing a limit request as shown
          below.
        </EmptyStateDescription>
      </EmptyStateHeader>
      <div className="mt-4 w-full max-w-lg">
        <div className="flex items-start gap-4 rounded-lg border bg-background p-4">
          <pre className="flex-1 text-xs text-left overflow-x-auto">
            <code>{EXAMPLE_SNIPPET}</code>
          </pre>
          <CopyButton value={EXAMPLE_SNIPPET} />
        </div>
      </div>
      <EmptyStateActions>
        <a
          href="https://www.unkey.com/docs/platform/ratelimiting/introduction"
          target="_blank"
          rel="noopener noreferrer"
        >
          <Button variant="outline" className="flex items-center gap-2">
            <IconBook2Outline18 className="w-4 h-4" />
            Read the docs
          </Button>
        </a>
      </EmptyStateActions>
    </EmptyState>
  );
}
