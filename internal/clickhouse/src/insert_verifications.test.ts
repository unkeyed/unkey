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
      outcome: "VALID",
      region: "test",
    } as const;

    await ch.verifications.insert(verification);

    const latestVerifications = await ch.verifications.logs({
      workspaceId: verification.workspace_id,
      keySpaceId: verification.key_space_id,
      keyId: verification.key_id,
    });

    expect(latestVerifications.length).toBe(1);
    expect(latestVerifications[0].time).toBe(verification.time);
    expect(latestVerifications[0].outcome).toBe("VALID");
    expect(latestVerifications[0].region).toBe(verification.region);
  },
);
