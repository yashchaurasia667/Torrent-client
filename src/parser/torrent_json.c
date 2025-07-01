#include "parser.h"

static const char *safe_str(const char *s)
{
  return s ? s : "";
}

void dumpToJson(FILE *out, Torrent *t)
{
  fprintf(out, "{\n");

  fprintf(out, "  \"announce\": \"%s\",\n", safe_str(t->announce));
  fprintf(out, "  \"createdBy\": \"%s\",\n", safe_str(t->createdBy));
  fprintf(out, "  \"creationDate\": %lld,\n", t->creationDate);
  fprintf(out, "  \"encoding\": \"%s\",\n", safe_str(t->encoding));
  fprintf(out, "  \"comment\": \"%s\",\n", safe_str(t->comment));

  fprintf(out, "  \"info\": {\n");
  fprintf(out, "    \"name\": \"%s\",\n", safe_str(t->info.name));
  fprintf(out, "    \"pieceLength\": %llu,\n", t->info.pieceLength);
  fprintf(out, "    \"pieceCount\": %zu,\n", t->info.pieceCount);

  if (t->hasMultipleFiles)
  {
    fprintf(out, "    \"files\": [\n");
    for (size_t i = 0; i < t->info.fileCount; i++)
    {
      File f = t->info.files[i];
      fprintf(out, "      {\n");
      fprintf(out, "        \"length\": %llu,\n", f.length);
      fprintf(out, "        \"path\": [");
      for (int j = 0; f.path[j]; j++)
      {
        fprintf(out, "\"%s\"", f.path[j]);
        if (f.path[j + 1])
          fprintf(out, ", ");
      }
      fprintf(out, "]\n");
      fprintf(out, "      }%s\n", i + 1 == t->info.fileCount ? "" : ",");
    }
    fprintf(out, "    ]\n");
  }
  else
  {
    fprintf(out, "    \"length\": %llu\n", t->info.length);
  }

  fprintf(out, "  },\n");

  fprintf(out, "  \"announceList\": [");
  for (uint32_t i = 0; i < t->announceUrlCount; i++)
  {
    fprintf(out, "\"%s\"", t->announceList[i]);
    if (i + 1 < t->announceUrlCount)
      fprintf(out, ", ");
  }
  fprintf(out, "]\n");

  fprintf(out, "}\n");
}