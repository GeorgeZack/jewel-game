#include "game_logic.h"
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
	debugLines[debugLineCount].text = text;
	debugLines[debugLineCount].color = color;
	debugLineCount++;
}

void drawDebugLines() {
	Color bkg = BLACK;
	bkg.a = 180;
	DrawRectangle(0, 0, GetScreenWidth(), debugLineCount * fontSize, bkg);
	for (int i = 0; i < debugLineCount; i++) {
		DrawText(debugLines[i].text, 2, i * fontSize, fontSize, debugLines[i].color);
	}
	debugLineCount = 0;
}

void drawMessageLines() {
	int baseY = GetScreenHeight() - fontSize * (messageLineCount + 1);

	for (int i = 0; i < messageLineCount; i++) {
		// Outlines
		DrawText(messageLines[i].text, 1, baseY + i * fontSize - 1, fontSize, BLACK);
		DrawText(messageLines[i].text, 3, baseY + i * fontSize - 1, fontSize, BLACK);
		DrawText(messageLines[i].text, 1, baseY + i * fontSize + 1, fontSize, BLACK);
		DrawText(messageLines[i].text, 3, baseY + i * fontSize + 1, fontSize, BLACK);
		// Text
		DrawText(messageLines[i].text, 2, baseY + i * fontSize, fontSize, messageLines[i].color);
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
	messageLines[messageLineCount].text = strdup(text);
	messageLines[messageLineCount].color = color;
	messageLineCount++;
}

void InitUI() {
	initializeDebugLines();
	initializeMessageLines();
}

void DrawUI() {
	drawMessageLines();
	addDebugLine(strdup(TextFormat("FPS: %i", GetFPS())), DARKGREEN);
	addDebugLine(strdup(TextFormat("Objects: %i", gemCount)), GREEN);
	addDebugLine(strdup(TextFormat("Objects (Batched): %i", 1 + gemCount / MAX_BATCH_ELEMENTS)), MAROON);
	addDebugLine(strdup(TextFormat("X: %i", GetMouseX())), GREEN);
	addDebugLine(strdup(TextFormat("Y: %i", GetMouseY())), GREEN);
	addDebugLine(strdup(TextFormat("Test: %i", 2)), GREEN);
	drawDebugLines();
}

void CleanupUI() {
	for (int i = 0; i < debugLineCount; i++) {
		free(debugLines[i].text);
	}
	free(debugLines);
	free(messageLines);
}

