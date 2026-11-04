'use client';

import React, { useState, useEffect } from 'react';

interface ServiceItem {
  id: string;
  type: string;
  priest_id?: string;
  priest_name?: string;
  original_title: string;
  title_en: string;
  title_te: string;
  title_hi: string;
  description_en: string;
  description_te: string;
  description_hi: string;
  category: string;
  base_price: number;
}

export default function ServicesTranslationPage() {
  const [services, setServices] = useState<ServiceItem[]>([]);
  const [filterType, setFilterType] = useState<'ALL' | 'PRIEST_CUSTOM' | 'CATALOG'>('ALL');
  const [isLoading, setIsLoading] = useState(true);
  const [statusMessage, setStatusMessage] = useState<string | null>(null);
  const [savingId, setSavingId] = useState<string | null>(null);

  const loadTranslations = async () => {
    setIsLoading(true);
    try {
      const res = await fetch('http://localhost:8085/api/v1/services/translations');
      if (res.ok) {
        const data = await res.json();
        setServices(data || []);
      }
    } catch (err: any) {
      console.error(err);
      setStatusMessage('Error loading translations: ' + err.message);
    } finally {
      setIsLoading(false);
    }
  };

  useEffect(() => {
    loadTranslations();
  }, []);

  const handleFieldChange = (id: string, field: keyof ServiceItem, value: string) => {
    setServices((prev) =>
      prev.map((s) => (s.id === id ? { ...s, [field]: value } : s))
    );
  };

  const saveTranslation = async (service: ServiceItem) => {
    setSavingId(service.id);
    try {
      const res = await fetch('http://localhost:8085/api/v1/services/translations', {
        method: 'PUT',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          id: service.id,
          type: service.type,
          title_en: service.title_en,
          title_te: service.title_te,
          title_hi: service.title_hi,
          description_en: service.description_en,
          description_te: service.description_te,
          description_hi: service.description_hi,
        }),
      });
      if (res.ok) {
        setStatusMessage(`✅ Saved translations for "${service.title_en || service.original_title}"`);
        setTimeout(() => setStatusMessage(null), 3000);
      }
    } catch (err: any) {
      setStatusMessage('Error saving: ' + err.message);
    } finally {
      setSavingId(null);
    }
  };

  const handleAutoTranslateAll = async () => {
    setIsLoading(true);
    try {
      const res = await fetch('http://localhost:8085/api/v1/services/auto-translate', {
        method: 'POST',
      });
      if (res.ok) {
        const data = await res.json();
        setStatusMessage(`✨ ${data.message || 'Auto-translated successfully!'}`);
        await loadTranslations();
      }
    } catch (err: any) {
      setStatusMessage('Auto-translation failed: ' + err.message);
    } finally {
      setIsLoading(false);
    }
  };

  const filteredServices = services.filter((s) => {
    if (filterType === 'ALL') return true;
    return s.type === filterType;
  });

  return (
    <div style={{ padding: '32px', maxWidth: '1400px', margin: '0 auto' }}>
      {/* Header */}
      <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '24px' }}>
        <div>
          <h1 style={{ fontSize: '26px', fontWeight: 800, margin: 0, color: '#831843' }}>
            🌐 3-Language Service Localizer & Vedic Converter
          </h1>
          <p style={{ margin: '6px 0 0 0', color: '#475569', fontSize: '14px' }}>
            Purohits enter services in their single preferred language. The CMS automatically translates and manages authentic titles in English, తెలుగు (Telugu), and हिन्दी (Hindi).
          </p>
        </div>

        <div style={{ display: 'flex', gap: '12px' }}>
          <button
            onClick={handleAutoTranslateAll}
            disabled={isLoading}
            style={{
              padding: '10px 18px',
              backgroundColor: '#D97706',
              color: 'white',
              border: 'none',
              borderRadius: '8px',
              fontWeight: 600,
              cursor: 'pointer',
              display: 'flex',
              alignItems: 'center',
              gap: '6px',
            }}
          >
            🪄 Auto-Translate Untranslated
          </button>
          <button
            onClick={loadTranslations}
            style={{
              padding: '10px 18px',
              backgroundColor: '#0F172A',
              color: 'white',
              border: 'none',
              borderRadius: '8px',
              fontWeight: 600,
              cursor: 'pointer',
            }}
          >
            🔄 Refresh
          </button>
        </div>
      </div>

      {statusMessage && (
        <div
          style={{
            padding: '12px 16px',
            backgroundColor: '#DCFCE7',
            color: '#15803D',
            borderRadius: '8px',
            marginBottom: '20px',
            fontWeight: 600,
          }}
        >
          {statusMessage}
        </div>
      )}

      {/* Filter Chips */}
      <div style={{ display: 'flex', gap: '10px', marginBottom: '20px' }}>
        {(['ALL', 'PRIEST_CUSTOM', 'CATALOG'] as const).map((type) => (
          <button
            key={type}
            onClick={() => setFilterType(type)}
            style={{
              padding: '8px 16px',
              borderRadius: '20px',
              border: 'none',
              fontWeight: 600,
              fontSize: '13px',
              cursor: 'pointer',
              backgroundColor: filterType === type ? '#831843' : '#E2E8F0',
              color: filterType === type ? '#FFFFFF' : '#334155',
            }}
          >
            {type === 'ALL' && 'All Offerings & Pujas'}
            {type === 'PRIEST_CUSTOM' && '🕉️ Priest Added Custom Offerings'}
            {type === 'CATALOG' && '🏛️ Central Temple Catalog'}
          </button>
        ))}
      </div>

      {/* Services List Cards */}
      {isLoading ? (
        <div style={{ padding: '40px', textAlign: 'center', color: '#64748B' }}>Loading 3-language service registry...</div>
      ) : (
        <div style={{ display: 'flex', flexDirection: 'column', gap: '20px' }}>
          {filteredServices.map((service) => (
            <div
              key={service.id}
              style={{
                backgroundColor: '#FFFFFF',
                borderRadius: '14px',
                padding: '20px',
                border: '1px solid #E2E8F0',
                boxShadow: '0 2px 6px rgba(0,0,0,0.04)',
              }}
            >
              {/* Card Header */}
              <div style={{ display: 'flex', justifyContent: 'space-between', alignItems: 'center', marginBottom: '14px', borderBottom: '1px solid #F1F5F9', paddingBottom: '10px' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '10px' }}>
                  <span
                    style={{
                      padding: '4px 10px',
                      borderRadius: '6px',
                      fontSize: '12px',
                      fontWeight: 700,
                      backgroundColor: service.type === 'PRIEST_CUSTOM' ? '#FEF3C7' : '#EFF6FF',
                      color: service.type === 'PRIEST_CUSTOM' ? '#B45309' : '#1D4ED8',
                    }}
                  >
                    {service.type === 'PRIEST_CUSTOM' ? '🕉️ Priest Offering' : '🏛️ Temple Catalog'}
                  </span>
                  <span style={{ fontWeight: 700, fontSize: '16px', color: '#0F172A' }}>
                    {service.original_title}
                  </span>
                  {service.priest_name && (
                    <span style={{ fontSize: '13px', color: '#64748B' }}>
                      by <strong>{service.priest_name}</strong>
                    </span>
                  )}
                </div>

                <div style={{ display: 'flex', alignItems: 'center', gap: '12px' }}>
                  <span style={{ fontWeight: 700, color: '#15803D', fontSize: '15px' }}>
                    ₹{service.base_price?.toFixed(0)}
                  </span>
                  <button
                    onClick={() => saveTranslation(service)}
                    disabled={savingId === service.id}
                    style={{
                      padding: '6px 14px',
                      backgroundColor: savingId === service.id ? '#94A3B8' : '#15803D',
                      color: '#FFFFFF',
                      border: 'none',
                      borderRadius: '6px',
                      fontWeight: 600,
                      cursor: 'pointer',
                      fontSize: '13px',
                    }}
                  >
                    {savingId === service.id ? 'Saving...' : '💾 Save Translations'}
                  </button>
                </div>
              </div>

              {/* 3-Language Grid */}
              <div style={{ display: 'grid', gridTemplateColumns: 'repeat(3, 1fr)', gap: '16px' }}>
                {/* 1. English */}
                <div style={{ backgroundColor: '#F8FAFC', padding: '12px', borderRadius: '10px', border: '1px solid #E2E8F0' }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '6px', marginBottom: '8px' }}>
                    <span style={{ fontSize: '16px' }}>🇬🇧</span>
                    <strong style={{ fontSize: '13px', color: '#0F172A' }}>English Title</strong>
                  </div>
                  <input
                    type="text"
                    value={service.title_en || ''}
                    onChange={(e) => handleFieldChange(service.id, 'title_en', e.target.value)}
                    placeholder="Enter English Title"
                    style={{ width: '100%', padding: '8px', borderRadius: '6px', border: '1px solid #CBD5E1', marginBottom: '8px', fontSize: '13px' }}
                  />
                  <div style={{ fontSize: '12px', color: '#64748B', marginBottom: '4px' }}>English Description</div>
                  <textarea
                    rows={2}
                    value={service.description_en || ''}
                    onChange={(e) => handleFieldChange(service.id, 'description_en', e.target.value)}
                    style={{ width: '100%', padding: '8px', borderRadius: '6px', border: '1px solid #CBD5E1', fontSize: '12px' }}
                  />
                </div>

                {/* 2. Telugu */}
                <div style={{ backgroundColor: '#FFFBEB', padding: '12px', borderRadius: '10px', border: '1px solid #FDE68A' }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '6px', marginBottom: '8px' }}>
                    <span style={{ fontSize: '16px' }}>🇮🇳</span>
                    <strong style={{ fontSize: '13px', color: '#92400E' }}>తెలుగు (Telugu Title)</strong>
                  </div>
                  <input
                    type="text"
                    value={service.title_te || ''}
                    onChange={(e) => handleFieldChange(service.id, 'title_te', e.target.value)}
                    placeholder="తెలుగు పూజా పేరు నమోదు చేయండి"
                    style={{ width: '100%', padding: '8px', borderRadius: '6px', border: '1px solid #FCD34D', marginBottom: '8px', fontSize: '13px', fontWeight: 600 }}
                  />
                  <div style={{ fontSize: '12px', color: '#92400E', marginBottom: '4px' }}>తెలుగు వివరాలు (Description)</div>
                  <textarea
                    rows={2}
                    value={service.description_te || ''}
                    onChange={(e) => handleFieldChange(service.id, 'description_te', e.target.value)}
                    style={{ width: '100%', padding: '8px', borderRadius: '6px', border: '1px solid #FCD34D', fontSize: '12px' }}
                  />
                </div>

                {/* 3. Hindi */}
                <div style={{ backgroundColor: '#FDF4FF', padding: '12px', borderRadius: '10px', border: '1px solid #F5D0FE' }}>
                  <div style={{ display: 'flex', alignItems: 'center', gap: '6px', marginBottom: '8px' }}>
                    <span style={{ fontSize: '16px' }}>🇮🇳</span>
                    <strong style={{ fontSize: '13px', color: '#86198F' }}>हिन्दी (Hindi Title)</strong>
                  </div>
                  <input
                    type="text"
                    value={service.title_hi || ''}
                    onChange={(e) => handleFieldChange(service.id, 'title_hi', e.target.value)}
                    placeholder="हिन्दी पूजा नाम दर्ज करें"
                    style={{ width: '100%', padding: '8px', borderRadius: '6px', border: '1px solid #F0ABFC', marginBottom: '8px', fontSize: '13px', fontWeight: 600 }}
                  />
                  <div style={{ fontSize: '12px', color: '#86198F', marginBottom: '4px' }}>हिन्दी विवरण (Description)</div>
                  <textarea
                    rows={2}
                    value={service.description_hi || ''}
                    onChange={(e) => handleFieldChange(service.id, 'description_hi', e.target.value)}
                    style={{ width: '100%', padding: '8px', borderRadius: '6px', border: '1px solid #F0ABFC', fontSize: '12px' }}
                  />
                </div>
              </div>
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
