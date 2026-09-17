# The import ID format is: [project/{project_id}/]cluster_id,node_pool_id
terraform import mgc_kubernetes_nodepool.nodepool my_cluster_id,my_nodepool_id
terraform import mgc_kubernetes_nodepool.nodepool ac9709ae-b298-41c6-bc58-92a6b8b92d8b,cc3ae8e5-07ae-4c45-8573-600bd26e9f40

# Node pool in a project
terraform import mgc_kubernetes_nodepool.nodepool project/550e8400-e29b-41d4-a716-446655440000/ac9709ae-b298-41c6-bc58-92a6b8b92d8b,cc3ae8e5-07ae-4c45-8573-600bd26e9f40
