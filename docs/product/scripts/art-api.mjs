import { renderEditorialApi } from "./art-editorial.mjs";

/** Each API card tells a separate substory; helpers share objects, not compositions. */
export function renderApi(card, g) {
  if (card.source === "api-management/index.mdx") return renderEditorialApi(card, g);
  const { p, rect, line, circle, text, arrow, cube, chip, brand } = g;
  const check = (x, y, size) => g.check(x, y - size / 2, size);
  const id = card.id.replace(/^api-management--/, "");
  const join = (...parts) => parts.flat().join("");
  const dots = (x, y, count = 6, color = "#d4dcda") =>
    Array.from({ length: count }, (_, i) => circle(x + i * 15, y, 3, color)).join("");
  const key = (x, y, width = 200, label = "sk_live", height = 104) =>
    join(
      rect(x, y, width, height, "#202525", 16, true),
      height >= 80 ? line(`M${x + 34} ${y + 25}a7 7 0 1 1-14 0 7 7 0 1 1 14 0`, "#a8b6b2", 2) : "",
      height >= 80 ? line(`M${x + 34} ${y + 25}h24m-7 0v7m-8 -7v5`, "#a8b6b2", 2) : "",
      text(x + 20, y + height - 24, label, "#f3f6f5", 18),
      width > 150 && height >= 80 ? dots(x + width - 77, y + 26, 4) : "",
    );
  const avatar = (x, y, radius = 30) =>
    join(
      circle(x, y, radius, p.raised),
      circle(x, y - 7, radius * 0.22, p.muted),
      line(
        `M${x - radius * 0.42} ${y + radius * 0.38}Q${x} ${y - radius * 0.1} ${x + radius * 0.42} ${y + radius * 0.38}`,
        p.muted,
        3,
      ),
    );
  const folder = (x, y, width, height, label) =>
    join(
      rect(x + 9, y, width * 0.43, 34, p.raised, 9),
      rect(x, y + 17, width, height - 17, p.panel, 16, true),
      text(x + 21, y + 53, label, p.muted, 18),
    );
  const badge = (x, y, label, good = true, width = 140) =>
    join(
      rect(x, y, width, 50, good ? p.tint : p.raised, 25, true),
      good ? check(x + 19, y + 25, 12) : line(`M${x + 21} ${y + 18}v9m0 6v1`, p.warning, 2.5),
      text(x + 42, y + 32, label, good ? p.accent : p.text, 19),
    );
  const coin = (x, y, radius = 28, active = true) =>
    join(
      circle(x, y + 5, radius, p.border),
      circle(x, y, radius, active ? p.tint : p.panel),
      line(
        `M${x + radius - 7} ${y}a${radius - 7} ${radius - 7} 0 1 1-${(radius - 7) * 2} 0a${radius - 7} ${radius - 7} 0 1 1 ${(radius - 7) * 2} 0`,
        p.border,
      ),
      text(x, y + 7, "U", active ? p.accent : p.muted, 21, "middle"),
    );
  const clock = (x, y, radius = 39) =>
    join(
      circle(x, y, radius, p.raised),
      line(`M${x} ${y - radius * 0.56}V${y}l${radius * 0.38} ${radius * 0.2}`, p.accent, 3),
      circle(x, y, 3, p.accent),
    );
  const calendar = (x, y, label, number = "1") =>
    join(
      rect(x, y, 146, 160, p.panel, 17, true),
      line(`M${x} ${y + 40}h146`, p.border),
      line(`M${x + 32} ${y - 7}v19m82 -19v19`, p.muted, 3),
      text(x + 73, y + 100, number, p.text, 43, "middle"),
      text(x + 73, y + 134, label, p.muted, 18, "middle"),
    );
  const spark = (x, y, width, height, color = p.accent) =>
    line(
      `M${x} ${y + height * 0.85}C${x + width * 0.15} ${y + height * 0.8} ${x + width * 0.14} ${y + height * 0.18} ${x + width * 0.3} ${y + height * 0.42}S${x + width * 0.53} ${y + height * 0.68} ${x + width * 0.66} ${y + height * 0.3}S${x + width * 0.84} ${y + height * 0.26} ${x + width} ${y}`,
      color,
      3,
    );
  const meter = (x, y, width, ratio = 0.65) =>
    join(rect(x, y, width, 12, p.raised, 6), rect(x, y, width * ratio, 12, p.accent, 6));
  const gauge = (x, y, radius = 78, label = "87 / 100", ratio = 0.87) =>
    join(
      line(`M${x - radius} ${y}A${radius} ${radius} 0 0 1 ${x + radius} ${y}`, p.border, 13),
      line(
        `M${x - radius} ${y}A${radius} ${radius} 0 0 1 ${x + radius * Math.cos(Math.PI * (1 - ratio))} ${y - radius * Math.sin(Math.PI * (1 - ratio))}`,
        p.accent,
        13,
      ),
      text(x, y + 4, label, p.text, 28, "middle"),
    );
  const brackets = (x, y, width, height) =>
    join(
      line(`M${x + 12} ${y}h-12v${height}h12M${x + width - 12} ${y}h12v${height}h-12`, p.muted, 2),
    );
  const document = (x, y, width = 140, height = 164) =>
    join(
      rect(x, y, width, height, p.panel, 14, true),
      [0, 1, 2].map((i) =>
        line(`M${x + 22} ${y + 38 + i * 26}h${width - 44 - (i % 2) * 22}`, p.border, 4),
      ),
    );

  const scenes = {
    "analytics--overview--1": () =>
      join(
        line("M207 108H276Q295 108 295 130V176H353M209 227H270Q295 227 295 203V176", p.border, 2),
        rect(60, 66, 170, 85, p.panel, 14, true),
        spark(82, 88, 60, 40),
        text(157, 118, "HTTP", p.muted, 18),
        rect(60, 179, 170, 85, p.panel, 14, true),
        line("M83 204h89m-89 17h64m-64 17h109", p.muted, 3),
        rect(352, 97, 187, 151, p.panel, 18, true),
        text(445, 154, "SQL", p.accent, 35, "middle"),
        text(445, 209, "Compute data", p.muted, 18, "middle"),
      ),
    "analytics--overview--2": () =>
      join(
        rect(65, 65, 320, 179, p.panel, 18, true),
        text(92, 108, "SELECT", p.accent, 22),
        text(92, 148, "outcome, count()", p.text, 22),
        text(92, 198, "GROUP BY outcome", p.muted, 18),
        rect(421, 137, 30, 111, p.tint, 5),
        rect(465, 188, 30, 60, p.accent, 5),
        rect(509, 221, 30, 27, p.raised, 5),
      ),
    "analytics--overview--3": () =>
      join(
        rect(128, 57, 345, 206, p.panel, 20, true),
        brackets(102, 88, 397, 142),
        text(300, 112, "SELECT", p.text, 28, "middle"),
        meter(180, 150, 240, 0.68),
        text(300, 205, "Bounded execution", p.muted, 20, "middle"),
      ),
    "analytics--overview--4": () =>
      join(
        rect(86, 64, 429, 192, p.panel, 20, true),
        text(115, 111, "Valid", p.muted, 18),
        rect(226, 91, 248, 25, p.accent, 6),
        text(115, 163, "Expired", p.muted, 18),
        rect(226, 143, 76, 25, p.raised, 6),
        text(115, 215, "Limited", p.muted, 18),
        rect(226, 195, 115, 25, p.tint, 6),
      ),
    "analytics--overview--5": () =>
      join(
        document(93, 75, 237, 166),
        circle(380, 147, 61, p.panel),
        line("M423 191l45 45", p.muted, 12),
        text(380, 158, "400", p.warning, 31, "middle"),
        chip(85, 236, "Inspect the query", { width: 210 }),
      ),
    "authorization--overview--1": () =>
      join(
        line("M280 111V156M280 156H150V181M280 156H418V181", p.border, 2),
        chip(204, 60, "Editor", { width: 152, accent: true }),
        rect(61, 182, 218, 70, p.panel, 14, true),
        check(82, 216, 13),
        text(109, 222, "documents.read", p.text, 18),
        rect(306, 182, 236, 70, p.panel, 14, true),
        check(327, 216, 13),
        text(354, 222, "documents.write", p.text, 18),
      ),
    "authorization--overview--2": () =>
      join(
        rect(64, 72, 471, 177, p.panel, 18, true),
        chip(104, 102, "admin", { width: 132, accent: true }),
        text(299, 132, "OR", p.muted, 19, "middle"),
        chip(361, 102, "editor", { width: 132 }),
        brackets(87, 91, 423, 71),
        text(164, 213, "AND", p.accent, 20),
        text(254, 213, "billing:view", p.text, 22),
      ),
    "authorization--overview--3": () =>
      join(
        line("M248 95H338Q369 95 369 126V167M250 235H336Q369 235 369 204V167", p.border, 2),
        chip(68, 70, "Direct grant", { width: 180 }),
        chip(68, 209, "Editor role", { width: 180, accent: true }),
        key(350, 120, 190, "Effective grants"),
        circle(288, 165, 23, p.panel),
        text(288, 173, "+", p.accent, 28, "middle"),
      ),
    "authorization--overview--4": () =>
      join(
        key(63, 117, 190, "Presented key"),
        line("M253 169H317", p.border, 2),
        [0, 1, 2].map((i) =>
          join(
            rect(320 + i * 29, 87 - i * 11, 13, 169 + i * 11, p.raised, 6),
            check(321 + i * 29, 119 - i * 11, 10),
          ),
        ),
        badge(415, 143, "Valid", true, 125),
      ),
    "cookbook--index--1": () =>
      join(
        line("M283 112V163H166V189M283 163H430V189", p.border, 2),
        avatar(283, 83, 36),
        key(69, 189, 195, "Mobile", 80),
        key(331, 189, 195, "Server", 80),
        chip(348, 63, "One user limit", { width: 184, accent: true }),
      ),
    "cookbook--index--2": () =>
      join(
        text(68, 98, "/search", p.text, 23),
        meter(246, 80, 287, 0.79),
        text(68, 167, "/export", p.text, 23),
        meter(246, 149, 287, 0.32),
        text(68, 236, "/reports", p.text, 23),
        meter(246, 218, 287, 0.57),
        line("M219 63V255", p.border, 1.5),
      ),
    "cookbook--index--3": () =>
      join(
        rect(65, 84, 269, 154, p.panel, 18, true),
        text(93, 128, "Credits remaining", p.muted, 18),
        text(93, 198, "990", p.text, 50),
        coin(421, 146, 41),
        chip(366, 220, "Cost 10", { width: 130, accent: true }),
        arrow(344, 152, 28),
      ),
    "cookbook--index--4": () =>
      join(
        rect(74, 166, 131, 86, p.panel, 14, true),
        rect(232, 117, 131, 135, p.panel, 14, true),
        rect(390, 68, 137, 184, p.tint, 14, true),
        text(139, 216, "Free", p.muted, 21, "middle"),
        text(297, 168, "Pro", p.text, 21, "middle"),
        text(458, 118, "Enterprise", p.accent, 19, "middle"),
        line("M256 201h83M414 155h89M414 190h89", p.border, 3),
      ),
    "cookbook--index--5": () =>
      join(
        rect(65, 105, 142, 127, p.panel, 20, true),
        brand("go", 94, 132, 85),
        line("M207 169H277V88H347M277 169H347M277 169V248H347", p.border, 2),
        chip(351, 64, "net/http", { width: 171 }),
        chip(351, 144, "Gin", { width: 171 }),
        chip(351, 223, "Echo", { width: 171, accent: true }),
      ),
    "cookbook--index--6": () =>
      join(
        rect(70, 91, 156, 144, p.panel, 20, true),
        brand("fastapi", 109, 111, 77),
        text(148, 214, "FastAPI", p.text, 19, "middle"),
        line("M226 163H349", p.border, 2),
        rect(280, 125, 27, 76, p.raised, 10),
        check(285, 162, 16),
        rect(352, 115, 174, 98, p.panel, 16, true),
        text(439, 155, "Depends", p.muted, 19, "middle"),
        text(439, 186, "Verified user", p.accent, 19, "middle"),
      ),
    "get-started--concepts--1": () =>
      join(
        folder(60, 61, 190, 173, "Keyspace"),
        key(88, 132, 186, "First key", 90),
        line("M274 177H350", p.border, 2),
        circle(433, 165, 60, p.tint),
        check(405, 165, 41),
        text(433, 258, "Verified", p.accent, 21, "middle"),
      ),
    "get-started--concepts--2": () =>
      join(
        folder(109, 61, 380, 197, "Payments keyspace"),
        [0, 1, 2].map((i) => key(136 + i * 105, 144, 91, ["API", "App", "Jobs"][i], 87)),
      ),
    "get-started--quickstart--1": () =>
      join(
        key(136, 83, 314, "sk_live", 147),
        circle(445, 91, 34, p.tint),
        line("M431 91h28M445 77v28", p.accent, 2.5),
        text(297, 271, "Show once. Store securely.", p.muted, 19, "middle"),
      ),
    "get-started--quickstart--2": () =>
      join(
        rect(114, 60, 373, 200, p.panel, 20, true),
        text(145, 106, "HTTP 200", p.muted, 18),
        text(146, 168, "valid:", p.text, 30),
        text(259, 168, "true", p.accent, 30),
        badge(319, 194, "VALID", true, 142),
      ),
    "get-started--quickstart--3": () =>
      join(
        text(132, 127, "1,000", p.muted, 36, "middle"),
        text(465, 127, "990", p.text, 42, "middle"),
        arrow(207, 116, 165),
        coin(285, 197, 35),
        text(346, 204, "−10", p.accent, 25),
        line("M67 257H533", p.border, 1.5),
      ),
    "get-started--quickstart--4": () =>
      join(
        rect(60, 57, 223, 90, p.panel, 16, true),
        brand("unkey", 79, 79, 40),
        text(135, 112, "Root key", p.text, 21),
        folder(325, 56, 215, 90, "Keyspace"),
        key(60, 185, 223, "API key", 80),
        avatar(355, 225, 30),
        text(400, 232, "Identity", p.text, 21),
      ),
    "get-started--quickstart--5": () =>
      join(
        key(54, 123, 164, "API key", 90),
        line("M218 168H275M335 168H422", p.border, 2),
        rect(270, 70, 24, 190, p.raised, 9),
        rect(312, 70, 24, 190, p.raised, 9),
        circle(303, 168, 29, p.tint),
        check(289, 168, 22),
        cube(429, 115, 90),
        text(471, 245, "Your app", p.muted, 19, "middle"),
      ),
    "guides--bun--1": () =>
      join(
        line("M140 159H202M266 159H328M392 159H454", p.border, 2),
        [0, 1, 2, 3].map((i) =>
          join(
            circle(108 + i * 126, 159, 32, i === 3 ? p.raised : p.tint),
            i < 3 ? check(94 + i * 126, 159, 22) : text(486, 167, "4", p.muted, 23, "middle"),
          ),
        ),
        text(171, 90, "Checks pass", p.text, 21, "middle"),
        text(443, 236, "Then spend", p.muted, 21, "middle"),
      ),
    "guides--bun--2": () =>
      join(
        calendar(70, 84, "Monthly"),
        line("M250 108C326 48 429 71 461 121", p.accent, 2),
        line("M444 116l18 7 1-19", p.accent, 2),
        coin(369, 187, 36),
        text(431, 198, "1,000", p.text, 30),
        text(354, 263, "Restored balance", p.muted, 19, "middle"),
      ),
    "guides--bun--3": () =>
      join(
        avatar(111, 110, 31),
        text(171, 118, "Per user", p.text, 21),
        meter(336, 104, 184, 0.58),
        text(83, 213, "/api", p.accent, 25),
        text(171, 214, "Per endpoint", p.text, 21),
        meter(336, 201, 184, 0.81),
        line("M70 160H530", p.border),
      ),
    "guides--cloudflare-workers--1": () =>
      join(
        key(75, 100, 229, "Unknown key", 121),
        circle(388, 158, 60, p.panel),
        text(388, 176, "?", p.muted, 53, "middle"),
        chip(329, 235, "NOT_FOUND", { width: 180 }),
      ),
    "guides--cloudflare-workers--2": () =>
      join(
        clock(132, 150, 65),
        text(132, 252, "Daily", p.muted, 21, "middle"),
        line("M224 154H328", p.border, 2),
        rect(331, 76, 204, 179, p.panel, 18, true),
        text(433, 129, "Allowance", p.muted, 19, "middle"),
        text(433, 197, "1,000", p.text, 43, "middle"),
      ),
    "guides--cloudflare-workers--3": () =>
      join(
        document(74, 63, 211, 190),
        text(99, 224, "Usage receipt", p.text, 19),
        coin(380, 116, 35),
        coin(431, 173, 35),
        coin(379, 231, 35),
        line("M308 151h30m-14-15v30", p.accent, 2),
      ),
    "guides--express--1": () =>
      join(
        key(75, 115, 265, "sk_live", 119),
        clock(419, 136, 63),
        chip(349, 233, "EXPIRED", { width: 167 }),
        line("M420 54V43M487 136h12", p.border, 2),
      ),
    "guides--express--2": () =>
      join(
        rect(60, 68, 216, 185, p.panel, 19, true),
        coin(168, 132, 33),
        text(168, 219, "Total credits", p.text, 20, "middle"),
        rect(322, 68, 216, 185, p.panel, 19, true),
        clock(430, 132, 36),
        text(430, 219, "Per window", p.text, 20, "middle"),
      ),
    "guides--express--3": () =>
      join(
        key(60, 89, 221, "meta.plan", 146),
        line("M281 162H347V79H381M347 162H381M347 162V245H381", p.border, 2),
        chip(381, 50, "Free", { width: 149 }),
        chip(381, 135, "Pro", { width: 149, accent: true }),
        chip(381, 220, "Enterprise", { width: 149 }),
      ),
    "guides--go--1": () =>
      join(
        rect(76, 70, 448, 180, p.panel, 20, true),
        text(109, 115, "documents.read", p.text, 23),
        check(467, 107, 20),
        text(300, 156, "AND", p.accent, 19, "middle"),
        brackets(101, 174, 399, 50),
        text(126, 205, "billing.read OR billing.write", p.muted, 19),
      ),
    "guides--go--2": () =>
      join(
        rect(86, 80, 250, 165, p.panel, 18, true),
        text(111, 122, "Balance", p.muted, 19),
        text(111, 203, "5", p.text, 58),
        coin(435, 128, 40, false),
        text(435, 220, "Cost 10", p.text, 23, "middle"),
        line("M348 158h30m-15-12v24", p.warning, 2),
        chip(85, 243, "Not deducted", { width: 190 }),
      ),
    "guides--go--3": () =>
      join(
        brand("go", 90, 102, 111),
        rect(276, 50, 247, 214, p.panel, 20, true),
        text(306, 102, "net/http", p.text, 24),
        text(306, 168, "Gin", p.text, 24),
        brand("echo", 304, 200, 30),
        text(352, 224, "Echo", p.text, 24),
        line("M233 87v150M233 162h30", p.border, 2),
      ),
    "guides--hono--1": () =>
      join(
        gauge(163, 163, 70, "At limit", 1),
        line("M269 161h68", p.border, 2),
        rect(303, 133, 15, 56, p.warning, 5),
        key(369, 101, 166, "Credits", 118),
        text(452, 261, "Unchanged", p.muted, 21, "middle"),
      ),
    "guides--hono--2": () =>
      join(
        rect(87, 73, 279, 170, p.panel, 18, true),
        text(112, 115, "credits.cost", p.muted, 22),
        text(113, 207, "0", p.text, 70),
        circle(439, 128, 40, p.tint),
        check(419, 128, 30),
        text(439, 209, "Verify only", p.accent, 20, "middle"),
      ),
    "guides--hono--3": () =>
      join(
        brand("fastapi", 70, 109, 90),
        rect(218, 70, 316, 188, p.panel, 18, true),
        text(246, 115, "APIKeyHeader", p.muted, 21),
        line("M246 146h241", p.border),
        check(250, 198, 19),
        text(289, 205, "Protected route", p.text, 23),
        arrow(171, 158, 32),
      ),
    "guides--nextjs--1": () =>
      join(
        key(60, 90, 259, "Full API key", 130),
        line("M319 155h71", p.border, 2),
        rect(389, 107, 144, 99, p.panel, 16, true),
        text(461, 150, "SHA-256", p.muted, 19, "middle"),
        dots(425, 178, 6, p.accent),
        text(295, 266, "An exact match", p.text, 21, "middle"),
      ),
    "guides--nextjs--2": () =>
      join(
        coin(117, 116, 38),
        text(117, 197, "1,000", p.muted, 25, "middle"),
        text(225, 151, "+", p.accent, 39, "middle"),
        rect(279, 70, 244, 184, p.panel, 19, true),
        text(401, 115, "Top-up 5,000", p.muted, 21, "middle"),
        text(401, 206, "6,000", p.text, 48, "middle"),
      ),
    "guides--nextjs--3": () =>
      join(
        avatar(142, 103, 34),
        line("M142 143v30H94v32M142 173h50v32", p.border, 2),
        key(62, 206, 84, "App", 60),
        key(161, 206, 84, "CLI", 60),
        line("M298 66v197", p.border),
        chip(357, 80, "/search", { width: 171, accent: true }),
        chip(357, 191, "/export", { width: 171 }),
      ),
    "guides--python--1": () =>
      join(
        key(87, 92, 314, "API key", 143),
        rect(356, 156, 148, 74, p.panel, 37, true),
        circle(396, 193, 23, p.muted),
        text(300, 273, "DISABLED", p.muted, 21, "middle"),
      ),
    "guides--python--2": () =>
      join(
        calendar(80, 77, "Monthly"),
        rect(310, 111, 214, 123, p.panel, 18, true),
        text(417, 155, "Refill amount", p.muted, 19, "middle"),
        text(417, 205, "10,000", p.text, 35, "middle"),
        line("M248 142h40m-9-7 9 7-9 7", p.accent, 2),
      ),
    "guides--python--3": () =>
      join(
        line("M130 121C178 42 401 45 455 125M452 209C395 278 187 279 130 207", p.border, 2),
        coin(121, 165, 40),
        coin(471, 165, 40),
        text(297, 109, "Allocate", p.text, 21, "middle"),
        text(297, 172, "Spend", p.muted, 21, "middle"),
        text(297, 243, "Refill", p.accent, 21, "middle"),
        arrow(266, 197, 62),
      ),
    "keyspaces--overview--1": () =>
      join(
        folder(93, 60, 321, 194, "Key defaults"),
        key(119, 139, 203, "sk_live", 80),
        rect(366, 117, 139, 110, p.panel, 17, true),
        line("M391 149h90M391 179h90M391 209h90", p.border, 3),
        circle(415, 149, 7, p.accent),
        circle(469, 179, 7, p.muted),
        circle(434, 209, 7, p.accent),
      ),
    "keyspaces--overview--2": () =>
      join(
        key(70, 60, 197, "key_01", 80),
        key(70, 172, 197, "key_02", 80),
        arrow(286, 156, 50),
        key(362, 116, 171, "key_03", 90),
        text(447, 255, "Next page", p.muted, 20, "middle"),
      ),
    "portal--overview--1": () =>
      join(
        avatar(114, 157, 42),
        line("M158 157H241M335 157H389", p.border, 2),
        rect(241, 90, 94, 134, p.panel, 15, true),
        text(288, 165, "Code", p.text, 20, "middle"),
        rect(390, 60, 145, 195, p.panel, 20, true),
        brand("unkey", 437, 89, 50),
        badge(373, 193, "Session", true, 165),
      ),
    "portal--overview--2": () =>
      join(
        rect(70, 55, 460, 210, p.panel, 21, true),
        avatar(112, 97, 22),
        text(150, 105, "Your portal", p.text, 23),
        key(95, 146, 182, "Your keys", 90),
        spark(317, 150, 181, 70),
        text(407, 247, "Usage", p.muted, 18, "middle"),
      ),
    "ratelimiting--overview--1": () =>
      join(
        rect(78, 70, 444, 185, p.panel, 20, true),
        [40, 63, 80, 55, 103, 73, 117, 96, 80].map((height, i) =>
          rect(105 + i * 40, 227 - height, 22, height, i < 4 ? p.raised : p.tint, 4),
        ),
        line("M274 93V235", p.accent, 2, "5 6"),
        arrow(294, 90, 174),
        text(155, 54, "Previous", p.muted, 18, "middle"),
        text(414, 54, "Current", p.text, 18, "middle"),
      ),
    "ratelimiting--overview--2": () =>
      join(
        brackets(60, 58, 480, 207),
        ["User", "Endpoint", "Workspace"].map((label, i) =>
          join(
            rect(86 + i * 145, 82, 132, 155, p.panel, 17, true),
            circle(152 + i * 145, 128, 23, p.tint),
            check(140 + i * 145, 128, 19),
            text(152 + i * 145, 201, label, p.text, 18, "middle"),
          ),
        ),
      ),
    "ratelimiting--overview--3": () =>
      join(
        chip(70, 137, "user_123", { width: 173 }),
        line("M243 163H303V94H341M303 163V229H341", p.border, 2),
        rect(341, 63, 190, 70, p.tint, 15, true),
        check(360, 98, 15),
        text(392, 105, "Exact match", p.accent, 19),
        rect(341, 197, 190, 60, p.panel, 15),
        text(436, 234, "Default", p.muted, 20, "middle"),
      ),
    "ratelimiting--overview--4": () =>
      join(
        key(60, 60, 213, "Key limit", 105),
        avatar(445, 111, 42),
        text(445, 185, "Identity limit", p.muted, 20, "middle"),
        line("M166 175V221H384Q445 221 445 200", p.border, 2),
        badge(213, 219, "Both pass", true, 179),
      ),
    "sdks--go--1": () =>
      join(
        rect(70, 70, 459, 180, "#202525", 20, true),
        rect(99, 101, 51, 51, p.raised, 10),
        brand("unkey", 108, 110, 33),
        text(173, 137, "$ unkey api", "#f1f5f3", 27),
        line("M104 191h188", "#56615d", 4),
        line("M310 191h83", "#56615d", 4),
        rect(416, 178, 13, 27, "#86cfbc", 2),
      ),
    "sdks--go--2": () =>
      join(
        brand("go", 68, 70, 85),
        rect(193, 70, 345, 180, p.panel, 19, true),
        chip(219, 90, "POST", { width: 103, accent: true }),
        text(218, 181, "keys.verifyKey", p.text, 25),
        text(218, 222, "valid + code", p.muted, 19),
      ),
    "sdks--overview--1": () =>
      join(
        rect(70, 82, 151, 151, p.panel, 22, true),
        brand("typescript", 105, 116, 80),
        rect(276, 60, 254, 194, p.panel, 20, true),
        text(302, 110, "import { Unkey }", p.text, 22),
        text(302, 152, "from", p.muted, 20),
        text(302, 198, "@unkey/api", p.accent, 25),
        arrow(233, 155, 29),
      ),
    "sdks--overview--2": () =>
      join(
        rect(89, 61, 421, 195, p.panel, 21, true),
        brand("go", 119, 90, 113),
        line("M267 91v129", p.border),
        text(300, 122, "Go SDK", p.text, 28),
        text(300, 167, "v3", p.accent, 23),
        text(300, 214, "Typed calls", p.muted, 19),
      ),
    "sdks--overview--3": () =>
      join(
        brand("python", 76, 111, 90),
        line("M188 156H255V102H315M255 156V225H315", p.border, 2),
        chip(318, 70, "Sync", { width: 192 }),
        chip(318, 199, "Async", { width: 192, accent: true }),
        text(125, 261, "unkey.py", p.text, 21, "middle"),
      ),
    "sdks--python--1": () =>
      join(
        rect(60, 88, 159, 150, p.panel, 20, true),
        brand("python", 103, 118, 75),
        arrow(235, 160, 45),
        key(304, 80, 232, "Python service", 148),
        circle(512, 227, 24, p.tint),
        line("M502 227h20M512 217v20", p.accent, 2),
      ),
    "sdks--python--2": () =>
      join(
        document(80, 69, 272, 190),
        text(106, 222, "Request schema", p.text, 20),
        rect(386, 107, 132, 115, p.panel, 18, true),
        text(452, 149, "{ }", p.accent, 33, "middle"),
        text(452, 192, "Response", p.text, 19, "middle"),
      ),
    "sdks--typescript--api--1": () =>
      join(
        rect(75, 70, 161, 183, p.panel, 22, true),
        brand("hono", 125, 95, 60),
        text(155, 222, "Hono", p.text, 22, "middle"),
        rect(359, 70, 165, 183, p.panel, 22, true),
        brand("nextjs", 390, 106, 103),
        text(441, 222, "Next.js", p.text, 22, "middle"),
        circle(297, 160, 28, p.tint),
        check(283, 160, 22),
      ),
    "sdks--typescript--api--2": () =>
      join(
        rect(71, 79, 458, 168, p.panel, 20, true),
        text(101, 125, "POST", p.accent, 22),
        text(191, 125, "/v2/keys.verifyKey", p.text, 23),
        line("M102 153h396", p.border),
        text(103, 207, "data.valid", p.text, 23),
        text(320, 207, "data.code", p.muted, 23),
      ),
    "sdks--typescript--hono--1": () =>
      join(
        rect(75, 60, 294, 198, p.panel, 20, true),
        [0, 1, 2, 3].map((i) =>
          join(
            circle(104, 90 + i * 43, 5, i === 2 ? p.warning : p.accent),
            line(`M126 ${90 + i * 43}h${i % 2 ? 173 : 205}`, p.border, 5),
          ),
        ),
        circle(456, 126, 49, p.tint),
        check(432, 126, 36),
        text(456, 224, "Recorded", p.text, 21, "middle"),
      ),
    "sdks--typescript--hono--2": () =>
      join(
        rect(76, 109, 152, 119, p.panel, 20, true),
        brand("typescript", 123, 138, 60),
        line("M228 169H295V90H354M295 169V235H354", p.border, 2),
        chip(355, 60, "Middleware", { width: 184, accent: true }),
        chip(355, 210, "Direct client", { width: 184 }),
      ),
    "sdks--typescript--nextjs--1": () =>
      join(
        rect(80, 60, 310, 199, p.panel, 20, true),
        text(107, 112, "HTTP 200", p.muted, 20),
        text(107, 168, "valid: false", p.text, 29),
        text(107, 226, "EXPIRED", p.warning, 23),
        circle(459, 158, 50, p.raised),
        line("M439 138l40 40m0-40-40 40", p.warning, 3),
      ),
    "sdks--typescript--nextjs--2": () =>
      join(
        rect(60, 56, 306, 211, p.panel, 20, true),
        brand("nextjs", 89, 66, 100),
        text(90, 153, "Server route", p.text, 23),
        key(89, 181, 247, "Root key", 60),
        line("M366 158H420", p.border, 2),
        rect(423, 111, 113, 104, p.tint, 17, true),
        text(479, 172, "SDK", p.accent, 29, "middle"),
      ),
  };

  const scene = scenes[id];
  if (!scene) throw new Error(`No API card artwork for ${card.id}`);
  return scene();
}
