SHELL := bash
BUILD_DIR := build
STAGE_DIR := $(BUILD_DIR)/mw-proxy-container
IMAGE_MARKER := $(STAGE_DIR)/marker
INITC_STAGE_DIR := $(BUILD_DIR)/mw-proxy-init-container
INITC_IMAGE_MARKER := $(INITC_STAGE_DIR)/marker
EXE := $(BUILD_DIR)/mw-proxy
SRC := */*.go
TAG ?= platform9/mw-proxy:latest
INITC_TAG ?= platform9/mw-proxy-init:latest

$(BUILD_DIR):
	mkdir -p $@

$(EXE): $(SRC) | $(BUILD_DIR)
	pushd cmd && GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
		go build --buildmode=exe -o ../$(EXE)

exe: $(EXE)

$(IMAGE_MARKER): $(EXE)
	cp -a containers/mw-proxy $(STAGE_DIR)
	cp $(EXE) $(STAGE_DIR)
	docker build --rm -t $(TAG) $(STAGE_DIR) && touch $(IMAGE_MARKER)

image: $(IMAGE_MARKER)

push: image
	docker push $(TAG) && docker rmi $(TAG) && rm -rf $(STAGE_DIR)

$(INITC_IMAGE_MARKER): | $(BUILD_DIR)
	cp -a containers/mw-proxy-init $(INITC_STAGE_DIR)
	docker build --rm -t $(INITC_TAG) $(INITC_STAGE_DIR) && touch $(INITC_IMAGE_MARKER)

initc-image: $(INITC_IMAGE_MARKER)

initc-push: initc-image
	docker push $(INITC_TAG) && docker rmi $(INITC_TAG) && rm -rf $(INITC_STAGE_DIR)

clean:
	rm -rf $(BUILD_DIR)
