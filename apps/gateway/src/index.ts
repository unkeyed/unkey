import { Response } from "@cloudflare/workers-types";
import { AesGCM, getDecryptionKeyFromEnv } from "@unkey/encryption";
import { createConnection } from "./db";
import { type Env, zEnv } from "./env";

export default {
  async fetch(request: Request, rawEnv: Env, _ctx: ExecutionContext): Promise<Response> {
    const env = zEnv.parse(rawEnv);

    const db = createConnection({
      host: env.DATABASE_HOST,
      username: env.DATABASE_USERNAME,
      password: env.DATABASE_PASSWORD,
    });

    const url = new URL(request.url);
    const subdomain = url.hostname.replace(`.${env.APEX_DOMAIN}`, "");

    const gateway = await db.query.gateways.findFirst({
      where: (table, { eq }) => eq(table.name, subdomain),
      with: {
        headerRewrites: {
          with: {
            secret: true,
          },
        },
      },
    });

    if (!gateway) {
      throw new Error("no gateway");
    }

    const headers = new Headers(request.headers);
    headers.delete("authorization");
    headers.set("Unkey-Gateway", "1");
    console.log("rewrite", gateway.headerRewrites);
    for (const rewrite of gateway.headerRewrites) {
      const decryptionKey = getDecryptionKeyFromEnv(env, rewrite.secret.encryptionKeyVersion);
      if (decryptionKey.err) {
        return new Response(
          `unable to load encryption key version ${rewrite.secret.encryptionKeyVersion}`,
        );
      }

      const aes = await AesGCM.withBase64Key(decryptionKey.val);
      const value = await aes.decrypt({
        iv: rewrite.secret.iv,
        ciphertext: rewrite.secret.ciphertext,
      });
      console.log({ value });
      headers.set(rewrite.name, value);
    }

    const res = fetch(new URL(url.pathname, `https://${gateway.origin}`), {
      ...request,
      headers,
    });
    // @ts-expect-error
    return res;
  },
};
