import os
import hashlib

def getInfoHash(path=""):
  if not os.path.exists(path):
    print("Invalid path")
    exit(1)

  content = ""
  with open(path, "rb") as f:
    content = f.read()
  
  info = content.split(b"4:info")[1]
  hash = hashlib.sha1(info)
  return hash


if __name__ == "__main__":
  path = input("Enter filePath: ")
  print(getInfoHash(path))