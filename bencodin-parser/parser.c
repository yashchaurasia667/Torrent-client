#include <stdio.h>

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
  char *announce;
  char *createdBy;
  long creationDate;
  char *encoding;
  char *info;
} torrent;

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

  return 0;
}