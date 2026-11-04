'use client';

import React from 'react';

export default function DashboardPage() {
  const stats = [
    { label: "Today's VIP Bookings", val: "148 Passes", change: "+18% from yesterday", color: "#831843" },
    { label: "Online Escrow Collected", val: "₹1,65,168", change: "100% reconciled", color: "#047857" },
    { label: "Gate Scans Processed", val: "124 Devotees", change: "Avg scan: 22ms", color: "#2563EB" },
    { label: "Live Sanctum Viewers", val: "1,840 CCU", change: "WebRTC broadcast active", color: "#D97706" }
  ];

  const recentBookings = [
    { name: "Smt. Lakshmi Prasanna", gotram: "Kashyapa", seva: "Special VIP Abhishekam & Sahasranama Archana", slot: "08:00 AM - 09:00 AM", status: "CHECKED_IN", amount: "₹1,116" },
    { name: "Srikanth Sharma", gotram: "Bharadwaja", seva: "Live Temple E-Pooja Video Call", slot: "10:00 AM - 11:00 AM", status: "CONFIRMED", amount: "₹2,116" },
    { name: "Venkata Ramanayya", gotram: "Haritasa", seva: "Special VIP Abhishekam & Sahasranama Archana", slot: "11:30 AM - 12:30 PM", status: "CONFIRMED", amount: "₹1,116" },
    { name: "Ananya Rao", gotram: "Vasishta", seva: "Special VIP Abhishekam & Sahasranama Archana", slot: "04:30 PM - 05:30 PM", status: "CONFIRMED", amount: "₹1,116" }
  ];

  return (
    <div>
      <div style={{ marginBottom: '28px' }}>
        <h1 style={{ fontSize: '26px', fontWeight: 800, margin: '0 0 6px 0', color: '#1E293B' }}>
          Temple Operations Dashboard
        </h1>
        <p style={{ margin: 0, color: '#64748B', fontSize: '14px' }}>
          Dhapurmari Sri Lakshmi Narasimha Swamy Temple • Real-time Sanctum & Gate Telemetry
        </p>
      </div>

      {/* Metrics Row */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '18px', marginBottom: '32px' }}>
        {stats.map((s, idx) => (
          <div key={idx} style={{
            backgroundColor: '#FFFFFF',
            padding: '20px',
            borderRadius: '12px',
            boxShadow: '0 1px 3px rgba(0,0,0,0.05)',
            borderTop: `4px solid ${s.color}`
          }}>
            <div style={{ fontSize: '13px', color: '#64748B', fontWeight: 600 }}>{s.label}</div>
            <div style={{ fontSize: '24px', fontWeight: 800, color: '#0F172A', marginTop: '6px' }}>{s.val}</div>
            <div style={{ fontSize: '12px', color: s.color, marginTop: '4px', fontWeight: 600 }}>{s.change}</div>
          </div>
        ))}
      </div>

      {/* Recent Activity Table */}
      <div style={{ backgroundColor: '#FFFFFF', borderRadius: '12px', padding: '24px', boxShadow: '0 1px 3px rgba(0,0,0,0.05)' }}>
        <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
          <h2 style={{ fontSize: '18px', fontWeight: 700, margin: 0 }}>Recent VIP Darshan Bookings</h2>
          <span style={{ fontSize: '12px', color: '#059669', fontWeight: 600 }}>● Live Auto-Sync Active</span>
        </div>

        <table style={{ width: '100%', borderCollapse: 'collapse', textAlign: 'left', fontSize: '14px' }}>
          <thead>
            <tr style={{ borderBottom: '2px solid #F1F5F9', color: '#64748B' }}>
              <th style={{ padding: '12px 8px' }}>Devotee Name</th>
              <th style={{ padding: '12px 8px' }}>Gotram</th>
              <th style={{ padding: '12px 8px' }}>Seva / Puja</th>
              <th style={{ padding: '12px 8px' }}>Slot Time</th>
              <th style={{ padding: '12px 8px' }}>Dakshina Amount</th>
              <th style={{ padding: '12px 8px' }}>Status</th>
            </tr>
          </thead>
          <tbody>
            {recentBookings.map((b, i) => (
              <tr key={i} style={{ borderBottom: '1px solid #F1F5F9' }}>
                <td style={{ padding: '12px 8px', fontWeight: 600 }}>{b.name}</td>
                <td style={{ padding: '12px 8px', color: '#64748B' }}>{b.gotram}</td>
                <td style={{ padding: '12px 8px' }}>{b.seva}</td>
                <td style={{ padding: '12px 8px', color: '#475569' }}>{b.slot}</td>
                <td style={{ padding: '12px 8px', fontWeight: 700, color: '#831843' }}>{b.amount}</td>
                <td style={{ padding: '12px 8px' }}>
                  <span style={{
                    padding: '4px 10px',
                    borderRadius: '20px',
                    fontSize: '11px',
                    fontWeight: 700,
                    backgroundColor: b.status === 'CHECKED_IN' ? '#D1FAE5' : '#FEF3C7',
                    color: b.status === 'CHECKED_IN' ? '#047857' : '#B45309'
                  }}>
                    {b.status}
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
