export type StepRequest<TRequestBody> = {
  url: string
  method: "POST" | "GET" | "PUT" | "DELETE",
  headers: Record<string, string>
  body: TRequestBody
}
export type StepResponse<TBody = unknown> = {
  status: number
  headers: Record<string, string>
  body: TBody
}


export async function step<TRequestBody = unknown, TResponseBody = unknown>(req: StepRequest<TRequestBody>): Promise<StepResponse<TResponseBody>> {
  const res = await fetch(req.url, {
    method: req.method,
    headers: req.headers,
    body: JSON.stringify(req.body)
  })
  return {
    status: res.status,
    headers: Object.fromEntries(res.headers.entries()),
    body: await res.json() as TResponseBody
  }



}
