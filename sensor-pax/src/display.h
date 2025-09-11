#ifndef DISPLAY_H
#define DISPLAY_H

#include <Wire.h>               
#include "HT_SSD1306Wire.h"

// Umbenennung der Variable, um Konflikte zu vermeiden
extern SSD1306Wire myDisplay;

void VextON(void);
void VextOFF(void);
void displayMcuInit();
void displayWriter(String text);

#endif // DISPLAY_H
