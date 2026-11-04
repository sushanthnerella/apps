"""
Live Backend Mock Server (HTTP + REST API)
Provides live endpoints for:
- GET /health
- GET /api/v1/poojas
- GET /api/v1/panchangam/today
- POST /api/v1/dispatch/match (Rapido Nearest Match & Scorer)
"""

import json
from http.server import HTTPServer, BaseHTTPRequestHandler
import math

active_priests = [
    {
        "id": "priest_101",
        "full_name": "Brahmasri Venkata Sharma Garu",
        "phone": "+91 98765 43210",
        "sampradaya": "Smartha",
        "rating_avg": 4.95,
        "total_completed_poojas": 184,
        "repeat_booking_count": 42,
        "acceptance_rate": 0.98,
        "distance_meters": 1800.0,
    },
    {
        "id": "priest_102",
        "full_name": "Purohit Narayana Acharya",
        "phone": "+91 98765 43211",
        "sampradaya": "Vaishnava",
        "rating_avg": 4.70,
        "total_completed_poojas": 96,
        "repeat_booking_count": 18,
        "acceptance_rate": 0.85,
        "distance_meters": 900.0,
    },
    {
        "id": "priest_103",
        "full_name": "Vedam Subrahmanya Sastry",
        "phone": "+91 98765 43212",
        "sampradaya": "Smartha",
        "rating_avg": 4.85,
        "total_completed_poojas": 210,
        "repeat_booking_count": 55,
        "acceptance_rate": 0.92,
        "distance_meters": 3400.0,
    }
]

def score_candidate(p, max_radius=5000.0, preferred_sampradaya="Smartha"):
    dist_ratio = min(p["distance_meters"] / max_radius, 1.0)
    distance_score = 1.0 - dist_ratio
    rating_score = min(p["rating_avg"] / 5.0, 1.0)
    repeat_score = min(p["repeat_booking_count"] / 50.0, 1.0)
    acceptance_score = min(p["acceptance_rate"], 1.0)
    sampradaya_score = 1.0 if p["sampradaya"].lower() == preferred_sampradaya.lower() else 0.5
    
    total = (0.35 * distance_score) + (0.25 * rating_score) + (0.20 * repeat_score) + (0.10 * acceptance_score) + (0.10 * sampradaya_score)
    return round(total, 3)

class PurohitHandler(BaseHTTPRequestHandler):
    def _send_json(self, data, status=200):
        self.send_response(status)
        self.send_header('Content-Type', 'application/json')
        self.send_header('Access-Control-Allow-Origin', '*')
        self.end_headers()
        self.wfile.write(json.dumps(data, indent=2).encode('utf-8'))

    def do_GET(self):
        if self.path == '/health':
            self._send_json({"status": "UP", "service": "Purohit Platform Gateway", "version": "1.0.0"})
        elif self.path == '/api/v1/poojas':
            catalog = [
                {
                    "id": "pooja_1",
                    "title": "Gruhapravesam (House Warming)",
                    "category": "Major Occasion",
                    "base_price": 5116.0,
                    "samagri_kit_price": 2500.0,
                    "duration_minutes": 180,
                    "is_online_eligible": False
                },
                {
                    "id": "pooja_2",
                    "title": "Sri Satyanarayana Swamy Vratam",
                    "category": "Prosperity & Devotion",
                    "base_price": 2516.0,
                    "samagri_kit_price": 1200.0,
                    "duration_minutes": 120,
                    "is_online_eligible": True
                },
                {
                    "id": "pooja_3",
                    "title": "Maha Ganapathi Homa / Yagam",
                    "category": "Obstacle Removal",
                    "base_price": 3116.0,
                    "samagri_kit_price": 1800.0,
                    "duration_minutes": 90,
                    "is_online_eligible": True
                }
            ]
            self._send_json(catalog)
        elif self.path == '/api/v1/panchangam/today':
            self._send_json({
                "tithi": "Shukla Paksha Panchami",
                "nakshatra": "Rohini Nakshatra",
                "yoga": "Siddhi Yoga",
                "karana": "Bava",
                "rahu_kalam": "01:30 PM - 03:00 PM (Avoid)",
                "yamagandam": "06:00 AM - 07:30 AM",
                "gulika_kalam": "09:00 AM - 10:30 AM",
                "is_auspicious": True
            })
        else:
            self._send_json({"error": "Not Found"}, status=404)

    def do_POST(self):
        if self.path == '/api/v1/dispatch/match':
            content_length = int(self.headers.get('Content-Length', 0))
            body = self.rfile.read(content_length).decode('utf-8') if content_length > 0 else "{}"
            try:
                req = json.loads(body)
            except:
                req = {}
            
            sampradaya = req.get("required_sampradaya", "Smartha")
            radius = float(req.get("search_radius_meters", 5000.0))
            
            ranked = []
            for p in active_priests:
                if p["distance_meters"] <= radius:
                    scored_p = dict(p)
                    scored_p["score"] = score_candidate(p, radius, sampradaya)
                    ranked.append(scored_p)
            
            ranked.sort(key=lambda x: x["score"], reverse=True)
            self._send_json({
                "booking_id": req.get("booking_id", "bk_test_88"),
                "total_candidates": len(ranked),
                "best_match": ranked[0] if ranked else None,
                "candidates": ranked
            })
        else:
            self._send_json({"error": "Not Found"}, status=404)

if __name__ == '__main__':
    port = 8090
    server = HTTPServer(('127.0.0.1', port), PurohitHandler)
    print(f"Backend Server running on http://127.0.0.1:{port}")
    server.serve_forever()
