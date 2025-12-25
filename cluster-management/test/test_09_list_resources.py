import sys
import os
sys.path.append(os.path.dirname(os.path.abspath(__file__)))
from config import BASE_URL, print_response, get_headers
import requests

def test_list_tenants():
    url = f"{BASE_URL}/tenants"
    response = requests.get(url, headers=get_headers())
    print_response(response, "列出租户")

def test_list_environments():
    url = f"{BASE_URL}/environments"
    response = requests.get(url, headers=get_headers())
    print_response(response, "列出环境")

def test_list_applications():
    url = f"{BASE_URL}/applications"
    response = requests.get(url, headers=get_headers())
    print_response(response, "列出应用")

if __name__ == "__main__":
    test_list_tenants()
    test_list_environments()
    test_list_applications()
