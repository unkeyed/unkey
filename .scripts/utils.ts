import fs from "fs";

export const parseEnv = (path: string) => {
  const env = {};
  const file = fs.readFileSync(path, "utf8");
  file.split("\n").forEach((line) => {
    const trimmedLine = line.trim();
    if (!trimmedLine || trimmedLine.startsWith("#")) return;

    const indexOfFirstEquals = trimmedLine.indexOf("=");
    if (indexOfFirstEquals === -1) return;

    const key = trimmedLine.substring(0, indexOfFirstEquals).trim();
    let value = trimmedLine.substring(indexOfFirstEquals + 1).trim();

    if (
      (value.startsWith("'") && value.endsWith("'")) ||
      (value.startsWith('"') && value.endsWith('"'))
    ) {
      value = value.substring(1, value.length - 1);
    }

    if (key) {
      env[key] = value;
    }
  });

  return env;
};
