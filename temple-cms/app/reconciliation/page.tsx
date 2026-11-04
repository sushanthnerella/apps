'use client';

import React, { useState } from 'react';

interface LedgerRow {
  date: string;
  onlineEscrow: number;
  cashCounter: number;
  totalRevenue: number;
  archakaHonorarium: number;
  status: 'BALANCED' | 'PENDING' | 'DISCREPANCY';
}

export default function FinancialReconciliationPage() {
  const [filter, setFilter] = useState<'ALL' | 'BALANCED' | 'PENDING'>('ALL');
  const [ledgerRows, setLedgerRows] = useState<LedgerRow[]>([
    { date: '2026-11-04', onlineEscrow: 135400, cashCounter: 42500, totalRevenue: 177900, archakaHonorarium: 44475, status: 'BALANCED' },
    { date: '2026-11-03', onlineEscrow: 124992, cashCounter: 40176, totalRevenue: 165168, archakaHonorarium: 41292, status: 'BALANCED' },
    { date: '2026-11-02', onlineEscrow: 98208, cashCounter: 35500, totalRevenue: 133708, archakaHonorarium: 33427, status: 'BALANCED' },
    { date: '2026-11-01', onlineEscrow: 142848, cashCounter: 52300, totalRevenue: 195148, archakaHonorarium: 48787, status: 'BALANCED' },
    { date: '2026-10-31', onlineEscrow: 89280, cashCounter: 28900, totalRevenue: 118180, archakaHonorarium: 29545, status: 'BALANCED' }
  ]);

  const totalEscrow = ledgerRows.reduce((acc, r) => acc + r.onlineEscrow, 0);
  const totalCash = ledgerRows.reduce((acc, r) => acc + r.cashCounter, 0);
  const totalHonorarium = ledgerRows.reduce((acc, r) => acc + r.archakaHonorarium, 0);

  const filteredRows = filter === 'ALL' ? ledgerRows : ledgerRows.filter(r => r.status === filter);

  const handleExportCSV = () => {
    const headers = ['Date', 'Online Escrow (INR)', 'Counter Cash (INR)', 'Total Revenue (INR)', 'Archaka Share 25% (INR)', 'Status'];
    const rows = filteredRows.map(r => [r.date, r.onlineEscrow, r.cashCounter, r.totalRevenue, r.archakaHonorarium, r.status]);
    const csvContent = 'data:text/csv;charset=utf-8,' + [headers.join(','), ...rows.map(e => e.join(','))].join('\n');
    const encodedUri = encodeURI(csvContent);
    const link = document.createElement('a');
    link.setAttribute('href', encodedUri);
    link.setAttribute('download', `financial_ledger_${new Date().toISOString().slice(0, 10)}.csv`);
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
  };

  return (
    <div>
      <div style={{ marginBottom: '24px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div>
          <h1 style={{ fontSize: '26px', fontWeight: 800, margin: '0 0 6px 0', color: '#1E293B' }}>
            🧾 Financial Reconciliation Ledger
          </h1>
          <p style={{ margin: 0, color: '#64748B', fontSize: '14px' }}>
            Daily automated reconciliation: Online Escrow deposits vs Cash-at-counter collections with 0% discrepancy.
          </p>
        </div>
        <button
          onClick={handleExportCSV}
          style={{
            backgroundColor: '#047857',
            color: '#FFFFFF',
            border: 'none',
            padding: '10px 18px',
            borderRadius: '8px',
            fontWeight: 700,
            cursor: 'pointer',
            display: 'flex',
            alignItems: 'center',
            gap: '6px'
          }}>
          📥 Export Audit CSV
        </button>
      </div>

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: '16px', marginBottom: '24px' }}>
        <div style={{ backgroundColor: '#FFFFFF', padding: '20px', borderRadius: '12px', boxShadow: '0 1px 3px rgba(0,0,0,0.05)' }}>
          <div style={{ fontSize: '12px', color: '#64748B', fontWeight: 600 }}>Total Period Online Escrow</div>
          <div style={{ fontSize: '22px', fontWeight: 800, color: '#047857', marginTop: '4px' }}>₹{totalEscrow.toLocaleString('en-IN')}</div>
          <div style={{ fontSize: '11px', color: '#059669', marginTop: '2px' }}>Processed via Gateway</div>
        </div>
        <div style={{ backgroundColor: '#FFFFFF', padding: '20px', borderRadius: '12px', boxShadow: '0 1px 3px rgba(0,0,0,0.05)' }}>
          <div style={{ fontSize: '12px', color: '#64748B', fontWeight: 600 }}>Total Counter Cash Collected</div>
          <div style={{ fontSize: '22px', fontWeight: 800, color: '#831843', marginTop: '4px' }}>₹{totalCash.toLocaleString('en-IN')}</div>
          <div style={{ fontSize: '11px', color: '#831843', marginTop: '2px' }}>Verified at physical counters</div>
        </div>
        <div style={{ backgroundColor: '#FFFFFF', padding: '20px', borderRadius: '12px', boxShadow: '0 1px 3px rgba(0,0,0,0.05)' }}>
          <div style={{ fontSize: '12px', color: '#64748B', fontWeight: 600 }}>Archaka Honorarium Disbursed</div>
          <div style={{ fontSize: '22px', fontWeight: 800, color: '#2563EB', marginTop: '4px' }}>₹{totalHonorarium.toLocaleString('en-IN')}</div>
          <div style={{ fontSize: '11px', color: '#2563EB', marginTop: '2px' }}>25% Vedic revenue share</div>
        </div>
      </div>

      <div style={{ backgroundColor: '#FFFFFF', borderRadius: '12px', padding: '20px', boxShadow: '0 1px 3px rgba(0,0,0,0.05)' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
          <h3 style={{ margin: 0, fontSize: '16px', fontWeight: 700 }}>Daily Audit Trail ({filteredRows.length} Days)</h3>
          <div style={{ display: 'flex', gap: '8px' }}>
            {(['ALL', 'BALANCED', 'PENDING'] as const).map((mode) => (
              <button
                key={mode}
                onClick={() => setFilter(mode)}
                style={{
                  padding: '6px 12px',
                  borderRadius: '6px',
                  border: filter === mode ? '1px solid #831843' : '1px solid #CBD5E1',
                  backgroundColor: filter === mode ? '#831843' : '#FFFFFF',
                  color: filter === mode ? '#FFFFFF' : '#64748B',
                  fontSize: '12px',
                  fontWeight: 600,
                  cursor: 'pointer'
                }}>
                {mode}
              </button>
            ))}
          </div>
        </div>

        <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: '14px', textAlign: 'left' }}>
          <thead>
            <tr style={{ borderBottom: '2px solid #F1F5F9', color: '#64748B' }}>
              <th style={{ padding: '10px' }}>Date</th>
              <th style={{ padding: '10px' }}>Online Escrow</th>
              <th style={{ padding: '10px' }}>Counter Cash</th>
              <th style={{ padding: '10px' }}>Gross Revenue</th>
              <th style={{ padding: '10px' }}>Archaka Share (25%)</th>
              <th style={{ padding: '10px' }}>Ledger Audit</th>
            </tr>
          </thead>
          <tbody>
            {filteredRows.map((r, i) => (
              <tr key={i} style={{ borderBottom: '1px solid #F1F5F9' }}>
                <td style={{ padding: '12px 10px', fontWeight: 600 }}>{r.date}</td>
                <td style={{ padding: '12px 10px', color: '#047857', fontWeight: 600 }}>₹{r.onlineEscrow.toLocaleString('en-IN')}</td>
                <td style={{ padding: '12px 10px', color: '#831843', fontWeight: 600 }}>₹{r.cashCounter.toLocaleString('en-IN')}</td>
                <td style={{ padding: '12px 10px', fontWeight: 700 }}>₹{r.totalRevenue.toLocaleString('en-IN')}</td>
                <td style={{ padding: '12px 10px', color: '#2563EB', fontWeight: 600 }}>₹{r.archakaHonorarium.toLocaleString('en-IN')}</td>
                <td style={{ padding: '12px 10px' }}>
                  <span style={{ backgroundColor: '#D1FAE5', color: '#047857', padding: '4px 8px', borderRadius: '4px', fontSize: '11px', fontWeight: 700 }}>
                    ✓ {r.status}
                  </span>
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </div>
  );
}
