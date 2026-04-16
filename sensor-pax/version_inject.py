Import("env")
import os

version = os.environ.get("FIRMWARE_VERSION", "dev")
# Use SCons CPPDEFINES API to correctly inject a string literal define.
# This avoids shell quoting problems with -D flags containing double quotes.
env.Append(CPPDEFINES=[("FIRMWARE_VERSION", '\\"' + version + '\\"')])
