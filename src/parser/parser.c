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

#include "parser.h"

Torrent meta = {0};

char *append(char *dest, const char *newStr);
char *parseString(FILE *fp);
uint64_t parseInteger(FILE *fp, char delimiter);
void parseTokens(FILE *fp);
void parseAnnounceList(FILE *fp);
void parseFiles(FILE *fp);
File parseFile(FILE *fp);
void parsePieces(FILE *fp);
void freeTorrent(Torrent *torrent);
uint32_t convertToMB(uint64_t byteSize);
void displayInformation();

int main(int argc, char **argv)
{
  if (argc < 2)
  {
    warn("A file name is needed...");
    return -1;
  }

  // info("Trying to open %s", argv[1]);
  FILE *fp = fopen(argv[1], "rb");
  if (!fp)
  {
    warn("Could not find %s", argv[1]);
    return -1;
  }

  parseTokens(fp);
  fclose(fp);

  FILE *json = fopen("./src/peers/meta.json", "w");
  if (!json)
  {
    warn("Failed to open meta.json for writing");
    return -1;
  }
  dumpToJson(json, &meta);
  fclose(json);
  okay("Dumped parsed torrent to meta.json");

  char ch = 'N';
  printf("Do you want to display the parsed information? [y/N] ");
  scanf("%c", &ch);

  if (ch == 'Y' || ch == 'y')
    displayInformation();

  freeTorrent(&meta);

  return 0;
}

void parseTokens(FILE *fp)
{
  int c;
  char *info = "d";

  while ((c = fgetc(fp)) != EOF && c != 'e')
  {
    if (c >= '0' && c <= '9')
    {
      ungetc(c, fp);
      char *str = parseString(fp);

      if (strcmp(str, "announce") == 0)
      {
        meta.announce = parseString(fp);
        // info("Announce: %s", meta.announce);
      }
      else if (strcmp(str, "announce-list") == 0)
      {
        parseAnnounceList(fp);
        // info("Announce List parsed.");
      }
      else if (strcmp(str, "created by") == 0)
      {
        meta.createdBy = parseString(fp);
        // info("Created by: %s", meta.createdBy);
      }
      else if (strcmp(str, "creation date") == 0)
      {
        fgetc(fp);
        meta.creationDate = parseInteger(fp, 'e');
        // info("Creation date: %llu", meta.creationDate);
      }
      else if (strcmp(str, "encoding") == 0)
      {
        meta.encoding = parseString(fp);
        // info("Encoding: %s", meta.encoding);
      }
      else if (strcmp(str, "comment") == 0)
      {
        meta.comment = parseString(fp);
      }

      else if (strcmp(str, "length") == 0)
      {
        char *len = bencodeString(str);

        fgetc(fp);
        meta.info.length = parseInteger(fp, 'e');
        meta.hasMultipleFiles = false;
        // info("Length: %llu", meta.info.length);
      }
      else if (strcmp(str, "name") == 0)
      {
        meta.info.name = parseString(fp);
        // info("Name: %s", meta.info.name);
      }
      else if (strcmp(str, "piece length") == 0)
      {
        fgetc(fp);
        meta.info.pieceLength = parseInteger(fp, 'e');
        // info("Piece length: %llu", meta.info.pieceLength);
      }
      else if (strcmp(str, "pieces") == 0)
      {
        parsePieces(fp);
        // info("Parsed pieces.");
      }
      else if (strcmp(str, "files") == 0)
      {
        meta.hasMultipleFiles = true;
        parseFiles(fp);
        // info("Parsed files.");
      }

      free(str);
    }
    else if (c == 'i')
    {
      parseInteger(fp, 'e');
    }
    else if (c == 'd' || c == 'l')
    {
      parseTokens(fp);
    }
  }
}

void parsePieces(FILE *fp)
{
  uint64_t len = parseInteger(fp, ':');
  meta.info.pieces = malloc(len);
  if (!meta.info.pieces)
  {
    warn("Failed to allocate memory for pieces");
    exit(EXIT_FAILURE);
  }

  fread(meta.info.pieces, 1, len, fp);
  meta.info.pieceCount = len / 20;
}

char *parseString(FILE *fp)
{
  uint64_t len = parseInteger(fp, ':');

  char *buf = malloc(len + 1);
  if (!buf)
  {
    warn("Malloc failed...");
    exit(EXIT_FAILURE);
  }

  if (fread(buf, 1, len, fp) != len)
  {
    warn("Failed to read expected string length.");
    free(buf);
    exit(EXIT_FAILURE);
  }

  buf[len] = '\0';
  return buf;
}

uint64_t parseInteger(FILE *fp, char delimiter)
{
  uint64_t num = 0;
  int c;
  while ((c = fgetc(fp)) != EOF && c != delimiter)
  {
    if (c >= '0' && c <= '9')
      num = num * 10 + (c - '0');
    else
    {
      warn("Invalid character in integer field!");
      exit(EXIT_FAILURE);
    }
  }
  return num;
}

void parseAnnounceList(FILE *fp)
{
  if (fgetc(fp) != 'l')
  {
    warn("Malformed announce-list");
    return;
  }

  char **urls = NULL;
  size_t urlCount = 0;

  int c;
  while ((c = fgetc(fp)) != EOF && c != 'e')
  {
    if (c != 'l')
    {
      warn("Expected nested list in announce-list");
      break;
    }

    while ((c = fgetc(fp)) != EOF && c != 'e')
    {
      ungetc(c, fp);
      char *url = parseString(fp);
      char **temp = realloc(urls, sizeof(char *) * (urlCount + 1));
      if (!temp)
      {
        warn("Failed to realloc announce list");
        exit(EXIT_FAILURE);
      }
      urls = temp;
      urls[urlCount++] = url;
    }
  }

  meta.announceUrlCount = urlCount;
  meta.announceList = urls;
}

File parseFile(FILE *fp)
{
  File file = {0};
  size_t path_count = 0;

  if (fgetc(fp) != 'd')
  {
    warn("Invalid file format");
    exit(EXIT_FAILURE);
  }

  int c;
  while ((c = fgetc(fp)) != EOF && c != 'e')
  {
    ungetc(c, fp);
    char *key = parseString(fp);

    if (strcmp(key, "length") == 0)
    {
      fgetc(fp);
      file.length = parseInteger(fp, 'e');
    }
    else if (strcmp(key, "path") == 0)
    {
      if (fgetc(fp) != 'l')
      {
        warn("Invalid path format.");
        exit(EXIT_FAILURE);
      }

      while ((c = fgetc(fp)) != EOF && c != 'e')
      {
        ungetc(c, fp);
        char *pathPart = parseString(fp);

        char **temp = realloc(file.path, sizeof(char *) * (path_count + 1));
        if (!temp)
        {
          warn("Memory allocation failed for path");
          exit(EXIT_FAILURE);
        }

        file.path = temp;
        file.path[path_count++] = pathPart;
      }

      file.path = realloc(file.path, sizeof(char *) * (path_count + 1));
      file.path[path_count] = NULL;
    }

    free(key);
  }

  return file;
}

void parseFiles(FILE *fp)
{
  if (fgetc(fp) != 'l')
  {
    warn("Invalid files format.");
    exit(EXIT_FAILURE);
  }

  meta.info.files = NULL;
  meta.info.fileCount = 0;

  int c;
  while ((c = fgetc(fp)) != EOF && c != 'e')
  {
    ungetc(c, fp);
    File f = parseFile(fp);

    File *temp = realloc(meta.info.files, sizeof(File) * (meta.info.fileCount + 1));
    if (!temp)
    {
      warn("Failed to realloc memory for files.");
      exit(EXIT_FAILURE);
    }

    meta.info.files = temp;
    meta.info.files[meta.info.fileCount++] = f;
  }
}

void freeTorrent(Torrent *torrent)
{
  free(torrent->announce);
  free(torrent->createdBy);
  free(torrent->encoding);
  free(torrent->info.name);
  free(torrent->info.pieces);

  if (torrent->announceList)
  {
    // for (size_t i = 0; torrent->announceList[i]; ++i)
    //   free(torrent->announceList[i]);
    free(torrent->announceList);
  }

  if (torrent->info.files)
  {
    for (size_t i = 0; i < torrent->info.fileCount; ++i)
    {
      for (size_t j = 0; torrent->info.files[i].path && torrent->info.files[i].path[j]; ++j)
        free(torrent->info.files[i].path[j]);
      free(torrent->info.files[i].path);
    }
    free(torrent->info.files);
  }
}

uint32_t convertToMB(uint64_t byteSize)
{
  return byteSize / (1024 * 1024);
}

void displayInformation()
{
  while (true)
  {
    printf("\n");
    printf("Print info about the torrent file: \n");
    printf("1. Type \n");
    printf("2. Announce \n");
    printf("3. Created by \n");
    printf("4. Creation date \n");
    printf("5. Encoding \n");
    printf("6. Name \n");
    printf("7. Piece Length \n");
    printf("8. Piece Count \n");
    printf("9. Pieces \n");
    printf("10. Announce List \n");

    if (meta.hasMultipleFiles)
    {
      printf("11. Files \n");
      printf("12. File Count \n");
    }
    else
    {
      printf("11. Length \n");
    }

    int choice = 0;
    printf("\nEnter your choice: ");
    scanf("%d", &choice);
    printf("\n");

    switch (choice)
    {
    case 1:
      printf("Type: ");
      if (meta.hasMultipleFiles)
        printf("Multiple file torrent \n");
      else
        printf("Single file torrent \n");
      break;
    case 2:
      printf("Announce: %s \n", meta.announce);
      break;
    case 3:
      printf("Created by: %s \n", meta.createdBy);
      break;
    case 4:
      // printf("Creation Date: %llu \n", meta.creationDate);
      struct tm *tm_info;
      tm_info = gmtime(&meta.creationDate);

      char buf[11];
      strftime(buf, sizeof(buf), "%d/%m/%y", tm_info);
      printf("Creation date: %s \n", buf);
      break;
    case 5:
      printf("Encoding: %s \n", meta.encoding);
      break;
    case 6:
      printf("Name: %s \n", meta.info.name);
      break;
    case 7:
      printf("Piece Length: %u MB \n", convertToMB(meta.info.pieceLength));
      break;
    case 8:
      printf("Piece Count: %llu \n", meta.info.pieceCount);
      break;
    case 9:
      printf("Pieces: %s \n", meta.info.pieces);
      break;
    case 10:
      if (meta.announceList)
      {
        printf("Announce List: \n");
        for (uint32_t i = 0; i < meta.announceUrlCount; i++)
          printf("%s \n", meta.announceList[i]);
        printf("\n");
      }
      else
      {
        printf("No Announce List found in this Torrent File. \n");
      }
      break;
    case 11:
      if (meta.hasMultipleFiles)
      {
        printf("Files: \n");
        for (size_t i = 0; i < meta.info.fileCount; i++)
        {
          printf("length: %llu", meta.info.files[i].length);
          printf("path: ");
          for (int j = 0; meta.info.files->path[j]; j++)
            printf("/%s", meta.info.files->path[j]);
          printf("\n");
        }
        printf("\n");
      }
      else
      {
        printf("Length: %u MB \n", convertToMB(meta.info.length));
      }
      break;
    case 12:
      if (meta.hasMultipleFiles)
        printf("File Count: %zu", meta.info.fileCount);
      break;

    default:
      printf("Please choose one of the above options... \n");
      break;
    }
  }
}