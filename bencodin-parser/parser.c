#include <stdio.h>
#include <stdlib.h>
#include <stdint.h>
#include <string.h>

// structure of a torrent file
/*
{
  announce: [URL OF THE TRACKER]
  created by: [CREATED BY]
  creation date: [CREATION DATE]
  encoding: [ENCODING]
  comment: [COMMENTS IF ANY]
  info: {
    [SINGLE FILE TYPE]
    {
      name: [NAME]
      length: [LENGTH]
      piece length: [PIECE LENGTH]
      pieces: [PIECES]
    }

    [MULTIPLE FILE TYPE]
    {
      name: [NAME]
      piece length: [PIECE LENGTH]
      pieces: [CONCATINATION OF 20 BYTE SHA1 SUM OF ALL PIECES]
    }
  }
}
*/

#define okay(msg, ...) printf("[+] " msg " \n", ##__VA_ARGS__)
#define info(msg, ...) printf("[*] " msg " \n", ##__VA_ARGS__)
#define warn(msg, ...) printf("[-] " msg " \n", ##__VA_ARGS__)

typedef struct
{
  uint64_t length;
  char path[][];
} file;

typedef struct
{
  char *name;
  uint64_t pieceLength;
  char *pieces;
  uint64_t length;
  file files[];
} Info;
typedef struct
{
  char *announce;
  char *createdBy;
  uint64_t creationDate;
  char *encoding;
  Info *info;
} torrent;

torrent meta = {0};

char *parseString(FILE *fp);
uint64_t parseInteger(FILE *fp, char delimiter);
char *getFileContent(FILE *fp);
void parseTokens(FILE *fp);

int main(int argc, char **argv)
{
  if (argc == 1)
  {
    warn("A file name is needed...");
    return -1;
  }

  info("Trying to open %s", argv[1]);
  FILE *fp = fopen(argv[1], "rb");

  if (fp == NULL)
  {
    warn("Could not find %s", argv[1]);
    return -1;
  }

  parseTokens(fp);
  fclose(fp);

  return 0;
}

char *getFileContent(FILE *fp)
{
  fseek(fp, 0, SEEK_END);
  long fileSize = ftell(fp);
  rewind(fp);

  char *buffer = (char *)malloc(fileSize + 1);
  if (buffer == NULL)
  {
    warn("Failed to allocate memory");
    fclose(fp);
    exit(EXIT_FAILURE);
  }

  fread(buffer, sizeof(char), fileSize, fp);
  buffer[fileSize] = '\0';
  return buffer;
}

void parseTokens(FILE *fp)
{
  int c;
  while ((c = fgetc(fp)) != EOF)
  {
    if (c >= '0' && c <= '9')
    {
      ungetc(c, fp);
      char *str = parseString(fp);

      if (strcmp(str, "announce") == 0)
      {
        meta.announce = parseString(fp);
        info("Got announce: %s", meta.announce);
      }
      else if (strcmp(str, "created by") == 0)
      {
        meta.createdBy = parseString(fp);
        info("Got created by: %s", meta.createdBy);
      }
      else if (strcmp(str, "creation date") == 0)
      {
        c = fgetc(fp);
        meta.creationDate = parseInteger(fp, 'e');
        info("Got creation date: %lld", meta.creationDate);
      }
      else if (strcmp(str, "encoding") == 0)
      {
        meta.encoding = parseString(fp);
        info("Got encoding: %s", meta.encoding);
      }

      free(str);
    }
    else if (c == 'i')
    {
      uint64_t val = parseInteger(fp, 'e');
      info("Parsed integer: %llu", val);
    }
    else if (c == 'e')
    {
      info("End of list or dictionary.");
      break;
    }
    else if (c == 'd' || c == 'l')
    {
      info("Current position: %ld", ftell(fp));
      info("Parsing List or Dictionary: %c", c);
      parseTokens(fp);
    }
  }
}

uint64_t parseInteger(FILE *fp, char delimiter)
{
  uint64_t num = 0;
  int c;

  while ((c = fgetc(fp)) != EOF && c != delimiter)
  {
    if (c >= '0' && c <= '9')
    {
      num = num * 10 + (c - '0');
    }
    else
    {
      warn("Invalid character in integer field!");
      exit(-1);
    }
  }

  return num;
}

char *parseString(FILE *fp)
{
  uint64_t len = (uint64_t)parseInteger(fp, ':');

  char *buf = (char *)malloc(len + 1);
  if (buf == NULL)
  {
    warn("Malloc failed...");
    exit(-1);
  }

  size_t read = fread(buf, 1, len, fp);
  if (read != len)
  {
    warn("Failed to read expected string length.");
    free(buf);
    exit(-1);
  }

  buf[len] = '\0';
  return buf;
}
