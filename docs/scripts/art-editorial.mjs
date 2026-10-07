import assert from "node:assert/strict";

// Overview art follows the dashboard's neutral surfaces; teal is reserved for
// verification and usage states, rather than used as a decorative panel tint.
function studio(g) {
  const dark = g.p.panel !== "#ffffff";
  const p = {
    surface: dark ? "#171717" : "#ffffff",
    subtle: dark ? "#202020" : "#fafafa",
    selected: dark ? "#292929" : "#f0f0f0",
    border: dark ? "#353535" : "#e0e0e0",
    ink: dark ? "#ededed" : "#212121",
    muted: dark ? "#a3a3a3" : "#737373",
    accent: dark ? "#37cdb3" : "#008577",
    tint: dark ? "#14352f" : "#edf8f4",
  };
  const box = (x, y, w, h, fill = p.surface, radius = 8) =>
    `<rect x="${x}" y="${y}" width="${w}" height="${h}" rx="${radius}" fill="${fill}" stroke="${p.border}" stroke-width="1"/>`;
  const rule = (d, color = p.border, width = 1) => g.line(d, color, width);
  const label = (x, y, value, color = p.ink, size = 20, anchor = "start") =>
    g.text(x, y, value, color, size, anchor);
  const mono = (x, y, value, color = p.ink, size = 18, anchor = "start") =>
    `<g font-family="ui-monospace, SFMono-Regular, Menlo, Consolas, monospace">${label(x, y, value, color, size, anchor)}</g>`;
  const tick = (x, y) => rule(`M${x} ${y + 6}l4 4 8-9`, p.accent, 1.8);
  const key = (x, y) =>
    rule(`M${x + 12} ${y + 6}a6 6 0 1 1-12 0 6 6 0 1 1 12 0m0 0h17m-4 0v5m-6-5v4`, p.muted, 1.5);
  const folder = (x, y) => rule(`M${x} ${y + 3}h8l4 4h12v15h-24Z`, p.muted, 1.5);
  return { p, box, rule, label, mono, tick, key, folder };
}

export function renderEditorialWorkspace(g) {
  const { p, box, rule, label, mono, key } = studio(g);
  return (
    box(56, 48, 488, 224) +
    g.brand("unkey", 80, 68, 24) +
    label(118, 88, "Workspace", p.ink, 22) +
    rule("M56 110h488M300 110v108M56 218h488") +
    rule("M84 144l10-6 10 6v12l-10 6-10-6Zm0 0 10 6 10-6m-10 6v12", p.muted, 1.5) +
    label(118, 155, "Compute", p.ink, 21) +
    key(324, 146) +
    label(366, 155, "API Management", p.ink, 20) +
    g.circle(87, 190, 3, p.accent) +
    mono(101, 196, "api-service", p.muted) +
    g.circle(327, 190, 3, p.accent) +
    mono(341, 196, "sk_live_…", p.muted) +
    label(300, 251, "Members · Billing · Access", p.muted, 18, "middle")
  );
}

export function renderEditorialBuild(g) {
  const { p, box, rule, label, mono, folder } = studio(g);
  return (
    box(72, 48, 456, 224) +
    g.brand("github", 96, 65, 24) +
    label(134, 85, "Build context", p.ink, 20) +
    mono(500, 85, "main", p.muted, 18, "end") +
    rule("M72 108h456") +
    box(92, 126, 416, 42, p.selected, 4) +
    folder(108, 135) +
    mono(146, 153, "/services/api", p.ink, 21) +
    rule("M120 168v72m0-42h28m-28 42h28") +
    g.brand("docker", 156, 180, 26) +
    mono(196, 202, "Dockerfile", p.ink, 20) +
    rule("M158 225h14l7 7v16h-21Zm14 0v7h7", p.muted, 1.3) +
    mono(196, 246, "package.json", p.muted, 20)
  );
}

export function renderEditorialApi(card, g) {
  const { p, box, rule, label, mono, tick, key } = studio(g);
  const scenes = {
    "api-management--index--1": () =>
      box(64, 48, 472, 224) +
      g.brand("unkey", 88, 66, 24) +
      label(128, 86, "Your first API key", p.ink, 20) +
      rule("M64 108h472M64 190h472") +
      mono(88, 141, "01", p.muted) +
      mono(134, 141, "keys.createKey", p.ink, 21) +
      mono(134, 172, "sk_live_…", p.muted) +
      mono(88, 222, "02", p.muted) +
      mono(134, 222, "keys.verifyKey", p.ink, 21) +
      tick(136, 240) +
      mono(160, 252, "VALID", p.accent),

    "api-management--index--2": () =>
      box(64, 48, 264, 160) +
      label(88, 81, "Keyspace", p.ink, 21) +
      rule("M64 98h264M264 128h24v24h88M264 176h24v-24") +
      box(88, 112, 176, 32, p.subtle, 4) +
      key(100, 122) +
      mono(142, 134, "Key A", p.muted) +
      box(88, 160, 176, 32, p.subtle, 4) +
      key(100, 170) +
      mono(142, 182, "Key B", p.muted) +
      box(376, 116, 160, 72) +
      label(456, 145, "Identity", p.ink, 20, "middle") +
      mono(456, 171, "user_123", p.muted, 18, "middle") +
      key(88, 243) +
      label(130, 257, "Root key", p.ink, 20) +
      rule("M230 251h130m-6-5 6 5-6 5", p.muted, 1.3) +
      g.brand("unkey", 382, 239, 24) +
      label(420, 258, "Unkey API", p.ink, 20),

    "api-management--index--3": () =>
      box(64, 48, 472, 224) +
      label(88, 84, "Keyspaces", p.ink, 21) +
      rule("M64 108h472M258 108v164") +
      box(80, 128, 162, 40, p.selected, 4) +
      label(96, 154, "Payments", p.ink, 19) +
      label(96, 202, "Notifications", p.muted, 19) +
      mono(282, 154, "sk_pay_…7v2m", p.ink, 19) +
      rule("M282 174h230") +
      mono(282, 204, "sk_pay_…9k4x", p.ink, 19) +
      label(282, 246, "Shared defaults", p.muted, 18),

    "api-management--index--4": () =>
      box(64, 48, 472, 224) +
      label(88, 86, "Create key", p.ink, 21) +
      rule("M64 108h472") +
      label(88, 143, "Prefix", p.muted, 18) +
      mono(254, 143, "sk_live", p.ink, 20) +
      label(88, 182, "Permissions", p.muted, 18) +
      mono(254, 182, "orders.read", p.ink, 20) +
      rule("M88 205h424") +
      label(88, 245, "Secret returned", p.muted, 18) +
      box(378, 220, 134, 34, p.ink, 5) +
      label(445, 243, "Create key", p.surface, 18, "middle"),

    "api-management--index--5": () =>
      box(80, 48, 440, 224) +
      mono(104, 86, "keys.verifyKey", p.ink, 21) +
      rule("M80 108h440") +
      ["Enabled", "Expiration", "Permissions"]
        .map((value, i) => label(104, 140 + i * 34, value, p.muted, 19) + tick(480, 128 + i * 34))
        .join("") +
      box(96, 224, 408, 32, p.tint, 4) +
      mono(112, 246, 'code: "VALID"', p.accent, 18),

    "api-management--index--6": () =>
      box(64, 48, 472, 224) +
      label(88, 84, "Credit balance", p.ink, 21) +
      mono(512, 84, "Refill 1,000", p.muted, 18, "end") +
      rule("M88 124h424M88 222h424") +
      `<path d="M88 124h50v24h52v22h52v20h52v32h52v-98h52v20h54v18h58v60H88Z" fill="${p.subtle}"/>` +
      rule("M88 124h50v24h52v22h52v20h52v32h52", p.muted, 2) +
      rule("M346 222v-98h52v20h54v18h58", p.accent, 2) +
      g.circle(346, 124, 3, p.accent) +
      label(88, 251, "Usage", p.muted, 18) +
      label(346, 251, "Scheduled refill", p.accent, 18),

    "api-management--index--7": () =>
      box(64, 48, 472, 224) +
      label(88, 84, "Rate limits", p.ink, 21) +
      mono(512, 84, "100 / min", p.muted, 20, "end") +
      rule("M88 126h424", p.muted, 1) +
      [46, 62, 54, 78, 88, 69, 102, 86, 110, 94, 70, 58]
        .map((height, i) =>
          box(92 + i * 35, 242 - height, 20, height, i < 6 ? p.selected : p.muted, 2),
        )
        .join("") +
      rule("M88 243h424"),

    "api-management--index--8": () =>
      box(64, 48, 472, 224) +
      mono(88, 85, "SELECT code, count()", p.ink, 21) +
      rule("M64 108h472M64 161h472M64 214h472") +
      ["VALID", "RATE_LIMITED", "EXPIRED"]
        .map(
          (value, i) =>
            mono(88, 141 + i * 53, value, i === 0 ? p.accent : p.muted, 19) +
            mono(512, 141 + i * 53, ["9,842", "128", "30"][i], p.ink, 20, "end"),
        )
        .join(""),
  };
  const render = scenes[card.id];
  assert.ok(render, `Missing API overview composition: ${card.id}`);
  return render();
}
