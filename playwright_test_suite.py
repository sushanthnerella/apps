import asyncio
import sys
import os

# Ensure UTF-8 output on Windows
if sys.platform == "win32":
    sys.stdout.reconfigure(encoding="utf-8")

from playwright.async_api import async_playwright

async def run():
    print("[INIT] Initializing Playwright End-to-End Browser Test Suite...")
    async with async_playwright() as p:
        browser = await p.chromium.launch(headless=True)
        context = await browser.new_context(viewport={"width": 1280, "height": 800})
        page = await context.new_page()

        output_dir = os.path.join(os.getcwd(), "output", "test-audit-2026-11-07-rerun")
        os.makedirs(output_dir, exist_ok=True)

        # 1. Main Dashboard
        print("[TEST 1] Temple CMS Dashboard (/)")
        await page.goto("http://localhost:3000", wait_until="networkidle")
        title = await page.title()
        print(f"   Page Title: {title}")
        content = await page.content()
        assert "Puja" in content or "Temple" in content or "Deity" in content, "Main dashboard content missing"
        dash_path = os.path.join(output_dir, "playwright_dashboard.png")
        await page.screenshot(path=dash_path)
        print(f"   [OK] Dashboard loaded & screenshot saved: {dash_path}")

        # 2. Gate Scanner
        print("[TEST 2] QR Gate Scanner (/gate-scanner)")
        await page.goto("http://localhost:3000/gate-scanner", wait_until="networkidle")
        content = await page.content()
        assert "Gate" in content or "Scanner" in content or "QR" in content, "Gate scanner content missing"
        gate_path = os.path.join(output_dir, "playwright_gate_scanner.png")
        await page.screenshot(path=gate_path)
        print(f"   [OK] Gate Scanner loaded & screenshot saved: {gate_path}")

        # 3. VIP Pujas
        print("[TEST 3] VIP Pujas Management (/vip-pujas)")
        await page.goto("http://localhost:3000/vip-pujas", wait_until="networkidle")
        content = await page.content()
        assert "VIP" in content or "Puja" in content, "VIP Pujas content missing"
        vip_path = os.path.join(output_dir, "playwright_vip_pujas.png")
        await page.screenshot(path=vip_path)
        print(f"   [OK] VIP Pujas loaded & screenshot saved: {vip_path}")

        # 4. Reconciliation Ledger
        print("[TEST 4] Financial Reconciliation (/reconciliation)")
        await page.goto("http://localhost:3000/reconciliation", wait_until="networkidle")
        content = await page.content()
        assert "Reconciliation" in content or "Escrow" in content or "Ledger" in content or "Dakshina" in content, "Reconciliation content missing"
        rec_path = os.path.join(output_dir, "playwright_reconciliation.png")
        await page.screenshot(path=rec_path)
        print(f"   [OK] Reconciliation loaded & screenshot saved: {rec_path}")

        # 5. Sanctum Studio
        print("[TEST 5] Sanctum Live Telecast Studio (/sanctum-studio)")
        await page.goto("http://localhost:3000/sanctum-studio", wait_until="networkidle")
        content = await page.content()
        assert "Studio" in content or "Sanctum" in content or "Telecast" in content or "CCTV" in content, "Sanctum studio content missing"
        sanctum_path = os.path.join(output_dir, "playwright_sanctum_studio.png")
        await page.screenshot(path=sanctum_path)
        print(f"   [OK] Sanctum Studio loaded & screenshot saved: {sanctum_path}")

        await browser.close()
        print("\nALL 5 PLAYWRIGHT END-TO-END TESTS PASSED SUCCESSFULLY!")

if __name__ == "__main__":
    asyncio.run(run())
