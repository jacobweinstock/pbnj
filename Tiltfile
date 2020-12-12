docker_build('pbnj', '.', 
    dockerfile='./Dockerfile')
k8s_yaml(helm(
    'kubernetes/',
    set=[
        'env=minikube',
        'location=local',
        'clusterFQDN=local',
        'pbnj.image.repository=pbnj',
        'pbnj.image.tag=latest'
        ]
    )
)
k8s_resource('pbnj', port_forwards=[50051,9090])