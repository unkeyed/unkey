// Mintlify cannot collapse top-level sidebar groups, so toggle them here.
(() => {
  const storageKey = "unkey-docs-expanded-groups";

  const readExpanded = () => {
    try {
      const parsed = JSON.parse(localStorage.getItem(storageKey) ?? "[]");
      return new Set(Array.isArray(parsed) ? parsed.filter((t) => typeof t === "string") : []);
    } catch {
      return new Set();
    }
  };

  const writeExpanded = (expanded) => {
    try {
      localStorage.setItem(storageKey, JSON.stringify([...expanded]));
    } catch {}
  };

  const groupTitle = (header) => header.textContent?.trim() ?? "";

  const containsCurrentPage = (list) => {
    const path = location.pathname.replace(/\/$/, "");
    return [...list.querySelectorAll("a[href]")].some(
      (a) => new URL(a.href, location.href).pathname.replace(/\/$/, "") === path,
    );
  };

  const setExpanded = (header, expanded) => {
    header.parentElement?.toggleAttribute("data-unkey-collapsed", !expanded);
    header.setAttribute("aria-expanded", String(expanded));
  };

  const toggle = (header) => {
    const expanded = readExpanded();
    const title = groupTitle(header);
    const expand = header.getAttribute("aria-expanded") !== "true";
    if (expand) {
      expanded.add(title);
    } else {
      expanded.delete(title);
    }
    writeExpanded(expanded);
    setExpanded(header, expand);
  };

  let lastPath = "";

  const sync = () => {
    const headers = document.querySelectorAll("#sidebar .sidebar-group-header");
    if (headers.length === 0) return;

    const expanded = readExpanded();
    const pathChanged = location.pathname !== lastPath;
    lastPath = location.pathname;
    for (const header of headers) {
      const list = header.nextElementSibling;
      if (!list?.classList.contains("sidebar-group")) continue;

      if (!header.hasAttribute("data-unkey-collapsible")) {
        header.setAttribute("data-unkey-collapsible", "");
        header.setAttribute("role", "button");
        header.setAttribute("tabindex", "0");
        header.addEventListener("click", () => toggle(header));
        header.addEventListener("keydown", (event) => {
          if (event.key !== "Enter" && event.key !== " ") return;
          event.preventDefault();
          toggle(header);
        });
      }

      if (pathChanged && containsCurrentPage(list)) {
        expanded.add(groupTitle(header));
      }
      setExpanded(header, expanded.has(groupTitle(header)));
    }
    if (pathChanged) {
      writeExpanded(expanded);
    }
  };

  let scheduled = false;
  const scheduleSync = () => {
    if (scheduled) return;
    scheduled = true;
    requestAnimationFrame(() => {
      scheduled = false;
      sync();
    });
  };

  const start = () => {
    sync();
    new MutationObserver(scheduleSync).observe(document.body, { childList: true, subtree: true });
  };

  if (document.readyState === "loading") {
    document.addEventListener("DOMContentLoaded", start);
  } else {
    start();
  }
})();
