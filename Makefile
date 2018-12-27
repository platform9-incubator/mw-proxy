STAGE_DIR := build
EXE := $(STAGE_DIR)/mw-proxy
SRC := */*.go
TAG ?= platform9/mw-proxy:latest

$(STAGE_DIR):
	mkdir -p $@

$(EXE): $(SRC) $(STAGE_DIR)
	pushd cmd && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
		go build --buildmode=exe -o ../$(EXE)

exe: $(EXE)

image: $(EXE)
	cp Dockerfile $(STAGE_DIR)
	docker build --rm -t $(TAG) $(STAGE_DIR)

push: image
	docker push $(TAG) && docker rmi $(TAG)

clean:
	rm -rf $(STAGE_DIR)
