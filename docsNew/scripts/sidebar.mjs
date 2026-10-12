// Turns the generated nav model into Blume sidebar items, header tabs, and
// the product selector. Page ids stay in the model so scripts/parity.ts can
// diff them against docs/docs.json; Blume itself only sees routes.

const mdxRoute = (id) => {
  if (id === "index") {
    return "/";
  }
  if (id.endsWith("/index")) {
    return `/${id.slice(0, -"/index".length)}`;
  }
  return `/${id}`;
};

const pageRoute = (page) =>
  typeof page === "string" ? mdxRoute(page) : page.route;

const pageId = (page) => (typeof page === "string" ? page : page.page);

const toItems = (pages) =>
  pages.map((page) => {
    if (page && typeof page === "object" && page.group) {
      return {
        collapsed: false,
        items: toItems(page.pages),
        label: page.group,
      };
    }
    return pageRoute(page);
  });

export const toBlumeSidebar = (model) => {
  const items = [];
  for (const tab of model.tabs) {
    const groups = toItems(tab.groups);
    if (tab.wrap) {
      items.push({
        collapsed: false,
        icon: tab.icon,
        items: groups,
        label: `${tab.product} / ${tab.tab}`,
        root: tab.path,
      });
    } else {
      items.push(...groups);
    }
  }
  return items;
};

export const tabsFromModel = (model) =>
  model.tabs.map((tab) => ({
    href: tab.href,
    icon: tab.icon,
    label: tab.tab,
    path: tab.path,
  }));

export const selectorsFromModel = (model) => model.selectors;

export { mdxRoute, pageId, pageRoute };
