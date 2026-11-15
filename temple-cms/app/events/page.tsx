'use client';

import { FormEvent, useCallback, useEffect, useState } from 'react';
import { apiFetch } from '../../lib/api';
import Switch from '../../components/Switch';

type Ghat = {
  id: string; event_id: string; name: string; name_te: string; name_hi: string;
  latitude: number | null; longitude: number | null; sort_order: number; is_active: boolean;
};
type EventItem = {
  id: string; slug: string; name: string; name_te: string; name_hi: string; river: string; status: string;
  booking_enabled: boolean; card_issuance_enabled: boolean; govt_cards_enabled: boolean;
  starts_at: string; ends_at: string; config_version: number; updated_at: string; ghats: Ghat[];
};
type Ritual = { id: string; name: string; name_te: string; name_hi: string; duration_minutes: number; sort_order: number; is_active: boolean };
type ParticipationRow = {
  priest_id: string; priest_name: string; sampradaya: string; experience_years: number; status: string;
  reviewer_reason: string; submitted_at: string; reviewed_at: string | null; services_count: number;
};
type AuditEntry = { id: number; actor_role: string; action: string; entity_type: string; created_at: string; after: Record<string, unknown> | null };

// Mirrors models.EventTransitions in the backend; the server re-checks every move.
const TRANSITIONS: Record<string, string[]> = {
  DRAFT: ['SCHEDULED'], SCHEDULED: ['DRAFT', 'ACTIVE'], ACTIVE: ['PAUSED', 'ENDED'], PAUSED: ['ACTIVE', 'ENDED'], ENDED: ['ARCHIVED'],
};
const ERRORS: Record<string, string> = {
  invalid_transition: 'That status change is not allowed from the current status.',
  another_event_live: 'Another event is already live. End or pause it first.',
  duplicate: 'That slug or ghat name already exists.',
  invalid_request: 'Some fields are missing or invalid.',
  not_found: 'Not found. Refresh the page.',
};

const card: React.CSSProperties = { background: '#FFFFFF', border: '1px solid #E2E8F0', borderRadius: 12, padding: 20 };
const input: React.CSSProperties = { padding: '10px 12px', border: '1px solid #CBD5E1', borderRadius: 8, font: 'inherit', minHeight: 44, boxSizing: 'border-box', width: '100%' };
const button: React.CSSProperties = { padding: '10px 16px', borderRadius: 8, border: 'none', background: '#831843', color: '#FFFFFF', fontWeight: 600, cursor: 'pointer', minHeight: 44 };

async function readError(res: Response) {
  const body = await res.json().catch(() => ({}));
  return ERRORS[body.error] ?? `Request failed (${res.status}).`;
}

export default function EventsPage() {
  const [events, setEvents] = useState<EventItem[]>([]);
  const [selectedId, setSelectedId] = useState<string | null>(null);
  const [audit, setAudit] = useState<AuditEntry[]>([]);
  const [message, setMessage] = useState<{ kind: 'ok' | 'error'; text: string } | null>(null);
  const [busy, setBusy] = useState(false);

  const selected = events.find((e) => e.id === selectedId) ?? null;

  const load = useCallback(async () => {
    const res = await apiFetch('api/v1/admin/events');
    if (!res.ok) { setMessage({ kind: 'error', text: await readError(res) }); return; }
    const list: EventItem[] = await res.json();
    setEvents(list);
    setSelectedId((cur) => cur ?? list[0]?.id ?? null);
  }, []);

  const [rituals, setRituals] = useState<Ritual[]>([]);
  const [participations, setParticipations] = useState<ParticipationRow[]>([]);

  const loadAudit = useCallback(async (id: string) => {
    const [a, r, p] = await Promise.all([
      apiFetch(`api/v1/admin/events/${id}/audit?limit=20`),
      apiFetch(`api/v1/admin/events/${id}/rituals`),
      apiFetch(`api/v1/admin/events/${id}/participations`),
    ]);
    setAudit(a.ok ? await a.json() : []);
    setRituals(r.ok ? await r.json() : []);
    setParticipations(p.ok ? await p.json() : []);
  }, []);

  async function addRitual(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!selected) return;
    const f = new FormData(e.currentTarget);
    const ok = await send(`api/v1/admin/events/${selected.id}/rituals`, 'POST', {
      name: f.get('name'), name_te: f.get('name_te'), name_hi: f.get('name_hi'),
      duration_minutes: Number(f.get('duration_minutes')), sort_order: Number(f.get('sort_order') || 0),
    }, 'Ritual added. Priests can now set their price for it.');
    if (ok) (e.target as HTMLFormElement).reset();
  }

  function setRitualActive(r: Ritual, active: boolean) {
    if (!selected) return;
    if (!active && !window.confirm(`Hide ${r.name}? Priests stop offering it for new bookings; existing bookings keep their agreed terms.`)) return;
    send(`api/v1/admin/events/${selected.id}/rituals/${r.id}`, 'PUT', { ...r, is_active: active }, active ? 'Ritual shown.' : 'Ritual hidden.');
  }

  function review(p: ParticipationRow, status: string) {
    if (!selected) return;
    let reason = '';
    if (status !== 'APPROVED') {
      reason = window.prompt(`Reason for ${status.replace('_', ' ').toLowerCase()} (shown to ${p.priest_name}):`)?.trim() ?? '';
      if (!reason) return;
    } else if (!window.confirm(`Approve ${p.priest_name} for ${selected.name}?`)) return;
    send(`api/v1/admin/events/${selected.id}/participations/${p.priest_id}`, 'PUT', { status, reason }, `${p.priest_name}: ${status}.`);
  }

  useEffect(() => { load(); }, [load]);
  useEffect(() => { if (selectedId) loadAudit(selectedId); }, [selectedId, loadAudit]);

  async function send(path: string, method: string, body: unknown, okText: string) {
    setBusy(true); setMessage(null);
    try {
      const res = await apiFetch(path, { method, headers: { 'Content-Type': 'application/json' }, body: JSON.stringify(body) });
      if (!res.ok) { setMessage({ kind: 'error', text: await readError(res) }); return false; }
      setMessage({ kind: 'ok', text: okText });
      await load();
      if (selectedId) await loadAudit(selectedId);
      return true;
    } finally { setBusy(false); }
  }

  function updateEvent(patch: Partial<Pick<EventItem, 'status' | 'booking_enabled' | 'card_issuance_enabled'>>, confirmText?: string) {
    if (!selected) return;
    if (confirmText && !window.confirm(confirmText)) return;
    send(`api/v1/admin/events/${selected.id}`, 'PATCH', patch, 'Saved. Apps refresh within seconds.');
  }

  async function createEvent(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    const f = new FormData(e.currentTarget);
    const toIso = (v: FormDataEntryValue | null) => (v ? new Date(String(v)).toISOString() : '');
    const ok = await send('api/v1/admin/events', 'POST', {
      slug: f.get('slug'), name: f.get('name'), name_te: f.get('name_te'), name_hi: f.get('name_hi'), river: f.get('river'),
      starts_at: toIso(f.get('starts_at')), ends_at: toIso(f.get('ends_at')),
    }, 'Event created as Draft.');
    if (ok) (e.target as HTMLFormElement).reset();
  }

  async function addGhat(e: FormEvent<HTMLFormElement>) {
    e.preventDefault();
    if (!selected) return;
    const f = new FormData(e.currentTarget);
    const num = (v: FormDataEntryValue | null) => (v === null || String(v).trim() === '' ? null : Number(v));
    const ok = await send(`api/v1/admin/events/${selected.id}/ghats`, 'POST', {
      name: f.get('name'), name_te: f.get('name_te'), name_hi: f.get('name_hi'),
      latitude: num(f.get('latitude')), longitude: num(f.get('longitude')), sort_order: num(f.get('sort_order')) ?? 0,
    }, 'Ghat added.');
    if (ok) (e.target as HTMLFormElement).reset();
  }

  function setGhatActive(g: Ghat, active: boolean) {
    if (!selected) return;
    if (!active && !window.confirm(`Hide ${g.name} from the apps? Existing bookings are not affected.`)) return;
    send(`api/v1/admin/events/${selected.id}/ghats/${g.id}`, 'PUT', { ...g, is_active: active }, active ? 'Ghat shown.' : 'Ghat hidden.');
  }

  const live = selected && (selected.status === 'ACTIVE' || selected.status === 'PAUSED');

  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 24, maxWidth: 1100 }}>
      <header>
        <h1 style={{ margin: 0, fontSize: 28 }}>Events (Pushkaralu)</h1>
        <p style={{ margin: '6px 0 0', color: '#475569' }}>Turn event mode on or off for the devotee and priest apps. Turning anything off never cancels existing bookings.</p>
      </header>

      {message && (
        <div role={message.kind === 'error' ? 'alert' : 'status'} style={{ padding: 12, borderRadius: 8, background: message.kind === 'error' ? '#FEE2E2' : '#DCFCE7', color: message.kind === 'error' ? '#991B1B' : '#166534' }}>
          {message.text}
        </div>
      )}

      <div style={{ display: 'flex', flexWrap: 'wrap', gap: 24, alignItems: 'flex-start' }}>
        <section style={{ ...card, flex: '1 1 280px' }} aria-labelledby="events-list">
          <h2 id="events-list" style={{ marginTop: 0, fontSize: 18 }}>All events</h2>
          {events.length === 0 && <p style={{ color: '#475569' }}>No events yet. Create one below.</p>}
          <ul style={{ listStyle: 'none', padding: 0, margin: 0, display: 'flex', flexDirection: 'column', gap: 8 }}>
            {events.map((ev) => (
              <li key={ev.id}>
                <button type="button" onClick={() => setSelectedId(ev.id)} aria-pressed={ev.id === selectedId}
                  style={{ width: '100%', textAlign: 'left', padding: 12, borderRadius: 8, cursor: 'pointer', minHeight: 44, font: 'inherit',
                    border: ev.id === selectedId ? '2px solid #831843' : '1px solid #E2E8F0', background: '#FFFFFF' }}>
                  <div style={{ fontWeight: 600 }}>{ev.name}</div>
                  <div style={{ fontSize: 13, color: '#475569' }}>{ev.status} · {new Date(ev.starts_at).toLocaleDateString()} to {new Date(ev.ends_at).toLocaleDateString()}</div>
                </button>
              </li>
            ))}
          </ul>

          <h3 style={{ fontSize: 16, marginTop: 24 }}>Create event</h3>
          <form onSubmit={createEvent} style={{ display: 'flex', flexDirection: 'column', gap: 10 }}>
            <label>Name<input name="name" required style={input} placeholder="Godavari Pushkaralu" /></label>
            <label>Name (Telugu)<input name="name_te" style={input} /></label>
            <label>Name (Hindi)<input name="name_hi" style={input} /></label>
            <label>River<input name="river" style={input} /></label>
            <label>Slug (lowercase, dashes)<input name="slug" required pattern="[a-z0-9]+(-[a-z0-9]+)*" style={input} placeholder="godavari-2027" /></label>
            <label>Starts<input name="starts_at" type="datetime-local" required style={input} /></label>
            <label>Ends<input name="ends_at" type="datetime-local" required style={input} /></label>
            <button type="submit" disabled={busy} style={button}>Create as Draft</button>
          </form>
        </section>

        {selected && (
          <section style={{ ...card, flex: '2 1 480px', display: 'flex', flexDirection: 'column', gap: 20 }} aria-labelledby="event-detail">
            <div>
              <h2 id="event-detail" style={{ margin: 0, fontSize: 22 }}>{selected.name}</h2>
              <div style={{ color: '#475569', fontSize: 14 }}>Status {selected.status} · config version {selected.config_version} · last change {new Date(selected.updated_at).toLocaleString()}</div>
            </div>

            <div>
              <h3 style={{ fontSize: 16, margin: '0 0 8px' }}>Status</h3>
              <div style={{ display: 'flex', flexWrap: 'wrap', gap: 8 }}>
                {(TRANSITIONS[selected.status] ?? []).map((to) => (
                  <button key={to} type="button" disabled={busy} style={{ ...button, background: to === 'ENDED' ? '#991B1B' : '#831843' }}
                    onClick={() => updateEvent({ status: to }, `Change status from ${selected.status} to ${to}?${to === 'ENDED' ? ' This turns booking and card issuance off. Existing bookings are kept.' : ''}`)}>
                    Move to {to}
                  </button>
                ))}
                {(TRANSITIONS[selected.status] ?? []).length === 0 && <span style={{ color: '#475569' }}>No further status changes.</span>}
              </div>
            </div>

            <div>
              <h3 style={{ fontSize: 16, margin: 0 }}>Switches</h3>
              <Switch label="Booking open" checked={selected.booking_enabled} disabled={busy || !live}
                description={live ? 'Devotees can send new requests while the event is Active.' : 'Available once the event is Active or Paused.'}
                onChange={(next) => updateEvent({ booking_enabled: next }, next ? undefined : 'Stop new bookings? Existing bookings are not cancelled.')} />
              <Switch label="Card issuance" checked={selected.card_issuance_enabled} disabled={busy || !live}
                description="Pooja Vahini event cards for approved priests."
                onChange={(next) => updateEvent({ card_issuance_enabled: next }, next ? undefined : 'Stop issuing new cards? Issued cards stay valid.')} />
              <Switch label="Government cards" checked={false} disabled
                description="Locked until a government authority signs written approval (PRD v2)." onChange={() => {}} />
            </div>

            {live && (selected.booking_enabled || selected.card_issuance_enabled) && (
              <button type="button" disabled={busy} style={{ ...button, background: '#991B1B', alignSelf: 'flex-start' }}
                onClick={() => updateEvent({ booking_enabled: false, card_issuance_enabled: false }, 'Emergency stop: turn off new bookings and card issuance now? Nothing is deleted.')}>
                Emergency stop
              </button>
            )}

            <div>
              <h3 style={{ fontSize: 16, margin: '0 0 8px' }}>Ghats</h3>
              <div style={{ overflowX: 'auto' }}>
                <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 14 }}>
                  <thead><tr style={{ textAlign: 'left', color: '#475569' }}><th style={{ padding: 8 }}>Order</th><th style={{ padding: 8 }}>Name</th><th style={{ padding: 8 }}>Telugu</th><th style={{ padding: 8 }}>Location</th><th style={{ padding: 8 }}>Shown in apps</th></tr></thead>
                  <tbody>
                    {selected.ghats.map((g) => (
                      <tr key={g.id} style={{ borderTop: '1px solid #E2E8F0' }}>
                        <td style={{ padding: 8 }}>{g.sort_order}</td>
                        <td style={{ padding: 8 }}>{g.name}</td>
                        <td style={{ padding: 8 }}>{g.name_te}</td>
                        <td style={{ padding: 8 }}>{g.latitude != null ? `${g.latitude.toFixed(5)}, ${g.longitude?.toFixed(5)}` : 'Not set'}</td>
                        <td style={{ padding: 8 }}><Switch label={`Show ${g.name}`} checked={g.is_active} disabled={busy} onChange={(next) => setGhatActive(g, next)} /></td>
                      </tr>
                    ))}
                    {selected.ghats.length === 0 && <tr><td colSpan={5} style={{ padding: 8, color: '#475569' }}>No ghats yet.</td></tr>}
                  </tbody>
                </table>
              </div>
              <form onSubmit={addGhat} style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(160px, 1fr))', gap: 10, marginTop: 12 }}>
                <label>Name<input name="name" required style={input} /></label>
                <label>Telugu<input name="name_te" style={input} /></label>
                <label>Hindi<input name="name_hi" style={input} /></label>
                <label>Latitude<input name="latitude" type="number" step="any" min={-90} max={90} style={input} /></label>
                <label>Longitude<input name="longitude" type="number" step="any" min={-180} max={180} style={input} /></label>
                <label>Order<input name="sort_order" type="number" style={input} /></label>
                <button type="submit" disabled={busy} style={{ ...button, alignSelf: 'end' }}>Add ghat</button>
              </form>
            </div>

            <div>
              <h3 style={{ fontSize: 16, margin: '0 0 8px' }}>Rituals offered at ghats</h3>
              <p style={{ margin: '0 0 8px', fontSize: 13, color: '#475569' }}>Priests choose from these and set their own price. Prices are frozen into each booking.</p>
              <div style={{ overflowX: 'auto' }}>
                <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 14 }}>
                  <thead><tr style={{ textAlign: 'left', color: '#475569' }}><th style={{ padding: 8 }}>Order</th><th style={{ padding: 8 }}>Ritual</th><th style={{ padding: 8 }}>Telugu</th><th style={{ padding: 8 }}>Minutes</th><th style={{ padding: 8 }}>Offered</th></tr></thead>
                  <tbody>
                    {rituals.map((r) => (
                      <tr key={r.id} style={{ borderTop: '1px solid #E2E8F0' }}>
                        <td style={{ padding: 8 }}>{r.sort_order}</td>
                        <td style={{ padding: 8 }}>{r.name}</td>
                        <td style={{ padding: 8 }}>{r.name_te}</td>
                        <td style={{ padding: 8 }}>{r.duration_minutes}</td>
                        <td style={{ padding: 8 }}><Switch label={`Offer ${r.name}`} checked={r.is_active} disabled={busy} onChange={(next) => setRitualActive(r, next)} /></td>
                      </tr>
                    ))}
                    {rituals.length === 0 && <tr><td colSpan={5} style={{ padding: 8, color: '#475569' }}>No rituals yet.</td></tr>}
                  </tbody>
                </table>
              </div>
              <form onSubmit={addRitual} style={{ display: 'grid', gridTemplateColumns: 'repeat(auto-fit, minmax(150px, 1fr))', gap: 10, marginTop: 12 }}>
                <label>Ritual<input name="name" required style={input} /></label>
                <label>Telugu<input name="name_te" style={input} /></label>
                <label>Hindi<input name="name_hi" style={input} /></label>
                <label>Minutes<input name="duration_minutes" type="number" min={5} max={480} required style={input} /></label>
                <label>Order<input name="sort_order" type="number" style={input} /></label>
                <button type="submit" disabled={busy} style={{ ...button, alignSelf: 'end' }}>Add ritual</button>
              </form>
            </div>

            <div>
              <h3 style={{ fontSize: 16, margin: '0 0 8px' }}>Priest applications</h3>
              <div style={{ overflowX: 'auto' }}>
                <table style={{ width: '100%', borderCollapse: 'collapse', fontSize: 14 }}>
                  <thead><tr style={{ textAlign: 'left', color: '#475569' }}><th style={{ padding: 8 }}>Priest</th><th style={{ padding: 8 }}>Status</th><th style={{ padding: 8 }}>Prices set</th><th style={{ padding: 8 }}>Submitted</th><th style={{ padding: 8 }}>Decision</th></tr></thead>
                  <tbody>
                    {participations.map((p) => (
                      <tr key={p.priest_id} style={{ borderTop: '1px solid #E2E8F0', verticalAlign: 'top' }}>
                        <td style={{ padding: 8 }}><div style={{ fontWeight: 600 }}>{p.priest_name}</div><div style={{ color: '#475569' }}>{p.sampradaya} · {p.experience_years} yrs</div></td>
                        <td style={{ padding: 8 }}>{p.status}{p.reviewer_reason && <div style={{ color: '#475569' }}>{p.reviewer_reason}</div>}</td>
                        <td style={{ padding: 8 }}>{p.services_count}</td>
                        <td style={{ padding: 8 }}>{new Date(p.submitted_at).toLocaleString()}</td>
                        <td style={{ padding: 8 }}>
                          <div style={{ display: 'flex', flexWrap: 'wrap', gap: 6 }}>
                            {p.status !== 'APPROVED' && <button type="button" disabled={busy} style={{ ...button, background: '#166534' }} onClick={() => review(p, 'APPROVED')}>Approve</button>}
                            {p.status === 'SUBMITTED' && <button type="button" disabled={busy} style={{ ...button, background: '#92400E' }} onClick={() => review(p, 'CHANGES_REQUESTED')}>Ask changes</button>}
                            {p.status !== 'REJECTED' && p.status !== 'APPROVED' && <button type="button" disabled={busy} style={{ ...button, background: '#991B1B' }} onClick={() => review(p, 'REJECTED')}>Reject</button>}
                            {p.status === 'APPROVED' && <button type="button" disabled={busy} style={{ ...button, background: '#991B1B' }} onClick={() => review(p, 'SUSPENDED')}>Suspend</button>}
                          </div>
                        </td>
                      </tr>
                    ))}
                    {participations.length === 0 && <tr><td colSpan={5} style={{ padding: 8, color: '#475569' }}>No applications yet.</td></tr>}
                  </tbody>
                </table>
              </div>
            </div>

            <div>
              <h3 style={{ fontSize: 16, margin: '0 0 8px' }}>Recent changes</h3>
              {audit.length === 0 ? <p style={{ color: '#475569', margin: 0 }}>No changes recorded.</p> : (
                <ul style={{ margin: 0, paddingLeft: 18, fontSize: 14, display: 'flex', flexDirection: 'column', gap: 4 }}>
                  {audit.map((a) => (
                    <li key={a.id}>{new Date(a.created_at).toLocaleString()} · {a.action}{a.after && typeof a.after.status === 'string' ? ` → ${a.after.status}` : ''}</li>
                  ))}
                </ul>
              )}
            </div>
          </section>
        )}
      </div>
    </div>
  );
}
