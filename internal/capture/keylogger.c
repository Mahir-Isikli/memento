#include <CoreFoundation/CoreFoundation.h>
#include <CoreGraphics/CoreGraphics.h>
#include <Carbon/Carbon.h>
#include "keylogger.h"

typedef enum State {
    Up,
    Down,
    Invalid
} State;

static const UniCharCount MAX_STRING_LENGTH = 4;
static CFRunLoopRef runLoop = NULL;

extern void handleKeyEvent(int k, int ch, int s, bool ctrl, bool opt, bool shift, bool cmd);

static inline CGEventRef CGEventCallback(CGEventTapProxy proxy,
                                         CGEventType type,
                                         CGEventRef event,
                                         void *refcon) {
    State state;
    if (type == kCGEventKeyDown) {
        state = Down;
    } else if (type == kCGEventKeyUp) {
        state = Up;
    } else if (type == kCGEventFlagsChanged) {
        state = Invalid;
    } else {
        return event;
    }

    CGKeyCode keyCode = (CGKeyCode) CGEventGetIntegerValueField(event, kCGKeyboardEventKeycode);

    const CGEventFlags flags = CGEventGetFlags(event);
    bool ctrl = (flags & kCGEventFlagMaskControl) != 0;
    bool opt = (flags & kCGEventFlagMaskAlternate) != 0;
    bool shift = (flags & kCGEventFlagMaskShift) != 0;
    bool cmd = (flags & kCGEventFlagMaskCommand) != 0;

    UInt16 modifierKeyState = shift << 1 | ctrl << 2 | opt << 3 | cmd << 4;

    TISInputSourceRef currentKeyboard = TISCopyCurrentKeyboardLayoutInputSource();
    CFDataRef layoutData = TISGetInputSourceProperty(currentKeyboard, kTISPropertyUnicodeKeyLayoutData);
    
    int ch = 0;
    if (layoutData != NULL) {
        const UCKeyboardLayout *keyboardLayout = (UCKeyboardLayout *)CFDataGetBytePtr(layoutData);

        static UInt32 deadKeyState = 0;
        UniCharCount actualStringLength = 0;
        UniChar unicodeString[MAX_STRING_LENGTH];
        OSStatus status = UCKeyTranslate(keyboardLayout,
                                         keyCode,
                                         kUCKeyActionDisplay,
                                         modifierKeyState,
                                         LMGetKbdType(),
                                         kUCKeyTranslateNoDeadKeysBit,
                                         &deadKeyState,
                                         MAX_STRING_LENGTH,
                                         &actualStringLength,
                                         unicodeString);
        if (status == noErr && actualStringLength > 0) {
            ch = (int)unicodeString[0];
        }
    }
    
    if (currentKeyboard != NULL) {
        CFRelease(currentKeyboard);
    }

    handleKeyEvent((int)keyCode, ch, (int)state, ctrl, opt, shift, cmd);

    return event;
}

static inline void startKeylogger(void) {
    CGEventMask eventMask = CGEventMaskBit(kCGEventKeyDown) | CGEventMaskBit(kCGEventKeyUp);

    CFMachPortRef eventTap = CGEventTapCreate(kCGSessionEventTap,
                                              kCGHeadInsertEventTap,
                                              kCGEventTapOptionListenOnly,
                                              eventMask,
                                              CGEventCallback,
                                              NULL);

    if (!eventTap) {
        fprintf(stderr, "ERROR: Unable to create event tap. Check Accessibility permissions.\n");
        return;
    }

    CFRunLoopSourceRef runLoopSource = CFMachPortCreateRunLoopSource(kCFAllocatorDefault, eventTap, 0);
    runLoop = CFRunLoopGetCurrent();
    CFRunLoopAddSource(runLoop, runLoopSource, kCFRunLoopCommonModes);
    CGEventTapEnable(eventTap, true);

    CFRunLoopRun();
}

static inline void stopKeylogger(void) {
    if (runLoop != NULL) {
        CFRunLoopStop(runLoop);
        runLoop = NULL;
    }
}
