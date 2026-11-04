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

        # 1. Main Dashboard
        print("[TEST 1] Temple CMS Dashboard (/)")
        await page.goto("http://localhost:3000", wait_until="networkidle")
        title = await page.title()
        print(f"   Page Title: {title}")
        content = await page.content()
        assert "Puja" in content or "Temple" in content or "Deity" in content, "Main dashboard content missing"
        await page.screenshot(path="playwright_dashboard.png")
        print("   [OK] Dashboard loaded & screenshot saved: playwright_dashboard.png")

        # 2. Gate Scanner
        print("[TEST 2] QR Gate Scanner (/gate-scanner)")
        await page.goto("http://localhost:3000/gate-scanner", wait_until="networkidle")
        content = await page.content()
        assert "Gate" in content or "Scanner" in content or "QR" in content, "Gate scanner content missing"
        await page.screenshot(path="playwright_gate_scanner.png")
        print("   [OK] Gate Scanner loaded & screenshot saved: playwright_gate_scanner.png")

        # 3. VIP Pujas
        print("[TEST 3] VIP Pujas Management (/vip-pujas)")
        await page.goto("http://localhost:3000/vip-pujas", wait_until="networkidle")
        content = await page.content()
        assert "VIP" in content or "Puja" in content, "VIP Pujas content missing"
        await page.screenshot(path="playwright_vip_pujas.png")
        print("   [OK] VIP Pujas loaded & screenshot saved: playwright_vip_pujas.png")

        # 4. Reconciliation Ledger
        print("[TEST 4] Financial Reconciliation (/reconciliation)")
        await page.goto("http://localhost:3000/reconciliation", wait_until="networkidle")
        content = await page.content()
        assert "Reconciliation" in content or "Escrow" in content or "Ledger" in content or "Dakshina" in content, "Reconciliation content missing"
        await page.screenshot(path="playwright_reconciliation.png")
        print("   [OK] Reconciliation loaded & screenshot saved: playwright_reconciliation.png")

        # 5. Sanctum Studio
        print("[TEST 5] Sanctum Live Telecast Studio (/sanctum-studio)")
        await page.goto("http://localhost:3000/sanctum-studio", wait_until="networkidle")
        content = await page.content()
        assert "Studio" in content or "Sanctum" in content or "Telecast" in content or "CCTV" in content, "Sanctum studio content missing"
        await page.screenshot(path="playwright_sanctum_studio.png")
        print("   [OK] Sanctum Studio loaded & screenshot saved: playwright_sanctum_studio.png")

        await browser.close()
        print("\nALL 5 PLAYWRIGHT END-TO-END TESTS PASSED SUCCESSFULLY!")

if __name__ == "__main__":
    asyncio.run(run())
