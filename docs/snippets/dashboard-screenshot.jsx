// Production serves the docs under /docs, and Mintlify does not rewrite image
// paths built at runtime. Local previews serve from /, so fall back to it.
const DOCS_BASE_PATH = "/docs";

const ThemedImage = ({ path, alt, width, className }) => (
  <img
    className={className}
    src={`${DOCS_BASE_PATH}${path}`}
    onError={(event) => {
      const img = event.currentTarget;
      if (img.dataset.fallback) {
        return;
      }
      img.dataset.fallback = "true";
      img.src = path;
    }}
    alt={alt}
    style={{ width, maxWidth: "100%", height: "auto" }}
  />
);

export const DashboardScreenshot = ({ src, alt, width }) => (
  <>
    <ThemedImage className="block dark:hidden" path={`${src}-light.png`} alt={alt} width={width} />
    <ThemedImage className="hidden dark:block" path={`${src}-dark.png`} alt={alt} width={width} />
  </>
);
