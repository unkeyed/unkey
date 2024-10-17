import { execa } from "execa";
import type { NextApiRequest, NextApiResponse } from "next";

export default async function(_req: NextApiRequest, res: NextApiResponse) {
  const out = await execa("bun", ["cli.deploy.ts"], {
    cwd: "../gateway",
    env: { NODE_ENV: "development", FAKE_GIT_BRANCH_NAME: "deploy" },
  });
  console.log(out.stdout);

  res.send("ok");
  res.end();
}
