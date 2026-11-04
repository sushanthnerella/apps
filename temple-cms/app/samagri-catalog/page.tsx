'use client';

import React, { useState, useEffect } from 'react';

interface SamagriItem {
  id: string;
  pooja_id: string;
  item_name: string;
  category: string;
  quantity: string;
  is_mandatory: boolean;
  estimated_price: number;
}

const POOJAS = [
  { id: '994c148d-a091-4531-8628-ed78f71d075f', title: '🏡 Gruhapravesam (House Warming)' },
  { id: '5dccda01-fcb9-4bc2-9ef9-6015419533bc', title: '✨ Sri Satyanarayana Swamy Vratam' },
  { id: '91b8ca3e-dfbc-4626-8bca-c2c64123b94e', title: '🕉️ Maha Ganapathi Homa / Yagam' },
  { id: 'e81975cc-49bd-497c-bc4c-a4d3153cc4e0', title: '👶 Namakaranam & Ayushya Homa' },
  { id: '61b03f23-c57b-443f-9e2c-f256ebdd3720', title: '🔱 Mrityunjaya Homa / Shanti' },
  { id: 'b0440927-e9a4-481b-87b5-351708baebb0', title: '☀️ Navagraha Shanti Homa' }
];

export default function SamagriCatalogPage() {
  const [selectedPoojaId, setSelectedPoojaId] = useState(POOJAS[0].id);
  const [items, setItems] = useState<SamagriItem[]>([]);
  const [isLoading, setIsLoading] = useState(false);
  const [showAddModal, setShowAddModal] = useState(false);

  // New item form state
  const [newItemName, setNewItemName] = useState('');
  const [newItemCategory, setNewItemCategory] = useState('Devotee Household Items');
  const [newItemQuantity, setNewItemQuantity] = useState('1 set');
  const [newItemPrice, setNewItemPrice] = useState(0);
  const [isMandatory, setIsMandatory] = useState(true);

  const fetchItems = async (poojaId: string) => {
    setIsLoading(true);
    try {
      const res = await fetch(`http://localhost:8085/api/v1/poojas/${poojaId}/samagri`, {
        headers: { 'ngrok-skip-browser-warning': '1' }
      });
      const data = await res.json();
      setItems(Array.isArray(data) ? data : []);
    } catch (e) {
      console.error('Failed to load items:', e);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    fetchItems(selectedPoojaId);
  }, [selectedPoojaId]);

  const handleAddItem = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!newItemName.trim()) return;

    try {
      const res = await fetch(`http://localhost:8085/api/v1/poojas/${selectedPoojaId}/samagri`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json', 'ngrok-skip-browser-warning': '1' },
        body: JSON.stringify({
          pooja_id: selectedPoojaId,
          item_name: newItemName,
          category: newItemCategory,
          quantity: newItemQuantity,
          is_mandatory: isMandatory,
          estimated_price: Number(newItemPrice)
        })
      });
      if (res.ok) {
        setShowAddModal(false);
        setNewItemName('');
        setNewItemQuantity('1 set');
        setNewItemPrice(0);
        fetchItems(selectedPoojaId);
      }
    } catch (e) {
      console.error('Failed to add item:', e);
    }
  };

  const handleDeleteItem = async (itemId: string) => {
    if (!confirm('Are you sure you want to remove this item from the standard checklist?')) return;
    try {
      const res = await fetch(`http://localhost:8085/api/v1/poojas/samagri/${itemId}`, {
        method: 'DELETE',
        headers: { 'ngrok-skip-browser-warning': '1' }
      });
      if (res.ok) {
        fetchItems(selectedPoojaId);
      }
    } catch (e) {
      console.error('Failed to delete item:', e);
    }
  };

  const householdItems = items.filter(it => it.category === 'Devotee Household Items');
  const sacredItems = items.filter(it => it.category === 'Sacred Ritual Samagri');

  return (
    <main style={{ padding: '32px', maxWidth: '1200px', margin: '0 auto', width: '100%' }}>
      {/* Page Header */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'flex-start', marginBottom: '24px' }}>
        <div>
          <h1 style={{ fontSize: '26px', fontWeight: 800, margin: '0 0 8px 0', color: '#831843' }}>
            📦 Vedic Samagri & Household Checklist Manager
          </h1>
          <p style={{ margin: 0, color: '#64748B', fontSize: '14px' }}>
            Configure standard Devotee household essentials and Sacred Purohit Samagri kits for all rituals.
          </p>
        </div>
        <button
          onClick={() => setShowAddModal(true)}
          style={{
            backgroundColor: '#831843',
            color: '#FFFFFF',
            border: 'none',
            borderRadius: '8px',
            padding: '10px 18px',
            fontWeight: 700,
            cursor: 'pointer',
            boxShadow: '0 2px 4px rgba(0,0,0,0.1)'
          }}
        >
          ➕ Add Standard Item
        </button>
      </div>

      {/* Pooja Service Selector Tabs */}
      <div style={{ display: 'flex', gap: '8px', flexWrap: 'wrap', marginBottom: '24px' }}>
        {POOJAS.map(p => {
          const isSelected = p.id === selectedPoojaId;
          return (
            <button
              key={p.id}
              onClick={() => setSelectedPoojaId(p.id)}
              style={{
                padding: '8px 16px',
                borderRadius: '20px',
                border: isSelected ? '2px solid #831843' : '1px solid #CBD5E1',
                backgroundColor: isSelected ? '#FDF2F8' : '#FFFFFF',
                color: isSelected ? '#831843' : '#334155',
                fontWeight: isSelected ? 700 : 500,
                cursor: 'pointer',
                fontSize: '13px'
              }}
            >
              {p.title}
            </button>
          );
        })}
      </div>

      {/* Two Column Layout: Household Items vs Sacred Samagri */}
      {isLoading ? (
        <div style={{ padding: '40px', textAlign: 'center', color: '#64748B' }}>Loading standard checklist...</div>
      ) : (
        <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '24px' }}>
          
          {/* Column 1: Devotee Household Items */}
          <div style={{ backgroundColor: '#FFFFFF', borderRadius: '12px', border: '1px solid #E2E8F0', padding: '20px' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
              <div>
                <h2 style={{ fontSize: '18px', fontWeight: 700, color: '#B45309', margin: 0 }}>
                  🏡 Devotee Household Items ({householdItems.length})
                </h2>
                <span style={{ fontSize: '12px', color: '#64748B' }}>
                  Standard items to be arranged by Devotee at home before Purohit arrives
                </span>
              </div>
              <span style={{ backgroundColor: '#FEF3C7', color: '#92400E', padding: '4px 10px', borderRadius: '12px', fontSize: '12px', fontWeight: 700 }}>
                Devotee Home
              </span>
            </div>

            <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
              {householdItems.map(item => (
                <div
                  key={item.id}
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    padding: '12px 14px',
                    borderRadius: '8px',
                    backgroundColor: '#FFFBEB',
                    border: '1px solid #FDE68A'
                  }}
                >
                  <div>
                    <div style={{ fontWeight: 600, fontSize: '14px', color: '#78350F' }}>{item.item_name}</div>
                    <div style={{ fontSize: '12px', color: '#92400E' }}>
                      Quantity: <strong>{item.quantity}</strong> {item.is_mandatory && '• Mandatory'}
                    </div>
                  </div>
                  <button
                    onClick={() => handleDeleteItem(item.id)}
                    style={{ background: 'none', border: 'none', color: '#DC2626', cursor: 'pointer', fontSize: '16px' }}
                    title="Delete item"
                  >
                    🗑️
                  </button>
                </div>
              ))}
              {householdItems.length === 0 && (
                <div style={{ padding: '20px', textAlign: 'center', color: '#94A3B8' }}>No household items configured.</div>
              )}
            </div>
          </div>

          {/* Column 2: Sacred Ritual Samagri */}
          <div style={{ backgroundColor: '#FFFFFF', borderRadius: '12px', border: '1px solid #E2E8F0', padding: '20px' }}>
            <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '16px' }}>
              <div>
                <h2 style={{ fontSize: '18px', fontWeight: 700, color: '#15803D', margin: 0 }}>
                  🪔 Sacred Ritual Samagri ({sacredItems.length})
                </h2>
                <span style={{ fontSize: '12px', color: '#64748B' }}>
                  Materials carried by Purohit or dispatched via local temple store kit
                </span>
              </div>
              <span style={{ backgroundColor: '#DCFCE7', color: '#166534', padding: '4px 10px', borderRadius: '12px', fontSize: '12px', fontWeight: 700 }}>
                Purohit / Kit
              </span>
            </div>

            <div style={{ display: 'flex', flexDirection: 'column', gap: '10px' }}>
              {sacredItems.map(item => (
                <div
                  key={item.id}
                  style={{
                    display: 'flex',
                    justifyContent: 'space-between',
                    alignItems: 'center',
                    padding: '12px 14px',
                    borderRadius: '8px',
                    backgroundColor: '#F0FDF4',
                    border: '1px solid #BBF7D0'
                  }}
                >
                  <div>
                    <div style={{ fontWeight: 600, fontSize: '14px', color: '#14532D' }}>{item.item_name}</div>
                    <div style={{ fontSize: '12px', color: '#166534' }}>
                      Quantity: <strong>{item.quantity}</strong> {item.estimated_price > 0 && `• Est: ₹${item.estimated_price}`}
                    </div>
                  </div>
                  <button
                    onClick={() => handleDeleteItem(item.id)}
                    style={{ background: 'none', border: 'none', color: '#DC2626', cursor: 'pointer', fontSize: '16px' }}
                    title="Delete item"
                  >
                    🗑️
                  </button>
                </div>
              ))}
              {sacredItems.length === 0 && (
                <div style={{ padding: '20px', textAlign: 'center', color: '#94A3B8' }}>No sacred samagri items configured.</div>
              )}
            </div>
          </div>

        </div>
      )}

      {/* Add New Item Modal */}
      {showAddModal && (
        <div style={{
          position: 'fixed', top: 0, left: 0, right: 0, bottom: 0,
          backgroundColor: 'rgba(0,0,0,0.5)', display: 'flex', alignItems: 'center', justifyContent: 'center', zIndex: 1000
        }}>
          <div style={{ backgroundColor: '#FFFFFF', borderRadius: '14px', padding: '24px', width: '480px', maxWidth: '90%' }}>
            <h2 style={{ margin: '0 0 16px 0', fontSize: '20px', fontWeight: 700, color: '#831843' }}>
              Add Standard Item to Checklist
            </h2>
            <form onSubmit={handleAddItem}>
              <div style={{ marginBottom: '14px' }}>
                <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: '#475569', marginBottom: '4px' }}>Item Name</label>
                <input
                  type="text"
                  required
                  placeholder="e.g. Pure Cow Ghee or Brass Plate"
                  value={newItemName}
                  onChange={e => setNewItemName(e.target.value)}
                  style={{ width: '100%', padding: '10px', borderRadius: '8px', border: '1px solid #CBD5E1', boxSizing: 'border-box' }}
                />
              </div>

              <div style={{ marginBottom: '14px' }}>
                <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: '#475569', marginBottom: '4px' }}>Category</label>
                <select
                  value={newItemCategory}
                  onChange={e => setNewItemCategory(e.target.value)}
                  style={{ width: '100%', padding: '10px', borderRadius: '8px', border: '1px solid #CBD5E1', boxSizing: 'border-box' }}
                >
                  <option value="Devotee Household Items">🏡 Devotee Household Items (Devotee prepares at home)</option>
                  <option value="Sacred Ritual Samagri">🪔 Sacred Ritual Samagri (Purohit carries or Kit)</option>
                </select>
              </div>

              <div style={{ display: 'grid', gridTemplateColumns: '1fr 1fr', gap: '12px', marginBottom: '14px' }}>
                <div>
                  <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: '#475569', marginBottom: '4px' }}>Quantity</label>
                  <input
                    type="text"
                    required
                    placeholder="e.g. 500g, 3 nos"
                    value={newItemQuantity}
                    onChange={e => setNewItemQuantity(e.target.value)}
                    style={{ width: '100%', padding: '10px', borderRadius: '8px', border: '1px solid #CBD5E1', boxSizing: 'border-box' }}
                  />
                </div>
                <div>
                  <label style={{ display: 'block', fontSize: '12px', fontWeight: 600, color: '#475569', marginBottom: '4px' }}>Est. Price (₹)</label>
                  <input
                    type="number"
                    value={newItemPrice}
                    onChange={e => setNewItemPrice(Number(e.target.value))}
                    style={{ width: '100%', padding: '10px', borderRadius: '8px', border: '1px solid #CBD5E1', boxSizing: 'border-box' }}
                  />
                </div>
              </div>

              <div style={{ display: 'flex', alignItems: 'center', gap: '8px', marginBottom: '20px' }}>
                <input
                  type="checkbox"
                  id="mandatoryCheck"
                  checked={isMandatory}
                  onChange={e => setIsMandatory(e.target.checked)}
                />
                <label htmlFor="mandatoryCheck" style={{ fontSize: '13px', color: '#334155' }}>Essential / Mandatory Item</label>
              </div>

              <div style={{ display: 'flex', justifyContent: 'flex-end', gap: '10px' }}>
                <button
                  type="button"
                  onClick={() => setShowAddModal(false)}
                  style={{ padding: '10px 16px', borderRadius: '8px', border: '1px solid #CBD5E1', backgroundColor: '#FFFFFF', cursor: 'pointer' }}
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  style={{ padding: '10px 20px', borderRadius: '8px', border: 'none', backgroundColor: '#831843', color: '#FFFFFF', fontWeight: 700, cursor: 'pointer' }}
                >
                  Save to Checklist
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </main>
  );
}
