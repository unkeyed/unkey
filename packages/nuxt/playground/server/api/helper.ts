import { useUnkey } from '#imports'

export default defineEventHandler(async event => {
  return {
    baseUrl: useUnkey().baseUrl
  }
})
