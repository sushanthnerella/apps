'use client';

import React, { useState } from 'react';

export default function SanctumStudioPage() {
  const [isBroadcasting, setIsBroadcasting] = useState(false);
  const [audioConnected, setAudioConnected] = useState(false);

  return (
    <div style={{ maxWidth: '900px' }}>
      <div style={{ marginBottom: '24px' }}>
        <h1 style={{ fontSize: '26px', fontWeight: 800, margin: '0 0 6px 0', color: '#1E293B' }}>
          📹 Sanctum Live E-Pooja Studio
        </h1>
        <p style={{ margin: 0, color: '#64748B', fontSize: '14px' }}>
          Real-time WebRTC 2-way broadcast from the sanctum sanctorum for scheduled VIP devotees.
        </p>
      </div>

      <div style={{ backgroundColor: '#0F172A', borderRadius: '16px', overflow: 'hidden', padding: '24px', color: '#FFFFFF', marginBottom: '24px' }}>
        <div style={{ height: '360px', backgroundColor: '#1E293B', borderRadius: '12px', display: 'flex', flexDirection: 'column', alignItems: 'center', justifyContent: 'center' }}>
          <span style={{ fontSize: '48px', marginBottom: '12px' }}>🪔</span>
          <div style={{ fontSize: '18px', fontWeight: 700 }}>
            {isBroadcasting ? '● LIVE SANCTUM BROADCAST ACTIVE' : 'Sanctum Camera Feed Standby'}
          </div>
          <div style={{ fontSize: '13px', color: '#94A3B8', marginTop: '4px' }}>
            {isBroadcasting ? '1080p60 • WebRTC STUN/TURN Connected • Opus Audio Chanting 48kHz' : 'Click "Start Sanctum Stream" to begin live transmission'}
          </div>
        </div>

        <div style={{ marginTop: '20px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <div style={{ display: 'flex', gap: '12px' }}>
            <button
              onClick={() => setIsBroadcasting(!isBroadcasting)}
              style={{
                backgroundColor: isBroadcasting ? '#EF4444' : '#10B981',
                color: '#FFFFFF',
                border: 'none',
                padding: '12px 24px',
                borderRadius: '8px',
                fontWeight: 700,
                fontSize: '14px',
                cursor: 'pointer'
              }}
            >
              {isBroadcasting ? 'End Broadcast' : '▶ Start Sanctum Stream'}
            </button>
            <button
              onClick={() => setAudioConnected(!audioConnected)}
              style={{
                backgroundColor: audioConnected ? '#2563EB' : '#334155',
                color: '#FFFFFF',
                border: 'none',
                padding: '12px 20px',
                borderRadius: '8px',
                fontWeight: 600,
                fontSize: '14px',
                cursor: 'pointer'
              }}
            >
              {audioConnected ? '🎤 2-Way Sankalpam Audio Active' : 'Enable Devotee Sankalpam Audio'}
            </button>
          </div>

          <div style={{ fontSize: '13px', color: '#94A3B8' }}>
            Connected Devotee: <strong style={{ color: '#FFFFFF' }}>Smt. Lakshmi Prasanna</strong> (Kashyapa Gotram)
          </div>
        </div>
      </div>
    </div>
  );
}
