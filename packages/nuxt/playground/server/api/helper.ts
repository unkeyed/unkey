import { useUnkey } from "#imports";

export default defineEventHandler(async (_event) => {
  return {
    baseUrl: useUnkey().baseUrl,
  };
});
