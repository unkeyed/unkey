import { renderEditorialWorkspace } from "./art-editorial.mjs";

// Each composition describes the linked topic in the context of its source page.
// Shared primitives establish a visual language without reusing complete scenes.
export function renderPlatform(card, g) {
  const { p, rect, line, circle, text, check, arrow, cube, chip, brand } = g;
  const t = (x, y, value, color = p.text, size = 20) => text(x, y, value, color, size);
  const tag = (x, y, value, accent = false, width) => chip(x, y, value, { accent, width });
  const ring = (x, y, r, color = p.border, width = 2) =>
    `<circle cx="${x}" cy="${y}" r="${r}" fill="none" stroke="${color}" stroke-width="${width}"/>`;
  const cross = (x, y, color = p.warning, size = 10) =>
    line(
      `M${x - size} ${y - size}l${size * 2} ${size * 2}m0 -${size * 2}l-${size * 2} ${size * 2}`,
      color,
      2.5,
    );
  const dots = (x, y, n = 6, color = p.muted) =>
    Array.from({ length: n }, (_, i) => circle(x + i * 15, y, 3, color)).join("");
  const bars = (x, y, widths = [100, 70, 88]) =>
    widths.map((w, i) => rect(x, y + i * 17, w, 5, p.border, 2)).join("");
  const avatar = (x, y, size = 24, active = false) =>
    circle(x, y, size, active ? p.tint : p.raised) +
    ring(x, y, size) +
    circle(x, y - size * 0.22, size * 0.25, active ? p.accent : p.muted) +
    line(
      `M${x - size * 0.5} ${y + size * 0.46}Q${x} ${y - size * 0.15} ${x + size * 0.5} ${y + size * 0.46}`,
      active ? p.accent : p.muted,
      3,
    );
  const key = (x, y, size = 1, color = p.accent) =>
    `<g transform="translate(${x} ${y}) scale(${size})">${ring(0, 0, 15, color, 3)}${line("M15 0h43m-14 0v12m13 -12v8", color, 3)}</g>`;
  const credential = (x, y, w = 210, accent = false) =>
    rect(x, y, w, 86, accent ? p.tint : p.raised, 16, true) +
    key(x + 35, y + 30, 0.65) +
    dots(x + 24, y + 63, Math.min(10, Math.floor((w - 40) / 15)));
  const shield = (x, y, size = 100, guarded = true) =>
    `<path d="M${x} ${y - size * 0.48}L${x + size * 0.42} ${y - size * 0.3}v${size * 0.4}Q${x + size * 0.38} ${y + size * 0.4} ${x} ${y + size * 0.58}Q${x - size * 0.38} ${y + size * 0.4} ${x - size * 0.42} ${y + size * 0.1}v-${size * 0.4}Z" fill="${p.raised}" stroke="${p.border}" stroke-width="1.5"/>` +
    (guarded ? check(x - 15, y - 2, 30) : cross(x, y + 5));
  const folder = (x, y, w = 260, h = 150) =>
    `<path d="M${x + 18} ${y}h${w * 0.3}l18 20h${w * 0.7 - 54}q18 0 18 18v${h - 56}q0 18 -18 18H${x + 18}q-18 0 -18 -18V${y + 18}q0 -18 18 -18Z" fill="${p.raised}" stroke="${p.border}" stroke-width="1.5"/>`;
  const server = (x, y, w = 110, active = true) =>
    rect(x, y, w, 112, p.raised, 16, true) +
    [0, 1, 2]
      .map(
        (i) =>
          rect(x + 15, y + 18 + i * 27, w - 30, 17, p.panel, 5) +
          circle(x + 28, y + 26 + i * 27, 3, active ? p.accent : p.border),
      )
      .join("");
  const paper = (x, y, w = 150, h = 174) =>
    rect(x, y, w, h, p.raised, 12, true) + bars(x + 22, y + 26, [w - 44, w - 62, w - 52]);
  const clock = (x, y, r = 52, warning = false) =>
    circle(x, y, r, p.raised) +
    ring(x, y, r) +
    ring(x, y, r - 10, p.border, 1) +
    line(
      `M${x} ${y - r * 0.55}v${r * 0.55}l${r * 0.36} ${r * 0.2}`,
      warning ? p.warning : p.accent,
      3,
    );
  const phone = (x, y, code = "248 619") =>
    rect(x, y, 122, 204, p.raised, 23, true) +
    rect(x + 41, y + 12, 40, 5, p.border, 3) +
    ring(x + 61, y + 78, 27, p.accent, 2) +
    check(x + 49, y + 68, 24) +
    text(x + 61, y + 139, code, p.text, 22, "middle") +
    rect(x + 40, y + 185, 42, 4, p.border, 2);
  const terminal = (x, y, w = 280, command = "unkey deploy") =>
    rect(x, y, w, 140, p.raised, 16, true) +
    line(`M${x + 24} ${y + 37}l9 9 -9 9m23 0h13`, p.accent, 2.5) +
    t(x + 23, y + 99, command, p.text, 22);
  const mail = (x, y, w = 120) =>
    rect(x, y, w, w * 0.68, p.raised, 13, true) +
    line(`M${x + 10} ${y + 13}l${w / 2 - 10} ${w * 0.29} ${w / 2 - 10} -${w * 0.29}`, p.border, 2);
  const meter = (x, y, width, fraction = 0.65, color = p.accent) =>
    rect(x, y, width, 13, p.border, 6) +
    (fraction > 0 ? rect(x, y, width * fraction, 13, color, 6) : "");
  const codeBraces = (x, y, size = 52) => text(x, y, "{ }", p.accent, size, "middle");
  const globe = (x, y, r = 52) =>
    ring(x, y, r) +
    `<ellipse cx="${x}" cy="${y}" rx="${r * 0.42}" ry="${r}" fill="none" stroke="${p.border}" stroke-width="1.5"/>` +
    line(
      `M${x - r} ${y}h${r * 2}M${x - r * 0.85} ${y - r * 0.5}h${r * 1.7}M${x - r * 0.85} ${y + r * 0.5}h${r * 1.7}`,
      p.border,
    );
  const plot = (x, y, w = 260, h = 140) =>
    rect(x, y, w, h, p.raised, 16, true) +
    [0.3, 0.6, 0.9].map((f) => line(`M${x + 22} ${y + h * f}h${w - 44}`, p.border, 1)).join("") +
    line(
      `M${x + 24} ${y + h - 26}C${x + w * 0.3} ${y + h - 15} ${x + w * 0.25} ${y + h * 0.48} ${x + w * 0.48} ${y + h * 0.6}S${x + w * 0.72} ${y + 35} ${x + w - 24} ${y + 27}`,
      p.accent,
      3,
    );
  const bracket = (x, y, w, h) =>
    line(`M${x + 12} ${y}h-12v${h}h12m${w - 24} -${h}h12v${h}h-12`, p.accent, 2);

  switch (card.id) {
    case "platform--index--1":
      return (
        folder(169, 83, 265, 173) +
        brand("unkey", 196, 126, 45) +
        t(254, 157, "Workspace", p.text, 24) +
        avatar(166, 61, 25) +
        avatar(224, 56, 25, true) +
        avatar(282, 61, 25) +
        rect(370, 193, 87, 63, p.panel, 12, true) +
        cube(393, 207, 31) +
        rect(457, 127, 74, 62, p.raised, 12, true) +
        key(478, 157, 0.45)
      );
    case "platform--index--2":
      return (
        rect(116, 60, 367, 202, p.raised, 20, true) +
        brand("unkey", 145, 85, 43) +
        t(204, 116, "acme", p.text, 28) +
        t(145, 171, "Workspace settings", p.muted) +
        line("M146 213h280", p.border, 3) +
        circle(218, 213, 11, p.accent) +
        circle(387, 213, 11, p.raised) +
        ring(387, 213, 11)
      );
    case "platform--index--3":
      return (
        line("M300 118v50m0 0H161v39m139 -39h139v39", p.border, 2) +
        avatar(300, 84, 39, true) +
        avatar(160, 217, 32) +
        avatar(300, 217, 32) +
        avatar(440, 217, 32) +
        tag(359, 62, "Admin", true, 108) +
        t(261, 280, "Your team")
      );
    case "platform--index--4":
      return (
        phone(155, 53) +
        rect(322, 101, 179, 117, p.raised, 18, true) +
        t(346, 140, "Sign-in") +
        dots(347, 168, 6) +
        circle(472, 202, 27, p.tint) +
        check(459, 192, 26) +
        arrow(283, 157, 29)
      );
    case "platform--index--5":
      return (
        line("M186 141v72h114m114 -72v72H300v20", p.border, 2) +
        rect(76, 60, 218, 84, p.raised, 16, true) +
        t(97, 110, "API Management") +
        rect(316, 60, 208, 84, p.raised, 16, true) +
        t(377, 110, "Compute") +
        tag(211, 230, "Shared billing", true, 180)
      );
    case "platform--index--6":
      return (
        rect(113, 63, 344, 194, p.raised, 18, true) +
        t(139, 104, "Resource limits", p.text, 24) +
        [0, 1, 2]
          .map((i) =>
            rect(139, 127 + i * 34, [205, 133, 236][i], 15, i === 1 ? p.accent : p.border, 5),
          )
          .join("") +
        line("M419 124v92", p.accent, 2) +
        tag(458, 212, "Plan", false, 94)
      );
    case "platform--index--7":
      return (
        plot(98, 64, 331, 180) +
        tag(323, 245, "This month", true, 173) +
        t(124, 103, "Usage", p.text, 24) +
        line("M470 99v81m-12 -67h24m-24 29h24m-24 29h24", p.border, 2)
      );
    case "platform--index--8":
      return (
        ring(302, 151, 121, p.border, 1) +
        shield(302, 146, 173) +
        credential(59, 168, 167) +
        avatar(454, 111, 32, true) +
        tag(388, 242, "Protected", true, 157)
      );
    case "platform--index--9":
      return (
        credential(66, 105, 183) +
        arrow(260, 148, 45) +
        rect(326, 78, 201, 160, p.raised, 18, true) +
        t(355, 114, "SHA-256", p.accent, 23) +
        [0, 1, 2, 3].map((i) => dots(352, 141 + i * 18, 10, i % 2 ? p.border : p.muted)).join("")
      );
    case "platform--index--10":
      return (
        rect(89, 78, 148, 157, p.raised, 20, true) +
        brand("github", 128, 99, 70) +
        t(111, 207, "Public repo") +
        line("M250 154h87", p.accent, 2, "4 7") +
        mail(365, 117, 146) +
        circle(512, 91, 22, p.tint) +
        t(508, 99, "!", p.accent, 24) +
        tag(284, 236, "Root key alert", true, 183)
      );
    case "platform--index--11":
      return (
        folder(121, 82, 281, 172) +
        cube(159, 137, 64) +
        shield(424, 175, 119) +
        t(250, 239, "Protected", p.accent)
      );
    case "platform--index--12":
      return (
        terminal(111, 82, 344, "unkey") +
        brand("unkey", 401, 65, 57) +
        tag(153, 235, "api", false, 78) +
        tag(244, 235, "deploy", true, 108) +
        tag(365, 235, "auth", false, 86)
      );
    case "platform--index--13":
      return (
        cube(136, 96, 104) +
        line("M324 74v109m-15 -15l15 15 15 -15", p.accent, 3) +
        rect(274, 207, 232, 50, p.raised, 12, true) +
        t(295, 238, "unkey --version", p.text, 22) +
        check(408, 107, 34)
      );
    case "platform--index--14":
      return (
        credential(74, 71, 204, true) +
        rect(109, 180, 169, 49, p.raised, 10) +
        t(130, 212, "Saved config", p.muted, 19) +
        line("M287 114h29q17 0 17 17v26h34m-80 47h29q17 0 17 -17v-30", p.border, 2) +
        terminal(373, 96, 155, "unkey") +
        tag(70, 249, "Explicit key first", true, 224)
      );
    case "platform--index--15":
      return (
        terminal(71, 67, 299, "unkey api --json") +
        rect(402, 134, 128, 123, p.raised, 16, true) +
        codeBraces(466, 194) +
        arrow(344, 234, 36) +
        t(95, 245, "Script-ready", p.muted)
      );
    case "platform--index--16":
      return (
        tag(74, 67, "POST", true, 103) +
        rect(75, 121, 440, 100, p.raised, 17, true) +
        t(107, 180, "api.unkey.com", p.text, 32) +
        line("M186 90h280v18", p.border, 2) +
        tag(339, 242, "meta + data", false, 176)
      );
    case "platform--index--17":
      return (
        rect(95, 70, 319, 168, p.raised, 18, true) +
        t(120, 113, "Authorization", p.muted) +
        t(120, 158, "Bearer", p.text, 25) +
        dots(213, 151, 9) +
        brand("unkey", 125, 184, 28) +
        shield(453, 185, 108)
      );
    case "platform--index--18":
      return (
        rect(65, 96, 471, 94, p.raised, 16, true) +
        t(87, 151, "/v2/", p.muted, 27) +
        t(149, 151, "keys", p.text, 27) +
        t(211, 151, ".", p.muted, 27) +
        t(224, 151, "verifyKey", p.accent, 27) +
        bracket(147, 126, 60, 40) +
        line("M176 191v37m111 -37v37", p.border, 1.5) +
        t(126, 257, "Service", p.muted, 18) +
        t(253, 257, "Procedure", p.muted, 18)
      );
    case "platform--index--19":
      return (
        rect(108, 62, 275, 195, p.raised, 17, true) +
        t(134, 109, "error.status", p.warning, 22) +
        t(134, 153, "error.type", p.text, 22) +
        t(134, 197, "error.detail", p.muted, 22) +
        tag(389, 219, "requestId", true, 159) +
        line("M432 83v85m-13 -67h26m-26 27h26m-26 27h26", p.border, 2)
      );
    case "platform--index--20":
      return (
        clock(164, 152, 70) +
        t(133, 260, "1 minute", p.muted, 21) +
        rect(279, 85, 233, 126, p.raised, 17, true) +
        t(305, 129, "Workspace", p.text, 25) +
        meter(306, 161, 177, 0.72) +
        tag(349, 235, "Remaining", true, 162)
      );
    case "platform--index--21":
      return (
        credential(74, 138, 207) +
        credential(318, 68, 207, true) +
        line(
          "M132 115C155 35 320 25 384 54m-12 -15l12 15 -20 2M466 183c-18 84 -181 97 -254 53m18 0l-18 0 8 17",
          p.accent,
          2.5,
        ) +
        tag(431, 242, "Rotate", true, 126)
      );
    case "platform--index--22":
      return (
        credential(76, 80, 223) +
        line("M313 123h43v-34m0 34v86h26", p.border, 2) +
        tag(371, 66, "Read", true, 128) +
        tag(384, 190, "Write", false, 117) +
        cross(504, 212, p.muted, 7) +
        t(103, 246, "Least privilege", p.text, 23)
      );
    case "platform--index--23":
      return (
        rect(82, 105, 431, 99, p.raised, 17, true) +
        tag(104, 134, "api", false, 75) +
        circle(196, 155, 3, p.muted) +
        tag(212, 134, "*", true, 55) +
        circle(284, 155, 3, p.muted) +
        tag(303, 134, "read_key", false, 179) +
        t(168, 255, "resource.id.action", p.muted, 23)
      );
    case "platform--index--24":
      return (
        folder(84, 56, 421, 191) +
        t(111, 120, "Workspace", p.muted, 22) +
        line("M124 134v28h30m0 12v29h30", p.border, 2) +
        t(168, 170, "Project", p.text, 22) +
        t(196, 212, "Keyspace / keys", p.text, 22) +
        tag(413, 248, "#read", true, 116)
      );
    case "platform--index--25":
      return (
        rect(137, 52, 265, 206, p.panel, 15) +
        rect(154, 65, 265, 206, p.raised, 15, true) +
        line("M193 65v206", p.border, 2) +
        t(221, 128, "Aa", p.text, 47) +
        t(219, 178, "Glossary", p.muted, 24) +
        rect(402, 104, 48, 25, p.tint, 6) +
        rect(402, 149, 48, 25, p.border, 6) +
        rect(402, 194, 48, 25, p.border, 6)
      );
    case "platform--index--26":
      return (
        brand("github", 76, 123, 60) +
        arrow(154, 154, 52) +
        cube(224, 96, 106) +
        arrow(347, 154, 40) +
        server(407, 100, 112) +
        tag(237, 241, "Build. Run. Route.", true, 239)
      );
    case "platform--index--27":
      return (
        globe(415, 150, 84) +
        credential(72, 77, 216) +
        shield(199, 223, 78) +
        line("M302 120h51m-51 49h51", p.accent, 2, "4 6") +
        t(351, 267, "Your API", p.text, 25)
      );
    case "platform--cli--overview--1":
      return (
        cube(93, 106, 98) +
        arrow(205, 158, 51) +
        terminal(284, 82, 247, "unkey --version") +
        circle(482, 231, 24, p.tint) +
        check(470, 220, 24)
      );
    case "platform--cli--overview--2":
      return (
        terminal(82, 112, 244, "unkey auth") +
        credential(297, 63, 212, true) +
        line("M405 164v65h-52", p.accent, 2) +
        tag(349, 244, "Root key", false, 151)
      );
    case "platform--cli--overview--3":
      return (
        rect(84, 61, 258, 195, p.raised, 16, true) +
        codeBraces(145, 133, 52) +
        bars(191, 103, [111, 82, 96]) +
        tag(115, 194, "--json", true, 135) +
        line("M359 149h49", p.accent, 2) +
        rect(426, 103, 93, 93, p.raised, 17, true) +
        check(452, 135, 35)
      );
    case "platform--cli--install--1":
      return (
        credential(155, 55, 277, true) +
        line("M293 154v30m-10 -10l10 10 10 -10", p.accent, 2) +
        rect(109, 200, 380, 78, p.raised, 16, true) +
        line("M133 228l10 10 -10 10", p.accent, 2.5) +
        t(164, 246, "unkey auth", p.text, 25)
      );
    case "platform--cli--install--2":
      return (
        terminal(75, 70, 277, "unkey api") +
        line("M371 138h33v-48h24m-24 48v82h24", p.border, 2) +
        tag(430, 67, "JSON", true, 100) +
        tag(430, 198, "Text", false, 95) +
        t(100, 257, "Choose your output", p.muted, 23)
      );
    case "platform--root-keys--overview--1":
      return (
        key(148, 91, 1.2) +
        line("M228 91h81v57m0 0h-98v45m98 -45h119v45", p.border, 2) +
        tag(99, 205, "Legacy tuple", false, 195) +
        tag(320, 205, "Resource name", true, 209)
      );
    case "platform--root-keys--overview--2":
      return (
        [0, 1, 2]
          .map((i) =>
            rect(92 + 145 * i, 73 + 31 * i, 125, 111, i === 1 ? p.tint : p.raised, 15, true),
          )
          .join("") +
        t(127, 132, "api", p.text, 29) +
        text(300, 164, "*", p.accent, 38, "middle") +
        t(397, 192, "read_key", p.text, 19) +
        line("M219 128h14m130 31h14", p.border, 2) +
        t(139, 268, "Three parts. One grant.", p.muted, 23)
      );
    case "platform--root-keys--overview--3":
      return (
        rect(86, 61, 302, 193, p.raised, 17, true) +
        t(110, 104, "unkey:v1", p.accent, 25) +
        line("M123 124v24h29m14 16v18h14m12 16v15h14", p.border, 2) +
        t(166, 155, "Workspace", p.muted, 19) +
        t(192, 189, "Project", p.muted, 19) +
        t(218, 220, "Resource", p.text, 19) +
        tag(415, 154, "#read", true, 113)
      );
    case "platform--security--key-storage--1":
      return (
        rect(99, 67, 177, 186, p.raised, 19, true) +
        ring(186, 143, 42, p.border, 3) +
        key(169, 143, 0.7) +
        bars(142, 213, [87]) +
        arrow(294, 161, 32) +
        credential(345, 117, 192, true) +
        t(354, 252, "Recoverable", p.muted, 22)
      );
    case "platform--security--overview--1":
      return (
        credential(65, 75, 198) +
        line("M281 117h48v46h32", p.accent, 2) +
        rect(366, 101, 161, 155, p.raised, 17, true) +
        t(390, 143, "Hash only", p.text, 22) +
        [0, 1, 2].map((i) => dots(392, 176 + i * 21, 7)).join("") +
        tag(85, 228, "One-way", true, 148)
      );
    case "platform--security--overview--2":
      return (
        brand("github", 88, 79, 76) +
        line("M180 118h60q21 0 21 21v50h50", p.accent, 2, "4 7") +
        mail(330, 151, 159) +
        credential(282, 59, 211) +
        tag(80, 224, "Rotate the key", true, 207)
      );
    case "platform--security--overview--3":
      return (
        folder(109, 78, 172, 128) +
        cube(310, 72, 85) +
        shield(437, 205, 112) +
        line("M139 231h229", p.border, 2) +
        t(133, 271, "Deletion guard", p.accent, 25)
      );
    case "platform--security--overview--4":
      return (
        avatar(137, 118, 39, true) +
        line("M192 121h90q21 0 21 21v46h48", p.border, 2) +
        phone(385, 57) +
        tag(83, 217, "Second factor", true, 202)
      );
    case "platform--workspace--settings--1":
      return (
        mail(85, 99, 161) +
        arrow(270, 154, 43) +
        avatar(395, 127, 50, true) +
        tag(333, 206, "Developer", false, 158) +
        t(111, 247, "Invite a teammate", p.muted, 22)
      );
    case "platform--workspace--settings--2":
      return (
        phone(109, 53, "••• •••") +
        rect(288, 93, 227, 144, p.raised, 17, true) +
        t(310, 132, "Authenticator", p.text, 23) +
        dots(312, 165, 6) +
        tag(317, 187, "Enabled", true, 144) +
        arrow(246, 150, 29)
      );
    case "platform--workspace--settings--3":
      return (
        paper(224, 63, 179, 201) +
        t(249, 156, "Billing", p.text, 25) +
        meter(249, 191, 116, 0.72) +
        tag(68, 74, "API", false, 108) +
        tag(427, 162, "Compute", true, 122) +
        line("M179 96h32m205 88h10", p.border, 2)
      );
    case "platform--workspace--settings--4":
      return (
        rect(102, 71, 383, 189, p.raised, 18, true) +
        t(128, 112, "Month to date", p.text, 25) +
        t(128, 159, "API", p.muted, 20) +
        meter(219, 147, 232, 0.46) +
        t(128, 209, "Compute", p.muted, 20) +
        meter(219, 197, 232, 0.76) +
        tag(474, 220, "Usage", true, 82)
      );

    case "errors--frontline--capacity--deployment_offline--1":
      return (
        server(125, 76, 159, false) +
        ring(204, 240, 26, p.accent, 2.5) +
        line("M204 207v32", p.accent, 3) +
        rect(346, 111, 172, 112, p.raised, 17, true) +
        t(371, 151, "Offline", p.text, 27) +
        tag(370, 174, "Start", true, 105)
      );
    case "errors--frontline--capacity--no_running_instances--1":
      return (
        server(99, 99, 110, false) +
        server(248, 77, 110, false) +
        server(397, 99, 110, false) +
        line("M127 244h350", p.border, 2, "4 7") +
        tag(216, 238, "Starting", true, 167)
      );
    case "errors--frontline--capacity--spend_limit_reached--1":
      return (
        rect(106, 74, 344, 180, p.raised, 18, true) +
        t(131, 119, "Spend budget", p.text, 27) +
        meter(132, 151, 287, 1, p.warning) +
        t(132, 211, "At the ceiling", p.muted, 22) +
        circle(472, 197, 46, p.panel) +
        ring(472, 197, 46) +
        line("M462 181v32m20 -32v32", p.warning, 5)
      );
    case "errors--frontline--client--firewall_denied--1":
      return (
        [0, 1, 2]
          .map(
            (i) =>
              line(`M85 ${109 + i * 42}h135`, p.muted, 2) +
              circle(123 + i * 20, 109 + i * 42, 5, p.border),
          )
          .join("") +
        shield(304, 150, 141, false) +
        server(427, 96, 96) +
        t(115, 268, "Review the firewall rule", p.text, 23)
      );
    case "errors--frontline--client--insufficient_permissions--1":
      return (
        credential(74, 79, 221, true) +
        check(250, 62, 20) +
        arrow(310, 124, 35) +
        rect(367, 64, 160, 178, p.raised, 16, true) +
        key(403, 109, 0.8, p.muted) +
        cross(447, 175, p.warning, 17) +
        t(97, 250, "Valid key. Missing grant.", p.text, 23)
      );
    case "errors--frontline--client--insufficient_permissions--2":
      return (
        credential(165, 157, 262, true) +
        tag(81, 60, "Direct grant", false, 183) +
        tag(337, 60, "Role grant", false, 178) +
        line("M177 107v24h119m129 -24v24H296v17", p.accent, 2) +
        t(191, 276, "documents.read", p.text, 22)
      );
    case "errors--frontline--client--invalid_key--1":
      return (
        credential(81, 105, 268) +
        cross(304, 91, p.warning, 15) +
        ring(435, 156, 62, p.border, 2) +
        key(417, 156, 0.8, p.muted) +
        line("M388 202l94 -94", p.warning, 3) +
        t(119, 263, "Check key validity", p.text, 25)
      );
    case "errors--frontline--client--invalid_key--2":
      return (
        credential(178, 60, 244) +
        rect(126, 189, 129, 62, p.tint, 31) +
        circle(223, 220, 23, p.accent) +
        rect(345, 189, 129, 62, p.raised, 31) +
        circle(377, 220, 23, p.border) +
        t(145, 285, "Enabled", p.muted, 19) +
        t(362, 285, "Disabled", p.muted, 19)
      );
    case "errors--frontline--client--missing_credentials--1":
      return (
        rect(91, 70, 413, 177, p.raised, 18, true) +
        t(119, 117, "Authorization", p.muted, 24) +
        rect(116, 144, 273, 62, p.panel, 10) +
        line("M137 174h137", p.border, 2, "5 8") +
        key(430, 176, 0.8) +
        line("M412 177h-16m6 -6l-6 6 6 6", p.accent, 2) +
        t(157, 281, "Send your API key", p.text, 24)
      );
    case "errors--frontline--client--missing_credentials--2":
      return (
        folder(83, 74, 190, 151) +
        key(130, 159, 1) +
        credential(326, 111, 205, true) +
        arrow(281, 151, 31) +
        tag(326, 231, "Copy once", true, 160)
      );
    case "errors--frontline--client--openapi_validation_failed--1":
      return (
        paper(96, 64, 182, 198) +
        codeBraces(185, 189, 57) +
        arrow(297, 160, 34) +
        rect(351, 101, 171, 107, p.raised, 16, true) +
        t(378, 144, "Schema", p.text, 25) +
        cross(436, 177, p.warning, 11) +
        tag(339, 240, "400", false, 101)
      );
    case "errors--frontline--client--rate_limited--1":
      return (
        clock(389, 141, 83) +
        [0, 1, 2, 3]
          .map((i) => circle(96 + i * 44, 149, 9, i < 3 ? p.border : p.warning))
          .join("") +
        line("M269 108v82", p.warning, 3) +
        tag(92, 236, "Wait for capacity", true, 239)
      );
    case "errors--frontline--client--rate_limited--2":
      return (
        avatar(306, 80, 33, true) +
        line("M306 121v25h-132v30m132 -30h132v30", p.border, 2) +
        credential(74, 179, 202) +
        credential(331, 179, 202) +
        tag(364, 66, "Shared limit", true, 190)
      );
    case "errors--frontline--client--usage_exceeded--1":
      return (
        rect(132, 73, 333, 174, p.raised, 18, true) +
        t(158, 115, "Key credits", p.text, 26) +
        text(299, 183, "0", p.warning, 51, "middle") +
        meter(159, 210, 280, 0) +
        tag(390, 239, "Refill", true, 112)
      );
    case "errors--frontline--client--usage_exceeded--2":
      return (
        rect(103, 59, 369, 203, p.raised, 18, true) +
        line("M130 203h313", p.accent, 2) +
        line("M141 96l88 38 64 66h91", p.text, 3) +
        circle(294, 200, 6, p.accent) +
        t(365, 186, "0", p.accent, 34) +
        t(130, 244, "Balance floor", p.muted, 21)
      );
    case "errors--frontline--config--invalid_configuration--1":
      return (
        paper(151, 53, 231, 216) +
        [0, 1, 2]
          .map((i) =>
            rect(176, 145 + i * 31, [145, 103, 137][i], 8, i === 1 ? p.warning : p.border, 4),
          )
          .join("") +
        circle(415, 183, 52, p.panel) +
        cross(415, 183, p.warning, 20) +
        tag(386, 247, "Review", true, 124)
      );
    case "errors--frontline--platform--config_load_failed--1":
      return (
        paper(94, 60, 165, 186) +
        line("M275 149h56m28 0h39", p.border, 2, "5 7") +
        cross(346, 149, p.warning, 10) +
        server(412, 94, 105) +
        t(99, 281, "Configuration unavailable", p.text, 23)
      );
    case "errors--frontline--platform--deployment_selection_failed--1":
      return (
        server(364, 58, 85, false) +
        server(452, 144, 85, false) +
        line("M191 155h99V114h54m-54 41v42h140", p.border, 2, "5 7") +
        ring(150, 155, 40, p.border, 2) +
        t(139, 166, "?", p.muted, 32) +
        tag(77, 238, "Reserved code", false, 218)
      );
    case "errors--frontline--platform--internal_server_error--1":
      return (
        rect(89, 92, 408, 126, p.raised, 18, true) +
        line("M112 154h85l24 -28 29 53 30 -47 28 22h158", p.border, 2.5) +
        circle(250, 179, 7, p.warning) +
        tag(333, 61, "500", false, 105) +
        tag(115, 238, "Trace with requestId", true, 269)
      );
    case "errors--frontline--routing--config_not_found--1":
      return (
        globe(152, 140, 58) +
        line("M224 140h83", p.border, 2, "5 7") +
        rect(335, 83, 169, 121, p.panel, 17) +
        cross(419, 142, p.warning, 23) +
        tag(109, 239, "Check the hostname", true, 275)
      );
    case "errors--frontline--routing--deployment_not_found--1":
      return (
        folder(102, 70, 297, 184) +
        circle(443, 164, 57, p.panel) +
        ring(443, 164, 57, p.border, 3) +
        line("M484 207l43 41", p.muted, 6) +
        t(431, 176, "?", p.muted, 37) +
        tag(128, 211, "Reserved", false, 154)
      );
    case "errors--frontline--upstream--bad_gateway--1":
      return (
        server(390, 67, 117) +
        line("M358 127h-54l-20 -20 -19 41 -20 -20H100", p.warning, 3) +
        line("M116 180h194", p.border, 2, "5 7") +
        tag(91, 61, "502", false, 104) +
        t(120, 256, "Connection reset", p.text, 26)
      );
    case "errors--frontline--upstream--gateway_timeout--1":
      return (
        server(93, 84, 127) +
        line("M239 140h78", p.border, 2, "5 7") +
        clock(409, 140, 79, true) +
        t(103, 269, "Response deadline", p.text, 24) +
        tag(394, 244, "504", false, 105)
      );
    case "errors--frontline--upstream--proxy_forward_failed--1":
      return (
        rect(81, 111, 114, 90, p.raised, 15, true) +
        arrow(117, 156, 43) +
        line("M214 156h150", p.border, 2, "5 7") +
        line("M283 126v59", p.muted, 2) +
        cube(401, 102, 94) +
        tag(178, 244, "Reserved forwarding code", false, 321)
      );
    case "errors--frontline--upstream--service_unavailable--1":
      return (
        server(340, 73, 167, false) +
        line("M95 150h125m34 0h68", p.border, 3) +
        circle(228, 150, 10, p.panel) +
        ring(228, 150, 10, p.warning, 2) +
        cross(278, 150, p.warning, 11) +
        tag(104, 232, "Check the listening port", true, 312)
      );
    case "errors--overview--1":
      return (
        rect(102, 72, 293, 179, p.raised, 18, true) +
        t(128, 117, "Bearer", p.text, 29) +
        rect(128, 151, 239, 55, p.panel, 10) +
        line("M149 179h195", p.border, 2, "5 8") +
        key(419, 144, 1.2) +
        t(126, 281, "Add authorization", p.muted, 22)
      );
    case "errors--overview--2":
      return (
        shield(180, 138, 144, false) +
        credential(331, 87, 211) +
        tag(329, 210, "Root key grant", true, 207) +
        line("M256 140h55", p.border, 2)
      );
    case "errors--overview--3":
      return (
        folder(76, 68, 220, 179) +
        folder(342, 110, 194, 137) +
        key(115, 159, 1) +
        line("M306 170h24", p.border, 2, "3 5") +
        t(425, 203, "?", p.muted, 34) +
        t(88, 281, "Check the workspace", p.text, 23)
      );
    case "errors--overview--4":
      return (
        cube(107, 105, 103) +
        line("M232 155h69", p.border, 2) +
        rect(328, 78, 178, 165, p.raised, 18, true) +
        circle(378, 119, 10, p.accent) +
        circle(378, 158, 10, p.border) +
        circle(378, 197, 10, p.border) +
        line("M406 119h69m-69 39h43m-43 39h58", p.border, 5) +
        tag(74, 241, "Check resource state", true, 282)
      );
    case "errors--overview--5":
      return (
        globe(153, 145, 67) +
        rect(279, 61, 233, 193, p.raised, 18, true) +
        t(305, 105, "Domains", p.text, 27) +
        [0, 1, 2]
          .map((i) => circle(319 + i * 65, 151, 16, p.tint) + check(310 + i * 65, 143, 18))
          .join("") +
        meter(305, 207, 181, 1, p.warning)
      );
    case "errors--overview--6":
      return (
        paper(100, 52, 191, 213) +
        codeBraces(194, 193, 56) +
        bracket(327, 104, 165, 118) +
        line("M350 163h119", p.warning, 2.5) +
        t(363, 147, "Body", p.text, 27) +
        t(337, 260, "Too large", p.muted, 23)
      );
    case "errors--overview--7":
      return (
        rect(88, 76, 260, 165, p.raised, 18, true) +
        t(114, 123, "SELECT", p.accent, 27) +
        bars(115, 159, [182, 138, 158]) +
        clock(426, 159, 67, true) +
        tag(126, 254, "Query timeout", false, 207)
      );
    case "errors--overview--8":
      return (
        rect(106, 81, 377, 153, p.raised, 18, true) +
        t(132, 125, "Workspace allowance", p.text, 26) +
        meter(132, 160, 322, 1, p.warning) +
        t(134, 207, "429", p.warning, 28) +
        clock(475, 239, 35) +
        t(147, 278, "Retry after reset", p.muted, 22)
      );
    case "errors--overview--9":
      return (
        credential(101, 66, 253) +
        line("M231 165v53h145", p.border, 2) +
        shield(429, 199, 109, false) +
        tag(98, 233, "Verify the key", true, 202)
      );
    case "errors--overview--10":
      return (
        cube(102, 128, 80) +
        cube(242, 93, 105) +
        cube(414, 128, 80) +
        circle(142, 231, 7, p.border) +
        circle(294, 231, 7, p.border) +
        circle(454, 231, 7, p.border) +
        t(170, 283, "No running instances", p.text, 23)
      );
    case "errors--overview--11":
      return (
        globe(132, 125, 51) +
        line("M198 126h97v77h57", p.border, 2, "5 7") +
        folder(373, 145, 140, 99) +
        t(409, 219, "?", p.muted, 31) +
        tag(80, 229, "Reserved route", false, 219)
      );
    case "errors--overview--12":
      return (
        rect(93, 90, 135, 123, p.raised, 17, true) +
        brand("unkey", 129, 118, 60) +
        server(394, 89, 113) +
        line("M247 133h120M367 175h-37l-16 -15 -17 30 -16 -15h-34", p.warning, 2.5) +
        t(173, 269, "Upstream response", p.text, 24)
      );
    case "errors--overview--13":
      return (
        rect(87, 67, 281, 180, p.raised, 18, true) +
        codeBraces(147, 131, 51) +
        bars(116, 169, [210, 141, 173]) +
        shield(439, 155, 120, false) +
        tag(311, 251, "Configuration", false, 201)
      );
    case "errors--overview--14":
      return (
        line("M123 178h350", p.border, 2) +
        [0, 1, 2, 3]
          .map((i) => circle(132 + i * 113, 178, i === 2 ? 15 : 9, i === 2 ? p.warning : p.border))
          .join("") +
        rect(277, 55, 251, 76, p.raised, 14, true) +
        t(300, 102, "meta.requestId", p.text, 26) +
        line("M358 143v15", p.warning, 2) +
        t(120, 264, "Trace the failure", p.muted, 24)
      );
    default:
      throw new Error(`Missing Platform illustration: ${card.id}`);
  }
}

// The home-page workspace strip uses its own wider, quieter product grouping.
export function renderWorkspaceOverview(g) {
  return renderEditorialWorkspace(g);
}
