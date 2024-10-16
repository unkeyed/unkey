import { execa } from "execa";
import type { NextApiRequest, NextApiResponse } from "next";

export default async function (_req: NextApiRequest, res: NextApiResponse) {
  await execa("bun", ["cli.deploy.ts"], {
    cwd: "../gateway",
    env: { NODE_ENV: "development", FAKE_GIT_BRANCH_NAME: "main" },
  });

  res.send("ok");
  res.end();
}
