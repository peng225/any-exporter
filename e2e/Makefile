KIND_VERSION = v0.17.0
HELM_VERSION = v3.11.2
KIND = bin/kind
HELM = bin/hel
PROMBLOCK := ../promblock

.PHONY: test
test: $(PROMBLOCK)
	$(PROMBLOCK) &
	go test -v
	pkill $(notdir $(PROMBLOCK))

.PHONY: test-with-helm
 test-with-helm:
	go test -v

$(PROMBLOCK):
	make -C ../

.PHONY: setup-by-helm
setup-by-helm: $(KIND) $(HELM)
	$(KIND) create cluster
	kubectl create ns promblock
	$(HELM) install --namespace promblock promblock ../charts/promblock
	sleep 10
	kubectl wait pods -n promblock -l app.kubernetes.io/name=promblock --for condition=Ready --timeout=30s
	kubectl get all -n promblock
	kubectl port-forward -n promblock service/promblock 8080:8080 &

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
clean:
	pkill kubectl
	$(KIND) delete cluster
