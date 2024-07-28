VERSION ?= valvalidating-v0.0.1

build:
    go build -o k8s-webhook main.go
build-image:
	docker build -t shenchangqing/k8s-webhook:$(VERSION)
	docker push shenchangqing/k8s-webhook:$(VERSION)
deploy-k8s:
    sed -i "s/#{VERSION}/${VERSION}/g" kustomize/overlays/dev/kustomization.yaml
    kustomize build kustomize/overlays/dev 
