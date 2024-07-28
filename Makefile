VERSION_TAG ?= validating-v0.1.0
REGISTRY_HOST ?= ccr.ccs.tencentyun.com/public-proxy/k8s-webhook

build:
	go build -o k8s-webhook main.go
build-image: build
	docker build -t $(REGISTRY_HOST):$(VERSION_TAG) .
	docker push $(REGISTRY_HOST):$(VERSION_TAG)
deploy-k8s: build-image
	kustomize build kustomize/overlays/dev/ | sed "s/VERSION_TAG/${VERSION_TAG}/g" | kubectl apply -f -

