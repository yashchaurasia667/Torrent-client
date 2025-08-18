#ifndef PARSER_H
#define PARSER_H

#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <stdbool.h>
#include <string.h>
#include <time.h>
#include <math.h>

#define okay(msg, ...) printf("[+] " msg "\n", ##__VA_ARGS__)
#define info(msg, ...) printf("[*] " msg "\n", ##__VA_ARGS__)
#define warn(msg, ...) printf("[-] " msg "\n", ##__VA_ARGS__)
#define HASH_LENGTH 20

typedef struct
{
  uint64_t length;
  char **path;
} File;

typedef struct
{
  char *name;
  uint64_t pieceLength;
  uint8_t *pieces;
  size_t pieceCount;
  uint64_t length;
  File *files;
  size_t fileCount;
} Info;

typedef struct
{
  bool hasMultipleFiles;
  char *announce;
  char **announceList;
  uint32_t announceUrlCount;
  char *createdBy;
  time_t creationDate;
  char *encoding;
  char *comment;
  Info info;
  char *infoHash;
} Torrent;

void dumpToJson(FILE *out, Torrent *t);
char *bencodeString(char *str);
char *bencodeInteger(uint64_t n);
char *append(char *str1, char *str2, char *str3);

#endif