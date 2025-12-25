import sys
import os
sys.path.append(os.path.dirname(os.path.abspath(__file__)))
from config import BASE_URL, print_response, get_headers
import requests

def test_create_cluster_backup():
    cluster_id = "test-cluster-import"
    payload = {
        "name": "test-cluster-backup",
        "description": "测试集群备份"
    }
    url = f"{BASE_URL}/clusters/{cluster_id}/backups"
    response = requests.post(url, json=payload, headers=get_headers())
    print_response(response, f"创建集群备份: {cluster_id}")

if __name__ == "__main__":
    test_create_cluster_backup()
