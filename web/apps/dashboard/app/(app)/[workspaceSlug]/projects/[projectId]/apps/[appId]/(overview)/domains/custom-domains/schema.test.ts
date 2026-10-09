import { describe, expect, it } from "vitest";
import { customDomainSchema } from "./schema";

function domainErrors(domain: string): string[] {
  const result = customDomainSchema.safeParse({ environmentId: "env_1", domain });
  return result.success ? [] : result.error.issues.map((issue) => issue.message);
}

const FORMAT_ERROR = "Enter a domain like api.example.com";

describe("customDomainSchema", () => {
  it("leaves the remaining hostname rules to the server", () => {
    for (const input of ["münchen.de", "api_v2.acme.com", "acme.co.uk"]) {
      expect(domainErrors(input), input).toEqual([]);
    }
  });

  it("rejects input that is not a hostname", () => {
    for (const input of [
      "localhost",
      "not a domain",
      "example.c",
      ".acme.com",
      "acme.com.",
      "a..com",
      "https://api.acme.com",
      "api.acme.com/v1",
      "api.acme.com:8080",
      "*.acme.com",
    ]) {
      expect(domainErrors(input), input).toEqual([FORMAT_ERROR]);
    }
  });
});
