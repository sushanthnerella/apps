import React from 'react';

export const metadata = {
  title: 'Puja Vaahini - Temple CMS & VIP Gate Portal',
  description: 'Enterprise administration, zero-lag gate check-in scanner and reconciliation',
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en">
      <body style={{ margin: 0, fontFamily: 'system-ui, -apple-system, sans-serif', backgroundColor: '#F8FAFC', color: '#0F172A' }}>
        <div style={{ display: 'flex', minHeight: '100vh' }}>
          {/* Sidebar */}
          <aside style={{
            width: '260px',
            backgroundColor: '#831843', // Sacred Deep Temple Maroon / Crimson
            color: '#FFFFFF',
            padding: '24px 16px',
            display: 'flex',
            flexDirection: 'column',
            justifyContent: 'space-between'
          }}>
            <div>
              <div style={{ display: 'flex', alignItems: 'center', gap: '10px', marginBottom: '32px' }}>
                <span style={{ fontSize: '28px' }}>🏛️</span>
                <div>
                  <h1 style={{ fontSize: '18px', fontWeight: 700, margin: 0, letterSpacing: '0.5px' }}>TEMPLE CMS</h1>
                  <span style={{ fontSize: '11px', color: '#FBCFE8', textTransform: 'uppercase' }}>Dhapurmari Shrine</span>
                </div>
              </div>

              <nav style={{ display: 'flex', flexDirection: 'column', gap: '8px' }}>
                <a href="/" style={{ padding: '10px 14px', borderRadius: '8px', color: '#FFFFFF', textDecoration: 'none', backgroundColor: 'rgba(255,255,255,0.15)', fontWeight: 600 }}>
                  📊 Dashboard Overview
                </a>
                <a href="/gate-scanner" style={{ padding: '10px 14px', borderRadius: '8px', color: '#FCE7F3', textDecoration: 'none', fontWeight: 600 }}>
                  ⚡ Gate Scanner (&lt; 50ms)
                </a>
                <a href="/vip-pujas" style={{ padding: '10px 14px', borderRadius: '8px', color: '#FCE7F3', textDecoration: 'none', fontWeight: 600 }}>
                  🎟️ VIP Pujas & Quotas
                </a>
                <a href="/samagri-catalog" style={{ padding: '10px 14px', borderRadius: '8px', color: '#FCE7F3', textDecoration: 'none', fontWeight: 600 }}>
                  📦 Samagri & Household Items
                </a>
                <a href="/dispatch-radar" style={{ padding: '10px 14px', borderRadius: '8px', color: '#FCE7F3', textDecoration: 'none', fontWeight: 600 }}>
                  🛰️ Live Dispatch & Geo-Radar
                </a>
                <a href="/services-translation" style={{ padding: '10px 14px', borderRadius: '8px', color: '#FCE7F3', textDecoration: 'none', fontWeight: 600 }}>
                  🌐 3-Language Service Localizer
                </a>
                <a href="/sanctum-studio" style={{ padding: '10px 14px', borderRadius: '8px', color: '#FCE7F3', textDecoration: 'none', fontWeight: 600 }}>
                  📹 Sanctum E-Pooja Studio
                </a>
                <a href="/reconciliation" style={{ padding: '10px 14px', borderRadius: '8px', color: '#FCE7F3', textDecoration: 'none', fontWeight: 600 }}>
                  🧾 Financial Reconciliation
                </a>
              </nav>
            </div>

            <div style={{ padding: '12px', backgroundColor: 'rgba(0,0,0,0.2)', borderRadius: '8px', fontSize: '12px' }}>
              <div style={{ fontWeight: 600 }}>Active Temple:</div>
              <div style={{ color: '#FDF2F8' }}>Dhapurmari Sri Lakshmi Narasimha Swamy Temple</div>
              <div style={{ color: '#34D399', marginTop: '4px' }}>● Edge Sync Active</div>
            </div>
          </aside>

          {/* Main Content Area */}
          <main style={{ flex: 1, padding: '32px 40px', overflowY: 'auto' }}>
            {children}
          </main>
        </div>
      </body>
    </html>
  );
}
