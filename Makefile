CC := gcc
CFLAGS := -Wall -Wextra -O2 -Iinclude

# Directories
SRC_DIR := ./src/parser
OBJ_DIR := build
BIN_DIR := bin
TARGET := $(BIN_DIR)/parser

# Source and object files
SRCS := $(wildcard $(SRC_DIR)/*.c)
OBJS := $(patsubst $(SRC_DIR)/%.c,$(OBJ_DIR)/%.o,$(SRCS))

# Default rule
all: $(TARGET)

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
	rm -r

parser: $(TARGET)
	./$(TARGET) ./src/parser/test_files/single_file.torrent

.PHONY: all clean