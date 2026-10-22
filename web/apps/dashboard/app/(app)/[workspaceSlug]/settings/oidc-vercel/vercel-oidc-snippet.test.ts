import { expect, it } from "vitest";
import { getVercelOIDCSnippet } from "./vercel-oidc-snippet";

it("creates a Vercel OIDC request example for the integration audience", () => {
  expect(
    getVercelOIDCSnippet("https://api.unkey.com/oidc/vercel/vco_test"),
  ).toBe(`import { Unkey } from "@unkey/api";
import { getVercelOidcToken } from "@vercel/oidc";

export const GET = async () => {
  const token = await getVercelOidcToken({
    audience: "https://api.unkey.com/oidc/vercel/vco_test",
  });

  const unkey = new Unkey({ rootKey: token });
  const { data } = await unkey.apis.getApi({
    apiId: "api_...",
  });

  return Response.json(data);
};`);
});
