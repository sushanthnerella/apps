'use client';

import React from 'react';
import { useRouter } from 'next/navigation';

export default function LogoutButton() {
  const router = useRouter();

  const handleLogout = async () => {
    try {
      await fetch('/api/session', { method: 'DELETE' });
    } catch {
      // Ignore network errors on logout
    }
    router.push('/login');
    router.refresh();
  };

  return (
    <button
      type="button"
      onClick={handleLogout}
      style={{
        width: '100%',
        marginTop: '10px',
        padding: '8px 12px',
        backgroundColor: 'rgba(255,255,255,0.12)',
        color: '#FCE7F3',
        border: '1px solid rgba(255,255,255,0.2)',
        borderRadius: '8px',
        fontSize: '12px',
        fontWeight: 600,
        cursor: 'pointer',
        display: 'flex',
        alignItems: 'center',
        justifyContent: 'center',
        gap: '6px',
        transition: 'background-color 0.2s',
      }}
    >
      🔒 Sign Out
    </button>
  );
}
