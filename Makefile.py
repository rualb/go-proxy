import os
import shutil
import subprocess
import sys

# BINARY_NAME = "app.exe" if os.name == "nt" else "app"
# local release  goreleaser build --snapshot


"""
git init
git add .
git commit -m "-"
git tag "$(cat VERSION)"
git tag (Get-Content VERSION)
"""


AppName = "go-proxy"

def test():
    print("Testing...")
    env = os.environ.copy()
    command = ['go', 'test'
               #, '-race'
               , '-timeout=60s', '-count=1', './...']
    subprocess.run(command, env=env) #, "-v"

def help():
    print("Usage:")
    print("  python build.py test     - Run test")
    print("  python build.py help     - Display this help message")
    
def build():
    print("Building the binary...")
    subprocess.run(["go", "build",  "-C", f"cmd/{AppName}", "-o",f"./../../dist/", "-ldflags", "-s -w", ])
 
def run():
    print("Building the binary...")
    subprocess.run([f"dist/{AppName}", "-config", f"./configs"])
def lint():
    print("Linter...")
    subprocess.run(["golangci-lint ", "run"])
def check():
    lint()
    test()
 
if len(sys.argv) > 1:
    command = sys.argv[1]
    if command == "test":
        test() 
    elif command == "help":
        help() 
    elif command == "build":
        build() 
    elif command == "run":
        run() 
    elif command == "lint":
        lint() 
    elif command == "check":
        check() 
    else:
        help()
        exit(1)
else:
    help()









