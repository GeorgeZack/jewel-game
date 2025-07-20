#ifndef GAME_LOGIC_H
#define GAME_LOGIC_H

#include <raylib.h>

#define MAX_BUNNIES 50000
#define MAX_BATCH_ELEMENTS 8192

typedef struct Bunny {
	Vector2 position;
	Vector2 speed;
	Color color;
} Bunny;

#ifdef __cplusplus
extern "C" {
#endif

	void init();
	void update(float delta);
	void render();
	void cleanup();

#ifdef __cplusplus
}
#endif

#endif

