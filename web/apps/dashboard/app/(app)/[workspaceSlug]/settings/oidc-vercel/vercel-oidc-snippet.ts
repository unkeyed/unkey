export function getVercelOIDCSnippet(audience: string): string {
  return `import { Unkey } from "@unkey/api";
import { getVercelOidcToken } from "@vercel/oidc";

export const GET = async () => {
  const token = await getVercelOidcToken({
    audience: ${JSON.stringify(audience)},
  });

  const unkey = new Unkey({ rootKey: token });
  const { data } = await unkey.apis.getApi({
    apiId: "api_...",
  });

  return Response.json(data);
};`;
}
