// Cloudflare Pages Functions: keep the Go API behind the same public origin.
export async function onRequest(context) {
  const origin = context.env.API_ORIGIN
  if (!origin) return Response.json({ error: 'API_ORIGIN 尚未配置' }, { status: 503 })
  const incoming = new URL(context.request.url)
  const upstream = new URL(incoming.pathname + incoming.search, origin)
  const headers = new Headers(context.request.headers)
  headers.set('X-Forwarded-Proto', incoming.protocol.replace(':', ''))
  headers.delete('host')
  const response = await fetch(upstream, {
    method: context.request.method,
    headers,
    body: ['GET', 'HEAD'].includes(context.request.method) ? undefined : context.request.body,
    redirect: 'manual',
  })
  return new Response(response.body, { status: response.status, headers: response.headers })
}
