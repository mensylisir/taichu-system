import sys
import os
sys.path.append(os.path.dirname(os.path.abspath(__file__)))
from config import BASE_URL, print_response, get_headers
import requests

def test_list_environments():
    url = f"{BASE_URL}/environments"
    response = requests.get(url, headers=get_headers())
    print_response(response, "获取环境列表")

if __name__ == "__main__":
    test_list_environments()
