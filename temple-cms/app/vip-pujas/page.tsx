'use client';

import React, { useState } from 'react';

interface VipPuja {
  id: string;
  title: string;
  price: number;
  maxSlots: number;
  duration: number;
  isOnline: boolean;
  active: boolean;
}

export default function VipPujasManagementPage() {
  const [pujas, setPujas] = useState<VipPuja[]>([
    { id: '1', title: 'Special VIP Abhishekam & Sahasranama Archana', price: 1116, maxSlots: 20, duration: 45, isOnline: false, active: true },
    { id: '2', title: 'Live Temple E-Pooja Video Call (Two-Way Sankalpam)', price: 2116, maxSlots: 10, duration: 60, isOnline: true, active: true },
    { id: '3', title: 'Sri Lakshmi Narasimha Swamy Kalyanam VIP Pass', price: 3116, maxSlots: 15, duration: 90, isOnline: false, active: true }
  ]);

  const [editingPuja, setEditingPuja] = useState<VipPuja | null>(null);
  const [showAddModal, setShowAddModal] = useState(false);
  const [newTitle, setNewTitle] = useState('');
  const [newPrice, setNewPrice] = useState('1116');
  const [newSlots, setNewSlots] = useState('20');
  const [newDuration, setNewDuration] = useState('45');
  const [newIsOnline, setNewIsOnline] = useState(false);
  const [notification, setNotification] = useState<string | null>(null);

  const handleSaveEdit = (e: React.FormEvent) => {
    e.preventDefault();
    if (!editingPuja) return;
    setPujas(prev => prev.map(p => p.id === editingPuja.id ? editingPuja : p));
    setEditingPuja(null);
    setNotification('✅ VIP Quota & Pricing updated successfully!');
    setTimeout(() => setNotification(null), 3000);
  };

  const handleCreateNew = (e: React.FormEvent) => {
    e.preventDefault();
    if (!newTitle.trim()) return;
    const newEntry: VipPuja = {
      id: String(Date.now()),
      title: newTitle.trim(),
      price: Number(newPrice) || 1116,
      maxSlots: Number(newSlots) || 20,
      duration: Number(newDuration) || 45,
      isOnline: newIsOnline,
      active: true
    };
    setPujas(prev => [...prev, newEntry]);
    setShowAddModal(false);
    setNewTitle('');
    setNotification('✅ New VIP Seva created successfully!');
    setTimeout(() => setNotification(null), 3000);
  };

  return (
    <div>
      <div style={{ marginBottom: '24px', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
        <div>
          <h1 style={{ fontSize: '26px', fontWeight: 800, margin: '0 0 6px 0', color: '#1E293B' }}>
            🎟️ VIP Pujas & Quota Management
          </h1>
          <p style={{ margin: 0, color: '#64748B', fontSize: '14px' }}>
            Configure daily VIP ticket allocations, pricing, and live sanctum WebRTC slots.
          </p>
        </div>
        <button
          onClick={() => setShowAddModal(true)}
          style={{
            backgroundColor: '#831843',
            color: '#FFFFFF',
            border: 'none',
            padding: '10px 18px',
            borderRadius: '8px',
            fontWeight: 700,
            cursor: 'pointer'
          }}>
          + Add New VIP Seva
        </button>
      </div>

      {notification && (
        <div style={{
          backgroundColor: '#ECFDF5',
          border: '1px solid #10B981',
          color: '#065F46',
          padding: '10px 16px',
          borderRadius: '8px',
          marginBottom: '16px',
          fontSize: '14px',
          fontWeight: 600
        }}>
          {notification}
        </div>
      )}

      <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: '16px' }}>
        {pujas.map((p) => (
          <div key={p.id} style={{
            backgroundColor: '#FFFFFF',
            borderRadius: '12px',
            padding: '20px',
            boxShadow: '0 1px 3px rgba(0,0,0,0.05)',
            borderTop: '4px solid #831843'
          }}>
            <h3 style={{ margin: '0 0 8px 0', fontSize: '16px', fontWeight: 700, color: '#0F172A' }}>{p.title}</h3>
            <div style={{ fontSize: '13px', color: '#64748B', marginBottom: '12px' }}>
              Duration: {p.duration} mins • {p.isOnline ? '📹 WebRTC Live E-Pooja' : '🏛️ In-Person Sanctum'}
            </div>

            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', backgroundColor: '#F8FAFC', padding: '10px 12px', borderRadius: '8px', marginBottom: '12px' }}>
              <div>
                <span style={{ fontSize: '11px', color: '#64748B' }}>Ticket Price:</span>
                <div style={{ fontSize: '18px', fontWeight: 800, color: '#831843' }}>₹{p.price}</div>
              </div>
              <div>
                <span style={{ fontSize: '11px', color: '#64748B' }}>Daily Cap:</span>
                <div style={{ fontSize: '18px', fontWeight: 800, color: '#0F172A' }}>{p.maxSlots} slots</div>
              </div>
            </div>

            <button
              onClick={() => setEditingPuja(p)}
              style={{
                width: '100%',
                padding: '8px',
                backgroundColor: '#F1F5F9',
                border: '1px solid #CBD5E1',
                borderRadius: '6px',
                fontSize: '13px',
                fontWeight: 600,
                cursor: 'pointer'
              }}>
              Edit Quota & Pricing
            </button>
          </div>
        ))}
      </div>

      {/* Edit Modal */}
      {editingPuja && (
        <div style={{
          position: 'fixed',
          top: 0,
          left: 0,
          right: 0,
          bottom: 0,
          backgroundColor: 'rgba(0,0,0,0.5)',
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
          zIndex: 1000
        }}>
          <div style={{
            backgroundColor: '#FFFFFF',
            borderRadius: '12px',
            padding: '24px',
            width: '420px',
            boxShadow: '0 20px 25px -5px rgba(0,0,0,0.1)'
          }}>
            <h2 style={{ margin: '0 0 16px 0', fontSize: '18px', fontWeight: 800 }}>Edit Quota & Pricing</h2>
            <form onSubmit={handleSaveEdit}>
              <div style={{ marginBottom: '12px' }}>
                <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: '#64748B', marginBottom: '4px' }}>Seva Title</label>
                <input
                  type="text"
                  value={editingPuja.title}
                  onChange={(e) => setEditingPuja({ ...editingPuja, title: e.target.value })}
                  style={{ width: '100%', padding: '8px 12px', borderRadius: '6px', border: '1px solid #CBD5E1', fontSize: '14px' }}
                  required
                />
              </div>
              <div style={{ display: 'flex', gap: '12px', marginBottom: '12px' }}>
                <div style={{ flex: 1 }}>
                  <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: '#64748B', marginBottom: '4px' }}>Ticket Price (₹)</label>
                  <input
                    type="number"
                    value={editingPuja.price}
                    onChange={(e) => setEditingPuja({ ...editingPuja, price: Number(e.target.value) })}
                    style={{ width: '100%', padding: '8px 12px', borderRadius: '6px', border: '1px solid #CBD5E1', fontSize: '14px' }}
                    required
                  />
                </div>
                <div style={{ flex: 1 }}>
                  <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: '#64748B', marginBottom: '4px' }}>Daily Slot Cap</label>
                  <input
                    type="number"
                    value={editingPuja.maxSlots}
                    onChange={(e) => setEditingPuja({ ...editingPuja, maxSlots: Number(e.target.value) })}
                    style={{ width: '100%', padding: '8px 12px', borderRadius: '6px', border: '1px solid #CBD5E1', fontSize: '14px' }}
                    required
                  />
                </div>
              </div>
              <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '8px', marginTop: '20px' }}>
                <button
                  type="button"
                  onClick={() => setEditingPuja(null)}
                  style={{ padding: '8px 14px', borderRadius: '6px', border: '1px solid #CBD5E1', backgroundColor: '#FFFFFF', cursor: 'pointer', fontWeight: 600 }}>
                  Cancel
                </button>
                <button
                  type="submit"
                  style={{ padding: '8px 16px', borderRadius: '6px', border: 'none', backgroundColor: '#831843', color: '#FFFFFF', cursor: 'pointer', fontWeight: 700 }}>
                  Save Changes
                </button>
              </div>
            </form>
          </div>
        </div>
      )}

      {/* Add Modal */}
      {showAddModal && (
        <div style={{
          position: 'fixed',
          top: 0,
          left: 0,
          right: 0,
          bottom: 0,
          backgroundColor: 'rgba(0,0,0,0.5)',
          display: 'flex',
          justifyContent: 'center',
          alignItems: 'center',
          zIndex: 1000
        }}>
          <div style={{
            backgroundColor: '#FFFFFF',
            borderRadius: '12px',
            padding: '24px',
            width: '420px',
            boxShadow: '0 20px 25px -5px rgba(0,0,0,0.1)'
          }}>
            <h2 style={{ margin: '0 0 16px 0', fontSize: '18px', fontWeight: 800 }}>Add New VIP Seva</h2>
            <form onSubmit={handleCreateNew}>
              <div style={{ marginBottom: '12px' }}>
                <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: '#64748B', marginBottom: '4px' }}>Seva Title</label>
                <input
                  type="text"
                  placeholder="e.g. Suprabhata Seva VIP Darshanam"
                  value={newTitle}
                  onChange={(e) => setNewTitle(e.target.value)}
                  style={{ width: '100%', padding: '8px 12px', borderRadius: '6px', border: '1px solid #CBD5E1', fontSize: '14px' }}
                  required
                />
              </div>
              <div style={{ display: 'flex', gap: '12px', marginBottom: '12px' }}>
                <div style={{ flex: 1 }}>
                  <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: '#64748B', marginBottom: '4px' }}>Price (₹)</label>
                  <input
                    type="number"
                    value={newPrice}
                    onChange={(e) => setNewPrice(e.target.value)}
                    style={{ width: '100%', padding: '8px 12px', borderRadius: '6px', border: '1px solid #CBD5E1', fontSize: '14px' }}
                    required
                  />
                </div>
                <div style={{ flex: 1 }}>
                  <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: '#64748B', marginBottom: '4px' }}>Daily Cap</label>
                  <input
                    type="number"
                    value={newSlots}
                    onChange={(e) => setNewSlots(e.target.value)}
                    style={{ width: '100%', padding: '8px 12px', borderRadius: '6px', border: '1px solid #CBD5E1', fontSize: '14px' }}
                    required
                  />
                </div>
              </div>
              <div style={{ marginBottom: '16px' }}>
                <label style={{ display: 'flex', alignItems: 'center', gap: '8px', fontSize: '13px', cursor: 'pointer' }}>
                  <input
                    type="checkbox"
                    checked={newIsOnline}
                    onChange={(e) => setNewIsOnline(e.target.checked)}
                  />
                  Live Temple E-Pooja (WebRTC Video Stream)
                </label>
              </div>
              <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '8px' }}>
                <button
                  type="button"
                  onClick={() => setShowAddModal(false)}
                  style={{ padding: '8px 14px', borderRadius: '6px', border: '1px solid #CBD5E1', backgroundColor: '#FFFFFF', cursor: 'pointer', fontWeight: 600 }}>
                  Cancel
                </button>
                <button
                  type="submit"
                  style={{ padding: '8px 16px', borderRadius: '6px', border: 'none', backgroundColor: '#831843', color: '#FFFFFF', cursor: 'pointer', fontWeight: 700 }}>
                  Create Seva
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
}
