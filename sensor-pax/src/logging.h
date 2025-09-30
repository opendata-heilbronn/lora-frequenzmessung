#ifndef LOGGING_H
#define LOGGING_H

#include <Arduino.h>
#include "customs.h"

// --- Logging Functions ---
// Conditional logging based on ENABLE_LOGGING flag in customs.h
inline void logMessage(const String& msg) {
#if ENABLE_LOGGING
  Serial.println(msg);
#endif
}

inline void logMessage(const char* msg) {
#if ENABLE_LOGGING
  Serial.println(msg);
#endif
}

inline void logMessageF(const char* format, ...) {
#if ENABLE_LOGGING
  char buffer[256];
  va_list args;
  va_start(args, format);
  vsnprintf(buffer, sizeof(buffer), format, args);
  va_end(args);
  Serial.println(buffer);
#endif
}
// --- End Logging Functions ---

#endif // LOGGING_H
