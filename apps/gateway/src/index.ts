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

    const proxy = await db.query.proxies.findFirst({
      where: (table, { eq }) => eq(table.name, subdomain),
      with: {
        headerRewrites: {
          with: {
            secret: true,
          },
        },
      },
    });

    if (!proxy) {
      throw new Error("no proxy");
    }

    const headers = new Headers(request.headers);
    headers.delete("authorization");
    headers.set("Unkey-Proxy", "1");
    console.log("rewrite", proxy.headerRewrites);
    for (const rewrite of proxy.headerRewrites) {
      const decryptionKey = getDecryptionKeyFromEnv(env, rewrite.secret.encryptionKeyVersion);

      const aes = await AesGCM.withBase64Key(decryptionKey);
      const value = await aes.decrypt({
        iv: rewrite.secret.iv,
        ciphertext: rewrite.secret.ciphertext,
      });
      console.log({ value });
      headers.set(rewrite.name, value);
    }

    return fetch(new URL(url.pathname, `https://${proxy.origin}`), {
      ...request,
      headers,
    });
  },
};
