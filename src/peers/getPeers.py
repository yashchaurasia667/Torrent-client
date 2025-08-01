import os
import socket
import argparse
import hashlib
from termcolor import cprint

def setupArgparse():
  parser = argparse.ArgumentParser()
  parser.add_argument("file_path")
  args = parser.parse_args()
  return args

def getInfo(path=""):
  meta = {}
  if not os.path.exists(path):
    print("Invalid path")
    exit(1)

  content = ""
  with open(path, "rb") as f:
    content = f.read()

  if not content:
    print("Invalid Torrent file")
    exit(1)

  info = content.split(b"4:info")[1]
  meta["info_hash"] = hashlib.sha1(info).hexdigest()

  announce = content.split(b"8:announce", 1)[1]
  announce_len = int(announce.split(b":", 1)[0])
  meta["announce"] = announce.split(b":", 1)[1][:announce_len]

  return meta


if __name__ == "__main__":
  args = setupArgparse()
  # print(args.file_path)
  meta = getInfo(args.file_path)
  soc = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
  