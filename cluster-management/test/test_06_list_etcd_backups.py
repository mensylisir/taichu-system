import sys
import os
sys.path.append(os.path.dirname(os.path.abspath(__file__)))
from config import BASE_URL, print_response, get_headers
import requests

def test_list_etcd_backups():
    cluster_id = "test-cluster-import"
    url = f"{BASE_URL}/clusters/{cluster_id}/etcd/backups"
    response = requests.get(url, headers=get_headers())
    print_response(response, f"获取etcd备份列表: {cluster_id}")

if __name__ == "__main__":
    test_list_etcd_backups()
