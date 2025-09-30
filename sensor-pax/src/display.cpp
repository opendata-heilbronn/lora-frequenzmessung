#include "display.h"

// Umbenannte Variable - jetzt myDisplay statt display
SSD1306Wire myDisplay(0x3c, 500000, SDA_OLED, SCL_OLED, GEOMETRY_128_64, RST_OLED);

void VextON(void)
{
  pinMode(Vext,OUTPUT);
  digitalWrite(Vext, LOW);
}

void VextOFF(void) //Vext default OFF
{
  pinMode(Vext,OUTPUT);
  digitalWrite(Vext, HIGH);
}

void displayMcuInit()
{
    VextON();
    digitalWrite(Vext,LOW);
    myDisplay.init();  // display -> myDisplay
    myDisplay.setFont(ArialMT_Plain_16);
    myDisplay.setTextAlignment(TEXT_ALIGN_CENTER);
    myDisplay.clear();
    myDisplay.drawString(myDisplay.getWidth()/2, myDisplay.getHeight()/2-10, "LFM CFHN");
    myDisplay.drawString(myDisplay.getWidth()/2, myDisplay.getHeight()/2+5, "STARTING");
    myDisplay.display();
    delay(500); // Reduced from 2000ms to 500ms for power efficiency
}

void displayWriter(String text)
{
    myDisplay.clear();  // display -> myDisplay
    myDisplay.drawString(0, 26, text);
    myDisplay.display(); // Hinzufügung: display() aufrufen, um Text anzuzeigen
}