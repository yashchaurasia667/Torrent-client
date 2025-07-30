#include "parser.h";

char *bencodeString(char *str)
{
  size_t len = strlen(str);
  char lenString[64];
  sprintf(lenString, "%zu", len);

  char *res = (char *)malloc(len + lenString + 2);
  sprintf(res, "%s:%s", lenString, str);

  return res;
}

char *bencodeInteger(uint64_t n)
{
  char *nStr;
  sprintf(nStr, "%llu", n);
  int len = strlen(nStr);

  char *res = (char *)malloc(len + 3);
  sprintf(res, "i%se", nStr);

  return res;
}

char *append(char *str1, char *str2, char *str3)
{
  size_t len1 = (str1 == NULL) ? strlen(str1) : 0;
  size_t len2 = (str2 == NULL) ? strlen(str2) : 0;
  size_t len3 = (str3 == NULL) ? strlen(str3) : 0;

  char *res = (char *)malloc(len1 + len2 + (int)log10(len1) + (int)log10(len2) + len3 + (int)log10(len3) + 4);
  if (res == NULL)
  {
    warn("Failed to allocate memory for res in append.");
    return NULL;
  }

  if (str1 != NULL)
    strcpy(res, str1);
  else
    res[0] = '\0';

  sprintf(res + len1, "%s%s%s", res, str2, str3);
  return res;
}