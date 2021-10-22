
APPLICATION = mireg
REGISTRY = mireg.wr25.org
KUBECTLOPTS =

all: apply.touch


mireg: main.go Makefile
	go build .

docker.digest: mireg Dockerfile
	podman build -t mireg.wr25.org/mireg:latest .

	podman tag mireg.wr25.org/mireg:latest $(REGISTRY)/mireg:latest
	podman push  $(REGISTRY)/mireg:latest
	
	echo -n "sha256:" > docker.digest
	curl -H "Accept: application/vnd.docker.distribution.manifest.v2+json" https://$(REGISTRY)/v2/$(APPLICATION)/manifests/latest | sha256sum | awk '{print $$1}' >> docker.digest

	sed -i "s#image: $(REGISTRY)/$(APPLICATION):latest.*#image: $(REGISTRY)/$(APPLICATION):latest@$$(cat docker.digest)#" deployment.yml


apply.touch: deployment.yml docker.digest
	kubectl $(KUBECTLOPTS) apply -f deployment.yml
	kubectl $(KUBECTLOPTS) rollout status deployment $(APPLICATION)
	touch apply.touch

restart:
	kubectl $(KUBECTLOPTS) scale --replicas=0 deployment mireg
	sleep 3
	kubectl $(KUBECTLOPTS) scale --replicas=1 deployment mireg


clean:
	rm -f mireg docker.digest apply.touch
