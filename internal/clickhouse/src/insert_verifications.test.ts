import { expect, test } from "vitest";
import { ClickHouse } from "./index";

import { ClickHouseContainer } from "./testutil";

test(
  "inserts a single row",
  {
    timeout: 300_000,
  },
  async (t) => {
    const container = await ClickHouseContainer.start(t);

    const ch = new ClickHouse({ url: container.url() });

    const verification = {
      request_id: "1",
      time: Date.now(),
      workspace_id: "workspace_id",
      key_space_id: "key_space_id",
      key_id: "key_id",
      outcome: "VALID" as const,
      region: "test",
      tags: ["tag"],
    };

    const { err: insertErr } = await ch.verifications.insert(verification);
    expect(insertErr).toBeUndefined();

    const latestVerifications = await ch.verifications.latest({
      workspaceId: verification.workspace_id,
      keySpaceId: verification.key_space_id,
      keyId: verification.key_id,
      limit: 1,
    });

    expect(latestVerifications.err).toBeUndefined();
    expect(latestVerifications.val!.length).toBe(1);
    expect(latestVerifications.val![0].time).toBe(verification.time);
    expect(latestVerifications.val![0].outcome).toBe("VALID");
    expect(latestVerifications.val![0].region).toBe(verification.region);
  }
);
