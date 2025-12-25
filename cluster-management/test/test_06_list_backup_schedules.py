import sys
import os
sys.path.append(os.path.dirname(os.path.abspath(__file__)))
from config import BASE_URL, print_response, get_headers
import requests

def test_list_backup_schedules():
    cluster_id = "af28a7e5-022b-4997-a6fc-eb2cb3a04395"
    url = f"{BASE_URL}/clusters/{cluster_id}/backup-schedules"
    response = requests.get(url, headers=get_headers())
    print_response(response, f"列出备份计划: {cluster_id}")

if __name__ == "__main__":
    test_list_backup_schedules()
