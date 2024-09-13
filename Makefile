VERSION_TAG ?= validating-v0.1.0
REGISTRY_ADDR ?= ccr.ccs.tencentyun.com/public-proxy/k8s-webhook

build:
	GOOS=linux go build -o k8s-webhook main.go
build-image: build
	docker build -t $(REGISTRY_ADDR):$(VERSION_TAG) .
	docker push $(REGISTRY_ADDR):$(VERSION_TAG)
deploy-k8s: build-image
	kustomize build kustomize/overlays/dev/ | sed -e "s|VERSION_TAG|${VERSION_TAG}|g" -e "s|REGISTRY_ADDR|${REGISTRY_ADDR}|g" | kubectl apply -f -

deploy-k8s-local:
	kustomize build kustomize/overlays/dev/ | sed -e "s|VERSION_TAG|${VERSION_TAG}|g" -e "s|REGISTRY_ADDR|${REGISTRY_ADDR}|g" | kubectl apply -f -
