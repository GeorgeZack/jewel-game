#include "game_logic.h"
#include <raylib.h>
#include <stdlib.h>
#include <stdio.h>
#include <string.h>

static Texture2D texBunny;
static Bunny *bunnies = NULL;
static int bunniesCount = 0;

void init() {
	printf("game_logic: init() called\n");
	texBunny = LoadTexture("resources/amethyst.png");
	bunnies = (Bunny *)RL_MALLOC(sizeof(Bunny) * MAX_BUNNIES);
	if (bunnies == NULL) {
		fprintf(stderr, "game_logic: Failed to allocate memory for bunnies!\n");
	} else {
		memset(bunnies, 0, sizeof(Bunny) * MAX_BUNNIES);
	}

	bunniesCount = 0;
}

void update(float delta) {
	if (bunnies == NULL) return;

	if (IsMouseButtonDown(MOUSE_BUTTON_LEFT)) {
		for (int i = 0; i < 100 && bunniesCount < MAX_BUNNIES; i++) {
			bunnies[bunniesCount].position = (Vector2){ GetMouseX() - texBunny.width/2, GetMouseY() - texBunny.height/2 };
			bunnies[bunniesCount].speed.x = (float)GetRandomValue(-250, 250)/60.0f;
			bunnies[bunniesCount].speed.y = (float)GetRandomValue(-250, 250)/60.0f;
			bunnies[bunniesCount].color = (Color){ GetRandomValue(20, 240), GetRandomValue(20, 240), GetRandomValue(20, 240), 255 };
			bunniesCount++;
		}
	}

	for (int i = 0; i < bunniesCount; i++) {
		bunnies[i].position.x += bunnies[i].speed.x;
		bunnies[i].position.y += bunnies[i].speed.y;

		if (((bunnies[i].position.x + texBunny.width/2) > GetScreenWidth()) ||
			((bunnies[i].position.x + texBunny.width/2) < 0)) bunnies[i].speed.x *= -1;
		if (((bunnies[i].position.y + texBunny.height/2) > GetScreenHeight()) ||
			((bunnies[i].position.y + texBunny.height/2) < 0)) bunnies[i].speed.y *= -1;
	}
}

void render() {
	if (bunnies != NULL && texBunny.id != 0) {
		for (int i = 0; i < bunniesCount; i++) {
			DrawTexture(texBunny, (int)bunnies[i].position.x, (int)bunnies[i].position.y, bunnies[i].color);
		}
	}
	DrawFPS(2, 2);
	DrawText(TextFormat("Objects: %i", bunniesCount), 2, 30, 20, GREEN);
	DrawText(TextFormat("Objects (Batched): %i", bunniesCount), 2, 50, 20, GREEN);
}

void cleanup() {
	printf("game_logic: cleanup() called\n");
	if (texBunny.id != 0) {
		UnloadTexture(texBunny);
		texBunny.id = 0;
	}

	if (bunnies != NULL) {
		RL_FREE(bunnies);
		bunnies = NULL;
	}

	bunniesCount = 0;
}

