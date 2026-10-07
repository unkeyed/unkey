import assert from "node:assert/strict";
import { renderEditorialBuild } from "./art-editorial.mjs";

// Landing-page compositions have a fixed optical center and their own content;
// detail-page motifs are not suitable substitutes for an overview of a feature.
export function renderComputeOverview(card, g) {
  const { p, rect, line, circle, text, check, cube, brand } = g;
  const panel = (x, y, w, h, fill = p.panel) => rect(x, y, w, h, fill, 16, true);
  const label = (x, y, value, color = p.text, size = 20, anchor = "start") =>
    text(x, y, value, color, size, anchor);
  const pill = (x, y, w, value, active = false) =>
    rect(x, y, w, 34, active ? p.tint : p.raised, 17) +
    label(x + w / 2, y + 23, value, active ? p.accent : p.muted, 18, "middle");
  const status = (x, y) => circle(x, y, 14, p.tint) + check(x - 7, y - 7, 14);
  const lock = (x, y, ink = p.accent) =>
    line(`M${x + 6} ${y + 17}v-7a10 10 0 0 1 20 0v7`, ink, 2) +
    `<rect x="${x}" y="${y + 17}" width="32" height="28" rx="6" fill="none" stroke="${ink}" stroke-width="2"/>` +
    circle(x + 16, y + 29, 2.5, ink) +
    line(`M${x + 16} ${y + 31}v5`, ink, 2);
  const masked = (x, y, color = p.muted) =>
    Array.from({ length: 8 }, (_, i) => circle(x + i * 13, y, 3, color)).join("");
  const imageBox = (x, y, size) =>
    cube(x, y, size) +
    [0.5, 0.66]
      .map((f) =>
        line(`M${x} ${y + size * f}l${size / 2} ${size / 4} ${size / 2} -${size / 4}`, p.border),
      )
      .join("");

  const scenes = {
    "compute--index--1": () =>
      line("M202 160H280", p.accent, 2) +
      panel(72, 88, 130, 144) +
      brand("github", 111, 108, 52) +
      label(137, 206, "main", p.muted, 20, "middle") +
      panel(280, 64, 248, 192) +
      label(304, 103, "Production", p.muted) +
      cube(304, 130, 65) +
      label(388, 159, "api-service", p.text, 20) +
      circle(397, 188, 4) +
      label(410, 194, "Ready", p.accent, 18) +
      line("M304 219h200", p.border) +
      label(304, 244, "Push to deploy", p.muted, 18),

    "compute--index--2": () =>
      line("M208 158H320", p.border, 2) +
      imageBox(88, 91, 120) +
      pill(80, 225, 136, "api:v1") +
      panel(320, 64, 208, 192) +
      label(344, 103, "Your app", p.muted) +
      [0, 1, 2]
        .map((i) => rect(344, 129 + i * 27, 160, 18, p.raised, 5) + circle(357, 138 + i * 27, 3))
        .join("") +
      status(358, 229) +
      label(382, 236, "Running", p.accent),

    "compute--index--3": () =>
      ["Starter", "Pro", "Business"]
        .map((name, i) => {
          const x = 64 + i * 164;
          return (
            panel(x, 68, 144, 184, i === 1 ? p.tint : p.panel) +
            label(x + 72, 108, name, i === 1 ? p.accent : p.text, 21, "middle") +
            [0, 1, 2]
              .map((j) =>
                rect(
                  x + 29 + j * 30,
                  212 - (35 + i * 21),
                  20,
                  35 + i * 21,
                  j <= i ? p.accent : p.border,
                  4,
                ),
              )
              .join("")
          );
        })
        .join(""),

    "compute--index--4": () =>
      line("M300 99v23M300 176v18H170v18M300 194h130v18", p.border, 2) +
      panel(174, 49, 252, 50) +
      label(300, 81, "Project", p.muted, 20, "middle") +
      panel(198, 122, 204, 54, p.tint) +
      label(300, 156, "App", p.accent, 21, "middle") +
      panel(72, 212, 196, 56) +
      label(170, 247, "Production", p.text, 20, "middle") +
      panel(332, 212, 196, 56) +
      label(430, 247, "Preview", p.text, 20, "middle"),

    "compute--index--5": () =>
      panel(64, 68, 472, 184) +
      label(88, 109, "Deployment lifecycle", p.muted) +
      line("M125 160H475", p.border, 2) +
      ["Pending", "Building", "Deploying", "Ready"]
        .map((name, i) => {
          const x = 125 + i * 116.6667;
          return (
            circle(x, 160, 17, i === 3 ? p.tint : p.raised) +
            (i === 3 ? check(x - 8, 152, 16) : circle(x, 160, 4, p.muted)) +
            label(x, 215, name, i === 3 ? p.accent : p.muted, 18, "middle")
          );
        })
        .join(""),

    "compute--index--6": () =>
      ["Production", "Preview"]
        .map((name, i) => {
          const x = 72 + i * 240;
          return (
            panel(x, 68, 216, 184) +
            label(x + 24, 106, name, p.text, 22) +
            cube(x + 24, 129, 54) +
            label(x + 99, 156, i === 0 ? "main" : "Pull request", p.muted, 18) +
            pill(x + 24, 199, 168, i === 0 ? "Live traffic" : "Isolated preview", i === 0)
          );
        })
        .join(""),

    "compute--index--7": () =>
      panel(72, 60, 456, 200) +
      label(96, 100, "Autoscaling", p.text, 22) +
      [0, 1, 2]
        .map((i) => {
          const x = 96 + i * 145;
          return (
            rect(x, 128, 118, 66, i === 0 ? p.tint : p.raised, 10) +
            circle(x + 19, 150, 4, i === 0 ? p.accent : p.muted) +
            line(`M${x + 35} 150h62M${x + 19} 173h78`, p.border, 2)
          );
        })
        .join("") +
      label(96, 235, "Min 1", p.muted, 19) +
      label(504, 235, "Max 3", p.muted, 19, "end"),

    "compute--index--8": () =>
      `<circle cx="300" cy="160" r="87" fill="${p.panel}" stroke="${p.border}" stroke-width="1.5"/><ellipse cx="300" cy="160" rx="39" ry="87" fill="none" stroke="${p.border}" stroke-width="1.2"/>` +
      line("M213 160H387M225 116H375M225 204H375", p.border, 1.2) +
      line("M216 107h29q18 0 18 18v10M337 186v11q0 16 20 16h27", p.accent, 2) +
      panel(56, 82, 160, 50) +
      label(136, 114, "us-east-1", p.text, 18, "middle") +
      panel(384, 188, 160, 50) +
      label(464, 220, "eu-central-1", p.text, 18, "middle") +
      circle(263, 139, 12, p.tint) +
      circle(263, 139, 4) +
      circle(337, 182, 12, p.tint) +
      circle(337, 182, 4),

    "compute--index--9": () =>
      panel(80, 60, 440, 200) +
      label(104, 101, "Build output", p.text, 22) +
      line("M104 119h392", p.border) +
      ["Install dependencies", "Build application", "Publish image"]
        .map((value, i) => label(104, 151 + i * 42, value, p.muted, 20) + status(480, 144 + i * 42))
        .join(""),

    "compute--index--10": () =>
      line("M300 160h66", p.border, 2) +
      panel(64, 60, 236, 200) +
      brand("docker", 88, 84, 36) +
      label(142, 111, "Dockerfile", p.text, 22) +
      label(88, 158, "FROM", p.accent, 19) +
      line("M160 152h110", p.border, 4) +
      label(88, 196, "COPY", p.accent, 19) +
      line("M160 190h87", p.border, 4) +
      label(88, 234, "RUN", p.accent, 19) +
      line("M160 228h101", p.border, 4) +
      imageBox(366, 95, 120) +
      label(426, 253, "Built image", p.muted, 19, "middle"),

    "compute--index--11": () =>
      line("M276 160h94", p.accent, 2) +
      panel(72, 72, 204, 176, "#202927") +
      lock(96, 99, "#b3ded2") +
      label(142, 122, "Build secret", "#edf4f1", 20) +
      masked(100, 177, "#c9ded7") +
      label(96, 225, "Temporary mount", "#c9ded7", 18) +
      imageBox(370, 92, 118) +
      label(429, 249, "Secret excluded", p.accent, 20, "middle"),

    "compute--index--12": () =>
      line("M186 160h50q22 0 22-22v-28h50M236 160q22 0 22 22v6q0 22 22 22h28", p.border, 2) +
      panel(72, 90, 114, 140) +
      brand("github", 103, 110, 52) +
      label(129, 205, "GitHub", p.muted, 19, "middle") +
      panel(308, 76, 220, 68) +
      circle(332, 110, 4) +
      label(350, 117, "Production", p.text, 22) +
      panel(308, 176, 220, 68) +
      circle(332, 210, 4, p.muted) +
      label(350, 217, "Preview", p.text, 22),

    "compute--index--13": () => renderEditorialBuild(g),

    "compute--index--14": () =>
      panel(80, 56, 208, 140) +
      label(104, 91, "CPU", p.muted, 19) +
      label(104, 136, "0.25", p.text, 36) +
      label(104, 172, "vCPU", p.muted, 18) +
      panel(312, 56, 208, 140) +
      label(336, 91, "Memory", p.muted, 19) +
      label(336, 136, "256", p.text, 36) +
      label(336, 172, "MiB", p.muted, 18) +
      panel(80, 216, 440, 48) +
      label(104, 247, "PORT", p.muted, 18) +
      label(496, 247, "8080", p.accent, 22, "end"),

    "compute--index--15": () =>
      panel(80, 60, 440, 200) +
      label(104, 102, "GET /healthz", p.text, 22) +
      pill(390, 78, 106, "200 OK", true) +
      line("M104 166h58l16-18 18 36 22-54 23 36h255", p.accent, 2.5) +
      line("M104 207h392", p.border) +
      status(118, 234) +
      label(143, 241, "Ready for traffic", p.muted, 19),

    "compute--index--16": () =>
      panel(80, 60, 440, 200) +
      label(104, 103, "Production", p.text, 22) +
      pill(374, 78, 122, "Encrypted", true) +
      line("M104 122h392", p.border) +
      label(104, 161, "API_URL", p.muted, 20) +
      masked(394, 154) +
      line("M104 184h392", p.border) +
      label(104, 226, "API_SECRET", p.muted, 20) +
      masked(394, 219),

    "compute--index--17": () =>
      panel(72, 60, 456, 200) +
      label(96, 103, "Workspace limits", p.text, 22) +
      rect(96, 126, 408, 110, p.raised, 12) +
      label(116, 155, "Per instance", p.muted, 18) +
      ["CPU", "Memory", "Disk"].map((value, i) => pill(116 + i * 125, 179, 118, value)).join(""),

    "compute--index--18": () =>
      panel(72, 60, 456, 200) +
      label(96, 103, "Monthly budget", p.text, 22) +
      label(504, 103, "75%", p.accent, 22, "end") +
      rect(96, 135, 408, 12, p.raised, 6) +
      `<rect x="96" y="135" width="306" height="12" rx="6" fill="${p.accent}"/>` +
      [0, 0.5, 0.75, 1].map((f) => line(`M${96 + 408 * f} 157v9`, p.border)).join("") +
      label(96, 195, "0%", p.muted, 18) +
      label(300, 195, "50%", p.muted, 18, "middle") +
      label(402, 195, "75%", p.muted, 18, "middle") +
      label(504, 195, "100%", p.muted, 18, "end") +
      label(96, 237, "Optional stop at 100%", p.muted, 18),

    "compute--index--19": () =>
      line("M138 183v20h324v-20M300 183v45", p.border, 2) +
      ["TypeScript", "Go", "Python"]
        .map((name, i) => {
          const x = 72 + i * 162;
          return (
            panel(x, 56, 132, 127) +
            brand(
              ["typescript", "go", "python"][i],
              x + (i === 1 ? 19 : 39),
              i === 1 ? 58 : 77,
              i === 1 ? 94 : 54,
            ) +
            label(x + 66, 160, name, p.muted, 18, "middle")
          );
        })
        .join("") +
      panel(216, 228, 168, 44, p.tint) +
      label(300, 257, "Unkey API", p.accent, 20, "middle"),
  };
  const render = scenes[card.id];
  assert.ok(render, `Missing Compute overview composition: ${card.id}`);
  return render();
}
