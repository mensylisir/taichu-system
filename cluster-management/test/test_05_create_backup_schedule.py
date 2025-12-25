import sys
import os
sys.path.append(os.path.dirname(os.path.abspath(__file__)))
from config import BASE_URL, print_response, get_headers
import requests

def test_create_backup_schedule():
    cluster_id = "af28a7e5-022b-4997-a6fc-eb2cb3a04395"
    payload = {
        "name": "test-etcd-schedule",
        "cron_expr": "0 2 * * *",
        "backup_type": "etcd",
        "retention_days": 7,
        "enabled": True,
        "created_by": "admin",
        
        "etcd_endpoints": "https://172.30.1.12:2379,https://172.30.1.14:2379,https://172.30.1.15:2379",
        "etcd_ca_cert": "/etc/ssl/etcd/ssl/ca.pem",
        "etcd_cert": "/etc/ssl/etcd/ssl/admin-node2.pem",
        "etcd_key": "/etc/ssl/etcd/ssl/admin-node2-key.pem",
        "etcd_data_dir": "/var/lib/etcd",
        "etcdctl_path": "/usr/local/bin/etcdctl",
        "ssh_username": "root",
        "ssh_password": "password",
        
        "etcd_deployment_type": "static",
        "k8s_deployment_type": "kubeadm"
    }
    url = f"{BASE_URL}/clusters/{cluster_id}/backup-schedules"
    response = requests.post(url, json=payload, headers=get_headers())
    print_response(response, f"创建备份计划: {cluster_id}")

if __name__ == "__main__":
    test_create_backup_schedule()
