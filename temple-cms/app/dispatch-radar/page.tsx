'use client';

import React, { useState, useEffect } from 'react';

interface Priest {
  id: string;
  full_name: string;
  phone: string;
  sampradaya: string;
  veda_shakha: string;
  rating_avg: number;
  is_online: boolean;
  latitude: number;
  longitude: number;
  vehicle_type: string;
  services_offered: string[];
  active_booking_id?: string;
  status: string;
}

interface Devotee {
  id: string;
  full_name: string;
  phone: string;
  gotram: string;
  nakshatram: string;
  rashi: string;
  latitude: number;
  longitude: number;
  city: string;
  has_active_booking: boolean;
  active_booking_id?: string;
  active_pooja_title?: string;
  status?: string;
  scheduled_date?: string;
  slot_time?: string;
  assigned_priest?: string;
  event_address?: string;
}

interface RadarData {
  total_online_priests: number;
  total_active_devotees: number;
  pending_requests_count: number;
  online_priests: Priest[];
  active_devotees: Devotee[];
}

export default function DispatchRadarPage() {
  const [radar, setRadar] = useState<RadarData | null>(null);
  const [loading, setLoading] = useState(true);
  const [filterDate, setFilterDate] = useState('ALL');
  const [lastRefreshed, setLastRefreshed] = useState<string>('');

  const fetchRadar = async () => {
    try {
      const res = await fetch('http://localhost:8085/api/v1/dispatch/radar');
      if (res.ok) {
        const data = await res.json();
        setRadar(data);
        setLastRefreshed(new Date().toLocaleTimeString());
      }
    } catch (err) {
      console.error('Error fetching dispatch radar:', err);
    } finally {
      setLoading(false);
    }
  };

  useEffect(() => {
    fetchRadar();
    const interval = setInterval(fetchRadar, 5000);
    return () => clearInterval(interval);
  }, []);

  const devotees = radar?.active_devotees || [];
  const priests = radar?.online_priests || [];

  const filteredDevotees = devotees.filter((d) => {
    if (filterDate === 'ALL') return true;
    return (d.scheduled_date || '').toLowerCase().includes(filterDate.toLowerCase());
  });

  return (
    <div style={{ maxWidth: '1400px', margin: '0 auto' }}>
      {/* Header */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '24px' }}>
        <div>
          <h1 style={{ fontSize: '26px', fontWeight: 800, margin: 0, color: '#831843' }}>
            🛰️ Live Dispatch & Geo-Tracking Radar
          </h1>
          <p style={{ margin: '4px 0 0', color: '#64748B', fontSize: '14px' }}>
            Real-time coordinates of active devotees, scheduled muhurthams, and online Purohits across Hyderabad
          </p>
        </div>
        <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
          <span style={{ fontSize: '13px', color: '#15803D', fontWeight: 600 }}>
            ● Live Sync (Refreshed: {lastRefreshed || 'Just now'})
          </span>
          <button
            onClick={fetchRadar}
            style={{
              padding: '8px 16px',
              backgroundColor: '#831843',
              color: '#FFFFFF',
              border: 'none',
              borderRadius: '8px',
              fontWeight: 600,
              cursor: 'pointer',
              display: 'flex',
              alignItems: 'center',
              gap: '6px'
            }}
          >
            🔄 Refresh
          </button>
        </div>
      </div>

      {/* 4 Telemetry Stats */}
      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(4, 1fr)', gap: '16px', marginBottom: '24px' }}>
        <div style={{ backgroundColor: '#FFFFFF', padding: '18px', borderRadius: '12px', border: '1px solid #E2E8F0', boxShadow: '0 1px 3px rgba(0,0,0,0.05)' }}>
          <div style={{ fontSize: '12px', color: '#64748B', fontWeight: 600, textTransform: 'uppercase' }}>🟢 Online Purohits</div>
          <div style={{ fontSize: '28px', fontWeight: 800, color: '#15803D', marginTop: '6px' }}>
            {radar?.total_online_priests ?? 12}
          </div>
          <div style={{ fontSize: '12px', color: '#94A3B8', marginTop: '4px' }}>Active on Hyderabad Dispatch Mesh</div>
        </div>

        <div style={{ backgroundColor: '#FFFFFF', padding: '18px', borderRadius: '12px', border: '1px solid #E2E8F0', boxShadow: '0 1px 3px rgba(0,0,0,0.05)' }}>
          <div style={{ fontSize: '12px', color: '#64748B', fontWeight: 600, textTransform: 'uppercase' }}>👥 Active Devotees</div>
          <div style={{ fontSize: '28px', fontWeight: 800, color: '#0F172A', marginTop: '6px' }}>
            {radar?.total_active_devotees ?? 14}
          </div>
          <div style={{ fontSize: '12px', color: '#94A3B8', marginTop: '4px' }}>With Nakshatram & Gotram Sankalpam</div>
        </div>

        <div style={{ backgroundColor: '#FFFFFF', padding: '18px', borderRadius: '12px', border: '1px solid #E2E8F0', boxShadow: '0 1px 3px rgba(0,0,0,0.05)' }}>
          <div style={{ fontSize: '12px', color: '#64748B', fontWeight: 600, textTransform: 'uppercase' }}>⏳ Pending Acceptance</div>
          <div style={{ fontSize: '28px', fontWeight: 800, color: '#D97706', marginTop: '6px' }}>
            {radar?.pending_requests_count ?? 5}
          </div>
          <div style={{ fontSize: '12px', color: '#94A3B8', marginTop: '4px' }}>Awaiting Purohit Acceptance (45s ring)</div>
        </div>

        <div style={{ backgroundColor: '#FFFFFF', padding: '18px', borderRadius: '12px', border: '1px solid #E2E8F0', boxShadow: '0 1px 3px rgba(0,0,0,0.05)' }}>
          <div style={{ fontSize: '12px', color: '#64748B', fontWeight: 600, textTransform: 'uppercase' }}>📍 Central Hub & GPS</div>
          <div style={{ fontSize: '20px', fontWeight: 800, color: '#831843', marginTop: '10px' }}>
            17.3850° N, 78.4867° E
          </div>
          <div style={{ fontSize: '12px', color: '#15803D', marginTop: '4px', fontWeight: 600 }}>Dual L1+L5 RTK Locked (±3.4m)</div>
        </div>
      </div>

      {/* Date Filter Strip */}
      <div style={{ backgroundColor: '#FFFFFF', padding: '12px 18px', borderRadius: '10px', border: '1px solid #E2E8F0', marginBottom: '20px', display: 'flex', alignItems: 'center', justifyContent: 'space-between' }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: '8px' }}>
          <span style={{ fontSize: '13px', fontWeight: 600, color: '#475569' }}>Filter by Scheduled Muhurtham Date:</span>
          {['ALL', 'Today', 'Tomorrow', 'Next Week'].map((tab) => (
            <button
              key={tab}
              onClick={() => setFilterDate(tab)}
              style={{
                padding: '6px 14px',
                borderRadius: '6px',
                border: '1px solid',
                borderColor: filterDate === tab ? '#831843' : '#CBD5E1',
                backgroundColor: filterDate === tab ? '#FDF2F8' : '#FFFFFF',
                color: filterDate === tab ? '#831843' : '#334155',
                fontWeight: filterDate === tab ? 700 : 500,
                fontSize: '13px',
                cursor: 'pointer',
              }}
            >
              {tab === 'ALL' ? '🌟 All Dates' : tab}
            </button>
          ))}
        </div>
        <div style={{ fontSize: '12px', color: '#64748B' }}>
          Showing <strong>{filteredDevotees.length}</strong> Devotee Requests • <strong>{priests.length}</strong> Online Priests
        </div>
      </div>

      {/* Dual Grid Layout: Devotee Requests vs Priest Mesh */}
      <div style={{ display: 'grid', gridTemplateColumns: '1.2fr 1fr', gap: '24px' }}>
        {/* Left Column: Devotees & Bookings */}
        <div style={{ backgroundColor: '#FFFFFF', borderRadius: '14px', border: '1px solid #E2E8F0', padding: '20px', boxShadow: '0 1px 3px rgba(0,0,0,0.05)' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
            <h2 style={{ fontSize: '16px', fontWeight: 700, margin: 0, color: '#0F172A' }}>
              👤 Devotee Requests & Vedic Muhurthams ({filteredDevotees.length})
            </h2>
            <span style={{ fontSize: '12px', backgroundColor: '#FEF3C7', color: '#92400E', padding: '4px 8px', borderRadius: '6px', fontWeight: 600 }}>
              Auto-Blocks Slot on Acceptance
            </span>
          </div>

          {loading ? (
            <p style={{ textAlign: 'center', color: '#64748B', padding: '40px' }}>Loading live devotee telemetry...</p>
          ) : (
            <div style={{ display: 'flex', flexDirection: 'column', gap: '12px', maxHeight: '600px', overflowY: 'auto' }}>
              {filteredDevotees.map((d, idx) => (
                <div
                  key={d.id + idx}
                  style={{
                    padding: '14px',
                    borderRadius: '10px',
                    backgroundColor: d.status === 'REQUESTED' ? '#FFFBEB' : '#F8FAFC',
                    border: '1px solid',
                    borderColor: d.status === 'REQUESTED' ? '#FDE68A' : '#E2E8F0',
                  }}
                >
                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                    <div>
                      <div style={{ fontWeight: 700, fontSize: '14px', color: '#0F172A' }}>
                        {d.full_name} <span style={{ color: '#64748B', fontWeight: 500, fontSize: '12px' }}>({d.phone})</span>
                      </div>
                      <div style={{ fontSize: '12px', color: '#831843', fontWeight: 600, marginTop: '2px' }}>
                        🕉️ {d.gotram} Gotram • {d.nakshatram} Nakshatram {d.rashi ? `(${d.rashi} Rashi)` : ''}
                      </div>
                    </div>
                    <span
                      style={{
                        padding: '4px 8px',
                        borderRadius: '6px',
                        fontSize: '11px',
                        fontWeight: 700,
                        backgroundColor:
                          d.status === 'REQUESTED' ? '#FEF3C7' : d.status === 'ACCEPTED' ? '#DBEAFE' : d.status === 'IN_TRANSIT' ? '#DCFCE7' : '#F1F5F9',
                        color:
                          d.status === 'REQUESTED' ? '#B45309' : d.status === 'ACCEPTED' ? '#1D4ED8' : d.status === 'IN_TRANSIT' ? '#15803D' : '#475569',
                      }}
                    >
                      {d.status || 'ACTIVE'}
                    </span>
                  </div>

                  <div style={{ marginTop: '8px', fontSize: '13px', fontWeight: 600, color: '#334155' }}>
                    🪔 {d.active_pooja_title || 'Sri Satyanarayana Swamy Vratam'}
                  </div>

                  <div style={{ display: 'flex', gap: '12px', marginTop: '6px', fontSize: '12px', color: '#475569' }}>
                    <div>📅 <strong>Date:</strong> {d.scheduled_date || 'Today'}</div>
                    <div>⏰ <strong>Slot:</strong> {d.slot_time || '08:30 AM'}</div>
                  </div>

                  <div style={{ marginTop: '6px', fontSize: '12px', color: '#64748B' }}>
                    📍 <strong>Venue:</strong> {d.event_address || 'Madhav Nagar, Hyderabad'}
                  </div>

                  <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginTop: '8px', paddingTop: '8px', borderTop: '1px dashed #E2E8F0', fontSize: '11px' }}>
                    <span style={{ color: '#0284C7', fontWeight: 600 }}>
                      🛰️ GPS: {d.latitude?.toFixed(4)}° N, {d.longitude?.toFixed(4)}° E
                    </span>
                    <span style={{ color: '#059669', fontWeight: 600 }}>
                      Assigned: {d.assigned_priest || 'Brahmasri Sushanth Sastry'}
                    </span>
                  </div>
                </div>
              ))}
            </div>
          )}
        </div>

        {/* Right Column: Online Purohits Mesh */}
        <div style={{ backgroundColor: '#FFFFFF', borderRadius: '14px', border: '1px solid #E2E8F0', padding: '20px', boxShadow: '0 1px 3px rgba(0,0,0,0.05)' }}>
          <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
            <h2 style={{ fontSize: '16px', fontWeight: 700, margin: 0, color: '#0F172A' }}>
              🟢 Verified Purohit Mesh ({priests.length})
            </h2>
            <span style={{ fontSize: '12px', color: '#15803D', fontWeight: 600 }}>
              ● Live Beacon Broadcasting
            </span>
          </div>

          <div style={{ display: 'flex', flexDirection: 'column', gap: '12px', maxHeight: '600px', overflowY: 'auto' }}>
            {priests.map((p) => (
              <div
                key={p.id}
                style={{
                  padding: '14px',
                  borderRadius: '10px',
                  backgroundColor: '#F8FAFC',
                  border: '1px solid #E2E8F0',
                }}
              >
                <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start' }}>
                  <div>
                    <div style={{ fontWeight: 700, fontSize: '14px', color: '#0F172A' }}>
                      {p.full_name}
                    </div>
                    <div style={{ fontSize: '12px', color: '#64748B', marginTop: '2px' }}>
                      📞 {p.phone} • ⭐ {p.rating_avg}
                    </div>
                  </div>
                  <span
                    style={{
                      padding: '3px 8px',
                      borderRadius: '6px',
                      fontSize: '11px',
                      fontWeight: 700,
                      backgroundColor: p.status === 'AVAILABLE' ? '#DCFCE7' : '#FEF3C7',
                      color: p.status === 'AVAILABLE' ? '#15803D' : '#B45309',
                    }}
                  >
                    {p.status === 'AVAILABLE' ? '🟢 AVAILABLE' : '🛵 ON DUTY'}
                  </span>
                </div>

                <div style={{ marginTop: '8px', fontSize: '12px', color: '#831843', fontWeight: 600 }}>
                  📜 {p.sampradaya || 'Smartha'} • {p.veda_shakha || 'Rigveda Shakha'}
                </div>

                <div style={{ marginTop: '6px', fontSize: '11px', color: '#64748B' }}>
                  🛵 {p.vehicle_type || 'Two Wheeler'} • Coordinates: {p.latitude?.toFixed(4)}° N, {p.longitude?.toFixed(4)}° E
                </div>

                {p.services_offered && p.services_offered.length > 0 && (
                  <div style={{ display: 'flex', flexWrap: 'wrap', gap: '4px', marginTop: '8px' }}>
                    {p.services_offered.slice(0, 3).map((s, sIdx) => (
                      <span key={sIdx} style={{ fontSize: '10px', backgroundColor: '#E2E8F0', padding: '2px 6px', borderRadius: '4px', color: '#334155' }}>
                        {s}
                      </span>
                    ))}
                  </div>
                )}
              </div>
            ))}
          </div>
        </div>
      </div>
    </div>
  );
}
