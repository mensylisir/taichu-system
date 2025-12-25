import sys
import os
sys.path.append(os.path.dirname(os.path.abspath(__file__)))
from config import BASE_URL, print_response, get_headers
import requests

def test_update_backup_schedule():
    cluster_id = "af28a7e5-022b-4997-a6fc-eb2cb3a04395"
    schedule_id = "b14cd01c-ef9f-49a2-9b53-7e67ce537baa"
    payload = {
        "cron_expr": "0 2 * * *",
        "backup_type": "etcd",
        "retention_days": 7,
        "enabled": True,
        
        "etcd_endpoints": "https://172.30.1.12:2379,https://172.30.1.14:2379,https://172.30.1.15:2379",
        "etcd_ca_cert": "/etc/ssl/etcd/ssl/ca.pem",
        "etcd_cert": "/etc/ssl/etcd/ssl/admin-node2.pem",
        "etcd_key": "/etc/ssl/etcd/ssl/admin-node2-key.pem",
        "etcd_data_dir": "/var/lib/etcd",
        "etcdctl_path": "/usr/local/bin/etcdctl",
        "ssh_username": "root",
        "ssh_password": "Def@u1tpwd",
        
        "etcd_deployment_type": "static",
        "k8s_deployment_type": "kubeadm"
    }
    url = f"{BASE_URL}/clusters/{cluster_id}/backup-schedules/{schedule_id}"
    response = requests.put(url, json=payload, headers=get_headers())
    print_response(response, f"更新备份计划: {cluster_id}/{schedule_id}")

if __name__ == "__main__":
    test_update_backup_schedule()
