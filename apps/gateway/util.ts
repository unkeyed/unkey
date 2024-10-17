import { exec } from "node:child_process";
import { spinner } from "@clack/prompts";

export async function task<T>(
  name: string,
  fn: (s: ReturnType<typeof spinner>) => Promise<T>,
): Promise<T> {
  const s = spinner();
  s.start(name);

  try {
    const res = await fn(s);
    return res;
  } catch (err) {
    s.stop((err as Error).message);
    process.exit(1);
    // just to make ts happy
    return undefined as T;
  }
}

export async function run(cmd: string, opts?: { cwd: string }) {
  await new Promise((resolve, reject) => {
    const p = exec(cmd, opts);

    p.on("exit", (code) => {
      if (code === 0) {
        resolve(code);
      } else {
        reject(code);
      }
    });
  });
}
