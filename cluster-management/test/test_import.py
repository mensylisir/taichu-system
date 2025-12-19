#!/usr/bin/env python3
import requests
import json
import uuid

# 读取kubeconfig文件
with open('test/kubeconfig.yaml', 'r') as f:
    kubeconfig_content = f.read()

# 获取认证token
token_response = requests.get('http://localhost:8081/api/v1/auth/token')
token_data = token_response.json()
token = token_data['data']['token']

# 生成唯一的集群名称
unique_name = f"test-cluster-{uuid.uuid4().hex[:8]}"

# 准备导入请求
import_data = {
    "import_source": "manual",
    "name": unique_name,
    "description": "Test cluster for import",
    "environment_type": "test",
    "kubeconfig": kubeconfig_content,
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
    'http://localhost:8081/api/v1/clusters/import',
    headers=headers,
    json=import_data
)

print(f"Status Code: {response.status_code}")
print(f"Response: {response.text}")