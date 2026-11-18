export async function apiFetch(path: string, init?: RequestInit): Promise<Response> {
  const cleanPath = path.replace(/^\/+/, '');
  const url = `/api/backend/${cleanPath}`;

  const res = await fetch(url, {
    ...init,
    credentials: 'same-origin',
  });

  if (res.status === 401) {
    if (typeof window !== 'undefined' && !window.location.pathname.startsWith('/login')) {
      const next = encodeURIComponent(window.location.pathname + window.location.search);
      window.location.href = `/login?next=${next}`;
    }
  }

  return res;
}
