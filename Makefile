APPLICATION = mireg

REGISTRY ?= mireg.wr25.org
KUBECTLOPTS ?=
RELEASE ?= latest
DOCKERCOMMAND ?= podman

GOCODE := $(shell find . -name \*.go -print)

.PHONY: clean

all: apply.touch

$(APPLICATION): $(GOCODE) Makefile
	go test
	go build .

docker.digest: Dockerfile docker_entrypoint.sh $(APPLICATION)
	$(DOCKERCOMMAND) build -t $(REGISTRY)/$(APPLICATION):$(RELEASE) .
	$(DOCKERCOMMAND) push $(REGISTRY)/$(APPLICATION):$(RELEASE)

	echo -n "sha256:" > docker.digest 
	curl -s -H "Accept: application/vnd.docker.distribution.manifest.v2+json" https://$(REGISTRY)/v2/$(APPLICATION)/manifests/$(RELEASE) | sha256sum | awk '{print $$1}' >> docker.digest

deployment.apply.yml: docker.digest deployment.yml
	cat deployment.yml | sed "s#image: [^/]*/$(APPLICATION):.*#image: $(REGISTRY)/$(APPLICATION):$(RELEASE)@$$(cat docker.digest)#" > deployment.apply.yml

apply.touch: deployment.apply.yml
	kubectl $(KUBECTLOPTS) apply -f deployment.apply.yml
	kubectl $(KUBECTLOPTS) rollout status deployment $(APPLICATION)
	touch apply.touch

clean:
	rm $(APPLICATION) docker.digest apply.touch -f
