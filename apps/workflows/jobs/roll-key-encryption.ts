import { client } from "@/trigger";

import { connectDatabase, eq, schema } from "@/lib/db";
import { connectVault } from "@/lib/vault";
import { cronTrigger } from "@trigger.dev/sdk";

export const createRollEncryptionJob = client.defineJob({
  id: "vault.roll.encryption",
  name: "Reencrypt all of the keys of a workspace",
  version: "0.0.2",
  trigger: cronTrigger({
    // sundays at midnight
    cron: "0 0 * * 0",
  }),

  run: async (_payload, io, _ctx) => {
    const db = connectDatabase();
    const vault = connectVault();

    const encryptedKeys = await io.runTask("Get all encrypted keys", () =>
      db.query.encryptedKeys.findMany(),
    );

    io.logger.info(`Found ${encryptedKeys.length} keys`);
    const workspaceIds = Array.from(new Set(encryptedKeys.map((k) => k.workspaceId)));
    for (const workspaceId of workspaceIds) {
      await io.runTask(`create new dek for ${workspaceId}`, async () => {
        await vault.createDEK({ keyring: workspaceId });
      });
    }
    for (const key of encryptedKeys) {
      await io.runTask(`reencrypt ${key.keyId}`, async () => {
        const res = await vault.reEncrypt({
          encrypted: key.encrypted,
          keyring: key.workspaceId,
        });

        await io.runTask(`saving reencrypted key ${key.keyId} to db`, async () => {
          await db
            .update(schema.encryptedKeys)
            .set({
              encrypted: res.encrypted,
              encryptionKeyId: res.keyId,
            })
            .where(eq(schema.encryptedKeys.keyId, key.keyId));
        });
      });
    }

    return;
  },
});
