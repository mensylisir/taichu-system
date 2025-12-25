import sys
import os
sys.path.append(os.path.dirname(os.path.abspath(__file__)))
from config import BASE_URL, print_response, get_headers
import requests

def test_create_etcd_backup():
    cluster_id = "af28a7e5-022b-4997-a6fc-eb2cb3a04395"
    payload = {
        "backup_name": "test-etcd-backup",
        "backup_type": "etcd",
        "description": "测试etcd备份"
    }
    url = f"{BASE_URL}/clusters/{cluster_id}/etcd/backups"
    response = requests.post(url, json=payload, headers=get_headers())
    print_response(response, f"创建etcd备份: {cluster_id}")

if __name__ == "__main__":
    test_create_etcd_backup()
