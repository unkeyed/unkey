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
