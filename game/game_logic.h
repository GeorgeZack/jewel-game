#ifndef GAME_LOGIC_H
#define GAME_LOGIC_H

#include <raylib.h>
#include <string.h>

#define MAX_BATCH_ELEMENTS 8192
#define MAX_GEMS 100000
#define MAX_TEXTURES 7
#define TIMER_START 5000 / 60.0f

typedef struct Gem {
	Vector2 position;
	Vector2 speed;
	Color color;
	Color discoloration;
	Texture2D texture;
} Gem;

typedef struct DebugLine {
	const char *text;
	Color color;
} DebugLine;

typedef struct MessageLine {
	const char *text;
	Color color;
} MessageLine;

static Gem *gems = NULL;
static int gemCount = 0;
static Texture2D textures[MAX_TEXTURES];

#ifdef __cplusplus
extern "C" {
#endif

	void GameLoopInit();
	void GameLoopUpdate();
	void GameLoopRender();
	void GameLoopCleanup();

#ifdef __cplusplus
}
#endif

#endif

