import assert from "node:assert/strict";
import fs from "node:fs";

export const palettes = {
  light: {
    panel: "#ffffff",
    raised: "#f4f5f5",
    border: "#d6dada",
    text: "#202525",
    muted: "#687070",
    accent: "#008577",
    tint: "#e7f5f1",
    warning: "#ac681b",
  },
  dark: {
    panel: "#191c1c",
    raised: "#222727",
    border: "#3d4544",
    text: "#eef2f1",
    muted: "#a0aeaa",
    accent: "#37cdb3",
    tint: "#193b33",
    warning: "#e8b36b",
  },
};

export function escape(value) {
  return String(value)
    .replaceAll("&", "&amp;")
    .replaceAll("<", "&lt;")
    .replaceAll(">", "&gt;")
    .replaceAll('"', "&quot;");
}

const marks = new Map();
for (const name of [
  "github",
  "docker",
  "unkey",
  "typescript",
  "go",
  "python",
  "hono",
  "bun",
  "nextjs",
  "fastapi",
  "express",
  "echo",
  "cloudflare",
]) {
  const source = fs.readFileSync(new URL(`./brands/${name}.svg`, import.meta.url), "utf8");
  const rootTag = source.slice(
    source.indexOf("<svg"),
    source.indexOf(">", source.indexOf("<svg")) + 1,
  );
  const dimensions = [
    rootTag.match(/\bwidth="(\d+)"/)?.[1],
    rootTag.match(/\bheight="(\d+)"/)?.[1],
  ];
  let viewBox =
    rootTag.match(/viewBox="([^"]+)"/)?.[1] ??
    (dimensions.every(Boolean) ? `0 0 ${dimensions.join(" ")}` : undefined);
  assert.ok(viewBox, `Missing brand viewBox: ${name}`);
  const start = source.indexOf("<svg");
  let content = source
    .slice(source.indexOf(">", start) + 1, source.lastIndexOf("</svg>"))
    .replace(/<metadata\b[\s\S]*?<\/metadata>/g, "")
    .replace(/<title\b[\s\S]*?<\/title>/g, "")
    .replace(/<sodipodi:namedview\b[\s\S]*?\/>/g, "");
  if (name === "cloudflare") {
    viewBox = "0 0 69 33";
    content = [...content.matchAll(/<path\b[^>]*\/>/g)]
      .slice(0, 2)
      .map((match) => match[0])
      .join("");
  }
  marks.set(name, { viewBox, content });
}

export function primitives(theme) {
  const p = palettes[theme];
  assert.ok(p, `Unknown theme: ${theme}`);
  let markId = 0;
  const rect = (x, y, w, h, fill = p.panel, r = 14, shadow = false) => {
    assert.ok([x, y, w, h, r].every(Number.isFinite) && w > 0 && h > 0, "Invalid SVG rectangle");
    return `<rect x="${x}" y="${y}" width="${w}" height="${h}" rx="${r}" fill="${fill === p.panel ? "url(#surface)" : fill}" stroke="${p.border}" stroke-width="1.5"${shadow ? ' filter="url(#lift)"' : ""}/>`;
  };
  const line = (d, color = p.border, width = 1.5, dash = "") =>
    `<path d="${d}" fill="none" stroke="${color}" stroke-width="${width}" stroke-linecap="round" stroke-linejoin="round"${dash ? ` stroke-dasharray="${dash}"` : ""}/>`;
  const circle = (x, y, r, fill = p.accent) =>
    `<circle cx="${x}" cy="${y}" r="${r}" fill="${fill}"/>`;
  const text = (x, y, value, color = p.text, size = 20, anchor = "start") => {
    assert.ok(size >= 18, `Illustration label too small: ${value}`);
    return `<text x="${x}" y="${y}" fill="${color}" font-size="${size}" text-anchor="${anchor}" font-weight="450">${escape(value)}</text>`;
  };
  const check = (x, y, size = 16) =>
    line(
      `M${x} ${y + size * 0.5}l${size * 0.35} ${size * 0.35} ${size * 0.65} -${size * 0.7}`,
      p.accent,
      2.5,
    );
  const arrow = (x, y, length = 30, color = p.accent) =>
    line(`M${x} ${y}h${length}m-7-6 7 6-7 6`, color, 2);
  const cube = (x, y, size = 60) => {
    const half = size / 2,
      quarter = size / 4;
    return `<g stroke="${p.border}" stroke-width="1.5" stroke-linejoin="round"><path d="M${x} ${y + quarter}l${half} -${quarter} ${half} ${quarter}-${half} ${quarter}Z" fill="url(#surface)"/><path d="M${x} ${y + quarter}v${half}l${half} ${quarter}V${y + half}Z" fill="${p.raised}"/><path d="M${x + half} ${y + half}l${half} -${quarter}v${half}l-${half} ${quarter}Z" fill="${p.tint}"/></g>`;
  };
  const chip = (
    x,
    y,
    label,
    { width = Math.max(82, label.length * 10 + 30), accent = false } = {},
  ) =>
    rect(x, y, width, 38, accent ? p.tint : p.panel, 19) +
    text(x + width / 2, y + 25, label, accent ? p.accent : p.muted, 18, "middle");
  const brand = (name, x, y, size = 48) => {
    const mark = marks.get(name);
    assert.ok(mark, `Unknown brand: ${name}`);
    const prefix = `brand-${name}-${markId++}-`;
    let content = mark.content
      .replace(/id="([^"]+)"/g, (_, id) => `id="${prefix}${id}"`)
      .replace(/url\(#([^)]+)\)/g, (_, id) => `url(#${prefix}${id})`);
    // Monochrome marks retain their official paths and use the approved inverse treatment.
    if (["github", "nextjs", "express"].includes(name))
      content = content
        .replace(/fill="(?:#000000|#000|black)"/gi, 'fill="currentColor"')
        .replace(/fill:black/g, "fill:currentColor");
    const inverse = ["cloudflare", "fastapi"].includes(name);
    return (
      (["typescript", "python"].includes(name)
        ? `<rect x="${x}" y="${y}" width="${size}" height="${size}" rx="4" fill="#ffffff"/>`
        : "") +
      (inverse
        ? `<rect x="${x - 7}" y="${y - 7}" width="${size + 14}" height="${size + 14}" rx="12" fill="#242828"/>`
        : "") +
      `<svg x="${x}" y="${y}" width="${size}" height="${size}" viewBox="${mark.viewBox}" preserveAspectRatio="xMidYMid meet" color="${p.text}" fill="${["github", "docker", "unkey", "nextjs"].includes(name) ? p.text : "#000000"}" xmlns:xlink="http://www.w3.org/1999/xlink">${content}</svg>`
    );
  };
  return { p, rect, line, circle, text, check, arrow, cube, chip, brand };
}

export function documentSvg(card, theme, content) {
  const p = palettes[theme];
  return `<svg xmlns="http://www.w3.org/2000/svg" width="600" height="320" viewBox="0 0 600 320" fill="none">\n<title>${escape(card.title)}</title>\n<desc>${escape(card.scene.heading + ". " + card.scene.note)}</desc>\n<defs><linearGradient id="surface" x1="0" y1="0" x2="0.7" y2="1"><stop stop-color="${p.panel}"/><stop offset="1" stop-color="${theme === "light" ? "#f9fafa" : "#151919"}"/></linearGradient><filter id="lift" x="-30%" y="-30%" width="160%" height="180%" color-interpolation-filters="sRGB"><feDropShadow dx="0" dy="7" stdDeviation="8" flood-color="#081815" flood-opacity="${theme === "light" ? "0.07" : "0.22"}"/><feDropShadow dx="0" dy="1" stdDeviation="1" flood-color="#081815" flood-opacity="0.06"/></filter></defs>\n<g font-family="Inter, -apple-system, BlinkMacSystemFont, Segoe UI, Arial, sans-serif">${content}</g>\n</svg>\n`;
}
