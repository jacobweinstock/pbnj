# PBnJ Helm Chart

Helm chart for deploying PBnJ to Kubernetes

## Local Development

```bash
# vanilla cluster
minikube start --driver virtualbox

# make sure helm is installed

# set up vault
kubectl create namespace vault
helm repo add hashicorp https://helm.releases.hashicorp.com
helm install vault hashicorp/vault --set "server.dev.enabled=true" --set "injector.enabled=false" -n vault
kubectl exec -it vault-0 -n vault -- sh -c '''
vault auth enable kubernetes

vault write auth/kubernetes/config \
    token_reviewer_jwt="$(cat /var/run/secrets/kubernetes.io/serviceaccount/token)" \
    kubernetes_host="https://$KUBERNETES_PORT_443_TCP_ADDR:443" \
	kubernetes_ca_cert=@/run/secrets/kubernetes.io/serviceaccount/ca.crt


vault write auth/kubernetes/role/kes \
    bound_service_account_names=kes \
    bound_service_account_namespaces=vault \
    policies=kes-policy \
    ttl=24h

vault secrets enable -path kubernetes/ kv-v2

vault policy write kes-policy - <<EOF
path "kubernetes/*" {
  capabilities = ["read", "list"]
}
EOF

vault kv put kubernetes/minikube/local/pbnj access-secret=1234 access-id=1234

vault kv get kubernetes/minikube/local/pbnj
'''

helm install kubernetes-external-secrets external-secrets/kubernetes-external-secrets --skip-crds --set env.LOG_LEVEL=debug --set env.VAULT_ADDR=http://vault:8200 --set serviceAccount.name=kes --set env.DEFAULT_VAULT_MOUNT_POINT="kubernetes" --set env.DEFAULT_VAULT_ROLE="kes" -n vault

# run tilt
tilt up

```