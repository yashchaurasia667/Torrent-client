#include <stdio.h>
#include <stdlib.h>

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

char c;

typedef struct
{
  char *announce;
  char *createdBy;
  long creationDate;
  char *encoding;
  char *info;
} torrent;

char *parseString(FILE *fp);
long int parseInteger(FILE *fp);
char *getFileContent(FILE *fp);
long reverseNumber(long num);
void parseTokens(FILE *fp);

int main(int argc, char **argv)
{
  if (argc == 1)
  {
    warn("A file name is needed...");
    return -1;
  }

  info("Trying to open %s", argv[1]);
  FILE *fp = fopen(argv[1], "r");

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

long reverseNumber(long num)
{
  int reversed = 0;
  while (num != 0)
  {
    int digit = num % 10;
    reversed = reversed * 10 + digit;
    num /= 10;
  }
  return reversed;
}

void parseTokens(FILE *fp)
{
  while ((c = fgetc(fp)) != EOF)
  {
    if (c >= 0x30 && c <= 0x39)
      parseString(fp);
    else if (c == 'i')
      parseInteger(fp);
    else if (c == 'e')
      break;
    else
    {
      info("Skipping non-digit char: %c", c);
      continue;
      // parseTokens(fp);
    }
  }
}

long int parseInteger(FILE *fp)
{
  long int num = 0;
  while ((c = getc(fp)) != EOF && c != 'e')
  {
    if (c >= '0' && c <= '9')
      num = (num * 10) + (c - '0');
  }

  info("Parsed integer: %ld", num);
  return num;
}

char *parseString(FILE *fp)
{
  // info("Parsing a String...\n");
  int len = 0;
  ungetc(c, fp);

  while ((c = fgetc(fp)) != EOF && c != ':')
  {
    if (c >= '0' && c <= '9')
      len = (len * 10) + (c - '0');
    else
    {
      warn("The given Torrent file is correpted!!\n");
      exit(-1);
    }
  }

  len += 1;
  info("Got string length: %d", len - 1);

  char *buf = (char *)malloc(len + 1);
  if (buf == NULL)
  {
    warn("Malloc failed...\n");
    exit(-1);
  }

  if (fgets(buf, len, fp) == NULL)
  {
    warn("Failed to read string...");
    exit(-1);
  }
  info("Got string: %s\n", buf);

  return buf;
}