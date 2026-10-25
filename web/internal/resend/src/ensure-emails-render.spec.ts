import { readdirSync } from "node:fs";
import path from "node:path";
import { render } from "@react-email/render";
import React from "react";
import { expect, test } from "vitest";
import { ComputeBudgetAlert } from "../emails/compute_budget_alert";
import { SecretScanningKeyDetected } from "../emails/secret_scanning_key_detected";

const pathToDirectoryWithEmails = path.resolve(__dirname, "../emails");
const emailFilenames = readdirSync(pathToDirectoryWithEmails, { recursive: true }).filter(
  (filename): filename is string => typeof filename === "string" && filename.endsWith(".tsx"),
);

test.each(emailFilenames)(
  "%s has readable HTML and plain text with one primary action",
  async (filename) => {
    const emailModule: unknown = await import(path.join(pathToDirectoryWithEmails, filename));
    if (
      typeof emailModule !== "object" ||
      emailModule === null ||
      !("default" in emailModule) ||
      typeof emailModule.default !== "function" ||
      !("PreviewProps" in emailModule.default) ||
      typeof emailModule.default.PreviewProps !== "object" ||
      emailModule.default.PreviewProps === null
    ) {
      throw new Error(`${filename} must export an email with PreviewProps`);
    }

    const element: unknown = emailModule.default(emailModule.default.PreviewProps);
    if (!React.isValidElement(element)) {
      throw new Error(`${filename} must return an email element`);
    }
    const html = await render(element);
    const text = await render(element, { plainText: true });

    expect(html.match(/<h1\b/g)).toHaveLength(1);
    expect(html).toMatch(/<title>[^<]+<\/title>/);
    expect(html).toContain('lang="en"');
    expect(html).toContain('dir="ltr"');
    expect(html).toContain('name="color-scheme" content="light dark"');
    expect(html).toContain("prefers-color-scheme: dark");
    expect(html).toContain("[data-ogsc]");
    expect(html.match(/class="email-button"/g)).toHaveLength(1);
    expect(html).not.toMatch(/href="(?:undefined|null|)"/);
    expect(html).not.toContain("Connect with us on social media");
    expect(text).toContain("support@unkey.com");
    expect(text).not.toMatch(/\b(?:undefined|null)\b/);
    expect(text).not.toMatch(/[\u200b-\u200f\ufeff]/);

    for (const url of [
      "https://x.com/unkeydev",
      "https://unkey.com/discord",
      "https://unkey.com/github",
    ]) {
      expect(html.split(`href="${url}"`).length - 1).toBe(filename === "welcome_email.tsx" ? 1 : 0);
      expect(text.includes(url)).toBe(filename === "welcome_email.tsx");
    }

    if (filename !== "secret_scanning_key_detected.tsx") {
      const emphasized = Array.from(
        html.matchAll(/<strong\b[^>]*>(.*?)<\/strong>/g),
        (match) => match[1],
      );
      for (const [field, value] of Object.entries(emailModule.default.PreviewProps)) {
        if (["usage", "budget", "used", "limit", "throttleDate", "date"].includes(field)) {
          expect(emphasized).toContain(value);
        }
        if (field === "requestsPerMinute") {
          expect(emphasized).toContain(`${value} per minute`);
        }
      }
    }

    for (const value of Object.values(emailModule.default.PreviewProps).flat()) {
      if (typeof value === "string" && value.length > 0) {
        expect(text).toContain(value);
      }
    }
  },
);

test("Resend variables survive rendering in content and links", async () => {
  const elements = [
    React.createElement(SecretScanningKeyDetected, {
      date: "{{{DATE}}}",
      source: "{{{SOURCE}}}",
      url: "{{{URL}}}",
    }),
    React.createElement(ComputeBudgetAlert, {
      workspaceName: "{{{WORKSPACE_NAME}}}",
      usage: "{{{USAGE}}}",
      budget: "{{{BUDGET}}}",
      percent: "{{{PERCENT}}}",
      billingUrl: "{{{BILLING_URL}}}",
      year: "{{{YEAR}}}",
    }),
  ];

  for (const element of elements) {
    const html = await render(element);
    const text = await render(element, { plainText: true });
    for (const value of Object.values(element.props)) {
      expect(html).toContain(value);
      expect(text).toContain(value);
    }
  }
});

test("plain and emphasized values are escaped without losing the billing destination", async () => {
  const element = React.createElement(ComputeBudgetAlert, {
    ...ComputeBudgetAlert.PreviewProps,
    workspaceName: '<script>alert("workspace")</script> & team',
    usage: "<b>$81.40</b>",
    billingUrl: "https://app.unkey.com/acme/settings/billing?from=email&tab=budget",
  });
  const html = await render(element);
  const text = await render(element, { plainText: true });

  expect(html).not.toContain("<script>");
  expect(html).not.toContain("<b>");
  expect(html).toContain("&lt;b&gt;$81.40&lt;/b&gt;</strong>");
  expect(text).toContain('<script>alert("workspace")</script> & team');
  expect(text).toContain("<b>$81.40</b>");
  expect(text).toContain("https://app.unkey.com/acme/settings/billing?from=email&tab=budget");
});

test("security details have readable labels in plain text", async () => {
  const text = await render(
    React.createElement(SecretScanningKeyDetected, SecretScanningKeyDetected.PreviewProps),
    { plainText: true },
  );

  expect(text).toMatch(/Source\s+content\s+Detected\s+Thu Oct 22 2026/);
});
