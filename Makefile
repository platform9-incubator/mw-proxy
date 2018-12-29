SHELL := bash
BUILD_DIR := build
STAGE_DIR := $(BUILD_DIR)/mw-proxy-container
INITC_STAGE_DIR := $(BUILD_DIR)/mw-proxy-init-container
EXE := $(BUILD_DIR)/mw-proxy
SRC := */*.go
TAG ?= platform9/mw-proxy:latest
INITC_TAG ?= platform9/mw-proxy-init:latest

$(BUILD_DIR):
	mkdir -p $@

$(EXE): $(SRC) $(BUILD_DIR)
	pushd cmd && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
		go build --buildmode=exe -o ../$(EXE)

exe: $(EXE)

image: $(EXE)
	cp -a containers/mw-proxy $(STAGE_DIR)
	cp $(EXE) $(STAGE_DIR)
	docker build --rm -t $(TAG) $(STAGE_DIR)

push: image
	docker push $(TAG) && docker rmi $(TAG)

initc-image:
	cp -a containers/mw-proxy-init $(INITC_STAGE_DIR)
	docker build --rm -t $(INITC-TAG) $(INITC_STAGE_DIR)

initc-push: initc-image
	docker push $(INITC-TAG) && docker rmi $(INITC-TAG)

clean:
	rm -rf $(BUILD_DIR)
