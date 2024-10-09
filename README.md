# k8sgpt-kyverno-custom-analyzer-draft
We are comparing "integrations" versus "custom analyzers", so trying Kyverno as an example.

Install Kyverno - `kubectl create -f https://github.com/kyverno/kyverno/releases/download/v1.11.1/install.yaml`

Build this custom analyzer - `docker image build -f Dockerfile -t kyverno-ca:v0.0.0 .`

Setup permissions for custom analayzer (retriving kyverno reports)):

- `kubectl create clusterrole getpols --resource policyreports.wgpolicyk8s.io --verb get,list`
- `kubectl create clusterrolebinding getpols --serviceaccount default:default --clusterrole getpols`
- `kubectl auth can-i --as system:serviceaccount:default:default list policyreports.wgpolicyk8s.io`

- `kubectl create clusterrole getcpols --resource clusterpolicyreports.wgpolicyk8s.io --verb get,list`
- `kubectl create clusterrolebinding getcpols --serviceaccount default:default --clusterrole getcpols`
- `kubectl auth can-i --as system:serviceaccount:default:default list clusterpolicyreports.wgpolicyk8s.io`

Create Deployment/Service for this custom analyzer:

```
 % cat kyverno-ca.yaml 
apiVersion: apps/v1
kind: Deployment
metadata:
  name: kyverno-ca
  labels:
    app: kyverno-ca
spec:
  replicas: 1
  selector:
    matchLabels:
      app: kyverno-ca
  template:
    metadata:
      labels:
        app: kyverno-ca
        team: makethecut
    spec:
      containers:
      - name: kyverno-ca
        image: kyverno-ca:v0.0.0
        ports:
        - containerPort: 8085
---
apiVersion: v1
kind: Service
metadata:
  name: kyverno-ca
spec:
  type: NodePort   
  selector:
    app: kyverno-ca
  ports:
  - name: http
    port: 8085
    protocol: TCP
    targetPort: 8085
    nodePort: 30000
```

`kubectl apply -f kyverno-ca.yaml`


Setup a debug pod to confirm RPC reachability:

- `kubectl run client --image=ubuntu:24.04 -- tail -f /dev/null`
- `kubectl exec client -- sh -c 'apt update; apt install curl -y'`
- `kubectl exec client -- curl -sLO https://github.com/fullstorydev/grpcurl/releases/download/v1.9.1/grpcurl_1.9.1_linux_amd64.deb`
- `kubectl exec client -- dpkg -i grpcurl_1.9.1_linux_amd64.deb`
- `kubectl exec client -- grpcurl --plaintext kyverno-ca:8085 schema.v1.AnalyzerService/Run`

Install Kyverno sample ClusterPolicy. This policy is across namespaces and requires "team" label key.

```
% cat testpol.yaml                                         
apiVersion: kyverno.io/v1
kind: ClusterPolicy
metadata:
  name: require-labels
spec:
  validationFailureAction: Enforce
  rules:
  - name: check-team
    match:
      any:
      - resources:
          kinds:
          - Pod
    validate:
      message: "label 'team' is required"
      pattern:
        metadata:
          labels:
            team: "?*"

kubectl apply -f testpol.yaml
```

This takes a moment before results come back.

`kubectl get policyreports,clusterpolicyreports -A`

From laptop (assuming 30000 is reachable):

```
/usr/local/Cellar/grpcurl/1.9.1/bin/grpcurl --plaintext localhost:30000 list
grpc.reflection.v1.ServerReflection
grpc.reflection.v1alpha.ServerReflection
schema.v1.AnalyzerService
```


## Install & configure K8sGPT

`brew reinstall k8sgpt` # think update happens

```
% cat ~/Library/Application\ Support/k8sgpt/k8sgpt.yaml
active_filters:
#    - PolicyReport
    - CronJob
    - PersistentVolumeClaim
    - Ingress
    - StatefulSet
    - Service
    - ValidatingWebhookConfiguration
#    - ClusterPolicyReport
    - Node
    - Pod
    - MutatingWebhookConfiguration
    - Deployment
    - ReplicaSet
ai:
    defaultprovider: ""
    providers:
        - customheaders: []
          maxtokens: 2048
          model: gpt-3.5-turbo
          name: openai
          password: sk-proj-...
          temperature: 0.7
          topk: 50
          topp: 0.5
        - baseurl: http://localhost:11434/
          customheaders: []
          maxtokens: 2048
          model: llama3
          name: ollama
          temperature: 0.7
          topk: 50
          topp: 0.5
kubeconfig: ""
kubecontext: ""
custom_analyzers:
  - name: kyverno-ca 
    connection:
      url: localhost
      port: 30000
```

## Calling from K8sGPT

```
k8sgpt analyze --custom-analysis --explain -b openai
 100% |█████████████████████████████████████████████████████████████████████████████████████████| (1/1, 3620 it/s)        
AI Provider: openai

0: kyverno-ca kyverno()
- Error: Kyverno reports pols: errors fix them
Error: Kyverno reports policy errors, fix them.
Solution: 
1. Identify the specific policy errors reported by Kyverno.
2. Review the policies and make necessary corrections to comply with Kubernetes best practices.
3. Apply the corrected policies to resolve the errors.
```
