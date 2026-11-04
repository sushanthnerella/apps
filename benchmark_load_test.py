import time
import sys
import json
import concurrent.futures
import urllib.request
import urllib.error
import statistics

# Ensure UTF-8 output
sys.stdout.reconfigure(encoding='utf-8')

BASE_URL = sys.argv[1] if len(sys.argv) > 1 else "http://localhost:8085"

def make_request(url, method="GET", data=None, headers=None):
    if headers is None:
        headers = {}
    if data:
        encoded_data = json.dumps(data).encode('utf-8')
        headers['Content-Type'] = 'application/json'
    else:
        encoded_data = None

    req = urllib.request.Request(url, data=encoded_data, headers=headers, method=method)
    start_time = time.perf_counter()
    try:
        with urllib.request.urlopen(req, timeout=5) as response:
            latency_ms = (time.perf_counter() - start_time) * 1000.0
            return response.status, latency_ms, None
    except urllib.error.HTTPError as e:
        latency_ms = (time.perf_counter() - start_time) * 1000.0
        return e.code, latency_ms, str(e)
    except Exception as e:
        latency_ms = (time.perf_counter() - start_time) * 1000.0
        return 0, latency_ms, str(e)

def run_load_test(name, endpoint, method="GET", data=None, total_requests=1000, concurrency=50):
    url = f"{BASE_URL}{endpoint}"
    print(f"\n=======================================================")
    print(f"🚀 RUNNING BENCHMARK: {name}")
    print(f"Target: {url} | Total Requests: {total_requests} | Concurrency: {concurrency}")
    print(f"=======================================================")

    latencies = []
    status_codes = {}
    errors = 0

    bench_start = time.perf_counter()

    with concurrent.futures.ThreadPoolExecutor(max_workers=concurrency) as executor:
        futures = [executor.submit(make_request, url, method, data) for _ in range(total_requests)]
        for future in concurrent.futures.as_completed(futures):
            code, latency, err = future.result()
            latencies.append(latency)
            status_codes[code] = status_codes.get(code, 0) + 1
            if code not in [200, 201] or err:
                errors += 1

    total_time = time.perf_counter() - bench_start
    rps = total_requests / total_time if total_time > 0 else 0

    latencies.sort()
    p50 = statistics.median(latencies)
    p90 = latencies[int(len(latencies) * 0.90)]
    p95 = latencies[int(len(latencies) * 0.95)]
    p99 = latencies[int(len(latencies) * 0.99)]
    avg = statistics.mean(latencies)

    print(f"⏱️  Duration:           {total_time:.3f} s")
    print(f"⚡ Throughput:         {rps:.2f} req/sec (RPS)")
    print(f"📊 Status Codes:       {status_codes}")
    print(f"❌ Error Rate:          {errors / total_requests * 100:.2f}% ({errors}/{total_requests})")
    print(f"📈 Latency Stats (ms):")
    print(f"   - Average:         {avg:.2f} ms")
    print(f"   - p50 (Median):     {p50:.2f} ms")
    print(f"   - p90:              {p90:.2f} ms")
    print(f"   - p95:              {p95:.2f} ms")
    print(f"   - p99:              {p99:.2f} ms")
    print(f"   - Min / Max:        {latencies[0]:.2f} ms / {latencies[-1]:.2f} ms")

    return {
        "name": name,
        "total_requests": total_requests,
        "concurrency": concurrency,
        "rps": rps,
        "p50": p50,
        "p95": p95,
        "p99": p99,
        "error_rate": errors / total_requests * 100
    }

if __name__ == "__main__":
    print(f"Initiating High-Throughput Load & Concurrency Benchmark Suite on {BASE_URL}...")
    
    # Check baseline health
    status, lat, err = make_request(f"{BASE_URL}/health")
    if status != 200:
        print(f"ERROR: Base URL {BASE_URL}/health responded with status {status}: {err}")
        sys.exit(1)
    print(f"[OK] Gateway Healthy (latency: {lat:.2f}ms)")

    results = []

    # 1. Health Probe Benchmark (Baseline gateway overhead)
    r1 = run_load_test("Health Probe (Ultra-Fast Baseline)", "/health", total_requests=2000, concurrency=100)
    results.append(r1)

    # 2. Panchangam Calculations (CPU-bound computation)
    r2 = run_load_test("Panchangam Engine (CPU Astro Calc)", "/api/v1/panchangam/today", total_requests=1000, concurrency=50)
    results.append(r2)

    # 3. Poojas Catalog Read (In-memory / DB query)
    r3 = run_load_test("Catalog Read (Database/Cache Retrieval)", "/api/v1/poojas", total_requests=1500, concurrency=75)
    results.append(r3)

    # 4. Nearby Priests Spatial Dispatch Query (Geo Spatial Redis lookup)
    r4 = run_load_test("Geo Spatial Dispatch Search", "/api/v1/priests/nearby?lat=17.3850&lng=78.4867&radius=15", total_requests=1000, concurrency=50)
    results.append(r4)

    # 5. Temple Directory Lookup
    r5 = run_load_test("Temple Directory Search", "/api/v1/temples", total_requests=1000, concurrency=50)
    results.append(r5)

    print("\n" + "="*60)
    print("🏆 BENCHMARK SUMMARY TABLE")
    print("="*60)
    print(f"{'Endpoint':<32} | {'Reqs':<6} | {'RPS':<9} | {'p50(ms)':<8} | {'p95(ms)':<8} | {'p99(ms)':<8}")
    print("-" * 80)
    for res in results:
        print(f"{res['name']:<32} | {res['total_requests']:<6} | {res['rps']:<9.1f} | {res['p50']:<8.2f} | {res['p95']:<8.2f} | {res['p99']:<8.2f}")
    print("="*60)
