import requests
import json
from typing import Optional, Dict, Any

BASE_URL = "http://localhost:8086/api/v1"

def print_response(response: requests.Response, title: str = ""):
    print(f"\n{'='*60}")
    if title:
        print(f"{title}")
    print(f"{'='*60}")
    print(f"Status Code: {response.status_code}")
    print(f"Response: {json.dumps(response.json(), indent=2, ensure_ascii=False)}")
    print(f"{'='*60}\n")

def get_headers() -> Dict[str, str]:
    return {
        "Content-Type": "application/json"
    }
