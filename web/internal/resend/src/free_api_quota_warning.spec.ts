import { readFileSync } from "node:fs";
import path from "node:path";
import { render } from "@react-email/render";
import { expect, test } from "vitest";
import FreeApiQuotaWarning from "../emails/free_api_quota_warning";

test("free API quota warning uses the approved copy", async () => {
  const text = await render(
    FreeApiQuotaWarning({
      workspaceName: "Acme",
      used: "150,000",
      limit: "150,000",
      billingUrl: "https://app.unkey.com/acme/settings/billing",
      year: "2026",
    }),
    { plainText: true },
  );

  expect(text).toContain("Hey,");
  expect(text).toContain(
    "Acme has used 150,000 of its 150,000 free API requests this month. That counts successful key verifications and rate limit checks.",
  );
  expect(text).toContain("Nothing is being rate limited yet. Requests are still going through.");
  expect(text).toContain(
    "If this workspace stays on the free plan, we'll start rate limiting it in 3 business days. Upgrading to a paid API plan raises the limit and avoids that.",
  );
  expect(text).toContain("Upgrade plan");
  expect(text).toContain("https://app.unkey.com/acme/settings/billing");
  expect(text).toContain("Need help? Email support@unkey.com or reply to this email.");
  expect(text).toContain("You are receiving this because you are an admin of this workspace.");
  expect(text).toContain("© 2026 Unkey");
  expect(text).not.toContain("—");

  const sync = readFileSync(path.join(__dirname, "../scripts/sync-templates.tsx"), "utf8");
  expect(sync).toContain('alias: "free-api-quota-warning"');
  expect(sync).toContain('subject: "{{{WORKSPACE_NAME}}} is over its free API limit"');
  expect(sync).toContain('from: "James | Unkey <james@updates.unkey.com>"');
});
