#!/usr/bin/env python3
import requests
import json
import uuid
import os
import time

# 获取当前脚本所在目录
script_dir = os.path.dirname(os.path.abspath(__file__))
kubeconfig_path = os.path.join(script_dir, 'kubeconfig.yaml')

# 读取kubeconfig文件
with open(kubeconfig_path, 'r') as f:
    kubeconfig_content = f.read()

# 获取认证token
token_response = requests.get('http://127.0.0.1:8081/api/v1/auth/token')
token_data = token_response.json()
token = token_data['data']['token']

# 生成唯一的集群名称
unique_name = f"test-cluster-{uuid.uuid4().hex[:8]}"

# 准备导入请求
import_data = {
    "import_source": "manual",
    "name": unique_name,
    "description": "Test cluster for import with base64 encoding",
    "environment_type": "test",
    "kubeconfig": kubeconfig_content,
    "region": "cn-east-1",
    "labels": {
        "env": "test",
        "team": "devops"
    }
}

print(f"Importing cluster with name: {unique_name}")

# 发送导入请求
headers = {
    'Content-Type': 'application/json',
    'Authorization': f'Bearer {token}'
}

response = requests.post(
    'http://127.0.0.1:8081/api/v1/clusters/import',
    headers=headers,
    json=import_data
)

print(f"Status Code: {response.status_code}")
print(f"Response: {response.text}")

# 如果导入成功，检查集群状态和相关数据
if response.status_code in [200, 201]:
    response_data = response.json()
    if 'data' in response_data and 'id' in response_data['data']:
        import_id = response_data['data']['id']
        cluster_id = response_data['data'].get('cluster_id', '')
        print(f"\nImport record created with ID: {import_id}")
        print(f"Cluster ID: {cluster_id}")
        
        # 如果cluster_id为空，等待一段时间让后台任务完成
        if not cluster_id:
            print("Cluster ID is empty, waiting for background tasks to complete...")
            time.sleep(10)
            
            # 获取导入状态以获取cluster_id
            import_status_response = requests.get(
                f'http://127.0.0.1:8081/api/v1/imports/{import_id}/status',
                headers=headers
            )
            
            if import_status_response.status_code == 200:
                import_status_data = import_status_response.json()
                cluster_id = import_status_data.get('data', {}).get('cluster_id', '')
                print(f"Got cluster ID from import status: {cluster_id}")
        
        # 如果仍然没有cluster_id，退出
        if not cluster_id:
            print("No cluster ID available, exiting...")
            exit(1)
        
        # 等待一段时间让后台任务完成
        print("Waiting for background tasks to complete...")
        time.sleep(10)
        
        # 检查集群状态和资源数据（通过GetCluster接口获取）
        cluster_response = requests.get(
            f'http://127.0.0.1:8081/api/v1/clusters/{cluster_id}',
            headers=headers
        )
        
        if cluster_response.status_code == 200:
            cluster_data = cluster_response.json()
            print(f"\nCluster Status: {cluster_data['data']['status']}")
            print(f"Cluster Node Count: {cluster_data['data'].get('node_count', 'N/A')}")
            print(f"Cluster CPU Usage: {cluster_data['data'].get('cpu_usage_percent', 'N/A')}%")
            print(f"Cluster Memory Usage: {cluster_data['data'].get('memory_usage_percent', 'N/A')}%")
            print(f"Cluster Storage Usage: {cluster_data['data'].get('storage_usage_percent', 'N/A')}%")
            print(f"Cluster Details: {json.dumps(cluster_data['data'], indent=2)}")
        else:
            print(f"\nFailed to get cluster details: {cluster_response.status_code} - {cluster_response.text}")
        
        # 等待更长时间让后台任务完成
        print("Waiting longer for background tasks to complete...")
        time.sleep(20)
        
        # 再次检查集群状态和资源数据
        final_cluster_response = requests.get(
            f'http://127.0.0.1:8081/api/v1/clusters/{cluster_id}',
            headers=headers
        )
        
        if final_cluster_response.status_code == 200:
            final_cluster_data = final_cluster_response.json()
            print(f"\nFinal Cluster Status: {final_cluster_data['data']['status']}")
            print(f"Final Cluster Node Count: {final_cluster_data['data'].get('node_count', 'N/A')}")
            print(f"Final Cluster CPU Usage: {final_cluster_data['data'].get('cpu_usage_percent', 'N/A')}%")
            print(f"Final Cluster Memory Usage: {final_cluster_data['data'].get('memory_usage_percent', 'N/A')}%")
            print(f"Final Cluster Storage Usage: {final_cluster_data['data'].get('storage_usage_percent', 'N/A')}%")
        else:
            print(f"\nFailed to get final cluster details: {final_cluster_response.status_code} - {final_cluster_response.text}")
else:
    print("Failed to import cluster")
