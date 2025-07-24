#include "gameLoop.h"
#include <raylib.h>
#include <stdlib.h>
#include <stdio.h>

DebugLine *debugLines = NULL;
int debugLineCount = 0;
int debugLineCapacity = 0;
int fontSize = 20;
MessageLine *messageLines = NULL;
int messageLineCount = 0;
int messageLineCapacity = 0;

void addDebugLine(const char *text, Color color) {
	if (debugLineCount >= debugLineCapacity) {
		debugLineCapacity *= 2;
		debugLines = realloc(debugLines, debugLineCapacity * sizeof(DebugLine));
	}
	strncpy(debugLines[debugLineCount].text, text, DEBUG_LINE_TEXT_MAX - 1);
	debugLines[debugLineCount].text[DEBUG_LINE_TEXT_MAX - 1] = '\0';
	debugLines[debugLineCount].color = color;
	debugLineCount++;
}

void drawDebugLines() {
	Color bkg = BLACK;
	bkg.a = 180;
	DrawRectangle(0, 0, GetScreenWidth(), (debugLineCount * fontSize) + 2, bkg);
	for (int i = 0; i < debugLineCount; i++) {
		DrawText(debugLines[i].text, 2, i * fontSize, fontSize, debugLines[i].color);
	}
}

void drawMessageLines() {
	int baseY = GetScreenHeight() - fontSize * messageLineCount;

	for (int i = 0; i < messageLineCount; i++) {
		int debugHeight = debugLineCount * fontSize;
		if (baseY + (i * fontSize) < debugHeight) {
			continue;
		}
		// Outlines
		DrawText(messageLines[i].text, 1, baseY + (i * fontSize - 1), fontSize, BLACK);
		DrawText(messageLines[i].text, 3, baseY + (i * fontSize - 1), fontSize, BLACK);
		DrawText(messageLines[i].text, 1, baseY + (i * fontSize + 1), fontSize, BLACK);
		DrawText(messageLines[i].text, 3, baseY + (i * fontSize + 1), fontSize, BLACK);
		// Text
		DrawText(messageLines[i].text, 2, baseY + (i * fontSize), fontSize, messageLines[i].color);
	}
}

void initializeDebugLines() {
	debugLineCapacity = 8;
	debugLines = malloc(debugLineCapacity * sizeof(DebugLine));
	debugLineCount = 0;
}

void initializeMessageLines() {
	messageLineCapacity = 8;
	messageLines = malloc(messageLineCapacity * sizeof(MessageLine));
	messageLineCount = 0;
}

void AddMessageLine(const char *text, Color color) {
	if (messageLineCount >= messageLineCapacity) {
		messageLineCapacity *= 2;
		messageLines = realloc(messageLines, messageLineCapacity * sizeof(MessageLine));
	}
	strncpy(messageLines[messageLineCount].text, text, MESSAGE_LINE_TEXT_MAX - 1);
	messageLines[messageLineCount].text[MESSAGE_LINE_TEXT_MAX - 1] = '\0';
	messageLines[messageLineCount].color = color;
	messageLineCount++;
}

void InitUI() {
	initializeDebugLines();
	initializeMessageLines();
}

void DrawUI() {
	debugLineCount = 0;
	addDebugLine(TextFormat("FPS: %i", GetFPS()), DARKGREEN);
	addDebugLine(TextFormat("Objects: %i", gemCount), GREEN);
	addDebugLine(TextFormat("Objects (Batched): %i", 1 + gemCount / MAX_BATCH_ELEMENTS), MAROON);
	addDebugLine(TextFormat("X: %i", GetMouseX()), GREEN);
	addDebugLine(TextFormat("Y: %i", GetMouseY()), GREEN);
	addDebugLine(TextFormat("baseY: %i", GetScreenHeight() - fontSize * messageLineCount), GREEN);
	addDebugLine(TextFormat("Debug Lines: %i (%ipx)", debugLineCount + 1, (debugLineCount + 1) * fontSize), GREEN);
	drawMessageLines();
	drawDebugLines();
}

void CleanupUI() {
	for (int i = 0; i < debugLineCount; i++) {
		free(debugLines[i].text);
	}
	free(debugLines);
	free(messageLines);
}

