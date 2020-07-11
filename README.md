# Admission Mutation Proxy (txn2/amp) implementation example project.

Example Webhook implementation for the [Admission Mutation Proxy (amp)](https://github.com/txn2/amp).

https://github.com/txn2/amp

## Install

```shell script
git clone git@github.com:txn2/amp-wh-example.git
cd amp-wh-example

# amp webhook handler
kubectl apply -f ./k8s/000-deployment-exampe-webhook.yml

# namespace with amp.txn2.com/enabled: "true"
kubectl apply -f ./k8s/100-namespace-amp-test.yml

# pod to be mutated
kubectl apply -f ./k8s/200-pod-mutate-test.yml
```

## Development
### Release
```bash
goreleaser --skip-publish --rm-dist --skip-validate
```

```bash
GITHUB_TOKEN=$GITHUB_TOKEN goreleaser --rm-dist
```