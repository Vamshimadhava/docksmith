#!/usr/bin/env python3
import os

def main():
    msg = os.environ.get('MSG', 'Hello')
    print(f"{msg} from Docksmith Python app!")
    print("Application is running successfully.")

if __name__ == "__main__":
    main()
