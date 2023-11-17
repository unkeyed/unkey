export const AppVersion = () => {
  return <p className="text-xs px-6 text-content-subtle">Version: v{process.env.APP_VERSION}</p>;
};
