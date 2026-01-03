export function marshalEnv(o: Record<string, Record<string, string>>): string {
  return Object.entries(o)
    .map(([comment, kvs]) => {
      const lines = [`# ${comment}`];
      for (const [k, v] of Object.entries(kvs)) {
        lines.push(`${k}="${v}"`);
      }
      return lines.join("\n");
    })
    .join("\n\n");
}
