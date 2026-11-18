import { NextResponse } from 'next/server';
import type { NextRequest } from 'next/server';
import { jwtVerify } from 'jose';

const JWT_SECRET = process.env.JWT_SECRET || 'your-32-byte-secret-key-change-in-production';

export async function middleware(request: NextRequest) {
  const token = request.cookies.get('pv_admin_token')?.value;
  const { pathname } = request.nextUrl;

  // Allow static files, Next.js internal requests, and public session endpoint
  if (
    pathname.startsWith('/_next') ||
    pathname.startsWith('/api/session') ||
    pathname === '/favicon.ico' ||
    pathname.match(/\.(png|jpg|jpeg|svg|css|js|webp)$/)
  ) {
    return NextResponse.next();
  }

  let isValidAdmin = false;
  if (token) {
    try {
      const secret = new TextEncoder().encode(JWT_SECRET);
      const { payload } = await jwtVerify(token, secret, { algorithms: ['HS256'] });
      if (payload.role === 'admin') {
        isValidAdmin = true;
      }
    } catch {
      isValidAdmin = false;
    }
  }

  // If visiting /login:
  if (pathname === '/login') {
    if (isValidAdmin) {
      return NextResponse.redirect(new URL('/', request.url));
    }
    const response = NextResponse.next();
    if (token && !isValidAdmin) {
      response.cookies.delete('pv_admin_token');
    }
    return response;
  }

  // Protected routes: require valid admin token
  if (!isValidAdmin) {
    const nextPath = encodeURIComponent(pathname + request.nextUrl.search);
    const loginUrl = new URL(`/login?next=${nextPath}`, request.url);
    const response = NextResponse.redirect(loginUrl);
    if (token) {
      response.cookies.delete('pv_admin_token');
    }
    return response;
  }

  return NextResponse.next();
}

export const config = {
  matcher: ['/((?!_next/static|_next/image|favicon.ico).*)'],
};
