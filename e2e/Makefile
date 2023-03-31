KIND_VERSION := v0.17.0
HELM_VERSION := v3.11.2
KIND := bin/kind_$(KIND_VERSION)
HELM := bin/helm_$(HELM_VERSION)
ANY_EXPORTER := ../any-exporter
IMAGE_NAME := ghcr.io/peng225/any-exporter
IMAGE_TAG := dev

.PHONY: test
test: $(ANY_EXPORTER)
	$(ANY_EXPORTER) &
	go test -v
	pkill $(notdir $(ANY_EXPORTER))

.PHONY: test-with-helm
test-with-helm:
	go test -v

$(ANY_EXPORTER):
	make -C ../

.PHONY: setup-by-helm
setup-by-helm: $(KIND) $(HELM)
	$(KIND) create cluster
	kubectl create ns any-exporter
	docker build -t $(IMAGE_NAME):$(IMAGE_TAG) ..
	$(KIND) load docker-image $(IMAGE_NAME):$(IMAGE_TAG)
	$(HELM) install --namespace any-exporter --set image.tag=$(IMAGE_TAG) any-exporter ../charts/any-exporter
	sleep 10
	kubectl wait pods -n any-exporter -l app.kubernetes.io/name=any-exporter --for condition=Ready --timeout=30s
	kubectl get all -n any-exporter
	kubectl port-forward -n any-exporter service/any-exporter 8080:8080 &

$(KIND): | bin
	curl -Lo ./kind https://kind.sigs.k8s.io/dl/$(KIND_VERSION)/kind-linux-amd64
	chmod +x ./kind
	mv ./kind $@

$(HELM): | bin
	curl -fsSL https://get.helm.sh/helm-$(HELM_VERSION)-linux-amd64.tar.gz | tar xzv
	mv linux-amd64/helm $@
	rm -rf linux-amd64

bin:
	mkdir $@

.PHONY: clean
clean: $(KIND)
	pkill kubectl
	$(KIND) delete cluster
