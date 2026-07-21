import { readdir } from "node:fs/promises";
import path from "node:path";
import { render } from "@react-email/render";
import React from "react";
import { expect, test } from "vitest";
import { templates } from "./template-manifest";

const pathToDirectoryWithEmails = path.resolve(__dirname, "../emails");

test("That all the emails render without errors", async () => {
  // Maybe later down the line this will need to have some filtering as well
  // to avoid files that are not email templates
  const emailFilenames = await readdir(pathToDirectoryWithEmails, {
    recursive: true,
  });

  for await (const emailFilename of emailFilenames) {
    const pathToEmail = path.join(pathToDirectoryWithEmails, emailFilename);
    const emailModule = (await import(pathToEmail)) as unknown;

    if (
      typeof emailModule === "object" &&
      emailModule !== null &&
      "default" in emailModule &&
      typeof emailModule.default === "function"
    ) {
      await render(
        React.createElement<Record<string, unknown>>(
          emailModule.default as React.FC,
          "PreviewProps" in emailModule
            ? (emailModule.PreviewProps as Record<string, unknown>)
            : {},
        ),
      );
    }
  }
});

test("That the quota follow-up explains enforced rate limiting", async () => {
  const followUp = templates.find(({ alias }) => alias === "api-usage-ratelimit-follow-up");
  if (!followUp) {
    throw new Error("api-usage-ratelimit-follow-up is missing from the template manifest");
  }

  expect(followUp.subject).toBe("We are rate limiting your API");
  expect(followUp.variables.map(({ key }) => key)).toEqual([
    "WORKSPACE_NAME",
    "USED",
    "LIMIT",
    "BILLING_URL",
    "YEAR",
  ]);

  const html = await render(followUp.element);

  expect(html).toContain("We are rate limiting your API");
  expect(html).toContain("will continue to be reduced to");
  expect(html).toContain("one request per hour");
  expect(html).toContain("Upgrade your workspace");
  expect(html).not.toContain("Configure rate limits");
});
