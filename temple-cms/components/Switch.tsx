'use client';

type SwitchProps = {
  checked: boolean;
  label: string;
  description?: string;
  disabled?: boolean;
  onChange: (next: boolean) => void;
};

/** Accessible on/off switch: a real button with role="switch" so keyboard and screen readers work. */
export default function Switch({ checked, label, description, disabled, onChange }: SwitchProps) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', gap: 16, padding: '14px 0', borderBottom: '1px solid #E2E8F0' }}>
      <div>
        <div style={{ fontWeight: 600 }}>{label}</div>
        {description && <div style={{ fontSize: 13, color: '#475569', marginTop: 2 }}>{description}</div>}
      </div>
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        aria-label={label}
        disabled={disabled}
        onClick={() => onChange(!checked)}
        style={{
          width: 52, height: 30, minWidth: 52, borderRadius: 999, border: 'none', padding: 3,
          background: disabled ? '#CBD5E1' : checked ? '#166534' : '#94A3B8',
          cursor: disabled ? 'not-allowed' : 'pointer', display: 'flex',
          justifyContent: checked ? 'flex-end' : 'flex-start',
        }}
      >
        <span style={{ width: 24, height: 24, borderRadius: '50%', background: '#FFFFFF', display: 'block' }} />
      </button>
    </div>
  );
}
