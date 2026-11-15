import { cookies } from 'next/headers';
import { NextRequest, NextResponse } from 'next/server';
import { jwtVerify, decodeJwt } from 'jose';

const JWT_SECRET = process.env.JWT_SECRET || 'your-32-byte-secret-key-change-in-production';
const API_BASE_URL = (process.env.API_BASE_URL || 'http://localhost:8085').replace(/\/+$/, '');

export async function POST(req: NextRequest) {
  try {
    const { email, password } = await req.json();
    if (!email || !password) {
      return NextResponse.json({ error: 'Email and password are required' }, { status: 400 });
    }

    const backendRes = await fetch(`${API_BASE_URL}/api/v1/auth/admin/login`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ email: email.trim().toLowerCase(), password }),
    });

    const data = await backendRes.json().catch(() => null);

    if (!backendRes.ok) {
      return NextResponse.json(
        { error: data?.message || 'Invalid email or password' },
        { status: backendRes.status }
      );
    }

    const token = data?.access_token;
    if (!token) {
      return NextResponse.json({ error: 'Authentication failed' }, { status: 401 });
    }

    let exp = Math.floor(Date.now() / 1000) + 86400; // default 24h
    try {
      const decoded = decodeJwt(token);
      if (decoded.exp) {
        exp = decoded.exp;
      }
    } catch {
      // fallback to default exp
    }

    const maxAge = Math.max(0, exp - Math.floor(Date.now() / 1000));

    cookies().set({
      name: 'pv_admin_token',
      value: token,
      httpOnly: true,
      secure: process.env.NODE_ENV === 'production',
      sameSite: 'strict',
      path: '/',
      maxAge,
    });

    return NextResponse.json({ ok: true });
  } catch {
    return NextResponse.json({ error: 'Internal authentication error' }, { status: 500 });
  }
}

export async function GET() {
  const cookieStore = cookies();
  const token = cookieStore.get('pv_admin_token')?.value;
  if (!token) {
    return NextResponse.json({ authenticated: false }, { status: 401 });
  }

  try {
    const secret = new TextEncoder().encode(JWT_SECRET);
    const { payload } = await jwtVerify(token, secret, { algorithms: ['HS256'] });
    if (payload.role !== 'admin') {
      return NextResponse.json({ authenticated: false }, { status: 403 });
    }
    return NextResponse.json({
      authenticated: true,
      role: payload.role,
      exp: payload.exp,
    });
  } catch {
    return NextResponse.json({ authenticated: false }, { status: 401 });
  }
}

export async function DELETE() {
  cookies().delete('pv_admin_token');
  return NextResponse.json({ ok: true });
}
