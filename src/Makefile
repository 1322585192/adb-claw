BINARY_NAME := adb-claw
BUILD_DIR := ../bin
MODULE := github.com/llm-net/adb-claw
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GO := go
LDFLAGS := -ldflags "-X $(MODULE)/cmd.Version=$(VERSION)"

DEX_SRC := ../helper/ADBClawBridge.java
DEX_DST := pkg/frame/classes.dex
AUDIO_DEX_SRC := ../helper/ADBClawAudio.java
AUDIO_DEX_DST := pkg/audio/classes.dex
INPUT_DEX_SRC := ../helper/ADBClawInput.java
INPUT_DEX_DST := pkg/input/classes.dex
JAVAC := javac
ANDROID_JAR ?= $(ANDROID_HOME)/platforms/android-33/android.jar
D8 := d8

.PHONY: build test lint clean tidy dex audio-dex input-dex

build: tidy
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) .

tidy:
	$(GO) mod tidy

test:
	$(GO) test ./...

lint:
	$(GO) vet ./...

clean:
	rm -rf $(BUILD_DIR)

dex:
	@echo "Building frame DEX..."
	@mkdir -p /tmp/adbclaw-dex
	$(JAVAC) --release 11 -classpath $(ANDROID_JAR) -d /tmp/adbclaw-dex $(DEX_SRC)
	$(D8) --output /tmp/adbclaw-dex /tmp/adbclaw-dex/*.class
	cp /tmp/adbclaw-dex/classes.dex $(DEX_DST)
	@rm -rf /tmp/adbclaw-dex
	@echo "DEX built: $(DEX_DST)"

audio-dex:
	@echo "Building audio capture DEX..."
	@mkdir -p /tmp/adbclaw-audio-dex
	$(JAVAC) --release 11 -classpath $(ANDROID_JAR) -d /tmp/adbclaw-audio-dex $(AUDIO_DEX_SRC)
	$(D8) --output /tmp/adbclaw-audio-dex /tmp/adbclaw-audio-dex/ADBClawAudio.class
	cp /tmp/adbclaw-audio-dex/classes.dex $(AUDIO_DEX_DST)
	@rm -rf /tmp/adbclaw-audio-dex
	@echo "Audio DEX built: $(AUDIO_DEX_DST)"

input-dex:
	@echo "Building Unicode input DEX..."
	@mkdir -p /tmp/adbclaw-input-dex
	$(JAVAC) --release 11 -classpath $(ANDROID_JAR) -d /tmp/adbclaw-input-dex $(INPUT_DEX_SRC)
	$(D8) --output /tmp/adbclaw-input-dex /tmp/adbclaw-input-dex/ADBClawInput.class
	cp /tmp/adbclaw-input-dex/classes.dex $(INPUT_DEX_DST)
	@rm -rf /tmp/adbclaw-input-dex
	@echo "Input DEX built: $(INPUT_DEX_DST)"
