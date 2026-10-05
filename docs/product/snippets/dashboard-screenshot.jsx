export const DashboardScreenshot = ({ src, alt, width }) => (
  <>
    <img
      className="block dark:hidden"
      src={`${src}-light.png`}
      alt={alt}
      style={{ width, maxWidth: "100%", height: "auto" }}
    />
    <img
      className="hidden dark:block"
      src={`${src}-dark.png`}
      alt={alt}
      style={{ width, maxWidth: "100%", height: "auto" }}
    />
  </>
);
