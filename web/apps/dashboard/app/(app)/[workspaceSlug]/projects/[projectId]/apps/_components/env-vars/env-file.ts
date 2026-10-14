export type EnvEntry = { key: string; value: string };

type ParseEnvResult = {
  entries: EnvEntry[];
};

export const parseEnvText = (text: string): ParseEnvResult => {
  const lines = text.replace(/\r\n/g, "\n").trim().split("\n");
  const entries: EnvEntry[] = [];

  for (let i = 0; i < lines.length; i++) {
    const trimmed = lines[i].trim();
    if (!trimmed || trimmed.startsWith("#")) {
      continue;
    }

    const eqIndex = trimmed.indexOf("=");
    if (eqIndex === -1) {
      continue;
    }

    const key = trimmed.slice(0, eqIndex).trim();
    let value = trimmed.slice(eqIndex + 1).trim();

    const quote = value[0];
    if ((quote === '"' || quote === "'") && value.indexOf(quote, 1) === -1) {
      const collected = [value.slice(1)];
      i++;
      while (i < lines.length) {
        const closeIndex = lines[i].indexOf(quote);
        if (closeIndex !== -1) {
          collected.push(lines[i].slice(0, closeIndex));
          break;
        }
        collected.push(lines[i]);
        i++;
      }
      entries.push({ key, value: collected.join("\n") });
      continue;
    }

    if (
      value.length >= 2 &&
      ((value.startsWith('"') && value.endsWith('"')) ||
        (value.startsWith("'") && value.endsWith("'")))
    ) {
      value = value.slice(1, -1);
    }

    entries.push({ key, value });
  }

  return { entries };
};

export function pastedEntries(text: string): EnvEntry[] {
  return text.includes("=") ? parseEnvText(text).entries : [];
}
