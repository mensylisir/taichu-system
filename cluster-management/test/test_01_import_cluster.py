import sys
import os
sys.path.append(os.path.dirname(os.path.abspath(__file__)))
from config import BASE_URL, print_response, get_headers
import requests

def test_import_cluster():
    script_dir = os.path.dirname(os.path.abspath(__file__))
    kubeconfig_path = os.path.join(script_dir, "kubeconfig.yaml")
    
    with open(kubeconfig_path, "r", encoding="utf-8") as f:
        kubeconfig_content = f.read()
    
    payload = {
        "import_source": "kubeconfig",
        "name": "test-cluster-import",
        "description": "测试集群导入",
        "environment_type": "production",
        "region": "default",
        "kubeconfig": kubeconfig_content,
        "labels": {
            "test": "true",
            "imported": "manual"
        }
    }
    
    url = f"{BASE_URL}/clusters/import"
    response = requests.post(url, json=payload, headers=get_headers())
    print_response(response, "测试集群导入")

if __name__ == "__main__":
    test_import_cluster()
