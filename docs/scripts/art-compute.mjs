import { renderComputeOverview } from "./art-compute-overview.mjs";

// Each occurrence has its own editorial brief, including links repeated on other pages.
const briefs = {
  "build--build-secrets--1": ["environment", "Build variables", "Encrypted", 0],
  "build--build-secrets--2": ["docker", "RUN --mount", "Dockerfile", 0],
  "build--dockerfile--1": ["secret", "Temporary mount", "RUN step", 1],
  "build--dockerfile--2": ["runtime", "Container", "8080", 0],
  "build--github--1": ["settings", "Watch paths", "/apps/api", 0],
  "build--github--2": ["branch", "main", "Pull request", 0],
  "build--overview--1": ["docker", "Image layers", "FROM → COPY", 1],
  "build--overview--2": ["secret", "Build step", "Clean image", 2],
  "build--overview--3": ["settings", "Build context", "./services/api", 1],
  "build--overview--4": ["release", "Image ready", "Deploy", 0],
  "cli--analytics--get-gateway-requests--1": ["query", "Gateway requests", "SELECT", 0],
  "cli--analytics--get-runtime-logs--1": ["logs", "Runtime output", "WHERE", 0],
  "cli--apps--create-app--1": ["project", "payments", "New app", 0],
  "cli--apps--create-app--2": ["github", "Connect repository", "api", 1],
  "cli--deploy--1": ["image", "Registry image", "Pull", 0],
  "cli--deploy--2": ["release", "Deployment status", "Ready", 1],
  "cli--deploy--3": ["credential", "--root-key", "CLI authentication", 0],
  "cli--deployments--create-deployment--1": ["release", "Pull image", "Starting", 2],
  "cli--deployments--create-deployment--2": ["terminal", "unkey deploy", "api:v1", 0],
  "cli--deployments--promote-deployment--1": ["promotion", "Candidate", "Production", 0],
  "cli--deployments--rollback-deployment--1": ["promotion", "Previous release", "Restored", 1],
  "cli--deployments--start-deployment--1": ["power", "Start deployment", "Running", 0],
  "cli--deployments--stop-deployment--1": ["power", "Stop deployment", "Stopped", 1],
  "cli--domains--create-domain--1": ["domain", "api.acme.com", "Add domain", 0],
  "cli--domains--delete-domain--1": ["domain", "Reconnect domain", "DNS", 1],
  "cli--domains--get-domain--1": ["certificate", "Domain status", "HTTPS", 0],
  "cli--domains--list-domains--1": ["domain", "Custom hostnames", "Domains", 2],
  "cli--domains--verify-domain--1": ["certificate", "Ownership verified", "TXT", 1],
  "cli--environments--list-environment-variables--1": [
    "environment",
    "Stored secrets",
    "••••••••",
    1,
  ],
  "cli--environments--list-environments--1": ["environments", "Production", "Preview", 0],
  "cli--environments--remove-environment-variables--1": [
    "environment",
    "Remove variable",
    "Next deployment",
    2,
  ],
  "cli--environments--set-environment-variables--1": ["environment", "Set variable", "API_URL", 3],
  "cli--environments--update-settings--1": ["resources", "CPU / memory", "Per instance", 0],
  "cli--environments--update-settings--2": ["settings", "Dockerfile path", "./Dockerfile", 2],
  "cli--environments--update-settings--3": ["health", "Readiness probe", "GET /healthz", 1],
  "cli--gateway--list-policies--1": ["policy", "Policy snapshot", "Enabled", 0],
  "cli--gateway--set-policies--1": ["match", "Path + method", "All conditions", 0],
  "cli--gateway--update-policy--1": ["policy", "Evaluation order", "First match", 1],
  "cli--github--install-app--1": ["github", "Repository access", "Authorized", 2],
  "configure--build-settings--1": ["build", "Source revision", "Container image", 0],
  "configure--build-settings--2": ["runtime", "Start command", "node server.js", 1],
  "configure--environment-variables--1": ["secret", "BuildKit secret", "Memory only", 3],
  "configure--environment-variables--2": ["runtime", "Runtime config", "PORT", 2],
  "configure--health-checks--1": ["scale", "Healthy replicas", "Keep serving", 1],
  "configure--health-checks--2": ["health", "Waiting for readiness", "Traffic held", 2],
  "configure--limits--1": ["limits", "Workspace", "Capacity", 0],
  "configure--limits--2": ["budget", "Monthly budget", "Alert thresholds", 0],
  "configure--runtime-settings--1": ["health", "GET /healthz", "200 OK", 3],
  "configure--runtime-settings--2": ["scale", "Replica bounds", "Min / max", 2],
  "configure--runtime-settings--3": ["limits", "Container limits", "CPU + memory", 1],
  "configure--runtime-settings--4": ["regions", "Near your users", "Running regions", 0],
  "configure--spend-budget--1": ["error", "402", "Spend limit reached", 0],
  "configure--spend-budget--2": ["plans", "Compute plans", "Choose capacity", 0],
  "configure--spend-budget--3": ["budget", "This month", "Workspace usage", 1],
  "gateway--api-key-auth--1": ["credential", "Create a key", "Gateway credential", 1],
  "gateway--api-key-auth--2": ["principal", "Verified identity", "X-Unkey-Principal", 0],
  "gateway--api-key-auth--3": ["error", "429", "Credits exhausted", 1],
  "gateway--errors--1": ["auth", "Permission check", "403", 0],
  "gateway--errors--2": ["catalog", "API errors", "Gateway errors", 0],
  "gateway--errors--3": ["policy", "Replace policies", "New snapshot", 2],
  "gateway--firewall--1": ["policy", "Update one policy", "Next deployment", 3],
  "gateway--firewall--2": ["rate", "Request allowance", "429 on excess", 0],
  "gateway--local-development--1": ["principal", "Local fixture", "Test identity", 1],
  "gateway--local-development--2": ["environments", "Local testing", "Preview", 1],
  "gateway--logging--1": ["schema", "OpenAPI", "Validate request", 0],
  "gateway--logging--2": ["policy", "Policy changes", "Redeploy to apply", 4],
  "gateway--openapi-validation--1": ["redact", "Authorization", "[REDACTED]", 0],
  "gateway--openapi-validation--2": ["catalog", "400 · request", "422 · configuration", 1],
  "gateway--overview--1": ["credential", "Issue API key", "Keyspace", 2],
  "gateway--overview--2": ["policy", "Ordered policies", "Request", 5],
  "gateway--overview--3": ["auth", "Accepted keyspaces", "Valid key", 1],
  "gateway--overview--4": ["principal", "Verified principal", "To your app", 2],
  "gateway--overview--5": ["error", "503", "No running instances", 2],
  "gateway--policies--1": ["auth", "Verified identity", "Forward to app", 2],
  "gateway--policies--2": ["rate", "Per identifier", "Separate buckets", 1],
  "gateway--policies--3": ["firewall", "/internal/*", "403 · denied", 0],
  "gateway--policies--4": ["redact", "Request capture", "Choose fields", 1],
  "gateway--principal--1": ["terminal", "curl localhost", "Test principal", 1],
  "gateway--principal--2": ["auth", "Key authentication", "Enabled", 3],
  "gateway--rate-limiting--1": ["auth", "Verify key", "Continue", 4],
  "gateway--rate-limiting--2": ["limits", "Policy bounds", "Validate before save", 2],
  "get-started--deploy-an-image--1": ["promotion", "Test image", "Promote when ready", 2],
  "get-started--deploy-an-image--2": ["github", "Push to deploy", "Build from source", 3],
  "get-started--deploy-an-image--3": ["scale", "One image", "Multiple replicas", 3],
  "get-started--deploy-an-image--4": ["environment", "Image configuration", "API_URL", 4],
  "get-started--deploy-an-image--5": ["auth", "Add authentication", "Gateway policy", 5],
  "get-started--deploy-from-github--1": ["branch", "main → production", "PR → preview", 1],
  "get-started--deploy-from-github--2": ["release", "First deployment", "Build → ready", 3],
  "get-started--deploy-from-github--3": ["resources", "Size your app", "CPU + memory", 1],
  "get-started--deploy-from-github--4": ["github-events", "Pull request", "Fork approval", 0],
  "get-started--deploy-from-github--5": ["auth", "Verify before proxy", "Trusted principal", 6],
  "get-started--plans--1": ["github", "Your first deploy", "Ready", 4],
  "get-started--plans--2": ["limits", "Plan allowances", "Workspace", 3],
  "observe--analytics-api--1": ["query", "Compute analytics", "Read-only SQL", 1],
  "observe--overview--1": ["logs", "GET /orders", "200", 1],
  "observe--overview--2": ["logs", "stdout / stderr", "Runtime logs", 2],
  "observe--overview--3": ["metrics", "Traffic + resources", "Live signals", 0],
  "observe--overview--4": ["build", "Build output", "Step complete", 2],
};

// Labels are kept outside plots and connectors so the card survives narrow layouts.
export function renderCompute(card, g) {
  if (card.source === "compute/index.mdx") return renderComputeOverview(card, g);
  const brief = briefs[card.id.replace(/^compute--/, "")];
  if (!brief) throw new Error(`Missing Compute art brief: ${card.id}`);
  const [motif, label, detail, mode] = brief;
  const { p, rect, line, circle, text, check, arrow, cube, chip, brand } = g;
  const raised = (x, y, w, h, r = 18) => rect(x, y, w, h, p.raised, r, true);
  const dots = (x, y, n = 7, color = p.muted) =>
    Array.from({ length: n }, (_, i) => circle(x + i * 15, y, 3.5, color)).join("");
  const bars = (x, y, widths, color = p.border) =>
    widths.map((w, i) => rect(x, y + i * 19, w, 5, color, 2)).join("");
  const labelPair = (a = label, b = detail) =>
    text(300, 263, a, p.text, 21, "middle") + text(300, 289, b, p.muted, 18, "middle");
  const lock = (x, y, s = 1, color = p.accent) =>
    `<g transform="translate(${x} ${y}) scale(${s})">${line("M8 22V12a14 14 0 0 1 28 0v10", color, 2.2)}${rect(0, 22, 44, 36, p.panel, 9)}${circle(22, 37, 3, color)}${line("M22 40v7", color, 2)}</g>`;
  const server = (x, y, n = 1, w = 78) =>
    Array.from({ length: n }, (_, i) => {
      const py = y - i * 23;
      return (
        raised(x, py, w, 33, 9) +
        circle(x + 14, py + 16, 3, p.accent) +
        line(`M${x + 30} ${py + 16}h${w - 45}`, p.border, 2)
      );
    }).join("");
  const key = (x, y, s = 1) =>
    `<g transform="translate(${x} ${y}) scale(${s})">${line("M29 24h47l10 10-10 10-9-9-8 8-8-8H29", p.accent, 3)}${line("M30 24a18 18 0 1 0 0 11", p.accent, 3)}${circle(10, 29, 3, p.accent)}</g>`;
  const chart = (x, y, w = 170, h = 90, trend = 0) => {
    const shapes = [
      `M${x} ${y + h - 8}C${x + 35} ${y + h - 8} ${x + 30} ${y + h / 2} ${x + 65} ${y + h / 2}S${x + 115} ${y + h / 2 + 10} ${x + w} ${y + 9}`,
      `M${x} ${y + h - 15}L${x + 35} ${y + h - 35}L${x + 70} ${y + h - 28}L${x + 100} ${y + 18}L${x + 130} ${y + 35}L${x + w} ${y + 8}`,
      `M${x} ${y + h / 2}C${x + 25} ${y + 5} ${x + 50} ${y + h - 5} ${x + 80} ${y + h / 2}S${x + 135} ${y + 5} ${x + w} ${y + 12}`,
    ];
    return (
      line(`M${x} ${y + h}h${w}`, p.border) +
      line(`M${x} ${y + h / 2}h${w}`, p.border, 1, "3 7") +
      line(shapes[trend % 3], p.accent, 3)
    );
  };
  const badge = (x, y, value, positive = true) =>
    chip(x, y, value, { width: Math.max(94, value.length * 10 + 38), accent: positive });
  const branch = (x, y) =>
    line(`M${x} ${y + 60}v-90q0-20 20-20h55M${x} ${y - 5}q0-20 20-20h55`, p.accent, 2.5) +
    circle(x, y + 60, 5) +
    circle(x + 75, y - 50, 5) +
    circle(x + 75, y - 25, 5);
  const shield = (x, y, ok = true) =>
    line(`M${x} ${y}q36 17 64 0v48q-2 32-32 49q-30-17-32-49Z`, p.accent, 2.4) +
    (ok ? check(x + 20, y + 34, 25) : line(`M${x + 24} ${y + 29}l17 25m0-25-17 25`, p.warning, 3));
  const renderers = {
    github: () => {
      const tileX = mode === 4 ? 96 : 74;
      const tileY = mode === 2 ? 75 : 100;
      const source =
        raised(tileX, tileY, 112, 112, 24) + brand("github", tileX + 27, tileY + 27, 58);
      if (mode === 2)
        return (
          source +
          line("M186 131H270", p.accent, 2, "5 7") +
          shield(292, 93) +
          badge(382, 126, detail) +
          text(300, 264, label, p.text, 22, "middle")
        );
      if (mode === 1)
        return (
          source +
          line("M186 156H257", p.accent, 2, "5 7") +
          raised(270, 73, 237, 159) +
          brand("unkey", 296, 100, 34) +
          text(296, 192, detail, p.text, 25) +
          badge(389, 96, "Linked") +
          text(300, 276, label, p.text, 22, "middle")
        );
      if (mode === 3)
        return (
          source +
          line("M186 156H242q20 0 20-20v-24h42", p.accent, 2, "5 7") +
          raised(315, 61, 184, 83) +
          bars(340, 84, [100, 73, 90]) +
          cube(337, 174, 66) +
          text(440, 213, "Build", p.muted, 18) +
          text(300, 279, label, p.text, 22, "middle")
        );
      return (
        source +
        line(`M${tileX + 112} 156H292`, p.accent, 2, "5 7") +
        raised(298, 74, 225, 159) +
        cube(322, 96, 56) +
        text(322, 196, label, p.text, 21) +
        badge(393, 110, detail) +
        badge(103, 231, mode === 4 ? "First commit" : "main", false)
      );
    },
    "github-events": () => {
      if (mode === 0)
        return (
          raised(68, 91, 104, 104, 23) +
          brand("github", 92, 115, 56) +
          line("M172 143H237q25 0 25-25V91h56M262 143v64h56", p.border, 2) +
          raised(326, 61, 208, 76) +
          text(347, 106, label, p.text, 20) +
          raised(326, 174, 208, 76) +
          lock(347, 181, 0.7) +
          text(395, 214, "Approve", p.accent, 20) +
          text(300, 285, detail, p.muted, 18, "middle")
        );
      return (
        raised(64, 105, 104, 104, 24) +
        brand("github", 88, 129, 56) +
        line("M168 157h56q25 0 25-25V98h58M249 157v61h58", p.accent, 2) +
        raised(318, 57, 218, 90) +
        circle(342, 86, 4) +
        text(342, 123, label, p.text, 22) +
        raised(318, 177, 218, 90) +
        line("M340 203h32", p.accent, 2, "3 5") +
        text(342, 245, detail, p.text, 22)
      );
    },
    secret: () => {
      const credentialSurface = (x, y, width, height) =>
        rect(x, y, width, height, "#202927", 18, true);
      const imageLayers = (x, y, size) =>
        cube(x, y + 18, size) + cube(x, y + 9, size) + cube(x, y, size);
      if (mode === 1)
        return (
          credentialSurface(85, 65, 210, 169) +
          text(111, 103, "RUN", "#dce8e3", 20) +
          lock(112, 127, 0.9) +
          dots(179, 162, 5, "#dce8e3") +
          line("M295 150h48", p.accent, 2, "4 7") +
          imageLayers(360, 95, 108) +
          text(405, 249, label, p.text, 20, "middle")
        );
      if (mode === 2)
        return (
          credentialSurface(75, 65, 177, 170) +
          lock(139, 88, 1.2) +
          dots(117, 195, 6, "#dce8e3") +
          line("M252 150h82", p.accent, 2, "5 8") +
          imageLayers(366, 121, 96) +
          check(450, 99, 23) +
          text(300, 280, detail, p.text, 22, "middle")
        );
      if (mode === 3)
        return (
          raised(108, 64, 380, 177) +
          text(137, 104, label, p.text, 22) +
          credentialSurface(139, 126, 220, 59) +
          lock(156, 125, 0.65) +
          dots(215, 156, 7, "#dce8e3") +
          badge(346, 195, "Memory") +
          line("M151 204h137", p.border, 2, "4 7")
        );
      return (
        credentialSurface(68, 82, 201, 151) +
        lock(92, 102, 0.85) +
        text(153, 122, "Secret", "#dce8e3", 19) +
        dots(110, 192, 8, "#dce8e3") +
        line("M269 157h55", p.accent, 2, "5 7") +
        imageLayers(354, 89, 106) +
        badge(336, 230, "Clean image") +
        text(69, 266, "Temporary mount", p.muted, 18)
      );
    },
    health: () => {
      const left = mode === 2 ? 108 : 87,
        width = mode === 2 ? 324 : 390;
      let art =
        raised(left, 57, width, 166, 20) +
        text(left + 25, 95, label, p.text, 20) +
        line(
          `M${left + 25} 155h48l15-22 23 52 23-75 24 45h${width - 183}`,
          mode === 2 ? p.muted : p.accent,
          3,
        );
      if (mode === 1)
        art +=
          badge(332, 238, "Scheduled") +
          circle(115, 256, 17, p.tint) +
          line("M115 246v11l8 5", p.accent, 2);
      else if (mode === 2)
        art +=
          raised(412, 112, 97, 113) +
          line("M444 145v40m23-40v40", p.warning, 5) +
          text(300, 278, detail, p.muted, 19, "middle");
      else if (mode === 3)
        art +=
          badge(99, 240, detail) +
          line("M330 256h68", p.accent, 2) +
          arrow(398, 256, 30) +
          circle(466, 256, 19, p.tint) +
          check(455, 247, 21);
      else
        art +=
          badge(110, 240, detail) +
          line("M366 250a20 20 0 1 1 0 20", p.muted, 2) +
          line("M360 246l8 4-6 7", p.muted, 2) +
          text(410, 266, "Repeat", p.muted, 18);
      return art;
    },
    scale: () => {
      if (mode === 1)
        return (
          server(83, 185, 2, 115) +
          server(242, 185, 2, 115) +
          server(401, 185, 2, 115) +
          circle(139, 97, 20, p.tint) +
          check(128, 88, 22) +
          circle(300, 97, 20, p.tint) +
          check(289, 88, 22) +
          line("M445 81l27 28m0-28-27 28", p.muted, 2) +
          labelPair()
        );
      if (mode === 2)
        return (
          line("M89 83v135h415V83", p.border, 2, "4 7") +
          server(118, 163, 1) +
          server(249, 163, 2) +
          server(380, 163, 3) +
          text(89, 64, "Min", p.muted, 18) +
          text(504, 64, "Max", p.muted, 18, "end") +
          labelPair()
        );
      if (mode === 3)
        return (
          cube(80, 109, 96) +
          line("M190 150h56q18 0 18-18V96h75M264 150v57h75", p.accent, 2) +
          server(355, 89, 1, 126) +
          server(355, 182, 1, 126) +
          text(129, 250, label, p.text, 20, "middle") +
          text(419, 251, detail, p.muted, 19, "middle")
        );
      return (
        raised(57, 85, 190, 145) +
        chart(80, 119, 140, 77, 1) +
        arrow(259, 165, 35) +
        server(314, 194, 1, 62) +
        server(390, 194, 2, 62) +
        server(466, 194, 3, 62) +
        line("M312 232v8h216v-8", p.border, 1.6) +
        text(80, 112, "Demand", p.muted, 18) +
        text(420, 275, detail, p.muted, 18, "middle")
      );
    },
    docker: () => {
      if (mode === 0)
        return (
          raised(82, 55, 261, 193) +
          brand("docker", 108, 77, 60) +
          text(111, 183, label, p.text, 23) +
          bars(112, 209, [138]) +
          line("M343 153h42", p.accent, 2, "4 7") +
          cube(404, 122, 70) +
          text(443, 241, "Image", p.muted, 19, "middle")
        );
      if (mode === 1)
        return (
          brand("docker", 91, 105, 94) +
          line("M204 151h52", p.accent, 2) +
          server(296, 190, 4, 188) +
          text(390, 265, label, p.text, 21, "middle")
        );
      return (
        raised(77, 68, 204, 179) +
        brand("docker", 104, 90, 62) +
        text(105, 197, label, p.text, 24) +
        arrow(295, 157, 41) +
        cube(368, 96, 108) +
        text(421, 260, "Image", p.muted, 20, "middle")
      );
    },
    image: () =>
      mode === 0
        ? raised(65, 83, 168, 152) +
          brand("docker", 110, 106, 76) +
          text(149, 212, "Registry", p.muted, 20, "middle") +
          line("M233 159h95", p.accent, 2, "5 7") +
          cube(370, 104, 102) +
          badge(355, 240, detail)
        : cube(80, 83, 126) +
          line("M222 157h63", p.accent, 2, "5 7") +
          raised(316, 87, 217, 153) +
          brand("unkey", 340, 112, 37) +
          server(402, 154, 2, 96) +
          text(338, 216, "Running", p.text, 21),
    runtime: () => {
      const art =
        raised(187, 63, 226, 181) +
        rect(248, 113, 104, 79, p.tint, 16) +
        brand("unkey", 279, 132, 40);
      const pins = [0, 1, 2, 3]
        .map((i) =>
          line(
            `M${263 + i * 23} 96v17M${263 + i * 23} 192v17M232 ${128 + i * 17}h16M352 ${128 + i * 17}h16`,
            p.border,
            2,
          ),
        )
        .join("");
      if (mode === 0)
        return (
          art +
          pins +
          badge(390, 91, detail) +
          badge(73, 182, "HTTP", false) +
          labelPair(label, "Container resources")
        );
      if (mode === 1)
        return art + pins + raised(102, 246, 390, 50) + text(126, 280, detail, p.text, 22);
      if (mode === 2)
        return (
          art +
          pins +
          badge(72, 79, "PORT", false) +
          badge(330, 254, "Environment") +
          text(90, 280, "Runtime config", p.text, 21)
        );
      return (
        art +
        pins +
        line("M105 150h77M418 150h79", p.accent, 2, "5 7") +
        circle(95, 150, 7) +
        circle(507, 150, 7) +
        text(300, 280, detail, p.text, 23, "middle")
      );
    },
    settings: () => {
      const positions = [
        [145, 282, 372],
        [252, 173, 395],
        [379, 270, 158],
        [190, 384, 273],
      ][mode];
      const controls = positions
        .map(
          (x, i) =>
            line(`M117 ${104 + i * 48}H479`, p.border, 3) +
            circle(x, 104 + i * 48, 13, p.panel) +
            circle(x, 104 + i * 48, 5, p.accent),
        )
        .join("");
      return (
        raised(80, 65, 440, 177) +
        controls +
        text(300, 279, label, p.text, 22, "middle") +
        (mode === 2 ? brand("docker", 443, 52, 36) : mode === 0 ? brand("github", 443, 52, 36) : "")
      );
    },
    environment: () => {
      const art =
        raised(94, 68, 360, 173) +
        text(122, 109, label, p.text, 21) +
        rect(121, 130, 306, 65, p.panel, 10);
      if (mode === 2)
        return (
          art +
          text(143, 172, "API_SECRET", p.muted, 20) +
          line("M376 153l20 20m0-20-20 20", p.warning, 2.5) +
          text(300, 279, detail, p.muted, 18, "middle")
        );
      if (mode === 3)
        return (
          art +
          text(143, 172, detail, p.text, 22) +
          circle(443, 210, 29, p.tint) +
          line("M430 210h26m-13-13v26", p.accent, 2) +
          text(300, 279, "Next deployment", p.muted, 18, "middle")
        );
      if (mode === 4)
        return (
          art +
          text(143, 172, detail, p.text, 22) +
          cube(400, 177, 71) +
          text(300, 280, "Separate from the image", p.muted, 19, "middle")
        );
      return (
        art +
        dots(146, 163, mode === 1 ? 9 : 7) +
        lock(402, 174, 1) +
        badge(117, 249, mode === 0 ? "Build" : mode === 1 ? "Stored" : "Production", false)
      );
    },
    branch: () =>
      mode === 0
        ? raised(71, 77, 136, 147) +
          brand("github", 108, 111, 60) +
          line("M207 151h52q20 0 20-20V96h44M279 151v62h44", p.accent, 2) +
          badge(339, 72, label) +
          badge(339, 191, detail, false)
        : raised(88, 57, 423, 210) +
          branch(136, 146) +
          text(248, 106, label, p.text, 21) +
          text(248, 193, detail, p.muted, 21),
    environments: () => {
      if (mode === 1)
        return (
          raised(78, 83, 201, 157) +
          text(102, 126, "localhost", p.muted, 20) +
          line("M104 160l16 12-16 12m28 0h30", p.accent, 2.5) +
          arrow(291, 160, 32) +
          raised(347, 83, 177, 157) +
          cube(397, 119, 60) +
          text(435, 219, detail, p.text, 20, "middle")
        );
      return (
        raised(70, mode === 0 ? 68 : 82, 224, 175) +
        circle(98, mode === 0 ? 99 : 113, 5) +
        text(96, mode === 0 ? 215 : 229, label, p.text, 21) +
        server(130, 150, mode === 0 ? 2 : 3, 104) +
        raised(322, mode === 0 ? 93 : 65, 207, 175) +
        server(375, mode === 0 ? 167 : 139, 1, 100) +
        text(348, mode === 0 ? 239 : 211, detail, p.muted, 20)
      );
    },
    project: () => {
      const top = mode === 0 ? "payments" : "Project";
      return (
        line("M300 136v42M185 178h230M185 178v25M415 178v25", p.border, 2) +
        raised(187, 49, 226, 98) +
        line("M213 83h50l12 12h87", p.accent, 2) +
        text(215, 124, top, p.text, 23) +
        raised(81, 201, 210, 65) +
        text(106, 243, mode === 0 ? "New app" : "Production", p.text, 21) +
        raised(321, 201, 198, 65) +
        text(348, 243, mode === 0 ? "Environments" : "Preview", p.muted, 21)
      );
    },
    release: () => {
      if (mode === 1)
        return (
          raised(115, 67, 370, 189) +
          cube(145, 103, 82) +
          text(254, 127, "Deployment", p.muted, 20) +
          badge(257, 157, detail) +
          bars(254, 215, [136])
        );
      if (mode === 2)
        return (
          cube(88, 91, 92) +
          line("M186 141h 80", p.accent, 2) +
          server(286, 165, 2, 110) +
          circle(466, 157, 30, p.tint) +
          check(450, 144, 31) +
          text(300, 276, label, p.text, 22, "middle")
        );
      if (mode === 3)
        return (
          raised(80, 67, 440, 175) +
          text(106, 107, label, p.text, 22) +
          [0, 1, 2, 3]
            .map(
              (v) =>
                circle(135 + v * 111, 163, 18, p.tint) +
                check(126 + v * 111, 155, 17) +
                (v < 3 ? line(`M${155 + v * 111} 163h70`, p.border, 2) : ""),
            )
            .join("") +
          text(300, 219, detail, p.muted, 19, "middle")
        );
      if (mode === 4)
        return (
          raised(113, 78, 364, 169) +
          brand("unkey", 140, 105, 44) +
          text(204, 135, label, p.text, 24) +
          server(143, 192, 1, 90) +
          badge(280, 174, "Ready") +
          line("M479 161h41", p.accent, 2) +
          arrow(516, 161, 20) +
          text(300, 283, detail, p.muted, 19, "middle")
        );
      return (
        cube(80, 102, 95) +
        arrow(197, 155, 42) +
        server(282, 176, 2, 118) +
        circle(478, 157, 28, p.tint) +
        check(464, 146, 26) +
        text(130, 260, label, p.muted, 20, "middle") +
        text(355, 260, detail, p.text, 22, "middle")
      );
    },
    promotion: () => {
      const reverse = mode === 1;
      return (
        raised(81, mode === 2 ? 62 : 87, 180, 167) +
        cube(132, mode === 2 ? 88 : 113, 68) +
        text(
          171,
          mode === 2 ? 205 : 230,
          mode === 2 ? "Preview" : reverse ? "Release A" : "Candidate",
          p.text,
          20,
          "middle",
        ) +
        raised(339, mode === 2 ? 93 : 61, 180, 167) +
        cube(390, mode === 2 ? 118 : 86, 68) +
        text(
          429,
          mode === 2 ? 237 : 204,
          reverse ? "Release B" : "Production",
          p.muted,
          20,
          "middle",
        ) +
        line(reverse ? "M361 265H186q-30 0-30-18" : "M239 62h152q37 0 37 28", p.accent, 2, "5 7") +
        (reverse ? line("M147 253l9-9 9 9", p.accent, 2) : line("M419 83l9 9 9-9", p.accent, 2)) +
        text(
          300,
          289,
          mode === 2 ? detail : reverse ? "Restore previous release" : "Promote",
          p.accent,
          19,
          "middle",
        )
      );
    },
    power: () =>
      raised(126, 63, 348, 190) +
      circle(300, 144, 50, p.tint) +
      line("M277 111a 40 40 0 1 0 46 0", mode ? p.muted : p.accent, 4) +
      line("M300  90v45", mode ? p.muted : p.accent, 4) +
      text(300, 232, detail, p.text, 24, "middle") +
      text(300, 286, label, p.muted, 20, "middle"),
    domain: () => {
      if (mode === 2)
        return [0, 1, 2]
          .map(
            (i) =>
              raised(90 + i * 19, 65 + i * 48, 382, 70) +
              circle(121 + i * 19, 100 + i * 48, 8, i === 2 ? p.accent : p.border) +
              text(
                147 + i * 19,
                107 + i * 48,
                ["api.acme.com", "app.acme.com", "preview.acme.com"][i],
                p.text,
                21,
              ),
          )
          .join("");
      return (
        circle(135, 147, 50, p.tint) +
        line("M85 147h100M135 97c-35 30-35 70 0 100m0-100c35 30 35 70 0 100", p.accent, 1.5) +
        raised(219, 99, 310, 101) +
        text(242, 158, mode === 0 ? label : "api.acme.com", p.text, 23) +
        line("M185 147h34", p.accent, 2, "4 6") +
        badge(282, 235, detail, mode === 0)
      );
    },
    certificate: () =>
      raised(151, 55, 282, 197) +
      text(182, 101, mode === 0 ? "api.acme.com" : "DNS ownership", p.text, 23) +
      bars(184, 123, [188, 140]) +
      circle(394, 211, 40, p.tint) +
      check(376, 195, 34) +
      text(184, 218, detail, p.accent, 24) +
      text(300, 287, label, p.muted, 19, "middle"),
    resources: () => {
      const widths = mode === 0 ? [156, 112] : [106, 166];
      return (
        raised(79, 68, 253, 182) +
        text(106, 109, label, p.text, 22) +
        widths
          .map(
            (w, i) =>
              rect(107, 139 + i * 49, 194, 12, p.border, 6) +
              rect(107, 139 + i * 49, w, 12, p.accent, 6),
          )
          .join("") +
        cube(379, 110, 90) +
        text(425, 251, detail, p.muted, 19, "middle")
      );
    },
    regions: () => {
      const globe =
        circle(300, 154, 102, p.panel) +
        line("M198 154a102 102 0 1 0 204 0a102 102 0 1 0-204 0", p.border, 1.3) +
        line(
          "M198 154h204M211 112h178M211 196h178M300 52c-69 57-69 147 0 204m0-204c69 57 69 147 0 204",
          p.border,
          1.3,
        );
      return (
        globe +
        line(
          mode === 0
            ? "M151 178Q224 22 365 98M365 98Q474 112 458 223"
            : "M140 112Q225 224 354 181M354 181Q419 51 488 130",
          p.accent,
          2,
          "5 7",
        ) +
        circle(mode === 0 ? 365 : 354, mode === 0 ? 98 : 181, 12, p.tint) +
        circle(mode === 0 ? 365 : 354, mode === 0 ? 98 : 181, 5) +
        circle(mode === 0 ? 151 : 140, mode === 0 ? 178 : 112, 7) +
        circle(mode === 0 ? 458 : 488, mode === 0 ? 223 : 130, 7, p.muted) +
        text(300, 287, label, p.text, 22, "middle")
      );
    },
    limits: () => {
      const counts = [3, 2, 5, 4, 3][mode];
      let art = raised(90, 61, 420, 193) + text(118, 102, label, p.text, 22);
      for (let i = 0; i < counts; i++)
        art +=
          rect(121 + i * (352 / counts), 131, 352 / counts - 14, 82, p.tint, 9) +
          rect(
            121 + i * (352 / counts),
            184 - (i % 3) * 17,
            352 / counts - 14,
            29 + (i % 3) * 17,
            p.raised,
            9,
          );
      art +=
        line("M114 119H486", p.accent, 1.5, "4 7") + text(300, 287, detail, p.muted, 19, "middle");
      return art;
    },
    budget: () => {
      if (mode === 1)
        return (
          raised(81, 61, 438, 195) +
          text(111, 102, label, p.text, 22) +
          chart(111, 131, 252, 81, 2) +
          badge(378, 181, "Usage") +
          text(300, 287, detail, p.muted, 19, "middle")
        );
      const end = mode === 0 ? 300 : 377;
      return (
        raised(90, 65, 420, 184) +
        text(118, 108, label, p.text, 24) +
        rect(119, 143, 360, 17, p.border, 8) +
        rect(119, 143, end - 119, 17, p.accent, 8) +
        [209, 299, 389, 479].map((x) => line(`M${x} 172v10`, p.border, 2)).join("") +
        text(119, 221, mode === 0 ? "50%" : "75%", p.muted, 20) +
        text(478, 221, "100%", p.muted, 20, "end") +
        badge(end - 43, 182, mode === 0 ? "Alert" : "Used")
      );
    },
    plans: () =>
      [0, 1, 2]
        .map(
          (i) =>
            raised(80 + i * 153, 136 - i * 31, 134, 109 + i * 31) +
            server(100 + i * 153, 181 - i * 15, i + 1, 90) +
            text(
              147 + i * 153,
              236,
              ["Starter", "Pro", "Business"][i],
              i === mode ? p.accent : p.text,
              19,
              "middle",
            ),
        )
        .join(""),
    build: () => {
      if (mode === 0)
        return (
          raised(60, 90, 146, 139) +
          brand("github", 104, 112, 50) +
          text(133, 210, "Commit", p.muted, 20, "middle") +
          line("M206 155h 90", p.accent, 2, "5 7") +
          cube(331, 95, 117) +
          text(390, 265, detail, p.text, 22, "middle")
        );
      if (mode === 1)
        return (
          raised(69, 95, 140, 123) +
          raised(80, 80, 140, 123) +
          raised(91, 65, 140, 123) +
          brand("github", 133, 98, 56) +
          line("M233 130h65q22 0 22 22v15h27", p.accent, 2, "4 7") +
          cube(370, 109, 94) +
          text(155, 259, label, p.muted, 21, "middle") +
          badge(355, 239, "Ready")
        );
      return (
        raised(90, 61, 420, 194) +
        text(118, 103, label, p.text, 22) +
        [0, 1, 2]
          .map(
            (i) =>
              circle(130, 139 + i * 39, 10, p.tint) +
              check(124, 133 + i * 39, 12) +
              rect(157, 135 + i * 39, [226, 164, 195][i], 7, p.border, 3),
          )
          .join("") +
        badge(373, 217, "Done")
      );
    },
    credential: () => {
      const art =
        raised(105, 83, 390, 153) +
        key(137, 116, 1.2) +
        dots(274, 155, mode === 0 ? 8 : 6) +
        text(136, 210, label, p.text, 22);
      return (
        art +
        (mode === 0
          ? badge(319, 248, "CLI", false)
          : mode === 1
            ? circle(474, 90, 26, p.tint) + line("M462 90h24m-12-12v24", p.accent, 2)
            : badge(332, 38, "Created")) +
        text(114, 275, detail, p.muted, 18)
      );
    },
    principal: () => {
      const art =
        raised(93, 74, 273, 167) +
        circle(135, 120, 18, p.tint) +
        circle(135, 115, 6) +
        line("M123 133q12-17 24 0", p.accent, 2) +
        text(171, 126, mode === 1 ? "user_test" : "user_123", p.text, 22) +
        bars(119, 167, [207, 154]);
      if (mode === 0)
        return (
          art +
          arrow(379, 157, 35) +
          cube(446, 123, 60) +
          text(300, 282, detail, p.muted, 20, "middle")
        );
      if (mode === 1)
        return (
          art +
          raised(388, 113, 130, 99) +
          line("M413 141l16 14-16 14m30 0h 30", p.accent, 2.5) +
          text(300, 282, label, p.text, 22, "middle")
        );
      return (
        art +
        line("M366 155h40", p.accent, 2, "4 6") +
        shield(428, 100) +
        text(300, 282, label, p.text, 22, "middle")
      );
    },
    error: () =>
      raised(118, 60, 364, 195) +
      text(148, 130, label, p.warning, 48) +
      text(148, 205, detail, p.text, 20) +
      [0, 1, 2]
        .map((i) =>
          rect(148 + i * 47, 151, mode === i ? 36 : 25, 4, mode === i ? p.warning : p.border, 2),
        )
        .join("") +
      circle(449, 90, 22, p.tint) +
      text(449, 98, "!", p.warning, 26, "middle"),
    catalog: () =>
      raised(68, 77, 221, 171) +
      raised(311, 77, 221, 171) +
      text(91, 120, label, p.text, mode === 1 ? 20 : 22) +
      text(334, 120, detail, p.text, mode === 1 ? 18 : 21) +
      text(91, 172, mode === 0 ? "type" : "Request", p.muted, 20) +
      text(91, 210, mode === 0 ? "detail" : "invalid", p.muted, 20) +
      text(334, 172, mode === 0 ? "code" : "Configuration", p.muted, 20) +
      text(334, 210, mode === 0 ? "message" : "invalid", p.muted, 20),
    auth: () => {
      if (mode === 0)
        return (
          raised(80, 83, 200, 148) +
          key(120, 109, 1) +
          text(108, 205, label, p.text, 19) +
          shield(354, 80, false) +
          badge(349, 227, "403", false)
        );
      if (mode === 1)
        return (
          raised(70, 74, 178, 174) +
          text(94, 112, "Keyspaces", p.muted, 21) +
          [0, 1, 2]
            .map(
              (i) =>
                rect(96, 132 + i * 30, 122, 17, i === 1 ? p.tint : p.panel, 5) +
                circle(109, 140 + i * 30, 3, i === 1 ? p.accent : p.border),
            )
            .join("") +
          line("M248 161h81", p.accent, 2) +
          shield(358, 100) +
          text(390, 265, detail, p.text, 21, "middle")
        );
      if (mode === 2)
        return (
          key(90, 124, 1.3) +
          line("M210 161h64", p.accent, 2) +
          raised(294, 80, 215, 159) +
          circle(328, 116, 14, p.tint) +
          text(354, 123, "user_123", p.text, 21) +
          text(321, 202, detail, p.muted, 19) +
          check(459, 98, 22)
        );
      if (mode === 3)
        return (
          raised(111, 67, 376, 184) +
          text(138, 112, label, p.text, 23) +
          shield(147, 133) +
          rect(329, 154, 112, 49, p.tint, 25) +
          circle(414, 178, 18, p.accent) +
          text(300, 283, detail, p.muted, 19, "middle")
        );
      if (mode === 4)
        return (
          key(90, 124, 1.4) +
          line("M216 160h 60", p.accent, 2) +
          shield(304, 105) +
          line("M377 160h42", p.accent, 2) +
          circle(473, 160, 34, p.tint) +
          check(456, 146, 32) +
          text(300, 271, label, p.text, 22, "middle")
        );
      if (mode === 5)
        return (
          raised(90, 80, 195, 167) +
          cube(144, 106, 75) +
          text(115, 221, "Your app", p.text, 22) +
          line("M285 162h 60", p.accent, 2, "5 7") +
          shield(383, 100) +
          circle(449, 211, 22, p.tint) +
          line("M438 211h22m-11-11v22", p.accent, 2) +
          text(300, 282, label, p.muted, 20, "middle")
        );
      return (
        raised(70, 109, 134, 103) +
        key(99, 127, 0.8) +
        line("M204 160h65", p.accent, 2) +
        shield(290, 105) +
        line("M354 160h 60", p.accent, 2) +
        cube(444, 130, 60) +
        text(300, 274, detail, p.text, 22, "middle")
      );
    },
    policy: () => {
      if (mode === 4)
        return (
          raised(85, 75, 215, 163) +
          text(111, 113, "Policy change", p.text, 21) +
          bars(111, 139, [157, 119, 142]) +
          arrow(312, 155, 35) +
          cube(385, 107, 91) +
          text(300, 280, detail, p.text, 21, "middle")
        );
      const selected = [0, 1, 2, 1, 0, 0][mode];
      return (
        [0, 1, 2]
          .map(
            (i) =>
              raised(113 + i * 24, 50 + i * 68, 330, 58) +
              text(
                139 + i * 24,
                87 + i * 68,
                ["Authenticate", "Rate limit", "Log request"][i],
                i === selected ? p.accent : p.muted,
                22,
              ) +
              (i === selected ? check(390 + i * 24, 69 + i * 68, 20) : ""),
          )
          .join("") +
        text(300, 289, label, p.text, 21, "middle") +
        (mode === 2
          ? line("M 80 90v142m-7-8 7 8 7-8", p.accent, 2)
          : mode === 3
            ? circle(486, 173, 20, p.tint) + line("M477 173h18m-9-9v18", p.accent, 2)
            : mode === 5
              ? arrow(62, 105, 30)
              : "")
      );
    },
    match: () =>
      raised(80, 72, 205, 158) +
      text(104, 112, "/api/*", p.text, 24) +
      badge(104, 156, "POST") +
      line("M285 153h67", p.accent, 2) +
      shield(383, 90) +
      text(300, 278, detail, p.text, 22, "middle"),
    rate: () =>
      mode === 0
        ? raised(108, 65, 384, 187) +
          line("M163 197a137 137 0 0 1 274 0", p.border, 15) +
          line("M163 197a137 137 0 0 1 223-106", p.accent, 15) +
          text(300, 190, "87 / 100", p.text, 32, "middle") +
          text(300, 282, detail, p.muted, 20, "middle")
        : [0, 1, 2]
            .map(
              (i) =>
                raised(80 + i * 153, 80, 131, 164) +
                text(145 + i * 153, 119, `user_${i + 1}`, p.muted, 18, "middle") +
                rect(105 + i * 153, 143, 81, 70, p.tint, 8) +
                rect(105 + i * 153, 185 - i * 13, 81, 28 + i * 13, p.accent, 8),
            )
            .join("") + text(300, 282, detail, p.text, 22, "middle"),
    firewall: () =>
      line("M 70 156h124M369 156h139", p.border, 2, "5 7") +
      [0, 1, 2].map((i) => circle(82 + i * 37, 156, 4)).join("") +
      raised(205, 67, 164, 181) +
      [0, 1, 2, 3].map((i) => rect(226 + i * 32, 95, 16, 117, p.tint, 7)).join("") +
      shield(255, 112, false) +
      text(300, 281, detail, p.text, 22, "middle"),
    schema: () =>
      raised(90, 61, 267, 195) +
      text(119, 105, "OpenAPI", p.text, 26) +
      text(119, 159, "{", p.accent, 30) +
      bars(151, 143, [143, 108, 130]) +
      text(119, 213, "}", p.accent, 30) +
      circle(437, 160, 49, p.tint) +
      check(413, 141, 45) +
      text(300, 286, detail, p.muted, 19, "middle"),
    redact: () =>
      mode === 0
        ? raised(80, 70, 440, 183) +
          text(109, 115, label, p.muted, 22) +
          rect(109, 139, 342, 60, p.tint, 12) +
          text(131, 179, detail, p.accent, 24) +
          lock(442, 188, 0.9)
        : raised(90, 61, 420, 194) +
          text(118, 103, label, p.text, 24) +
          [0, 1, 2]
            .map(
              (i) =>
                text(119, 146 + i * 38, ["Headers", "Query", "Body"][i], p.muted, 20) +
                rect(396, 127 + i * 38, 70, 20 + 6, i === 1 ? p.panel : p.tint, 13) +
                circle(i === 1 ? 409 : 453, 140 + i * 38, 9, i === 1 ? p.muted : p.accent),
            )
            .join(""),
    terminal: () =>
      raised(80, 70, 440, 181) +
      line("M110 105l16 13-16 13m29 0h23", p.accent, 2.5) +
      text(112, 174, label, p.text, 27) +
      text(112, 216, detail, p.muted, 21) +
      (mode ? badge(396, 35, "Local", false) : brand("docker", 439, 90, 44)),
    query: () =>
      raised(70, 65, 340, 185) +
      text(100, 106, label, p.text, 22) +
      chart(100, 134, 269, 80, mode) +
      raised(340, 239, 191, 52) +
      text(364, 273, mode ? "SELECT" : "SQL", p.accent, 26),
    logs: () => {
      const rows =
        mode === 2
          ? ["stdout", "stderr", "stdout"]
          : mode === 1
            ? ["GET /orders", "GET /healthz", "POST /events"]
            : ["SELECT", "WHERE", "LIMIT"];
      return (
        raised(80, 59, 440, 199) +
        rows
          .map(
            (row, i) =>
              circle(109, 103 + i * 51, 4, i === 1 && mode === 2 ? p.warning : p.accent) +
              text(129, 110 + i * 51, row, p.text, 21) +
              rect(329, 98 + i * 51, [130, 90, 113][i], 5, p.border, 2),
          )
          .join("") +
        text(300, 290, mode === 1 ? "Request history" : label, p.muted, 19, "middle")
      );
    },
    metrics: () =>
      raised(60, 70, 287, 180) +
      text(87, 110, "Requests", p.text, 21) +
      chart(87, 140, 231, 70, 0) +
      raised(369, 90, 171, 141) +
      text(393, 126, "CPU", p.muted, 20) +
      [28, 47, 70, 53].map((h, i) => rect(394 + i * 30, 211 - h, 17, h, p.accent, 4)).join(""),
    sdk: () =>
      line("M154 137h292M300 137v72", p.border, 2) +
      [
        ["typescript", 93],
        ["go", 251],
        ["python", 409],
      ]
        .map(([name, x]) => raised(x, 60, 98, 98, 22) + brand(name, x + 23, 83, 52))
        .join("") +
      badge(226, 213, "Unkey API") +
      text(300, 286, label, p.text, 21, "middle"),
  };
  const renderer = renderers[motif];
  if (!renderer) throw new Error(`Unknown Compute motif: ${motif}`);
  return renderer();
}
