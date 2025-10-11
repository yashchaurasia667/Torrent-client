CC := gcc
CFLAGS := -Wall -Wextra -O2 -Iinclude

GO := go
GO_SRC := ./src
GO_OUT := ./out

# Directories
SRC_DIR := ./src/parser
OBJ_DIR := ./build
BIN_DIR := ./bin
TARGET := $(BIN_DIR)/parser

# Source and object files
SRCS := $(wildcard $(SRC_DIR)/*.c)
OBJS := $(patsubst $(SRC_DIR)/%.c,$(OBJ_DIR)/%.o,$(SRCS))

# Default rule
all: run

# Link the target
$(TARGET): $(OBJS) | $(BIN_DIR)
	$(CC) $(CFLAGS) -o $@ $^

# Compile source files to object files
$(OBJ_DIR)/%.o: $(SRC_DIR)/%.c | $(OBJ_DIR)
	$(CC) $(CFLAGS) -c $< -o $@

# Create necessary directories
$(BIN_DIR) $(OBJ_DIR):
	mkdir -p $@

# Clean build artifacts
clean:
	rm -r $(GO_OUT)
	rm -r $(BIN_DIR)

parser: $(TARGET)
	./$(TARGET) ./src/parser/test_files/single_file.torrent

.PHONY: all clean

go-build: $(GO_SRC) | $(BIN_DIR)
	$(GO) -o ./$(BIN_DIR) $(GO_SRC)

run: $(GO_SRC)
	$(GO) run $(GO_SRC)/main.go $(GO_SRC)/test_files/multiple_files.torrent $(GO_OUT)

torrent-client: go-build
	./$(BIN_DIR)/torrent-client