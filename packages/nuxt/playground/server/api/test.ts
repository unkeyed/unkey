import { defineEventHandler } from 'h3'

export default defineEventHandler(async event => {
  return {
    unkey: event.context.unkey
  }
})
