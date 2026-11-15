import { cookies } from 'next/headers';
import { NextRequest, NextResponse } from 'next/server';

const API_BASE_URL = (process.env.API_BASE_URL || 'http://localhost:8085').replace(/\/+$/, '');

async function handleProxy(req: NextRequest, { params }: { params: { path: string[] } }) {
  const pathSegments = params.path || [];
  const targetPath = pathSegments.join('/');

  // SSRF Protection: Only allow paths starting with api/v1/ and reject '..'
  if (!targetPath.startsWith('api/v1/') || targetPath.includes('..') || targetPath.includes('://')) {
    return NextResponse.json({ error: 'Forbidden API path' }, { status: 400 });
  }

  // CSRF Protection: State-changing methods require Origin matching the CMS origin
  const method = req.method.toUpperCase();
  if (['POST', 'PUT', 'DELETE', 'PATCH'].includes(method)) {
    const origin = req.headers.get('origin');
    if (!origin || origin !== req.nextUrl.origin) {
      return NextResponse.json({ error: 'Invalid or missing CSRF origin header' }, { status: 403 });
    }
  }

  // Authentication: Read httpOnly token from cookie
  const cookieStore = cookies();
  const token = cookieStore.get('pv_admin_token')?.value;
  if (!token) {
    return NextResponse.json({ error: 'Authentication required' }, { status: 401 });
  }

  const search = req.nextUrl.search || '';
  const destinationUrl = `${API_BASE_URL}/${targetPath}${search}`;

  const forwardHeaders = new Headers();
  forwardHeaders.set('Authorization', `Bearer ${token}`);

  const contentType = req.headers.get('content-type');
  if (contentType) {
    forwardHeaders.set('Content-Type', contentType);
  }

  const idempotencyKey = req.headers.get('idempotency-key');
  if (idempotencyKey) {
    forwardHeaders.set('Idempotency-Key', idempotencyKey);
  }

  const acceptLang = req.headers.get('accept-language');
  if (acceptLang) {
    forwardHeaders.set('Accept-Language', acceptLang);
  }

  let body: BodyInit | undefined = undefined;
  if (method !== 'GET' && method !== 'HEAD') {
    const rawBody = await req.text();
    if (rawBody) {
      body = rawBody;
    }
  }

  try {
    const backendRes = await fetch(destinationUrl, {
      method,
      headers: forwardHeaders,
      body,
      cache: 'no-store',
    });

    const resBody = await backendRes.text();
    const clientRes = new NextResponse(resBody, {
      status: backendRes.status,
      statusText: backendRes.statusText,
    });

    const resContentType = backendRes.headers.get('content-type');
    if (resContentType) {
      clientRes.headers.set('Content-Type', resContentType);
    }

    // On backend 401, clear the session cookie
    if (backendRes.status === 401) {
      clientRes.cookies.delete('pv_admin_token');
    }

    return clientRes;
  } catch {
    return NextResponse.json({ error: 'Backend service communication failure' }, { status: 502 });
  }
}

export const GET = handleProxy;
export const POST = handleProxy;
export const PUT = handleProxy;
export const PATCH = handleProxy;
export const DELETE = handleProxy;
