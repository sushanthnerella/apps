'use client';

import React, { useState } from 'react';
import { apiFetch } from '../../lib/api';

export default function GateScannerPage() {
  const [qrInput, setQrInput] = useState('');
  const [scanResult, setScanResult] = useState<{
    valid: boolean;
    devoteeName?: string;
    gotram?: string;
    status?: string;
    timestamp?: string;
    message: string;
  } | null>(null);
  const [latencyMs, setLatencyMs] = useState<number | null>(null);
  const [isVerifying, setIsVerifying] = useState(false);
  const [recentScans, setRecentScans] = useState<Array<{
    name: string;
    token: string;
    latency: string;
    time: string;
    status: string;
  }>>([]);

  const handleScan = async (e?: React.FormEvent) => {
    if (e) e.preventDefault();
    if (!qrInput.trim()) return;

    setIsVerifying(true);
    const start = performance.now();

    try {
      const res = await apiFetch('/api/v1/temples/tickets/verify', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ qr_token: qrInput.trim() }),
      });
      const end = performance.now();
      const elapsed = Math.round((end - start) * 100) / 100;
      setLatencyMs(elapsed);

      if (res.ok) {
        const data = await res.json();
        const devoteeName = data.ticket?.devotee_name || 'VIP Devotee';
        const gotram = data.ticket?.devotee_gotram || '';
        const status = data.ticket?.status || 'CHECKED_IN';
        const timestamp = new Date().toLocaleTimeString();

        setScanResult({
          valid: true,
          devoteeName: data.ticket?.devotee_name,
          gotram: data.ticket?.devotee_gotram,
          status,
          timestamp,
          message: 'Verification successful. Admitted to sanctum.',
        });

        setRecentScans((prev) => [
          {
            name: devoteeName,
            token: qrInput,
            latency: `${elapsed}ms`,
            time: timestamp,
            status: 'ADMITTED',
          },
          ...prev.slice(0, 9),
        ]);
      } else {
        const err = await res.json().catch(() => ({}));
        setScanResult({
          valid: false,
          message: err.message || err.error || 'Unable to verify, do not admit, retry',
        });
        setRecentScans((prev) => [
          {
            name: 'Verification Failed',
            token: qrInput,
            latency: `${elapsed}ms`,
            time: new Date().toLocaleTimeString(),
            status: 'REJECTED',
          },
          ...prev.slice(0, 9),
        ]);
      }
    } catch {
      const end = performance.now();
      const elapsed = Math.round((end - start) * 100) / 100;
      setLatencyMs(elapsed);
      setScanResult({
        valid: false,
        message: 'Unable to verify, do not admit, retry',
      });
      setRecentScans((prev) => [
        {
          name: 'Verification Error',
          token: qrInput,
          latency: `${elapsed}ms`,
          time: new Date().toLocaleTimeString(),
          status: 'ERROR',
        },
        ...prev.slice(0, 9),
      ]);
    } finally {
      setIsVerifying(false);
    }
  };

  return (
    <div style={{ maxWidth: '800px' }}>
      <div style={{ marginBottom: '24px' }}>
        <h1 style={{ fontSize: '26px', fontWeight: 800, margin: '0 0 6px 0', color: '#1E293B' }}>
          ⚡ Sanctum VIP Gate Scanner
        </h1>
        <p style={{ margin: 0, color: '#64748B', fontSize: '14px' }}>
          Real-time atomic verification and admittance control for temple VIP pooja tickets.
        </p>
      </div>

      {/* Input Scanner Simulation Box */}
      <div style={{ backgroundColor: '#FFFFFF', padding: '24px', borderRadius: '12px', boxShadow: '0 1px 3px rgba(0,0,0,0.05)', marginBottom: '24px' }}>
        <form onSubmit={handleScan} style={{ display: 'flex', gap: '12px', alignItems: 'center' }}>
          <input
            type="text"
            value={qrInput}
            onChange={(e) => setQrInput(e.target.value)}
            placeholder="Scan QR or enter token..."
            style={{
              flex: 1,
              padding: '14px 16px',
              borderRadius: '8px',
              border: '2px solid #E2E8F0',
              fontSize: '15px',
              fontFamily: 'monospace',
            }}
          />
          <button
            type="submit"
            disabled={isVerifying}
            style={{
              padding: '14px 28px',
              backgroundColor: '#831843',
              color: '#FFFFFF',
              border: 'none',
              borderRadius: '8px',
              fontWeight: 700,
              fontSize: '15px',
              cursor: isVerifying ? 'not-allowed' : 'pointer',
            }}
          >
            {isVerifying ? 'Scanning...' : 'Verify Entry Scan'}
          </button>
        </form>

        {latencyMs !== null && (
          <div style={{ marginTop: '12px', display: 'flex', alignItems: 'center', gap: '8px', fontSize: '13px', color: '#475569', fontWeight: 600 }}>
            <span>Verification Latency:</span>
            <span style={{ backgroundColor: '#F1F5F9', padding: '2px 8px', borderRadius: '4px' }}>
              {latencyMs} ms
            </span>
          </div>
        )}
      </div>

      {/* Active Scan Result Box */}
      {scanResult && (
        <div style={{
          backgroundColor: scanResult.valid ? '#ECFDF5' : '#FEF2F2',
          border: `2px solid ${scanResult.valid ? '#10B981' : '#EF4444'}`,
          borderRadius: '12px',
          padding: '24px',
          marginBottom: '28px',
        }}>
          <div style={{ display: 'flex', alignItems: 'center', gap: '12px', marginBottom: '8px' }}>
            <span style={{ fontSize: '28px' }}>{scanResult.valid ? '✅' : '❌'}</span>
            <div>
              <h3 style={{ margin: 0, fontSize: '18px', fontWeight: 700, color: scanResult.valid ? '#065F46' : '#991B1B' }}>
                {scanResult.valid ? 'ENTRY GRANTED - SANCTUM VIP PASS' : 'ENTRY REJECTED - DO NOT ADMIT'}
              </h3>
              <span style={{ fontSize: '13px', color: '#4B5563' }}>{scanResult.message}</span>
            </div>
          </div>

          {scanResult.valid && (
            <div style={{ marginTop: '16px', display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: '12px', backgroundColor: '#FFFFFF', padding: '16px', borderRadius: '8px' }}>
              {scanResult.devoteeName && (
                <div>
                  <span style={{ fontSize: '12px', color: '#6B7280' }}>Devotee Name:</span>
                  <div style={{ fontWeight: 700 }}>{scanResult.devoteeName}</div>
                </div>
              )}
              {scanResult.gotram && (
                <div>
                  <span style={{ fontSize: '12px', color: '#6B7280' }}>Family Gotram:</span>
                  <div style={{ fontWeight: 700 }}>{scanResult.gotram}</div>
                </div>
              )}
              {scanResult.timestamp && (
                <div>
                  <span style={{ fontSize: '12px', color: '#6B7280' }}>Scanned At:</span>
                  <div style={{ fontWeight: 700, color: '#831843' }}>{scanResult.timestamp}</div>
                </div>
              )}
            </div>
          )}
        </div>
      )}

      {/* Audit Log Table */}
      {recentScans.length > 0 && (
        <div style={{ backgroundColor: '#FFFFFF', padding: '20px', borderRadius: '12px', boxShadow: '0 1px 3px rgba(0,0,0,0.05)' }}>
          <h3 style={{ margin: '0 0 12px 0', fontSize: '16px', fontWeight: 700 }}>Gate Scan Local Audit Log</h3>
          <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '13px' }}>
            <thead>
              <tr style={{ borderBottom: '1px solid #E2E8F0', color: '#64748B', textAlign: 'left' }}>
                <th style={{ padding: '8px' }}>Devotee</th>
                <th style={{ padding: '8px' }}>Token</th>
                <th style={{ padding: '8px' }}>Latency</th>
                <th style={{ padding: '8px' }}>Time</th>
                <th style={{ padding: '8px' }}>Result</th>
              </tr>
            </thead>
            <tbody>
              {recentScans.map((s, idx) => (
                <tr key={idx} style={{ borderBottom: '1px solid #F1F5F9' }}>
                  <td style={{ padding: '8px', fontWeight: 600 }}>{s.name}</td>
                  <td style={{ padding: '8px', fontFamily: 'monospace', color: '#475569' }}>{s.token.slice(0, 24)}...</td>
                  <td style={{ padding: '8px', color: '#475569', fontWeight: 600 }}>{s.latency}</td>
                  <td style={{ padding: '8px', color: '#64748B' }}>{s.time}</td>
                  <td style={{ padding: '8px', fontWeight: 700, color: s.status === 'ADMITTED' ? '#15803D' : '#DC2626' }}>{s.status}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
